# Change Proposal

- ID: CHG-009-search
- Title: 快速搜索与条件检索
- Status: Approved（2026-11-15，F1 按实施计划批准）
- Owner: 主 Agent
- Related Spec: SPEC-SEARCH-001、SPEC-SEARCH-002、SPEC-SEARCH-003、SPEC-SEARCH-004、SPEC-SEARCH-005
- Related Issue: roadmap §CHG-009；FR-27~FR-33、FR-62；US-007、US-008、US-012；PG-01；AC-01、AC-08、AC-23；EX-26、EX-27

## Problem / Opportunity

CHG-003～008 已交付体系、人物、势力、地点、榜单与回收站，但用户仍只能在各实体列表内分别搜索：没有一个入口能同时按名称、别名、称号、简介或关联实体名定位内容；条件组合（势力 + 境界 + 地点 + 榜单）只能在单页内近似筛选；跨类型结果无法统一分组、排序与跳转。`search_documents` 在数据模型与 API 契约中一直处于未实现状态，SPEC-SEARCH-001~005 全部为 `planned`。

## User Value

- 打开 `/search` 即可在当前作品内一次查到人物、势力、地点、榜单与体系节点。
- 输入别名或称号即可命中人物，并在结果中看到「名称/别名/称号/简介/关联实体」的命中原因与高亮。
- 条件检索可把势力、境界、地点、榜单与实体类型 AND 组合，快速缩小范围。
- 结果按类型分组、组头显示总命中、全局每页 20 条、翻页稳定；点击直达详情并可用浏览器返回恢复检索态。
- 回收站与其它作品的数据不会混入；无结果时不编造内容，错误可重试且不丢关键词。

## Goal

交付一条完整可走通的检索垂直切片：迁移 `0009` 建立 `search_documents`；后端提供同事务检索文档维护、`GET /api/v1/search` 查询与 `cli reindex`；前端提供 `/search` 快速/条件检索页、分组结果、高亮、分页与完整状态；全量门禁、集成测试、单测、E2E 与性能基线通过。

## In Scope

- 迁移 `0009_search_documents`：表、CHECK、唯一键、`(work_id, entity_type, is_deleted)` 索引与 5 个部分 GIN trgm 索引；补 `review_status` 列。
- 检索文档维护：人物、势力、地点、榜单、体系节点新增/编辑/改名/删除/恢复/整组删除/永久删除后，同一事务内重建该作品文档；启动时与 CLI 可全量重建。
- 一致性：按类型比较文档数与源实体数；`cli reindex --check` 可只读校验。
- 查询：`GET /api/v1/search` 支持 `work_id`、`q`、`entity_types`、`faction_id`、`faction_scope`、`system_category_id`、`system_node_id`、`location_id`、`ranking_id`、`review_status`、`page`。
- 语义：空格分词 AND；名称/别名/称号/简介/关联实体名命中；英文不区分大小写；不检索大事记正文；仅当前作品；回收站不参与。
- 结果：按人物→势力→地点→榜单→体系节点分组；组头总命中；全局每页 20；排序 `score DESC, updated_at DESC, id ASC`；`matched_fields`、`snippet`、`subtitle`、`url`。
- 前端：快速搜索（300ms 防抖、Enter 立即、IME 保护）、条件检索（AND、chip、重置）、分组结果、`<mark>` 高亮、分页、空查询/无作品/无结果/错误/加载态、URL 状态与返回恢复、a11y 与键盘。
- 导航与深链：`GlobalNav` 增加「检索」；`/system?category=&node=` 支持检索结果跳转。
- 文档：F10 回写 SPEC-SEARCH-001~005、PG-01、`api-contract.md` §3.9、`data-model.md` §3.15、roadmap 进度。

## Out of Scope

- 同义词、拼音、错字容忍、语义搜索、自然语言理解、大模型生成。
- 跨作品全局检索、检索历史持久化、离线缓存、保存的检索条件、布尔 OR/NOT、排序自定义。
- 大事记正文检索、人物-人物关系检索。
- 外部搜索引擎（Elasticsearch 等）与相关性机器学习排序。
- 检索结果缓存 HIT/MISS（本期 `X-Cache: BYPASS`）。
- 导入/导出（CHG-010/011）与关系一致性收口（CHG-012）。

