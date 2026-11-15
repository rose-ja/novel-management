# CHG-010 任务拆解

- Change ID：`CHG-010-backup`
- 依据：`proposal.md`、`design.md`
- 规则：每项完成后独立提交；门禁命令见 `design.md` §8。

## 1. 任务列表

### T1 迁移 0010、导出模型与 schema 版本

- **文件**：`backend/internal/migrations/0010_export_runs.sql`、`backend/internal/model/export.go`、`backend/internal/migrations/migrations.go`、`backend/internal/migrations/migrations_test.go`。
- **验收**：Up/Down 成功；四态 CHECK 与索引；`LatestVersion=10` 与嵌入 SQL 最大版本一致。
- **验证**：gofmt/vet/build/迁移用例。

### T2 `internal/backup` 工具包

- **文件**：`backend/internal/backup/{archive,checksum,manifest,disk_linux,disk_other}.go` 与单测。
- **验收**：zip-slip 拒绝；checksum 篡改/缺文件失败；manifest 版本校验；确定性排序。
- **验证**：`go test ./internal/backup/...`。

### T3 导出服务与后台任务

- **文件**：`backend/internal/service/export.go`、`backend/internal/service/export_test.go`、`backend/cmd/server/main.go`。
- **验收**：两种模式产物与 manifest 正确；失败置 failed 并清理；启动中断标记；`last_export_at` 更新。
- **验证**：单测 + build/vet。

### T4 导出 HTTP API

- **文件**：`backend/internal/httpapi/exports.go`、`backend/internal/httpapi/router.go`、`backend/internal/integration/exports_test.go`。
- **验收**：202/409、列表/详情/download/status；错误码与中文文案。
- **验证**：集成 PASS。

### T5 CLI 恢复

- **文件**：`backend/internal/service/restore.go`、`restore_test.go`、`backend/cmd/cli/main.go`、`backend/internal/service/snapshot.go`。
- **验收**：版本/校验/非空/快照失败分支；fake runner 成功写 `last_restore_at`；恢复前快照；摘要。
- **验证**：单测 + 集成（fake runner）。

### T6 设置页导出区块

- **文件**：`frontend/lib/api/exports.ts`、`frontend/components/settings/export-section.tsx`、`frontend/app/settings/page.tsx`、`frontend/lib/messages.ts`、`frontend/styles/components.css`。
- **验收**：Dialog 默认包含回收站；轮询/成功/失败/重试/无作品禁用；可下载与复制路径。
- **验证**：typecheck/lint/test。

### T7 CLI 区块、快照路径与系统信息

- **文件**：`frontend/components/settings/cli-restore-section.tsx`、`snapshot-list.tsx`、`page.tsx`。
- **验收**：两条命令复制；路径列复制/Tooltip；访问地址显示。
- **验证**：typecheck/lint/test。

### T8 前端单测、a11y 与键盘

- **文件**：`frontend/lib/api/exports.test.ts`、`frontend/e2e/a11y.spec.ts`、`frontend/e2e/keyboard.spec.ts`。
- **验收**：API 单测；`/settings` axe 无 serious/critical；Tab/Esc 可达。
- **验证**：npm test + Playwright 相关用例。

### T9 Docker 构建、E2E 与视觉基线

- **文件**：`frontend/e2e/settings-backup.spec.ts`、`frontend/e2e/visual.spec.ts`。
- **验收**：核心 E2E PASS；视觉基线更新；全量 Playwright PASS。
- **验证**：`docker compose build app` + `npx playwright test`。

### T10 收口

- **文件**：`changes/CHG-010-backup/{verification,review,acceptance,pr}.md`、归档目录、Spec/PG/契约/数据模型/roadmap。
- **验收**：F3~F10 完成；空环境演练记录；文档同步并推送。
- **验证**：全量门禁 + 手工演练。

## 2. 依赖图

```text
T1 ─ T2 ─ T3 ─ T4 ─ T5 ─ T6 ─ T7 ─ T8 ─ T9 ─ T10
```

## 3. 门禁

- 每个后端任务：gofmt（改动文件）、go vet、go build、相关 go test。
- T4/T5 后：全量集成。
- 每个前端任务：typecheck、lint、test。
- T9 后：Docker 构建 + 全量 Playwright。
