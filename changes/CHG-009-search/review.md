# CHG-009 独立审查与修复记录（F4/F5）

- Change ID：`CHG-009-search`
- 日期：2026-11-15
- 审查方式：未参与实现的独立审查代理，对照 `design.md`、`tasks.md`、`checklist.md`、SPEC-SEARCH 与 API 契约阅读 `2d428cb..HEAD` 全部变更
- 结论：0 个 S0；3 个 S1 与 5 个 S2 已处理，其余 2 项记录为已确认偏差；F6 全量回归通过

## 1. 审查发现与处置

| 编号 | 级别 | 发现 | 处置 | 状态 |
|---|---|---|---|---|
| R1 | S1 | `page` 解析错误被忽略；超大整数使 `(page-1)*20` 溢出并切片 panic；`page=abc/0` 返回 200 | HTTP 层校验 `page` 为正整数（非法 422 `VALIDATION_ERROR`，field=page）；服务层改为溢出安全的起点计算，超出末页返回空切片 | 已修复并加测试 |
| R2 | S1 | 同作品并发写事务各自整作品刷新，`ON CONFLICT DO UPDATE` 可能用旧快照覆盖新文本，并有交叉行锁死锁风险 | `RefreshWork` 开头加作品级 `pg_advisory_xact_lock(hashtext(work_id))`，串行化同作品刷新；锁顺序为实体/作品行锁 → 刷新 advisory，无反向获取 | 已修复 |
| R3 | S1 | 条件模式仅输入关键词时 `submitDisabled` 恒为 true，用户无法点击「检索」 | `submitDisabled` 同时考虑输入框待提交关键词；新增 E2E「仅输入关键词可提交检索」 | 已修复并加测试 |
| R4 | S2 | 人物 `subtitle` 缺「人物 · 」前缀，与 design §3.3/契约示例不一致 | `characterSubtitle` 固定以「人物」开头；集成断言改为前缀校验 | 已修复 |
| R5 | S2 | 前端 URL 使用 `realm` 而非 design §4.2 的 `category/node`，且未暴露 `review` 筛选 | 记录为已确认偏差：前端以 `realm=<system_node_id>` 表达境界节点；`review_status` 仅后端支持，PG-01 §4.4 未要求 UI；F10 回写 design 与 PG-01 时同步说明 | 已确认偏差 |
| R6 | S2 | 作品软删除未刷新文档，`is_deleted` 与 `works.deleted_at` 语义漂移（检索因 404 不泄漏） | `Works.Delete` 事务末尾调用 `RefreshWork`；集成用例覆盖作品删除后全部文档 `is_deleted=true`、恢复后还原 | 已修复并加测试 |
| R7 | S2 | 查询在 Go 内排序/分页，偏离 D3「SQL 内完成评分排序」；`COALESCE(...) ILIKE` 无法命中 trgm 索引 | 记录为实现偏差：1000 人物 P95 7.11ms，满足 AC 14；保持实现简单与可验证，F10 在 design/Spec 记录已知限制 | 已确认偏差 |
| R8 | S2 | 非首页中「组总命中>0 但当前页无条目」的类型仍渲染空组头 | 服务层仅返回当前页有条目的分组；集成用例覆盖 25 人物 + 1 势力的第 2 页只返回人物组 | 已修复并加测试 |
| R9 | S2 | 仅筛选时 `query.tokens` 序列化为 `null` | 空分词归一为 `[]string{}`；集成断言 `tokens` 为空数组 | 已修复 |
| R10 | S2 | 测试缺口：非法 page、条件模式纯关键词、作品删除、空分组、非法 review_status | 已在 `search_api_test.go`、`search_trash_test.go`、`e2e/search.spec.ts` 补齐 | 已补齐 |

## 2. 复审结论

- 事务与锁：人物/地点/势力/榜单创建先锁作品行；刷新统一再取作品级 advisory，避免旧快照覆盖；回收站恢复/永久删除仍在事务内刷新。
- 文档一致性：软删除、恢复、物理删除、作品删除/恢复、清空回收站后 `drift=0`；多别名/多关系聚合无膨胀。
- 查询：空 400、非法参数 422、失效筛选 422、作品隔离、回收站排除、token AND、英文大小写、组 total、当前页切片、`X-Cache: BYPASS` 均符合契约。
- 前端：URL 往返、返回恢复、防抖/Enter/IME/Esc、失效筛选移除重试、错误重试、空态与深链均已覆盖。
- 性能：1000 人物 100 次抽样 P95 7.11ms；首屏可交互 200ms。

## 3. 修复后回归证据

- 后端：`go vet ./...`、`go build ./...`、`go test -count=1 ./...` PASS；集成全量 PASS（20.0s）。
- 前端：`npm run typecheck`、`npm run lint`、`npm run test`（86 PASS）PASS。
- 构建：`docker compose build app`（node:22-alpine）PASS。
- E2E：`npx playwright test` **105 PASS**（含检索 7 条、a11y 12 条、键盘 7 条、视觉 20 张）。
