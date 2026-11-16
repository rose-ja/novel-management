# CHG-011 飞书《正文》迁移导入与校对

## Summary

- 新增固定模板 v1.0 导入：Markdown（≤20MB）+ 图片 zip（≤6GB）流式上传、后台校验、可定位到行/字段/图片的错误清单、预览计数与二次确认。
- 提交前自动快照，图片先落盘，单事务写入作品/体系/势力/地点/人物/榜单/大事记与关系，重建检索并自动切换当前作品；失败清理图片与暂存，进程中断可恢复。
- 新增导入报告（八类计数、图片匹配、耗时、快照）与校对清单（人物/势力/地点/榜单/体系节点），支持单条与批量确认；导入与导出任务互斥。
- 前端交付 PG-11 五步向导（上传/校验/确认/报告/校对）与设置页入口，含 1024 布局、键盘与 a11y。

## Changes

### 数据与后端

- `backend/internal/migrations/0011_import_runs.sql`：导入任务表（六态 CHECK、计数/错误/图片/报告 JSON、索引）；`migrations.LatestVersion = 11`。
- `backend/internal/imports/*`：模板与示例生成、Markdown 解析（8 区块）、校验（类别 ≤10、两级节点、引用、图片缺失/多余/格式/10MB、路径穿越与盘符）、zip 清单。
- `backend/internal/service/{import,import_plan,review}.go`：上传落盘与磁盘预检、后台校验、`startCommit` 串行化导入/导出、快照、原子写入、检索重建、当前作品切换、失败/中断清理；校对幂等与批量。
- `backend/internal/httpapi/{imports,reviews}.go`、`router.go`、`request.go`：模板/示例下载、multipart 上传、validate/detail/report/commit、review 单条/批量；错误码映射。
- `backend/cmd/server/main.go`：装配导入与校对，启动清理中断任务。

### 前端

- `frontend/lib/api/{imports,reviews}.ts`：XHR 上传进度、校验/详情/报告/提交、校对；模板/示例 URL。
- `frontend/components/import/*`：步骤条、上传（拖拽/键盘/模板下载）、校验（计数/错误清单/图片匹配/摘要）、报告、校对（分组/批量/空状态）、向导编排与轮询。
- `frontend/app/import/page.tsx`、`components/settings/import-section.tsx`、设置页入口与 `.import-*` 样式；顶栏作品列表在导入后自动刷新。

### 文档与测试

- `changes/CHG-011-import/*`（proposal/design/tasks/checklist/verification/review/acceptance）。
- 集成：合法导入 + 校对 + 幂等、缺图阻断、快照失败回滚、导入/导出互斥、模板下载。
- E2E：`import.spec.ts`（E2E-09 全链路 + 缺图阻断）、`a11y.spec.ts` 导入页、`keyboard.spec.ts` 步骤条与文件选择、`visual.spec.ts` 校验错误态 1280/1024。

## Test Plan

- [x] `go vet ./...`、`go build ./...`、`go test -count=1 ./...`
- [x] `go test -count=1 -tags=integration ./internal/integration/...`（22.2s）
- [x] `npm run typecheck`、`npm run lint`、`npm run test`（91）
- [x] `docker compose build app`
- [x] `npx playwright test`（115/115）
- [x] 真实示例包走完上传 → 校验 → 提交 → 报告（commit 124ms，report elapsed 232ms）
- [x] 导入后清理演练作品（works=0 trash=0）

## Compatibility / Rollback

- 迁移 0011 只新增 `import_runs` 表，Down 可安全删除；不修改既有表结构。
- 导入不改动既有作品，失败不影响当前数据；必要时用导入前快照回退。
- 导入与导出互斥：任一任务进行中，另一入口返回中文冲突提示。
