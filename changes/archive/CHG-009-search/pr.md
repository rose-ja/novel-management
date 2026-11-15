# PR/MR 描述 — CHG-009 快速搜索与条件检索

> 状态：已推送（2026-11-15，`origin/main` `18ce34c`）。本次变更直接在 `main` 上按阶段提交，实现范围 `2d428cb..18ce34c`（12 个提交，54+ 文件，+4832/−203，含 F10 回写）。**未创建远程 PR**（单人本地仓库流程）。
> 门禁：F3 自测 PASS、F4 独立审查 0×S0 / 3×S1 / 5×S2、F5 修复复审 **PASS**、F6 全量回归 **105 passed**、F7 人工验收 **ACCEPT**。

## 1. 标题与目标

**feat: 快速搜索与条件检索（CHG-009）**

为当前作品提供跨人物、势力、地点、榜单、体系节点的统一检索：反规范化 `search_documents` + `pg_trgm` 索引；任何可检索写入口在同一事务内整作品重建检索文档；`GET /api/v1/search` 支持关键词、条件筛选、分组、评分排序与固定分页；前端 `/search` 提供快速/条件双模式、URL 固化检索态、高亮与深链；`cli reindex` 负责全量重建与漂移检查。

## 2. 关联

- Requirement：FR-51、FR-52、FR-53
- 验收：AC-12、AC-15、AC-16
- Spec：SPEC-SEARCH-001～006
- Page Spec：PG-01 检索
- 契约：`project/api-contract.md` §3.7；`project/data-model.md` §3.8（F10 回写实际形态）
- 变更文档：`changes/CHG-009-search/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 迁移 `0009_search_documents`：唯一键 `(work_id, entity_type, entity_id)`、`is_deleted`、5 个部分 GIN `gin_trgm_ops` 索引、`(work_id, entity_type, is_deleted)` 索引；复用 0001 已启用的 `pg_trgm`；Down 可逆。
- 刷新器：人物/势力/地点/榜单/体系节点 5 条集合式 upsert + 物理不存在清理；聚合别名/称号/关联文本，避免笛卡尔膨胀；作品级 `pg_advisory_xact_lock` 串行化；`RebuildAll` 与 `Consistency`（drift）。
- 写路径接入：人物/势力/地点/榜单/席位/体系类别与节点的创建、更新、删除；地点-势力关系增删；回收站 Restore/Undo/Purge 各根类型；作品软删除/恢复；`PurgeAll` 后全量重建。
- 查询：`GET /api/v1/search`；空格分词 AND、英文大小写不敏感、五类筛选（势力直属/含下级、境界节点/类别、地点、榜单、校对状态）、评分排序、按类型分组、全局 20/页；空查询 400、非法参数/失效筛选 422；`X-Cache: BYPASS`。
- CLI：`cli reindex [--work <id>] [--check]`，advisory 锁、全量重建与漂移报告。
- 前端：`/search` 快速/条件模式、300ms 防抖、AbortController + 请求序号、IME 组合保护、Esc 清空、URL 查询参数固化、返回恢复、组头组总命中、`<mark>` 高亮、失效筛选移除重试、错误重试、空态。
- 深链：结果分别跳转 `/characters?id=`、`/factions?id=`、`/locations?id=`、`/rankings?id=`、`/system?category=&node=`；系统页按类别与节点定位。
- 导航/样式/可访问性：主导航新增「检索」；1024/1280 响应式；axe 无 serious/critical；检索页键盘路径与视觉基线。
- 文档：F10 回写 SPEC-SEARCH、`api-contract.md`、`data-model.md`、roadmap、PG-01。

**非目标（已登记延后）**

| 项目 | 承接 |
|---|---|
| 校对确认端点（仅只读 `review_status`） | CHG-011 |
| 大事记全文检索（仅关联摘要文本） | 需求边界 |
| 搜索历史、热门词、跨作品检索 | 需求边界 |
| 分页 cursor / 每页条数切换 | 性能专项 |
| 查询下推 SQL 评分与窗口分页 | 性能专项（当前 P95 7.11ms） |

## 4. 主要文件/模块变更

**后端**

- 迁移/模型：`migrations/0009_search_documents.sql`、`model/search.go`、`model/errors.go`。
- 检索：`search/refresher.go`（刷新/一致性）、`search/query.go`（动态 WHERE、评分、命中窗口）。
- 服务/HTTP：`service/search.go`、`httpapi/search.go`、`httpapi/router.go`、`service/{characters,factions,locations,rankings,system,trash,works}.go`、`cmd/server/main.go`。
- CLI：`cmd/cli/main.go`（`reindex`、`--check`、`--work`）。
- 测试：`integration/{search_refresh,search_write,search_relations,search_trash,search_api,migrations}_test.go`、`integration/suite_test.go`。

**前端**

- API/纯函数：`lib/api/search.ts`、`lib/search.ts` 及单测。
- 组件：`components/search/{search-route,search-page,quick-search-bar,search-mode-switch,conditional-filter-panel,search-result-group,search-result-row,matched-snippet}.tsx`。
- 系统深链：`components/system/system-route.tsx`、`app/system/page.tsx`、`components/system/{system-page,category-detail,node-tree}.tsx`。
- 页面/导航/文案/样式：`app/search/page.tsx`、`components/shell/global-nav.tsx`、`lib/messages.ts`、`styles/components.css`。
- 测试：`e2e/search.spec.ts`（7 例）、`e2e/visual.spec.ts`（检索 1280/1024 基线）、`e2e/a11y.spec.ts`、`e2e/keyboard.spec.ts`。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/search-1280-chromium-win32.png`、`search-1024-chromium-win32.png`。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; go build ./...` | 通过 |
| `cd backend; go vet ./...` | 0 |
| `cd backend; go test -count=1 ./...` | 全部 `ok` |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | **全量 PASS**（20.0s） |
| `cd frontend; npm run typecheck` / `npm run lint` | 通过 / 0 warning |
| `cd frontend; npm run test` | **86 passed**（14 文件） |
| `docker compose build app`（node:22-alpine） | 静态导出成功，`/search` 9.32 kB / 首载 133 kB |
| `docker compose up -d app` | 容器 healthy，迁移至 **v9** |
| `cd frontend; npx playwright test` | **105 passed**（1.7m） |
| 性能：1000 人物 100 次 `GET /search` | avg 6.28ms、p50 6.18ms、p95 7.11ms、max 10.12ms |
| 首屏：导航 `/search` 至输入框可交互 | 200ms；输入到首条结果 828ms |
| `TestMigrationRoundTrip` | 0009 Down/Up 往返通过（v9） |

## 7. E2E 与人工验收证据

- E2E 检索 7 例：自动聚焦/单字命中/双 `<mark>`/组头与返回恢复；防抖合并与 Enter；条件筛选与 chip 移除；条件模式仅关键词提交；空查询 0 请求与无结果清除；500 错误重试；体系节点深链 `aria-selected`。
- 集成覆盖：迁移 v9、五类文档与多别名/称号/关联文本、软删/恢复/物理删除、作品删除/恢复、回收站各根与 PurgeAll、别名/称号/URL/大小写/AND/分组/排序/分页/失效筛选/空查询、非法 page/超大 page/非法 review_status、当前页空分组。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-15）。
- 审查：`review.md` F4 发现 3 个 S1 与 5 个 S2；F5 全部修复或登记偏差后复审 **PASS**。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0009 仅新增 `search_documents` 与索引，不改既有表；`pg_trgm` 已由 0001 启用；无数据回填。
- **兼容性**：新增 `/search` 路由与 `cli reindex` 子命令；`/system` 新增可选 `category`/`node` 查询参数；未改既有响应结构。
- **一致性**：整作品重建在写事务内完成；作品级 advisory 锁串行化并发刷新；启动与 `reindex` 可自愈；`--check` 输出漂移。
- **性能**：1000 人物 P95 7.11ms；分页固定 20；后续可下推 SQL 评分与窗口分页。
- **回滚方案**：回退本变更提交并重新部署；若已迁移执行 0009 Down（仅删表，不影响源实体）；或从操作前快照恢复。

## 9. Reviewer 重点关注

1. `search/refresher.go` 的 5 条 upsert SQL、聚合子查询与清理条件（软删/物理删除/作品删除）。
2. `search/query.go` 的动态 WHERE、token AND、势力子树递归与评分权重。
3. `service/search.go` 的失效筛选 422、分页切片溢出安全、组 total/当前页切片、subtitle/URL 装配。
4. 各写服务事务末端的 `RefreshWork` 调用是否完整，回收站与作品生命周期是否覆盖。
5. `httpapi/search.go` 的 page 校验、空查询 400 与 `X-Cache`。
6. 前端 URL 往返、防抖/序号/Abort、IME、失效筛选移除重试与深链。
7. 文档一致性：`api-contract.md` §3.7、`data-model.md`、SPEC-SEARCH、PG-01 由 F10 回写。

## 10. 已知遗留（不阻断合并）

| ID | 内容 | 承接 |
|---|---|---|
| — | 查询在 Go 内评分/排序/分页，未下推 SQL；聚合列不命中 trgm 索引 | 性能专项 |
| — | `review_status` 仅后端筛选，前端不展示校对筛选；校对端点 | CHG-011 |
| — | URL 以 `realm` 表达境界节点，未使用 `category`/`node` 双参数 | 接受 |
| — | 检索固定 20/页，无每页条数切换与 cursor | 需求边界 |
| — | 启动 `RebuildAll` 失败只告警 | 运维约定 |
