# ADR-007: Redis 仅缓存与非关键路径

- Status: Accepted（2026-11-07）
- Date: 2026-11-07
- Decision owner: 项目发起人；技术方案：架构师

## Context

项目发起人要求技术栈包含 Redis（学习路线），并明确「方案一基础上加入 Redis 做缓存」。本项目为单用户、本地运行、单作品 1000 人物规模，PostgreSQL 已能承担查询；Redis 不是数据源。需要明确 Redis 的职责边界，避免其成为关键路径或事实来源。

## Decision Drivers

- 用户明确要求使用 Redis，但不得增加数据不一致风险。
- 检索 P95 ≤150ms，缓存命中可降低重复查询开销。
- 零成本、离线、本地 Docker。
- Redis 故障时系统必须继续可用（QA-07）。
- 不引入无需求的队列/会话/分布式锁。

## Options Considered

| 方案 | 说明 | 优点 | 代价 |
|---|---|---|---|
| A. 不使用 Redis | 仅 PostgreSQL | 最简单、少一个容器 | 不符合用户学习路线要求 |
| B. Redis 作为缓存 | cache-aside 检索结果与作品摘要 | 降低重复查询；可练习缓存模式 | 需失效策略；故障需旁路 |
| C. Redis 作为队列 | 导入/导出任务入队 | 任务解耦 | Redis 变关键路径；重启丢任务；状态一致性复杂 |
| D. Redis 作为会话/主存储 | 存业务数据 | 读写快 | 违反「单一可信数据源」；持久化/备份复杂 |

## Decision

采用 **B，仅缓存**：

1. **缓存对象**：`GET /api/v1/search` 结果页（按参数）与作品摘要计数（`GET /api/v1/works/{id}/summary` 等只读聚合）。
2. **键格式**（版本前缀，便于整体失效）：
   - `nm:v1:{work_id}:epoch` → 整数，作品写操作 `INCR`；
   - `nm:v1:{work_id}:{epoch}:search:{params_hash}` → 检索结果（TTL 60s）；
   - `nm:v1:{work_id}:{epoch}:summary` → 作品摘要（TTL 60s）。
3. **失效**：任何写操作（实体 CRUD、关系、席位、导入、删除/恢复、回收站清空）在同一 service 事务提交后 `INCR epoch`；旧 epoch 键自然过期，不做扫描删除。
4. **TTL**：默认 60s（`CACHE_TTL_SEC`，上限 300）；即使 epoch 失效失败，TTL 兜底保证最终一致。
5. **降级**：`Cache` 接口含 `RedisCache` 与 `NoopCache`；连接失败/超时（≤50ms）→ 直接回源 PostgreSQL，响应头 `X-Cache: BYPASS`；不重试、不阻塞请求。
6. **部署**：独立 `cache` 容器；`--save "" --appendonly no`；`maxmemory 256mb`、`allkeys-lru`；不映射宿主端口；不设口令（本地）。
7. **禁止**：不用 Redis 做队列、分布式锁、会话、唯一 ID、业务数据存储、跨实例协调；不缓存个人信息以外的敏感正文（仅缓存名称/摘要/ID 等检索结果所需字段）。
8. **可观测**：health 返回 `cache: ok|degraded|disabled`；日志记录 `cache_status`，不记录键值内容。

## Consequences

- 正面：满足用户对 Redis 的要求；检索重复查询更快；Redis 故障不影响正确性；缓存可随时清空。
- 负面：多一个容器与失效逻辑；缓存与数据库短暂不一致（≤60s 或至 epoch 更新）；测试需覆盖 HIT/MISS/BYPASS。
- 约束：任何代码不得把缓存读取放在写事务或正确性判断中；缓存值必须可重建。

## Risks and Mitigations

| 风险 | 缓解 |
|---|---|
| epoch 失效遗漏 | 所有写操作统一在 service 层调用 `cache.BumpEpoch`；集成测试断言 |
| Redis 故障拖慢请求 | 短超时 + 旁路 + 熔断（连续失败 3 次后 30s 内直接 BYPASS） |
| 缓存脏数据 | TTL 60s + epoch；缓存值只含展示字段，不作为校验依据 |
| 内存不足 | maxmemory 256mb + LRU；键数量上限估算（1000 人物 × 查询组合） |
| 被误用为队列 | ADR 明确禁止；代码评审；不引入 Redis Stream 依赖 |

## Follow-up Actions

- [x] 项目发起人确认 D-08（2026-11-07）。
- [ ] V-07 验证停止 Redis 后检索/保存可用。
- [ ] 集成测试覆盖 HIT/MISS/BYPASS 与 epoch 失效。
- [ ] 监控缓存命中率（日志聚合，不引入 APM）。
