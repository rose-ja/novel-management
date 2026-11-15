# CHG-009 设计

- Change ID：`CHG-009-search`
- 依据：`proposal.md`、`project/specs/search/specs.md`、`project/page-specs/PG-01-search.md`、`project/api-contract.md` §2.7/§3.9、`project/data-model.md` §3.15
- 目标：在不新增第三方依赖的前提下，交付当前作品内的统一检索与条件筛选。

## 1. 设计决策

| ID | 决策 | 理由 |
|---|---|---|
| D1 | 使用 `search_documents` 反规范化文档 + `pg_trgm` GIN 索引 | 与 data-model/契约一致；中文子串与英文大小写统一用 `ILIKE` |
| D2 | 写事务内 `Refresher.RefreshWork` 整作品重建 | 保证 SEARCH-005 的「不允许长期漂移」；实现简单、可验证；规模 ≤1500 文档 |
| D3 | 查询在 SQL 内完成命中、评分、排序、分组计数 | 避免拉全量到 Go；GIN 索引可用；分页稳定 |
| D4 | 结果 DTO 由服务层按当前页批量装配 subtitle/thumbnail | 禁止 N+1；保持 `search_documents` 表结构最小 |
| D5 | 检索态固化在 URL 查询参数 | 静态导出兼容；浏览器返回可恢复；结果链接保持契约的查询参数路由 |
| D6 | 条件筛选只对适用类型生效；失效 ID 返回 422 | 避免静默忽略；前端可移除 chip 后重试 |
| D7 | 系统节点结果跳 `/system?category=&node=` | 复用现有 `/system` 路由；不新增页面 |
| D8 | 启动时 `RebuildAll`，并提供 `cli reindex` | 兼容 v8 升级与导入/恢复后的修复入口 |

## 2. 数据设计（迁移 0009）

```sql
CREATE TABLE search_documents (
  id TEXT(26) PRIMARY KEY,
  work_id TEXT(26) NOT NULL REFERENCES works(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL CHECK (entity_type IN ('character','faction','location','ranking','system_node')),
  entity_id TEXT(26) NOT NULL,
  name_text TEXT NOT NULL,
  name_key TEXT NOT NULL,
  alias_text TEXT,
  title_text TEXT,
  desc_text TEXT,
  relation_text TEXT,
  review_status TEXT NOT NULL DEFAULT 'reviewed' CHECK (review_status IN ('reviewed','pending')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  is_deleted BOOLEAN NOT NULL DEFAULT false,
  UNIQUE (work_id, entity_type, entity_id)
);
```

- 索引：`(work_id, entity_type, is_deleted)`；`name_text/alias_text/title_text/desc_text/relation_text` 各自 `GIN (… gin_trgm_ops) WHERE is_deleted = false`。
- `id` 由 SQL `substr(replace(gen_random_uuid()::text,'-',''),1,26)` 生成，仅内部使用。
- 0001 已 `CREATE EXTENSION IF NOT EXISTS pg_trgm`，0009 不重复。

### 2.1 文档内容

| 类型 | name_text | alias/title | desc_text | relation_text |
|---|---|---|---|---|
| character | 主名称 | 别名/称号空格拼接 | 简介 | 活跃势力、地点、榜单、体系节点（含父节点与类别名） |
| faction | 名称 | — | 简介 | 活跃上级势力、首领人物 |
| location | 名称 | — | 类型 + 简介 | 活跃上级地点、关联人物、关联势力 |
| ranking | 名称 | — | — | 在榜活跃人物 |
| system_node | 节点名称 | — | — | 类别名称、活跃父节点 |

- 关联名只取活跃实体；软删除实体置 `is_deleted=true`，不参与查询。
- `review_status` 同步源实体 DB 值；API 输出时映射 `reviewed→confirmed`。

### 2.2 刷新策略

- `Refresher.RefreshWork(ctx, tx, workID)`：
  1. 5 条 `INSERT … SELECT … ON CONFLICT (work_id,entity_type,entity_id) DO UPDATE`；
  2. 每类关联先聚合为一行（`string_agg(DISTINCT … ORDER BY …)` 或 `LATERAL`），避免笛卡尔积；
  3. `DO UPDATE … WHERE` 仅在文本/is_deleted/review_status 变化时更新；
  4. `DELETE` 清理源实体已物理不存在的同作品文档。
