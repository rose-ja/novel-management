# ADR-003: PostgreSQL 中文检索与回退

- Status: Accepted（2026-11-07；V-01 验证前不得视为性能已达标）
- Date: 2026-11-07
- Decision owner: 项目发起人（已确认 2026-11-07）；技术方案：架构师

## Context

检索是本项目的最高优先能力（constitution §0）。`FR-27~33` 要求：

- 空格分隔多词 AND；
- 命中人物/势力/地点的名称、别名、称号、简介；人物/势力/地点还需命中关联实体名称；
- 结果按实体类型分组；
- 名称优先于别名/称号，再简介，再关联名；
- 单作品 1000 人物规模下 P95 ≤150ms；
- 不搜大事记；不生成内容；无结果明确提示。

P2 基线（v0.1）曾选用 SQLite FTS5 + `instr` 短词回退；技术栈切换为 PostgreSQL 16 后重新决策。

## Decision Drivers

- 中文无空格，用户常输入 2 字（如「林岚」「剑宗」），需要子串匹配。
- 1000 人物量级不需要外部搜索引擎。
- 零成本、无额外服务、离线运行。
- 实现可解释、可调试，失败可替换。
- 必须能在 Go/gorm 中维护索引。

## Options Considered

| 方案 | 原理 | 优点 | 代价 |
|---|---|---|---|
| A. `pg_trgm` GIN + 短词 `ILIKE`（选用） | 三元组索引加速 `ILIKE '%词%'`；1–2 字顺序扫描 | 无额外服务；PG 原生；维护简单 | 1–2 字无法用索引；需 V-01 验证 1500 行内延迟 |
| B. `pg_bigm` | 二元组索引，适合中日韩短词 | 2 字可走索引 | 需编译扩展；镜像与运维复杂度增加 |
| C. `zhparser` + `tsvector` | 中文分词 + 全文检索 | 词级检索、排序成熟 | 分词词典与配置复杂；2 字子串/别名匹配不自然 |
| D. 外部搜索引擎（Meilisearch 等） | 独立索引服务 | 检索体验最好 | 引入第四容器与同步一致性；违反零成本/简单原则 |
| E. 纯 `ILIKE` 无索引 | 顺序扫描 | 最简单 | 1000 人物 + 关联文本下不保证 150ms |

## Decision

采用 **A**：

- `search_documents` 表按实体维护 `name_text`、`name_key`、`alias_text`、`title_text`、`desc_text`、`relation_text`、`is_deleted`。
- 对 5 个文本列建立 `GIN (… gin_trgm_ops)` 部分索引（`WHERE is_deleted = false`）。
- 查询：空格分词 → 每词 `ILIKE '%token%'` → 各字段 AND；3 字符以上由 trigram 索引加速；1–2 字走同一 SQL（预计顺序扫描）。
- 排序：名称命中 4 > 别名/称号 3 > 简介 2 > 关联名 1；同权重 `updated_at DESC`。
- 索引维护：写事务内由 `search.Refresher` 重建受影响文档；被引用实体改名时刷新引用方；`cmd/cli reindex` 全量重建。
- 缓存：检索结果可缓存 60s，key 含作品 epoch（ADR-007）；缓存不改变检索语义。

## Consequences

- 正面：不引入额外服务；PG 原生；索引与业务表同库同事务，无同步延迟；实现可替换（`SearchService` 接口）。
- 负面：1–2 字查询可能全表扫描（1500 行内可接受，需 V-01 证实）；`pg_trgm` 对中文连续文本的召回率需用预设查询集验证；`ILIKE` 对大小写不敏感但与 `name_key` 归一策略需一致。
- **阻断条件**：V-01 未通过前，本 ADR 保持 Proposed，不得宣称「检索已达标」，不得在文档/界面写「P95 已验证」。

## Risks and Mitigations

| 风险 | 缓解 |
|---|---|
| 2 字查询 P95 >150ms | V-01 加入 40% 短词查询；失败则评估 `pg_bigm`（方案 B） |
| 关联名文本膨胀导致索引大 | `relation_text` 只存名称，不存简介；定期 `VACUUM`/观察索引大小 |
| `pg_trgm` 扩展不可用（非超级用户） | 本地 compose 用户为超级用户；启动时 `CREATE EXTENSION IF NOT EXISTS`，失败则退出并提示 |
| 索引与实体不同步 | 所有写操作经 service；集成测试断言文档与实体一致；`reindex` 兜底 |
| 排序不稳定 | 同权重按 `updated_at DESC, entity_id ASC` 稳定排序 |

## Follow-up Actions

- [ ] 实现 V-01 数据集与 30 条预设查询集。
- [ ] 记录 P50/P95/P99、索引大小、召回率、空结果率。
- [ ] 未达标时写 ADR-003 修订或新 ADR，评估 `pg_bigm`/`zhparser`。
- [ ] V-01 通过后，在 architecture D-02 标注「性能已验证」。
