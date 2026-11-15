# CHG-010 设计

- Change ID：`CHG-010-backup`
- 依据：`proposal.md`、`project/specs/backup/specs.md`、`project/page-specs/PG-13-settings-backup.md`、`project/adr/ADR-006-export-restore-cli.md`、`project/api-contract.md` §3.12/§3.13、`project/data-model.md` §3.17
- 目标：不新增第三方依赖，交付全量导出、CLI 恢复与快照收口。

## 1. 设计决策

| ID | 决策 | 理由 |
|---|---|---|
| D1 | 导出包按 ADR-006：`manifest.json + checksums.sha256 + data/app.dump + images/ + README.txt` | 与 pg_restore 恢复、快照结构一致；用户 2026-11-15 裁决 |
| D2 | 导出为进程内后台 goroutine，状态落 `export_runs` | 与 architecture §6.7 一致；无新增队列依赖 |
| D3 | 不含回收站导出使用临时数据库过滤后 `pg_dump` | 确保包内真实不含软删除数据（AC-12） |
| D4 | 恢复单事务 `pg_restore --clean --if-exists --single-transaction` | 失败原子回滚，不产生半恢复状态 |
| D5 | 图片在 DB 恢复成功后原子替换，失败回滚旧目录 | 图片与 DB 不出现跨版本错配 |
| D6 | 恢复前强制快照（`reason=cli_restore`），失败即中止 | 满足 FR-54/FR-56 回滚要求 |
| D7 | 非空实例必须 `--yes` 才覆盖 | 避免静默清空（SPEC-BACKUP-002 场景 4） |
| D8 | 快照 manifest 增加 `schema_version/app_version`，旧 manifest 兼容 | 支持版本校验且不破坏既有快照 |

## 2. 数据设计（迁移 0010）

```sql
CREATE TABLE export_runs (
  id            TEXT PRIMARY KEY,
  status        TEXT NOT NULL DEFAULT 'queued'
                CHECK (status IN ('queued','running','completed','failed')),
  include_trash BOOLEAN NOT NULL DEFAULT true,
  file_name     TEXT,
  file_path     TEXT,
  byte_size     BIGINT,
  error_code    TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at   TIMESTAMPTZ
);
CREATE INDEX export_runs_status_created_idx ON export_runs (status, created_at DESC);
CREATE INDEX export_runs_created_idx ON export_runs (created_at DESC);
```

- `file_path` 相对 `DATA_DIR`，形如 `exports/novel-settings-20261115-143012.zip`。
- 错误码：`EXPORT_IN_PROGRESS/EXPORT_NO_WORK/EXPORT_INTERRUPTED/EXPORT_INSUFFICIENT_STORAGE/EXPORT_FAILED`。

## 3. 导出设计

### 3.1 包结构

```text
novel-settings-YYYYMMDD-HHmmss.zip
├─ manifest.json
├─ checksums.sha256
├─ README.txt
├─ data/app.dump          # pg_dump -Fc
└─ images/                # 活跃 media_assets 的 original/thumb
```

### 3.2 manifest

```json
{
  "export_version": 1,
  "schema_version": 10,
  "app_version": "0.1.0",
  "generated_at": "2026-11-15T14:30:12Z",
  "includes_trash": true,
  "works": [{"id": "…", "name": "山海"}],
  "counts": {"works": 1, "characters": 1000, "factions": 200, "locations": 200,
             "rankings": 5, "ranking_seats": 40, "system_categories": 3,
             "system_nodes": 150, "chronicles": 80, "media_assets": 480},
  "images": {"count": 480, "files": ["images/…jpg"]},
  "files": [{"path": "data/app.dump", "size": 123}]
}
```

- `counts` 均为活跃记录（`deleted_at IS NULL`）；`works` 只列活跃作品。
- `checksums.sha256` 覆盖 manifest、README、app.dump、images 全部文件，不覆盖自身。

### 3.3 流程

1. `Create`：检查无 queued/running → 插入 `queued` → 返回；HTTP 启动 goroutine。
2. `Run`：置 `running` → 生成临时目录 → dump → 复制活跃图片 → README → manifest → checksums → zip → 更新 `completed/byte_size/finished_at` 与 `app_settings.last_export_at`。
3. 失败：清理半成品，置 `failed/error_code/finished_at`。
4. 启动：`MarkInterrupted` 把 queued/running 置 failed。
5. 文件名冲突追加 ID 末 6 位，不覆盖旧包。

### 3.4 不含回收站