- 接入点见 `tasks.md` T3~T5。
- `RebuildAll(ctx)`：按 work 分批调用 `RefreshWork`；启动时调用一次，失败记录日志不阻断。
- `Consistency(ctx, workID)`：按类型比较 `search_documents` 与源实体计数，返回 `total_docs/total_entities/drift`。

## 3. 查询设计

### 3.1 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `work_id` | 是 | 当前作品；不存在/已删除返回 404 |
| `q` | 否 | trim 后 ≤200 rune；空且无收窄筛选 → 400 |
| `entity_types` | 否 | 逗号分隔五类；默认全部；非法值 422 |
| `faction_id` | 否 | 人物直属势力或子树（配合 `faction_scope`） |
| `faction_scope` | 否 | `direct`（默认）/`subtree` |
| `system_category_id` | 否 | 人物引用了该类别任一节点 |
| `system_node_id` | 否 | 人物引用该节点或其直接子节点 |
| `location_id` | 否 | 人物-地点关系 |
| `ranking_id` | 否 | 人物在榜单占席 |
| `review_status` | 否 | `pending` / `confirmed`，五类生效 |
| `page` | 否 | 默认 1；固定 20/页 |

- 收窄筛选：`faction_id/system_category_id/system_node_id/location_id/ranking_id/review_status` 任一非空；仅 `entity_types` 不算。
- 筛选 ID 校验：存在、同作品、未删除；失败 `422 VALIDATION_ERROR`，details `{field,value}`。

### 3.2 匹配与评分

- `tokens = strings.Fields(q)`；每个 token 对 `name_text/alias_text/title_text/desc_text/relation_text` 任一 `ILIKE '%escaped%' ESCAPE '\'`。
- 评分：`name` 4；`alias|title` 3；`desc` 2；`relation` 1；最终 `score DESC, updated_at DESC, id ASC`。
- 过滤 `work_id`、`is_deleted=false`、`entity_type ∈ selected`、候选筛选适用条件。
- 组总数用 `COUNT(*) OVER (PARTITION BY entity_type)` 或一次分组查询；全局 `total` 为组总数之和。

### 3.3 响应 DTO（契约 §2.7 + 扩展）

```json
{
  "groups": [{ "entity_type": "character", "total": 12, "items": [] }],
  "total": 12,
  "page": 1,
  "page_size": 20,
  "query": { "q": "山海 帝君", "tokens": ["山海","帝君"], "filters": { "entity_types": ["character"], "faction_id": null } }
}
```

- item：`entity_id/entity_type/title/subtitle/matched_fields[{field,matched_text}]/snippet/score/updated_at/url/review_status`；人物额外 `image_thumb_url`。
- `matched_text` 取字段内首个命中窗口（≤60 rune，超长加省略号）；`snippet` 取最高权重命中字段，格式 `名称：…`。
- `subtitle`：人物「人物 · 势力 · 境界摘要」；势力「势力 · 上级/首领」；地点「地点 · 类型 · 上级」；榜单「榜单 · N 席 · M 人在榜」；节点「体系节点 · 类别 · 父级」。
- `url`：`/characters?id=`、`/factions?id=`、`/locations?id=`、`/rankings?id=`、`/system?category=<category_id>&node=<node_id>`。
- 响应头 `X-Cache: BYPASS`。

### 3.4 错误

| 场景 | 状态 | code |
|---|---|---|
| `work_id` 缺失 | 422 | `VALIDATION_ERROR`（field=work_id） |
| 作品不存在/已删除 | 404 | `NOT_FOUND` |
| 空 q 且无收窄筛选 | 400 | `VALIDATION_ERROR` |
| 非法 `entity_types/faction_scope/review_status/page` | 422 | `VALIDATION_ERROR` |
| 失效筛选 ID | 422 | `VALIDATION_ERROR`（field=对应参数） |

## 4. 前端设计

### 4.1 文件

