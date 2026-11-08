# Verification — CHG-002 作品生命周期与回收站/快照基座

- 分支：`feat/chg-002-works-lifecycle`
- 关联：`proposal.md`（D-01/D-02/D-03）、`design.md`、`tasks.md`、`checklist.md`
- 验证日期：2026-11-08
- 提交：`07c78f6`（`feat/chg-002-works-lifecycle`）
- 验收：ACCEPT（`acceptance.md`）
- 发布状态：未发布（本地合并到 `main`，无远程发布记录）
- 归档：`changes/archive/CHG-002-works-lifecycle/`

## 1. 环境

- Windows 11，PowerShell 5.1，Node 22，Go 1.26，Docker Desktop
- 栈：`docker compose up -d`（app / db / cache 均 healthy，应用端口 `127.0.0.1:17860`）
- 数据库：PostgreSQL 16；迁移 0001 + 0002 已执行（`goose_db_version` version 2）
- 容器内 `pg_dump (PostgreSQL) 16.15`

## 2. 自动化门禁

| 检查 | 命令 | 结果 |
|---|---|---|
| 后端格式 | `gofmt -l .` | 无输出 |
| 后端 Lint | `golangci-lint run` | 0 issues |
| 后端单测 | `go test ./...` | 通过（config/httpapi/model/service） |
| 后端集成 | `go test -tags=integration ./internal/integration/...` | 通过（独立 scratch 库 `{db}_it`，事务回滚） |
| 前端 Lint/类型/单测 | `npm run lint`、`npm run typecheck`、`npm test` | 通过（4 文件 / 19 条） |
| 前端构建 | `npm run build` | 通过（`/`、`/settings`、`/trash`、404 静态导出） |
| 镜像 | `docker compose build app` | 通过 |
| E2E | `npm run test:e2e` | 34 passed（含 axe 4 页、视觉基线 5 张、作品/回收站/快照/键盘用例） |
| 本地 CI | `powershell -File scripts/ci.ps1` | `CI PASSED`（退出码 0；34 E2E passed；单元 + 集成 + 构建；2026-11-08 18:38） |

集成测试用例（`backend/internal/integration/works_test.go`）：

- `TestCreateWorkCreatesBuiltinCategoryAndActivates`
- `TestCreateWorkDuplicateName`
- `TestListWorksSearchAndPagination`
- `TestUpdateWorkIfMatch`、`TestUpdateWorkNameConflict`
- `TestDeleteWorkConfirmNameAndVersion`
- `TestSnapshotFailureBlocksDelete`
- `TestDeleteThenUndo`、`TestUndoExpired`
- `TestRestoreConflictThenRename`
- `TestPurgeWorkRequiresConfirm`、`TestPurgeAllTrash`
- `TestSettingsCurrentWorkPatch`
- `TestSnapshotsEndpointReturnsRecentItems`
- `TestCrossOriginWriteIsForbidden`
- `snapshot_test.go`：保留 3 份、失败记录；`migrations_test.go`：up→down→up 往返；`regression_test.go`：`/health`、`/meta`、未知 API 404

## 3. 验收逐条证据

| AC | 证据 |
|---|---|
| AC-001 | `TestCreateWorkCreatesBuiltinCategoryAndActivates`：201、`version=1`、内置「境界」类别同事务创建并设为当前作品 |
| AC-002 | `TestCreateWorkDuplicateName`：409 `NAME_CONFLICT`，原记录保留 |
| AC-003 | `TestListWorksSearchAndPagination`：`q=山` + 分页字段完整，只返回未删除作品 |
| AC-004 | `TestUpdateWorkIfMatch`：缺/错 `If-Match` 409 `VERSION_CONFLICT`；正确时 `version+1` |
| AC-005 | `TestDeleteWorkConfirmNameAndVersion`：`confirm_name` 不符 422 `VALIDATION_ERROR`，作品未删除 |
| AC-006 | `TestSnapshotFailureBlocksDelete`（`flakyDumper`）：503 `SNAPSHOT_FAILED`，作品保留 |
| AC-007 | `TestDeleteThenUndo` + E2E「删除作品需要名称确认，撤销后回到当前作品」：8 秒内撤销恢复并补回当前作品 |
| AC-008 | `TestUndoExpired` 断言 409 `UNDO_EXPIRED`；API 实测重复撤销 409（`ALREADY_DELETED`） |
| AC-009 | `TestRestoreConflictThenRename`：422 `RESTORE_CONFLICT` + `details.items`，不部分恢复 |
| AC-010 | 同上：提供 `new_name` 后成功且名称更新；E2E「重名时改名恢复」覆盖 UI 路径 |
| AC-011 | `snapshot_test.go` 保留 3 份；集成 `snapshot_test.go` 连续 4 次；容器内 `/data/snapshots` 实测仅 3 个目录 |
| AC-012 | 容器内实测五件套：`app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`（见 §4） |
| AC-013 | E2E「切换作品并持久化到设置页」：顶栏切换后 `/settings` 当前作品一致 |
| AC-014 | E2E「创建作品并成为当前作品」：新建后顶栏与设置页立即显示新作品 |
| AC-015 | E2E「恢复作品后从回收站消失」「永久删除需要确认并生成快照」：名称/删除时间/影响条数/操作齐备 |
| AC-016 | E2E「清空回收站…」结束后断言「回收站是空的」且无不可用操作 |
| AC-017 | E2E「删除作品后快照列表只读展示」：时间/原因（删除作品）/大小/状态（成功），无恢复按钮；空状态由 `EmptyState` 提供 |
| AC-101 | E2E「作品切换器可用键盘打开、方向键遍历与 Esc 关闭」：聚焦进入、ArrowUp/Down 移动、Esc 关闭并回焦触发按钮 |
| AC-102 | E2E `/trash` 1280 与 1024 无横向溢出（差值 ≤1px） |
| AC-103 | `a11y.spec.ts` 覆盖 `/`、`/settings`、`/trash`、404，serious/critical = 0 |
| AC-104 | 部分满足（实现差异，见 §5）：模态弹窗下无法点击切换器；已实现「关闭/取消时未保存确认」等价保护 |
| AC-201 | `error-states.spec.ts`：网络异常与 500 均显示文案 + 重试，无堆栈；`client.test.ts` 覆盖 NetworkError/ApiError |
| AC-202 | 后端实测 422 `VALIDATION_ERROR`，`details.field=name`（空名称 / >200 字符）；前端空名称禁用提交并提示，超长由 `maxLength=200` 约束（见 §5） |
| AC-203 | E2E「删除作品需要名称确认…」：名称不匹配时确认按钮 `disabled` |
| AC-301 | `TestCrossOriginWriteIsForbidden`：跨源写请求 403 `FORBIDDEN_ORIGIN` |
| AC-302 | `works.test.ts` 断言写请求携带 `X-Requested-With: XMLHttpRequest`（含 `If-Match`）；服务端 `originGuard` 校验同源 |
| AC-401 | `regression_test.go`（`/health`、`/meta`、未知 API 404）+ `shell.spec.ts` 壳层/404 用例 |
| AC-402 | `scripts/ci.ps1` 输出 `CI PASSED`（退出码 0） |
| AC-403 | `migrations_test.go`：0002 up → down → up，5 张表存在且无残留；`app_settings` 单行约束保持 |
| AC-404 | `visual.spec.ts` 5 条：概览 1280/1024、设置 1280、回收站 1280/1024（差异阈值 0.02） |

