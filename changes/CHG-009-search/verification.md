# CHG-009 验证记录（F3）

- Change ID：`CHG-009-search`
- 日期：2026-11-15
- 范围：T1–T10 全部实现（迁移 v9、检索文档刷新/查询/CLI、前端检索页与系统深链、集成/单测/E2E/性能基线）
- 依据：`proposal.md`、`design.md`、`tasks.md`、`checklist.md`

## 1. 交付清单

| 任务 | 结果 | 关键文件 |
|---|---|---|
| T1 迁移与模型 | 完成 | `migrations/0009_search_documents.sql`、`model/search.go`、`model/errors.go` |
| T2 刷新器与一致性 | 完成 | `search/refresher.go`、`integration/search_refresh_test.go` |
| T3 人物/势力写入接入 | 完成 | `service/characters.go`、`service/factions.go`、`integration/search_write_test.go` |
| T4 地点/榜单/体系接入 | 完成 | `service/{locations,rankings,system}.go`、`integration/search_relations_test.go` |
| T5 回收站与生命周期 | 完成 | `service/trash.go`、`integration/search_trash_test.go` |
| T6 查询服务与 HTTP | 完成 | `search/query.go`、`service/search.go`、`httpapi/search.go`、`cmd/server/main.go`、`integration/search_api_test.go` |
| T7 CLI reindex | 完成 | `cmd/cli/main.go`（`reindex`、`--check`、`--work`） |
| T8 前端 API 与 URL 状态 | 完成 | `lib/api/search.ts`、`lib/search.ts` 及单测 |
| T9 检索页与交互 | 完成 | `app/search/page.tsx`、`components/search/*`、`global-nav.tsx`、`styles/components.css` |
| T10 系统深链与 E2E 收口 | 完成 | `components/system/system-route.tsx`、`app/system/page.tsx`、`e2e/search.spec.ts`、a11y/键盘/视觉更新 |

## 2. 门禁结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| 后端格式/静态/构建 | `gofmt -l .`（仅检查新增与改动文件）、`go vet ./...`、`go build ./...` | PASS（`gofmt` 仅列出既有 CRLF 文件，检索相关文件无输出） |
| 后端单元测试 | `go test -count=1 ./...` | PASS（config/httpapi/model/service） |
| 后端集成测试 | `TEST_DATABASE_URL=... go test -count=1 -tags=integration ./internal/integration/...` | **全量 PASS（20.0s，含 6 个检索用例）** |
| 迁移往返 | 同上，`TestMigrationRoundTrip` | PASS（版本 9；`search_documents` Up/Down，`rankings` 保留） |
| 前端类型检查 | `npm run typecheck` | PASS |
| 前端 lint | `npm run lint` | PASS |
| 前端单测 | `npm run test` | **86 PASS（14 文件）** |
| 前端构建（Node 22） | `docker compose build app`（node:22-alpine 执行 `next build`） | PASS，`/search` 9.32 kB / 首载 133 kB |
| E2E 全量 | `npx playwright test` | **105 PASS（1.6m）** |
| 检索专项 E2E | `npx playwright test e2e/search.spec.ts` | **7 PASS** |
| 无障碍 | `npx playwright test e2e/a11y.spec.ts` | **12 PASS**（含 `/search` 空态与有数据/条件筛选） |
| 键盘 | `npx playwright test e2e/keyboard.spec.ts` | **7 PASS**（含检索输入、Esc、Tab 顺序） |
| 视觉基线 | `npx playwright test e2e/visual.spec.ts` | **20 PASS**（新增 `search-1280/1024.png`） |
| 检索性能（1000 人物） | 100 次 `GET /api/v1/search?q=角色` | **avg 6.28ms、p50 6.18ms、p95 7.11ms、max 10.12ms、total=1000** |
| 首屏可交互 | Playwright 导航 `/search` 至输入框可用并聚焦 | **200ms**；输入到首条结果 828ms（含 300ms 防抖） |

## 3. 验收点证据

