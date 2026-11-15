# CHG-010 独立审查与修复记录（F4/F5）

- 审查范围：`3f34dee..f9429ae`（CHG-010 全部改动）
- 审查方式：独立只读审查（非实现者视角），随后由实现者逐条修复并回归
- 结论：无 P0；P1 已修复；P2 已处理或记录为已知限制

## 1. P1 问题与处置

| # | 问题 | 处置 | 验证 |
|---|---|---|---|
| 1 | 不含回收站导出删除软删除人物时，`factions.leader_character_id`（RESTRICT）可能阻断 | 清理顺序中加入 `UPDATE factions SET leader_character_id=NULL` | 集成 + Docker 恢复演练 PASS |
| 2 | 快照 `entity_counts` 未过滤 `deleted_at`，与恢复后活跃计数核对不一致 | `entityCounts` 增加 `deleted_at IS NULL` | 集成 PASS |
| 3 | 旧 schema（<10）包恢复后 `export_runs` 残留，goose 0010 建表冲突 | 恢复后、迁移前按版本删除新增表（`dropTablesNewerThan`） | 单测/集成 PASS；真实恢复 schema v10 正常 |
| 4 | 先解压后校验，恶意 zip 可耗尽磁盘 | 新增条目/解压总量上限；恢复前从 zip 内只读 manifest 并做磁盘预检 | `TestReadZipFile`、恢复单测 PASS |
| 5 | 含口令 DSN 作为 `-d` 参数暴露在进程列表 | 改为 `PGHOST/PGPORT/PGUSER/PGPASSWORD/PGSSLMODE` 环境变量 + `-d <db>` | `TestPGEnvKeepsPasswordOutOfArguments` PASS |
| 6 | DB 恢复成功后的非事务步骤失败只能手工回滚 | 保留恢复前快照、错误输出含回滚命令；记录为已知限制 | 恢复演练 + 文档 |

## 2. P2 问题与处置

| # | 问题 | 处置 |
|---|---|---|
| 1 | 导出并发仅靠 `app_settings` 行锁 | 单用户场景下事务行锁已串行化创建；未新增部分唯一索引，记录为可接受风险 |
| 2 | 进程崩溃可能遗留临时库 | 临时库以导出 ID 命名；DROP 失败写 stderr；启动清理未实现，记录为运维项 |
| 3 | `EXPORT_INSUFFICIENT_STORAGE` 未产生 | 保留错误码供后续磁盘预检；导出自身不做空间预检（恢复侧已做） |
| 4 | 缺失图片静默跳过 | 记录 Warn 日志；manifest 以实际复制文件计数 |
| 5 | 前端轮询依赖 `run` 对象导致定时器重建 | effect 改为依赖 `run.id` 与 `phase` |
| 6 | 测试盲区 | 新增 zip 读取、pgEnv 单测；真实 pg_dump/临时库过滤在 Docker 演练覆盖 |

## 3. 复审结果

- 修复后重新执行 F6 全量回归：后端单测、集成、前端类型/lint/单测、Docker 构建、Playwright 109 用例全部 PASS。
- 未发现新增 P0/P1。
