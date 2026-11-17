# CHG-012 发布收口：关系一致性、异常状态与发布验收

## Summary

- 回收站支持 `type`/`highlight` 深链定位：跨页查找、高亮、滚动与回焦；人物/势力/地点删除对话框引用直达对应回收站条目，E2E-15 覆盖阻断→定位→恢复→解除全链路。
- 统一未保存守卫：势力内联编辑、体系类别/节点对话框、榜单新建/容量/调席/改名与席位备注；新增 `components/ui/unsaved-dialog.tsx`。
- 状态与错误收口：428 文案统一为「页面数据已过期，请刷新后重试」；E2E-12 扩展列表 500 重试保留筛选、筛选空态与无作品空态。
- 质量门禁：a11y 覆盖人物/势力/榜单详情与对话框、回收站有数据；键盘主流程（人物增改删、回收站恢复、Esc 回焦）；E2E-08 跨页一致性；E2E-13 CSP 基线。
- 性能基线：新增 `cli seed`（默认 1000 人物、单事务批量写入并重建检索）、`e2e/perf.spec.ts` 与 `scripts/perf.ps1`；实测首屏 221ms、列表 P95 5ms、检索 P95 10ms。
- 发布演练：导出→CLI 恢复计数一致、快照创建/恢复、迁移回滚、恶意 zip/路径穿越/伪造图片复测通过；README 增加运行/门禁/性能/发布步骤。

## Changes

### 前端

- `app/trash/page.tsx`：深链参数解析、跨页定位、一次性高亮/滚动/回焦与未命中提示。
- `components/{characters,factions,locations}/`：删除对话框回收站引用链接指向 `/trash?type=…&highlight=…`；势力内联编辑守卫。
- `components/system/{system-dialogs,reference-dialog}.tsx`：类别/节点对话框关闭守卫与深链链接。
- `components/rankings/{create-ranking-dialog,capacity-dialog,adjust-seat-dialog,ranking-detail-page,seat-row}.tsx`：新建/容量/调席关闭守卫；改名离开拦截；备注取消确认。
- `components/ui/{dialog,unsaved-dialog}.tsx`：统一确认框；Dialog 关闭后显式回焦。
- `lib/messages.ts`：428 文案；`styles/components.css`：高亮样式。

### 测试与工具

- `frontend/e2e/*`：E2E-08、E2E-12、E2E-13、E2E-15，a11y/键盘/错误态/回收站/榜单用例扩展，性能用例。
- `backend/cmd/cli/main.go`：新增 `seed` 命令（`--work/--characters/--factions/--locations`，上限校验，不占用排他锁）。
- `scripts/perf.ps1`：造数、HTTP P95 采样、Playwright 首屏测量与阈值判断，输出 `artifacts/perf/<时间>/report.json`。
- `.gitignore`：忽略 `artifacts/`；`README.md`：本项目启动/门禁/性能/发布与恢复说明。

## Test Plan

- [x] `gofmt -l cmd/cli`、`go vet ./...`、`go test -count=1 ./...`
- [x] `TEST_DATABASE_URL=… go test -count=1 -tags=integration ./internal/integration/...`（14.6s）
- [x] `npm run lint`、`npm run typecheck`、`npm test`（91）
- [x] `docker compose build app` + `scripts/smoke.ps1`
- [x] `npx playwright test`（129 passed / 1 skipped）
- [x] `scripts/perf.ps1`（首屏 221ms / 列表 P95 5ms / 检索 P95 10ms）
- [x] 导出 → CLI 恢复 → 计数一致；快照恢复；`TestMigrationRoundTrip`
- [x] 安全用例复跑（恶意 zip、路径穿越、伪造图片）与 E2E-13 CSP
- [x] 演练数据清理：`works=0 trash=0`