## Acceptance Criteria

1. 打开 `/search` 输入框自动聚焦；空查询不发起请求，显示输入提示与示例。
2. 输入 1 个字符即可检索；300ms 防抖；Enter 立即检索；中文输入法组合期间不误触发。
3. 人物别名「小圣」可命中该人物，结果标注 `alias` 命中；输入 `SWORD` 可命中名称为 `sword` 的人物。
4. 输入「青丘 狐」时，两个词必须在同一实体命中；不因大事记正文命中人物。
5. 结果按人物/势力/地点/榜单/体系节点分组，组头显示组总命中；列表仅渲染当前页切片；无结果组不显示。
6. 每条结果显示命中字段标签与 `matched_text` 高亮；不伪造摘要；关联命中显示关联实体名。
7. 排序符合名称 4 > 别名/称号 3 > 简介 2 > 关联实体 1；同权重 `updated_at DESC`、`id ASC`。
8. 每页 20 条；第 2 页与第 1 页无重复漏项；翻页保留关键词、模式与筛选。
9. 条件检索 AND 组合；实体类型=全部/非人物时人物筛选隐藏；势力筛选可切换「直属/含下级」。
10. 失效筛选 ID 返回 422 字段级错误，前端移除该条件并提示；空查询 + 无收窄筛选返回 400。
11. 只返回当前作品数据；人物进入回收站后不命中，恢复后重新命中；改名后新名命中、旧名不命中。
12. 两本作品同名实体互不可见；作品删除/恢复/永久删除后检索文档无残留、无漂移。
13. 错误态显示中文提示 + 重试并保留输入；无堆栈/SQL/内部 ID；`X-Cache: BYPASS`。
14. 1000 人物、本机 Docker，关键词检索 P95 ≤150ms（20 次抽样）；首屏可交互 ≤2s。
15. 1024/1280 无横向溢出；axe 无 serious/critical；Tab/Enter/Esc 可达；结果行是链接；`<mark>` 不只靠颜色。
16. 点击结果到达对应详情（查询参数路由）；浏览器返回恢复关键词、筛选与页码；系统节点结果到达 `/system?category=&node=`。

## Constraints and Assumptions

- 技术栈不变：Go + Gin + GORM + PostgreSQL + goose；Next.js App Router 静态导出；不新增第三方依赖。
- 复用既有模式：`model.ValidationError`、`abortServiceError`、`normalizePage`、软删除三件套、`work_id` 隔离、`search_documents` 既定表结构（补 `review_status`）。
- `pg_trgm` 已在 0001 启用；检索走 PostgreSQL GIN/trgm，不引入外部索引。
- 检索文档采用「整作品重建」保证一致性；作品规模上限约 1500 文档，写侧可接受，若写 P95 不达标在 CHG-012 切换增量刷新。
- 现有设计系统没有 TreeSelect/SegmentedControl；本期用 `Select` + 层级路径标签、按钮组实现，F1 记录为设计系统简化。

## Dependencies

- CHG-003（体系类别/节点）、CHG-004（人物）、CHG-005（人物图片）、CHG-006（势力）、CHG-007（地点）、CHG-008（榜单）。
- CHG-002（作品隔离、回收站、撤销、适配器基座）。

## Risks

| 风险 | 影响 | 缓解 |
|---|---|---|
| 整作品重建写放大 | 写请求变慢、锁竞争 | 集合式 SQL + `DO UPDATE … WHERE`；记录写侧观察；预留增量刷新回滚路径 |
| 多别名×多关系行放大 | 同键重复、relation_text 膨胀 | 关联先用聚合子查询/`LATERAL` 收敛为一行；集成用例覆盖多关系 |
| 分页在数据更新时漂移 | 跨页重复/漏项 | 固定 `score/updated_at/id` 排序；E2E 覆盖翻页；F10 同步 Spec 已知限制 |
| 失效筛选语义冲突 | 静默忽略或前端死循环 | 后端 422 字段级 details；前端移除 chip 后重试一次并 Toast |
| 系统节点无详情路由 | 结果死链 | 实现 `/system?category=&node=` 深链并纳入 E2E |
| 旧数据升级后文档缺失 | 升级后检索为空 | 启动时 `RebuildAll`；`cli reindex` 可手动修复 |
