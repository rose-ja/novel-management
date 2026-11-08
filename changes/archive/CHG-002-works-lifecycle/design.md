# Design — CHG-002 作品生命周期与回收站/快照基座

## 1. 现状

- 后端只有 `config`、`database`、`httpapi`、`migrations`、`observability` 五个包；路由仅健康检查、`/meta`、`/settings`、静态托管。
- 数据库只有 `goose_db_version` + `pg_trgm`（`0001_init.sql`）。
- `/settings` 的 `current_work_id`、`last_export_at`、`last_restore_at` 恒为 `null`。
- 前端顶栏作品位是只读占位；导航只有 `/` 与 `/settings`。
- 无 service/repository 分层，无乐观锁实现，无事务边界。

## 2. 方案总览

沿用「handler → service → repository」三层，先建立最小可复用的领域骨架，后续变更在同一骨架上加实体。

```
cmd/server → httpapi(router)
                 ├─ works.go     ─┐
                 ├─ trash.go      ├→ service/{works,trash,settings,snapshot}.go
                 └─ snapshots.go ─┘        │
                                           ├→ repository/{works,trash,settings,snapshot}.go
                                           └→ snapshot.Service（文件产出）
                                                       └→ pg_dump / 文件系统
```

## 3. 数据模型与迁移 `0002_works_trash_snapshot.sql`

### 3.1 `works`

| 列 | 类型 | 约束 |
|---|---|---|
| id | TEXT(26) | PK（ULID） |
| name | TEXT | NOT NULL，1–200（应用层校验） |
| name_key | TEXT | NOT NULL；部分唯一索引 `works_name_key_uq ON works(name_key) WHERE deleted_at IS NULL` |
| description | TEXT | NULL，≤20,000 |
| created_at / updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| version | INTEGER | NOT NULL DEFAULT 1，CHECK ≥1 |
| deleted_at / deleted_root_id / deleted_by_action | TIMESTAMPTZ / TEXT / TEXT | 软删除三件套，CHECK 同步（有 deleted_at 必须有 root_id） |

- `name_key` 规范化：`lower(trim(name))`（中文不折叠全半角，保持与检索一致；后续如需扩展在 CHG-009 统一）。

### 3.2 `app_settings`（单行表）

| 列 | 类型 | 说明 |
|---|---|---|
| id | SMALLINT | PK，CHECK id=1 |
| current_work_id | TEXT(26) | FK → works.id `ON DELETE SET NULL`（作品恢复/永久删除后自动置空） |
| last_export_at / last_restore_at | TIMESTAMPTZ | NULL |
| snapshot_keep | SMALLINT | NOT NULL DEFAULT 3（只读展示，页面不可改） |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

初始化 `INSERT ... (1, NULL)`，由迁移完成。

### 3.3 `snapshots`

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT(26) | PK |
| reason | TEXT | `work_delete` / `trash_purge` / `purge_all` / `import` / `cli_restore` |
| path | TEXT | `/data/snapshots/{ts}-{reason}` |
| db_bytes / image_count | BIGINT / INTEGER | 展示用 |
| status | TEXT | `ok` / `failed` |
| created_at | TIMESTAMPTZ | 倒序索引 |

### 3.4 `undo_tokens`

| 列 | 类型 | 说明 |
|---|---|---|
| token | TEXT(32) | PK（随机 hex） |
| root_type / root_id | TEXT / TEXT(26) | 被删根 |
| created_at / expires_at | TIMESTAMPTZ | expires = created + 8s |
| used_at | TIMESTAMPTZ | 一次性 |

清理：撤销成功或过期后不主动删，`expires_at` 索引 + 每次 undo 顺带删除 1 天前的记录（批量 DELETE，无定时任务）。

## 4. API 映射

| 端点 | service | 关键行为 |
|---|---|---|
| `GET /works` | `works.List` | 未删除 + `q` 模糊（`name_key LIKE %q%`），分页 |
| `POST /works` | `works.Create` | 事务：插入作品 + 内置「境界」类别（`system_categories` 表在本变更先建最小结构，只存 id/work_id/name/is_builtin/is_enabled/category_type） |
| `GET /works/{id}` | `works.Get` | 不存在或已删 → 404 NOT_FOUND |
| `PATCH /works/{id}` | `works.Update` | `If-Match` 必填；冲突 409 VERSION_CONFLICT；改名冲突 409 NAME_CONFLICT |
| `DELETE /works/{id}` | `works.Delete` | 校验 `confirm_name`；快照 → 软删除整组 → 写 undo_tokens |
| `GET /trash` | `trash.List` | 只返回根记录；本变更 `root_type=work` 有数据，其余为空列表 |
| `POST /trash/{type}/{id}/restore` | `trash.Restore` | 整组原子；名称冲突 422 RESTORE_CONFLICT（details.items 列出冲突项） |
| `DELETE /trash/{type}/{id}` | `trash.Purge` | 快照 `trash_purge` → 物理删除 |
| `DELETE /trash` | `trash.PurgeAll` | `confirm=PURGE_ALL`；快照 `purge_all` |
| `POST /trash/undo` | `trash.Undo` | 8 秒窗口；过期 `UNDO_EXPIRED`，重复 `ALREADY_DELETED` |
| `GET /snapshots` | `snapshot.List` | 倒序，最多 10 条 |
| `GET /settings` | `settings.Get` | 读取 `app_settings` + 配置项 |