| 验收点 | 证据 |
|---|---|
| AC 1 自动聚焦、空查询不请求 | `search.spec.ts`「空查询不请求」断言 500ms 内 0 次检索请求；「快速搜索」断言输入框自动聚焦与空态提示 |
| AC 2 单字/防抖/Enter/IME | `search.spec.ts`「300ms 防抖」用 `pressSequentially` 断言仅 1 次请求，Enter 立即触发且不重复；IME 组合保护见 `quick-search-bar.tsx` |
| AC 3 别名与英文大小写 | `search_api_test.go`：别名「青丘狐」、称号、`SWORD`→`sword`；E2E 断言名称与别名双 `<mark>` |
| AC 4 空格 AND、不过大事记 | `search_api_test.go`：`青丘 狐` 命中人物，未命中仅大事记含关键词的实体 |
| AC 5 分组与组总命中 | `search_api_test.go` 分组顺序与组 total；E2E 断言「人物」组头 Badge=1、无结果组不渲染 |
| AC 6 命中标签与高亮 | `SearchResultRow` + `MatchedSnippet`；E2E 断言 `mark.search-mark` 含关键词 |
| AC 7 权重排序 | `search.RankDocs`/`Score` 单测路径与 `search_api_test.go` 顺序断言 |
| AC 8 每页 20 与翻页 | `search_api_test.go`：25 条 → 第 1 页 20、第 2 页 5；E2E 返回恢复关键词 |
| AC 9 条件 AND 与筛选可见性 | `lib/search.test.ts` 适用性用例；E2E 条件模式选手势力后仅筛选项即命中，移除 chip 回空态 |
| AC 10 失效筛选与空查询 | `search_api_test.go`：失效 ID 422 `VALIDATION_ERROR`、空查询 400；前端 `search-page.tsx` 移除 chip 后重试 |
| AC 11 作品隔离/回收站/改名 | `search_refresh/search_trash/search_write` 集成用例；E2E 体系节点深链 |
| AC 12 作品生命周期无残留 | `search_trash_test.go` PurgeAll 后 drift=0；`TestMigrationRoundTrip` |
| AC 13 错误态与 `X-Cache` | E2E 错误重试（500 → 重试成功）；`httpapi/search.go` 固定 `X-Cache: BYPASS` |
| AC 14 性能 | 1000 人物 P95 7.11ms ≤150ms；首屏可交互 200ms ≤2s |
| AC 15 a11y/键盘/响应式 | axe 12 页无 serious/critical；键盘 7 条；E2E 全量含 1024/1280 视觉基线 |
| AC 16 结果跳转与返回恢复 | E2E：人物结果 → `/characters?id=`；浏览器返回恢复 `q=沉`；系统节点 → `/system?category=&node=` 且节点 `aria-selected=true` |

## 4. 已确认决策落实

- D1 整作品重建：写事务内 `Refresher.RefreshWork`，启动与 CLI 可全量重建；一致性 drift=0。
- D2 查询分层：SQL 负责 token AND/筛选/回收站过滤，Go 负责评分、排序、分组与分页切片。
- D3 失效筛选：后端 422 + `details.field`，前端按字段移除 chip、提示并重试一次。
- D4 系统节点无独立详情：以 `/system?category=&node=` 深链定位类别并滚动/选中节点。
- D5 前端状态固化：URL 查询参数 + `router.replace`，浏览器返回由初始参数恢复。

## 5. 实现说明与偏差

1. 设计系统无 TreeSelect/SegmentedControl：境界使用两级路径标签 `Select`，模式切换使用按钮组（`aria-pressed`）。
2. `/search` 为非默认页，需从导航进入；与设计一致。
3. 分页为全局 offset（`page`），未实现 cursor；固定 `score/updated_at/id` 排序降低漂移。
4. `review_status` 只读展示「待校对」，未实现校对写接口（延后 CHG-011）。
5. `faction_scope`/`system_category_id` 后端参数已支持，前端仅暴露 `system_node_id`（境界）与势力直属/含下级。
6. 启动时 `RebuildAll` 失败只记录告警，不阻断服务启动；可用 `cli reindex` 修复。
7. 检索相关新增/修改文件已 `gofmt`；仓库既有 CRLF 文件未批量格式化。

## 6. 复现命令

```powershell
cd backend
go vet ./...; go build ./...; go test -count=1 ./...
$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
go test -count=1 -tags=integration ./internal/integration/...

cd ..\frontend
npm run typecheck; npm run lint; npm run test

cd ..
docker compose build app; docker compose up -d app
cd frontend
npx playwright test --reporter=list
```
