# ADR-005: 导入原子性、临时目录与后台任务

- Status: Accepted（2026-11-07）
- Date: 2026-11-07
- Decision owner: 项目发起人；技术方案：架构师
- Database: PostgreSQL 16

## Context

`FR-39~46` 要求从飞书迁移：上传 zip（Markdown + 图片，≤6GB）、解析模板 v1.0、校验（重复名、缺失引用、缺图、未知类别/节点）、预览计数与错误清单、确认后原子写入、图片绑定、报告、重复提交幂等。验收要求失败时 0 残留（FR-43、AC-08），导入 ≤90s（1000 人物 + 500 图片，NFR §7）。

本版无消息队列；后台任务必须本地可运行、可观测、可恢复。

## Decision Drivers

- 原子性优先于吞吐：任何错误不得产生半成品。
- 单用户、低并发，不需要分布式任务队列。
- 大文件解析不得阻塞 HTTP 请求。
- 失败必须可定位（文件/行/字段），且可修正后重跑。
- 图片与数据库最终一致。

## Options Considered

| 方案 | 说明 | 优点 | 缺点 |
|---|---|---|---|
| A. 请求内同步导入 | HTTP 请求内完成全部解析与写入 | 简单 | 超时、无法预览、内存峰值高 |
| B. 逐实体提交 | 每解析一条提交一次 | 失败保留部分数据 | 违反原子性；回滚复杂 |
| C. 临时库/暂存表 + 切换 | 先写暂存表，成功后合并 | 隔离好 | 需要额外 schema/清理；当前规模过度 |
| D. 进程内单 worker + 单事务（选用） | 上传后异步解析，确认后单事务写入 | 原子、简单、可观测 | 大事务内存与锁；进程重启需状态恢复 |
| E. 外部队列 + worker | Redis Stream/BullMQ 等 | 可扩展 | 引入无需求基础设施；Redis 变关键路径 |

## Decision

采用 **D**：

1. **上传与预览分离**：`POST /imports` 保存 zip 到 `/data/tmp/imports/{import_id}/`，创建 `import_runs(status=uploaded)`，返回 `202`；`POST /imports/{id}/validate` 异步解析生成预览/错误；`POST /imports/{id}/commit` 在单事务内写入。
2. **安全解压**：拒绝绝对路径、`..`、符号链接、超数量、超大小、压缩炸弹比率；解压目录按 `import_id` 隔离。
3. **解析与校验**：`internal/imports/parser.go` 纯函数解析 Markdown 模板；`internal/domain` 校验名称、类别、节点、引用、席位、图片；错误含 `file/line/field/code`。
4. **提交**：单 goroutine worker（全局串行，同一时间仅一个导入）；提交前生成快照（`reason=import`）；在 PostgreSQL 单事务内按固定顺序批量插入（`CreateInBatches`，每批 200）；图片在事务前移动到 `images/{work_id}/{import_id}/`，事务失败则删除本次移动的文件；事务提交后更新 `import_runs=completed` 并把 `review_status=pending`。
5. **幂等**：`commit` 对同一 `import_id` 重复调用返回首次结果（`200` + 相同 counts/status）；`status` 非 `validated` 时返回 `409 IMPORT_STATE_CONFLICT`。
6. **失败与重启**：任何错误 → 事务回滚 + 清理临时目录 + `status=failed` + 错误清单；进程启动扫描 `validating/running` → `failed`，临时目录按 TTL（120 分钟）清理。
7. **限制**：同一时间仅一个导入；与导出互斥（`409 IMPORT_IN_PROGRESS` / `EXPORT_IN_PROGRESS`）；解析内存上限按文件大小分块读取，图片单张 ≤10MB。

## Consequences

- 正面：失败 0 残留；预览与提交一致；无外部队列；状态可查询、可重试；1000 人物在 PG 单事务内开销可接受（V-02 验证）。
- 负面：单事务持有时间较长，期间该作品写操作可能等待；进程重启会使运行中任务失败（需用户重跑，已有快照保护）。
- 约束：导入期间禁止其他写事务修改同一作品（service 级锁 + advisory lock）；不得在事务内做图片重编码等耗时操作。

## Risks and Mitigations

| 风险 | 缓解 |
|---|---|
| 大事务内存/锁竞争 | 分块解析、批量插入、单事务仅写；V-02 测量；必要时缩短事务（先图片后 DB） |
| 图片移动后事务失败 | 记录本次移动清单，失败逆序删除；集成测试覆盖 |
| 重复 commit 产生两套数据 | 幂等键 = `import_id` + 状态机 + 唯一导入锁 |
| 恶意 zip | 路径/符号链接/大小/比率校验；独立安全测试 |
| 进程重启 | 启动扫描 running→failed；临时目录 TTL；快照可回退 |
| 模板解析歧义 | 固定模板 v1.0 + 示例 + 错误行号；V-06 |

## Follow-up Actions

- [x] 项目发起人确认 D-04（2026-11-07）。
- [ ] V-02/V-06 验证性能与错误定位。
- [ ] 集成测试覆盖失败回滚、图片移动回滚、重复 commit。