### 4.1 `If-Match` 解析

`If-Match: "3"` → 去引号转 int；缺失、非数字、与当前 `version` 不等 → `409 VERSION_CONFLICT`。删除作品同样需要 `If-Match`。

### 4.2 删除事务边界

```
snapshot.Create(reason="work_delete")   // 失败 → 503 SNAPSHOT_FAILED，不进入删除
tx: update works set deleted_at/deleted_root_id/deleted_by_action, version=version+1
    update system_categories ... (同一 root)
    insert undo_tokens
    update app_settings.current_work_id = null（若指向被删作品）
commit
```
快照在事务外先行，避免长时间锁表；快照失败直接返回。

## 5. 快照实现

- 目录：`{DATA_DIR}/snapshots/{YYYYMMDDTHHMMSSZ}-{reason}/`
- 产出：`app.dump`（`pg_dump --format=custom`，经 `DATABASE_URL` 的 host/port/user/pass/db 参数化）、`images/`（拷贝 `{DATA_DIR}/images`，本变更目录可能为空）、`snapshot.json`（id/reason/created_at/entity_counts）、`manifest.json`（文件清单+大小）、`checksums.sha256`（`sha256sum` 等价的 Go 实现）。
- 保留：生成成功后按 `created_at DESC` 保留 3 份，超出删除目录与记录。
- 失败判定：`pg_dump` 非零退出、目录不可写、校验和生成失败 → 整体失败并写 `status=failed` 记录（不阻断后续重试）。
- 镜像：运行阶段 `apk add --no-cache postgresql16-client`；启动时记录 `pg_dump` 版本到日志（不新增健康字段，避免改契约）。

## 6. 前端

| 组件/页面 | 内容 |
|---|---|
| `components/shell/work-switcher.tsx` | 顶栏下拉：作品列表（含「新建作品」）、当前项打勾；键盘可达；列表加载失败显示重试 |
| `components/works/work-dialogs.tsx` | 新建/重命名 Dialog（名称必填 ≤200，FormField 错误）、删除 Dialog（输入作品名一致才启用确认按钮） |
| `components/works/undo-toast.tsx` | 删除成功后 8 秒 Toast +「撤销」，超时自动消失；失败显示错误 Toast |
| `app/trash/page.tsx` | 作品筛选（默认当前作品）、列表（名称/删除时间/影响条数）、恢复/永久删除/清空；空状态 |
| `components/settings/snapshot-list.tsx` | 设置页快照卡片：时间、原因、大小、状态；空状态说明「暂无快照」 |
| `lib/api/{works,trash,snapshots}.ts` | 类型化封装，复用 `fetchJson`；写请求带 `If-Match` 与 `X-Requested-With` |
| `components/providers/settings-provider.tsx` | 扩展 `reload()` 供切换作品后刷新；保持单次请求 |

未保存保护：`useUnsavedGuard` 轻量 hook（`isDirty` 时切换作品前弹确认 Dialog）。本变更只有作品名称表单，用于重命名/删除场景。

## 7. 错误处理

- 所有新增错误码复用 `abortError*`；`RESTORE_CONFLICT` 使用 `details.items`。
- 前端按 code 映射文案：`NAME_CONFLICT`「名称已存在」、`VERSION_CONFLICT`「内容已被修改，请刷新后重试」、`SNAPSHOT_FAILED`「快照失败，操作已取消」、`UNDO_EXPIRED`「撤销窗口已关闭」。
- 500 不含堆栈（沿用 CHG-001 契约）。

## 8. 测试策略

| 层级 | 覆盖 |
|---|---|
| 单元 | `name_key` 规范化、If-Match 解析、undo 过期判定、快照校验和计算、保留 3 份裁剪 |
| 集成（Go，需 DB） | 迁移 0002 幂等；创建作品含内置类别；软删除整组；恢复原子性；快照失败阻止删除（注入失败的快照服务） |
| E2E | 新建→切换→重命名→删除→撤销；回收站恢复与永久删除；设置页快照列表；空状态；1024 无溢出；axe |

集成测试将首次建立 `backend/internal/integration`（`-tags=integration`，用 `TEST_DATABASE_URL`），并在 `ci.ps1` 中以 compose 内 DB 运行。

## 9. 回滚

- 迁移 `0002` 提供 `Down`：删表 `undo_tokens`、`snapshots`、`app_settings`、`system_categories`、`works`（顺序按 FK）。
- 代码回滚：`git revert` 本变更提交；前端静态导出随镜像回退。
- 数据回滚：删除前已有快照，可用 `restore-snapshot` 恢复（该 CLI 在 CHG-010 交付，本变更的快照已为未来恢复准备）。

## 10. Open Questions

- Q-01 快照命名冲突（backup spec vs api-contract）→ 已裁决 D-03，以 api-contract 为准并回写 backup spec。
- Q-02 内置「境界」类别的最小结构是否提前到本变更 → 是（创建作品必须原子创建，否则 CHG-003 需回填）。
- Q-03 回收站是否展示 `affected_count` → 是（作品根下实体数暂为 0/1，字段保留）。
