# CHG-009 任务拆解

- Change ID：`CHG-009-search`
- 依据：`proposal.md`、`design.md`
- 规则：任务为可验证的垂直切片；每项完成后提交一次；门禁命令见 `design.md` §6.5。

## 1. 任务列表

### T1 迁移 0009 与检索模型

- **目标**：建立 `search_documents` 表、约束与索引；补检索 DTO 与文档模型。
- **文件**：`backend/internal/migrations/0009_search_documents.sql`、`backend/internal/model/search.go`。
- **验收**：空库 Up/Down 成功；五类 CHECK、唯一键、`(work_id,entity_type,is_deleted)` 与 5 个 GIN trgm 索引存在；DTO 与 `design.md` §3.3 一致。
- **依赖**：无。
- **验证**：`go build ./...`、`go vet ./...`、迁移集成用例。

### T2 检索文档刷新器

- **目标**：实现 `Refresher.RefreshWork/RebuildAll/Consistency`，覆盖五类文档、软删除、物理删除与多关系聚合。
- **文件**：`backend/internal/search/{types,refresher}.go`、`backend/internal/integration/search_refresh_test.go`。
- **验收**：多别名/多地点/多榜单/多体系引用仍为单文档；`relation_text` 无重复膨胀；软删除 `is_deleted=true`；物理删除后文档清理；两次刷新幂等；一致性 drift=0。
- **依赖**：T1。
- **验证**：集成 `search_refresh_test.go` PASS；`go vet ./...`。

### T3 人物/势力写入接入

- **目标**：人物、势力写事务末尾刷新；改名/删除/恢复后引用方文本同步。
- **文件**：`backend/internal/service/{characters,factions}.go`、`backend/internal/integration/search_test.go`。
- **验收**：人物创建/更新/删除、势力创建/更新/删除后文档一致；人物改名后旧名不命中；势力改名后人物关联命中更新。
- **依赖**：T2。
- **验证**：相关集成用例 PASS；`go test ./...`。

### T4 地点/榜单/体系接入

- **目标**：地点、榜单、体系节点写路径接入；无事务方法补 `tx`。
- **文件**：`backend/internal/service/{locations,rankings,system}.go`、`backend/internal/integration/search_refresh_test.go`。
- **验收**：地点关系增删、榜单席位指派、节点/类别改名后文档同步；软删除后不命中；恢复后重新命中。
- **依赖**：T2。
- **验证**：相关集成用例 PASS；`go vet ./...`。

### T5 回收站与作品生命周期

- **目标**：`Restore/Undo/Purge/PurgeAll` 与作品恢复接入刷新；作品永久删除依赖 FK 级联。
- **文件**：`backend/internal/service/{trash,works}.go`、`backend/internal/integration/search_test.go`。
- **验收**：删除进回收站不命中，恢复后命中；永久删除无残留；清空回收站后 drift=0；作品隔离。
- **依赖**：T3、T4。
- **验证**：trash/works 相关集成 PASS。

### T6 检索查询服务与 HTTP 端点

- **目标**：实现 `GET /api/v1/search`：参数解析、分词/转义、评分、筛选、分组、分页、DTO、错误与 `X-Cache`。
- **文件**：`backend/internal/search/query.go`、`backend/internal/service/search.go`、`backend/internal/httpapi/search.go`、`backend/internal/httpapi/router.go`、`backend/cmd/server/main.go`、`backend/internal/integration/suite_test.go`。
- **验收**：别名/称号/英文大小写/简介/关联命中；AND；筛选与失效 ID；空查询 400；非法类型 422；组 total 与全局分页正确；当前页 subtitle/URL 正确；`X-Cache: BYPASS`。
- **依赖**：T5。
- **验证**：全量后端门禁 + 检索集成用例。

### T7 CLI reindex

- **目标**：新增 `cli reindex` 与 `--check`。
- **文件**：`backend/cmd/cli/main.go`。
- **验收**：全量重建输出计数；`--check` 漂移时非 0 退出且不写数据；advisory lock 冲突提示。
- **依赖**：T2。
- **验证**：`go build ./...`、手工 smoke。

### T8 前端 API 与检索 URL 状态

- **目标**：`lib/api/search.ts`、`lib/search.ts` 与单测。
- **文件**：`frontend/lib/api/search.ts`、`frontend/lib/api/search.test.ts`、`frontend/lib/search.ts`、`frontend/lib/search.test.ts`。
- **验收**：URL 参数序列化/解析往返；筛选适用性；分组顺序；API 参数拼接正确。
- **依赖**：T6 契约冻结。
- **验证**：`npm run typecheck`、`npm run lint`、`npm run test`。

### T9 检索页面与交互状态

- **目标**：`/search` 页面、模式切换、条件面板、结果分组/高亮/分页、状态与 a11y、导航入口。
- **文件**：`frontend/app/search/page.tsx`、`frontend/components/search/*`、`frontend/components/shell/global-nav.tsx`、`frontend/lib/messages.ts`、`frontend/styles/components.css`。
- **验收**：PG-01 §12 的快速/条件/分组/高亮/分页/空/无结果/错误/加载/键盘项全部可用；1024/1280 不溢出。
- **依赖**：T8。
- **验证**：typecheck/lint/test；E2E 初版。

### T10 系统深链、E2E 与收口

- **目标**：`/system?category=&node=` 深链；补齐 E2E、axe、键盘、视觉与性能基线；F3~F10 文档与回写。
- **文件**：`frontend/components/system/*`、`frontend/app/system/page.tsx`、`frontend/e2e/search.spec.ts`、`frontend/e2e/{a11y,keyboard,visual}.spec.ts`、`changes/CHG-009-search/*`。
- **验收**：系统节点结果可定位；全量 Playwright PASS；axe 无 serious/critical；1000 人物 P95 ≤150ms 记录；文档回写并归档。
- **依赖**：T9。
- **验证**：全部门禁 + 手工验收。

## 2. 依赖图

```text
T1 ── T2 ── T3 ──┐
          ├── T4 ── T5 ── T6 ── T8 ── T9 ── T10
          └── T7
```

## 3. 并行边界

- T3/T4/T7 可在 T2 后并行。
- T5 依赖 T3/T4；T6 依赖 T5；T8 依赖 T6。
- T9 依赖 T8；T10 最后执行。

## 4. 门禁

- 每个后端任务：`gofmt -l .`、`go vet ./...`、`go build ./...`、相关 `go test`。
- T5/T6 后：全量集成。
- 每个前端任务：`npm run typecheck`、`npm run lint`、`npm run test`。
- T9/T10 后：Node 22 构建 + 全量 Playwright。
- F3 回填 `verification.md`；F4 独立审查；F6 全量回归。