- `frontend/app/search/page.tsx`：Suspense + `SearchRoute`。
- `frontend/components/search/search-route.tsx`：读取 URL 参数并渲染页面。
- `frontend/components/search/search-page.tsx`：模式、请求、状态、结果。
- `frontend/components/search/search-mode-switch.tsx`：快速/条件按钮组。
- `frontend/components/search/quick-search-bar.tsx`：44px 输入、清除、自动聚焦。
- `frontend/components/search/conditional-filter-panel.tsx`：实体类型、势力/分支、境界、地点、榜单、chip、重置/检索。
- `frontend/components/search/search-result-group.tsx`、`search-result-row.tsx`、`matched-snippet.tsx`。
- `frontend/lib/api/search.ts`、`frontend/lib/search.ts`（URL 状态纯函数）。
- 修改 `frontend/lib/messages.ts`、`frontend/components/shell/global-nav.tsx`、`frontend/styles/components.css`。

### 4.2 状态与交互

- 快速模式：输入即搜；300ms 防抖；Enter 立即；`compositionstart/end` 期间不触发；Esc 清空。
- 条件模式：切换不清空关键词；筛选只对人物生效；实体类型=全部时隐藏人物筛选；筛选保留但不生效（PG01-Q2）。
- 请求：AbortController + 自增序号，过期响应丢弃；加载时 Skeleton 6 行；错误 Alert + 重试。
- URL：`q/mode/type/faction/scope/category/node/location/ranking/review/page` 与状态双向同步；浏览器返回恢复。
- 结果：整行 `<Link>`；`<mark>` 渲染后端 `matched_text`；组头显示组总命中；分页保留状态并滚动到结果区顶部。
- 状态：无作品页面级空态（搜索框禁用）；空查询提示；无结果 + 清除筛选；错误；加载。
- a11y：可见标签、`section+aria-labelledby`、`aria-live` 总数、`aria-busy`、chip aria-label、键盘可达。

### 4.3 设计系统简化

- 现有组件库无 TreeSelect/SegmentedControl：势力/境界/地点用 `Select` + 层级路径标签（如 `青丘 / 狐族`），模式切换用按钮组（`role=group` + `aria-pressed`）。
- 检索页默认路由仍为 `/` 概览；`/search` 作为导航入口。PG-01「默认页」在 F10 按现状同步说明。

## 5. 接口与数据影响

- 新增端点 `GET /api/v1/search`；不修改既有端点。
- 新增表 `search_documents`；不修改既有表结构。
- 新增依赖注入：`httpapi.Options.Search`、`service.NewSearch`；启动时 `RebuildAll`。
- CLI 新增 `reindex`（`--check`）。

## 6. 测试策略

- 单元：分词/转义、评分、snippet、筛选适用性、URL 状态。
- 集成：命中矩阵、AND、隔离、改名/删除/恢复、筛选、分页、错误码、一致性、迁移。
- E2E：快速/条件检索、分组、高亮、跳转/返回、空/无结果/错误、a11y、键盘。
- 性能：1000 人物 20 次抽样 P95 ≤150ms，记录于 `verification.md`。

## 6.5 门禁

- 后端：`gofmt -l .`、`go vet ./...`、`go build ./...`、`go test -count=1 ./...`
- 集成：`TEST_DATABASE_URL=… go test -count=1 -tags=integration ./internal/integration/...`
- 前端：`npm run typecheck`、`npm run lint`、`npm run test`
- 构建：`npx -y node@22 node_modules/next/dist/bin/next build`
- E2E：`docker compose build app; docker compose up -d app; npx playwright test --reporter=list`

## 7. 风险与回滚

- 写放大：记录写侧观察；不达标时切换 `RefreshEntity` 增量刷新（不改表结构）。
- 分页漂移：固定排序键；F10 在 Spec 记录已知限制。
- 回滚：回退代码与 `0009` down；检索文档可从源表 `RebuildAll` 重建，不影响业务数据。

## 8. 变更历史

| 日期 | 版本 | 说明 |
|---|---|---|
| 2026-11-15 | v0.1 | F1 初版：迁移、刷新、查询、前端、CLI 与测试策略 |
