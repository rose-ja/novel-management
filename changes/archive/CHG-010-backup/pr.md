# CHG-010 导出、CLI 恢复与快照收口

## Summary

- 新增全量导出：后台任务、设置页导出 Dialog/进度/结果下载、`export_runs` 状态表与 HTTP API；导出包按 ADR-006 产出 `manifest.json + checksums.sha256 + data/app.dump + images/ + README.txt`，支持包含/排除回收站两种模式。
- 新增 CLI `restore <zip>` 与 `restore-snapshot <dir>`：manifest/checksum/schema 校验、磁盘预检、非空实例 `--yes` 确认、恢复前自动快照、单事务 `pg_restore`、图片原子替换、迁移与检索重建、计数核对、`last_restore_at` 回写。
- 设置页补齐 PG-13：导出区块、快照路径与复制、两条 CLI 恢复命令与注意事项、系统信息（版本/数据目录/访问地址/最近导出恢复）。

## Changes

### 数据与后端

- `backend/internal/migrations/0010_export_runs.sql`：导出任务表与索引；`migrations.LatestVersion = 10`。
- `backend/internal/backup/*`：zip 打包/解包（zip-slip 与 32GB/20 万条目上限）、checksum、manifest、磁盘检测。
- `backend/internal/service/{export,export_pg,restore,snapshot}.go`：导出后台任务、临时库过滤回收站、恢复流程与恢复前快照；快照 manifest 增加 schema/app 版本。
- `backend/internal/httpapi/exports.go`：`POST/GET /exports`、`GET /exports/{id}`、`GET /exports/{id}/download`、`GET /export/status`。
- `backend/cmd/{server,cli}/main.go`：启动清理中断任务；新增 restore 命令。

### 前端

- `frontend/components/settings/export-section.tsx`：Dialog 默认包含回收站、轮询、成功卡片、下载/复制、失败重试、无作品禁用。
- `frontend/components/settings/cli-restore-section.tsx`：恢复命令与复制。
- `snapshot-list.tsx` 增加路径列与复制；设置页标题/顺序/系统信息更新。
- `frontend/lib/api/exports.ts`、`clipboard.ts`、`format.ts`（中间省略）与文案/样式。

### 文档

- `changes/CHG-010-backup/*`（proposal/design/tasks/checklist/verification/review/acceptance）。
- 回写 backup Spec、PG-13、api-contract、data-model、roadmap。

## Test Plan

- [x] `go vet ./...`、`go build ./...`、`go test -count=1 ./...`
- [x] `go test -count=1 -tags=integration ./internal/integration/...`（21.2s）
- [x] `npm run typecheck`、`npm run lint`、`npm run test`（91）
- [x] `docker compose build app`
- [x] `npx playwright test`（109/109）
- [x] `docker compose stop app` → `cli restore <zip> --yes` → `cli restore-snapshot <dir> --yes` → 启动健康检查
- [x] 真实 zip 解包核对五件套与 manifest 计数

## Compatibility / Rollback

- 迁移 0010 只新增表，可安全回滚（Down 删除 `export_runs`）。
- 导出包格式版本 `export_version=1`；恢复拒绝高于当前 `schema_version` 的包。
- 恢复失败不覆盖数据库（单事务），恢复前快照保留；非事务步骤失败按 CLI 输出的 `restore-snapshot` 命令回滚。
- 旧快照缺少 `schema_version` 时按兼容模式恢复并告警。