### API 边界实测（curl）

```text
POST /api/v1/works  name=201 字符
422 {"error":{"code":"VALIDATION_ERROR","message":"作品名称不能超过 200 个字符","details":{"field":"name"}}}

POST /api/v1/works  name=""
422 {"error":{"code":"VALIDATION_ERROR","message":"作品名称不能为空","details":{"field":"name"}}}
```

## 4. 容器内真实快照核实

```text
$ docker compose exec -T app pg_dump --version
pg_dump (PostgreSQL) 16.15

$ docker compose exec -T app sh -c 'ls -1 /data/snapshots | tail -3 && ls -la /data/snapshots/$(ls -1 /data/snapshots | tail -1)'
<3 个目录，最新一个内含>
app.dump
images/
snapshot.json
manifest.json
checksums.sha256
```

说明：E2E 删除/永久删除流程触发真实 `pg_dump`，目录命名 `{YYYYMMDDTHHMMSSZ}-{reason}`，同秒冲突以 ULID 末 6 位区分；保留 3 份生效。

## 5. 已记录的实现差异（均已在文档标注）

1. **AC-104**：作品编辑/新建为模态 Dialog，打开期间无法点击顶栏切换器。已实现 `useUnsavedGuard`：按 Esc、点遮罩、取消或关闭时若有修改则弹「放弃未保存的修改？」，放弃才关闭。自动切换作品场景随多页面编辑（CHG-003+）接入同一守卫。
2. **删除当前作品后的当前作品**：按 `tasks.md` T4 清空 `current_work_id`，顶栏显示「未选择作品」；PG-00 §5「自动切换到另一本」延后到 CHG-003，已在 PG-00 §0 标注。
3. **回收站阶段范围**：仅作品根；「原作品」列恒为「—」，无原作品/父级筛选，无 `renames[]` 整组改名 UI；已写入 PG-12 §0。
4. **快照卡片**：仅时间/原因/数据库大小/状态，不显示 `path`；页面无恢复入口；已写入 PG-13 §0 与 backup SPEC-BACKUP-003。
5. **AC-202 字段级提示**：后端返回 `details.field`，前端以表单预校验（必填禁用 + `maxLength`）呈现，未把 `details.field` 定位到输入框边框/错误文案。
6. **AC-302 抓包**：以单测断言请求头替代 E2E 抓包；写请求统一经 `requestJson`，不存在漏带路径。
7. **视觉基线**：概览/设置对数据相关块（`.info-list`、`.ui-empty`、`.ui-card`）打码；回收站仅截取顶部 320px 并对页头描述/操作打码，避免本机数据影响基线。

## 6. 回滚方案

1. 代码：revert 本变更提交（前端新路由与组件、后端 0002 相关代码）。
2. 数据：`go run ./cmd/cli migrate down`（或容器内 `migrate down`）回退到 0001；`Down` 按 FK 逆序删除 `undo_tokens`、`snapshots`、`app_settings`、`system_categories`、`works`。
3. 快照：回滚不删除 `/data/snapshots`；如需清理，人工删除对应目录（保留最近 3 份逻辑不反向执行）。
4. 静态产物：`docker compose build app && docker compose up -d` 回到上一镜像。

## 7. 遗留风险

- 真实 `pg_dump` 仅在容器内验证；本机无 PostgreSQL 客户端，本地直接运行 server 时快照会失败并阻止删除（返回 503），符合设计。
- E2E 数据以 `E2E-` 前缀隔离清理；若测试被强杀可能残留作品/回收站项，重跑用例会自动清理。
- 8 秒撤销窗口依赖客户端时钟与网络；过期由服务端判定，UI 仅展示 Toast 8 秒。