- 创建临时库 `novel_export_tmp_<id>` → `pg_restore` 当前库转储到临时库 → 清理：
  1. 关系/子表：`character_names`、`character_system_refs`、`character_locations`、`location_factions`、`ranking_seats`、`chronicles`、`media_assets` 中引用软删除实体的行；
  2. 软删除子表：`rankings`、`characters`、`factions`、`locations`、`system_nodes`、`system_categories`、`works` 中 `deleted_at IS NOT NULL`；
  3. `undo_tokens` 全清。
- 再 `pg_dump -Fc` 临时库 → 删除临时库。
- 任一步失败：删除临时库并让导出失败。

## 4. 恢复设计

1. 校验：zip 解压（防 zip-slip）；manifest `export_version==1` 且 `schema_version <= LatestVersion`；逐文件 checksums；必需 `data/app.dump`。
2. 磁盘预检（Linux 实际检查，其它平台跳过）。
3. 空实例检查：`works` 非空且无 `--yes` → 拒绝。
4. 恢复前快照：`Snapshot.Create(ctx, cli_restore)`；失败中止。
5. `pg_restore --clean --if-exists --no-owner --no-privileges --single-transaction -d $DATABASE_URL data/app.dump`。
6. 图片替换：旧 `images/` → `images.pre-restore-<ts>`，解压图片 → `images/`；失败恢复旧目录。
7. `migrations.Up` → `search.RebuildAll` → 写 `app_settings.last_restore_at`。
8. 计数核对：与 manifest `counts` 比较，不一致返回错误并给出回滚命令。
9. 回填恢复前快照记录；删除路径不存在的快照行。
10. `restore-snapshot <dir>` 使用 `app.dump/images`，额外校验 `snapshot.json`；manifest 缺 `schema_version` 时告警但允许。

- CLI 先获取 advisory lock；app 未停止时返回 `ErrLockBusy` 并提示 `docker compose stop app`。
- 失败输出回滚命令：`docker compose run --rm cli restore-snapshot <恢复前快照目录>`。

## 5. API

| 方法 | 路径 | 行为 |
|---|---|---|
| POST | `/api/v1/exports` | `{include_trash}` → 202 `{export_id,status}`；进行中 409 `EXPORT_IN_PROGRESS` |
| GET | `/api/v1/exports` | 最近 20 条 |
| GET | `/api/v1/exports/{id}` | 详情；不存在 404 |
| GET | `/api/v1/exports/{id}/download` | completed 才可下载；流式 zip + Content-Disposition |
| GET | `/api/v1/export/status` | `{active, last}`，供页面恢复进度 |

DTO：`id/status/include_trash/file_name/file_path/byte_size/error_code/created_at/finished_at`。

## 6. 前端

- `lib/api/exports.ts`：类型与请求封装；下载 URL 纯函数。
- `export-section.tsx`：主按钮 → Dialog（说明 + 包含回收站 Checkbox 默认勾选）→ 轮询 → 成功卡片/失败 Alert；无作品禁用；生成中禁用；`aria-live` 播报。
- `cli-restore-section.tsx`：两条命令 + 复制 + Warning Alert。
- `snapshot-list.tsx`：新增路径列（等宽、中间省略、复制、Tooltip）。
- `app/settings/page.tsx`：顺序 导出 → 快照 → CLI → 系统信息；补访问地址。
- 不新增设计系统组件依赖；Checkbox 用原生 input + 既有 FormField 样式。

## 7. 测试策略

- 单元：zip/checksum/manifest 版本、导出两种模式与失败、恢复校验分支、CLI 标志、前端 API。
- 集成：202/409/轮询完成、download、failed 重试、include_trash 标志、恢复（fake runner）写 `last_restore_at` 与快照记录。
- E2E：设置页区块、导出 Dialog 与成功/失败、快照路径复制、命令复制、a11y、键盘、视觉基线。
- 演练：真实导出 → 停 app → `restore` → 启动抽查；`restore-snapshot` 一次。

## 8. 门禁

- 后端：`gofmt -l <改动文件>`、`go vet ./...`、`go build ./...`、`go test -count=1 ./...`
- 集成：`TEST_DATABASE_URL=… go test -count=1 -tags=integration ./internal/integration/...`
- 前端：`npm run typecheck`、`npm run lint`、`npm run test`
- 构建：`docker compose build app`
- E2E：`npx playwright test --reporter=list`

## 9. 风险与回滚

- 记录在 `proposal.md` Risks；恢复失败不触碰数据/图片，可用恢复前快照回退。
- 代码回滚：回退 0010 迁移与本次代码；导出文件与快照目录保留。

## 10. 变更历史

| 日期 | 版本 | 说明 |
|---|---|---|
| 2026-11-15 | v0.1 | F1 初版：ADR-006 导出包、后台任务、CLI 恢复、PG-13 与测试策略 |
