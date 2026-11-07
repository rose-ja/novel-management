# ADR-006: 导出包与 CLI 全量恢复

- Status: 修订（P5b，2026-11-07，待复审）；Rev.2
- Date: 2026-11-07
- Decision owner: 项目发起人；技术方案：架构师
- Supersedes: Rev.1（Accepted，2026-11-07）
- Database: PostgreSQL 16

## Context

`FR-52~56` 要求全量导出（含/不含回收站两种模式）、自动快照、CLI 恢复、图片不丢失、恢复前快照、空环境恢复演练。`FR-56` 与 constitution §9 要求迁移前/公网部署前完成可恢复备份并演练。

技术栈为 PostgreSQL 16，运行环境为 Docker Desktop；无云存储、无对象存储。

## Decision Drivers

- 恢复必须可靠、可验证、可重复。
- 导出包应包含数据库与图片，且可离线保存。
- 不含回收站的导出必须**真的不含**回收站数据。
- 恢复失败不得破坏原数据。
- 不引入额外备份服务或云依赖。

## Options Considered

| 方案 | 说明 | 优点 | 缺点 |
|---|---|---|---|
| A. 直接复制 `pgdata` 目录 | 文件级复制 | 快 | 一致性无保证；版本/平台耦合；禁止 |
| B. 应用层 JSON 导出 | 逐表查询写 JSON，恢复时插入 | 可控过滤 | 需实现恢复器、顺序/类型/引用复杂；易漏 |
| C. `pg_dump -Fc` + 图片（选用） | 数据库自定义转储 + 图片目录 + manifest | 一致、成熟、可用 `pg_restore` | 恢复需停写；不含回收站需临时库过滤 |
| D. 外部备份服务 | 云/对象存储 | 自动 | 违反零成本与隐私 |

## Decision

采用 **C**：

1. **快照**：`SnapshotService` 执行 `pg_dump -Fc --no-owner --no-acl` 到 `snapshots/{ts}-{reason}/app.dump`；图片通过同 volume 硬链接（失败降级复制）；写入 `snapshot.json`、`manifest.json`、`checksums.sha256`；保留最近 3 份。
   - 快照目录**固定**包含：`app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`（P5b RT-06）。
   - **触发矩阵（以 requirements FR-55 为准，P5b RT-17）**：CLI 全量恢复、导入、删除作品、永久删除、清空回收站生成快照；回收站单项恢复不生成；手动快照仅 CLI 可选（`cli snapshot`），无页面入口。
2. **导出包**：
   ```text
   novel-settings-YYYYMMDD-HHmmss.zip
   ├─ manifest.json      # export_version=1, schema_version, includes_trash, works[], counts, images[]
   ├─ checksums.sha256
   ├─ data/app.dump      # pg_dump -Fc
   └─ images/
   ```
3. **含回收站**：直接 `pg_dump -Fc` 当前数据库。
4. **不含回收站**：创建临时数据库 → `pg_restore` 复制 → 删除软删除根及其隐藏关系/席位/节点 → `pg_dump -Fc` → 删除临时库。确保导出包真实不含回收站数据（AC-12）。
5. **恢复（唯一命令入口，P5b RT-18）**：停止 app 后执行 `docker compose run --rm cli restore <zip>`；流程：校验 manifest/checksums/版本/空间 → 生成恢复前快照 → 解压到临时目录 → 校验图片与引用 → `pg_restore --clean --if-exists --no-owner --single-transaction -d "$DATABASE_URL"` → 成功后替换 `images/` → `goose.Up` 对齐 → 写入 `app_settings.last_restore_at` → 输出计数。`scripts/restore.sh` 不作为权威路径。
   - **快照恢复（P5b RT-06）**：`docker compose run --rm cli restore-snapshot <dir>`；校验 `manifest.json`、`checksums.sha256`、`schema_version`/`app_version` 兼容性 → 恢复 DB + images → 写 `last_restore_at`；版本不兼容时拒绝并提示使用导出包跨版本恢复。
   - CLI 恢复前需 `docker compose stop app`；CLI 与 app 使用同一 advisory lock，避免并发写入；迁移同样遵循该锁（P5b RT-22）。
6. **失败处理**：`pg_restore` 单事务回滚，原数据库不变；图片在 DB 成功后才替换；替换失败用恢复前快照回退。
7. **验证**：恢复后核对作品数、各实体计数、图片数量/checksum、外键完整、检索文档数量；`/api/v1/settings` 的 `last_restore_at` 读取 `app_settings`（P5b RT-06）；`cmd/cli doctor` 可重复自检。
8. **安全**：导出包与快照含私密内容；不写入日志；不自动上传；界面提示用户自行加密备份。

## Consequences

- 正面：`pg_dump/pg_restore` 成熟；恢复失败原子回滚；导出包可离线；图片与 DB 同一包。
- 负面：恢复需停写；不含回收站导出需要临时数据库与额外磁盘/时间；`pg_dump` 需与 PG 16 客户端版本匹配；5GB 图片包耗时较长（V-03）。
- 约束：禁止直接复制 `pgdata` 作为备份；恢复必须走 `docker compose run --rm cli restore`/`restore-snapshot`；恢复前必须有快照；快照触发以 FR-55 为准（P5b RT-17/18）。

## Risks and Mitigations

| 风险 | 缓解 |
|---|---|
| 导出包损坏 | `checksums.sha256` 逐文件校验；恢复前拒绝损坏包 |
| 版本不兼容 | `export_version`/`schema_version` 校验；恢复后 `goose.Up` |
| 临时库过滤遗漏隐藏关系 | 固定删除顺序 + 恢复后引用完整性检查；集成测试覆盖两种包 |
| 图片替换失败 | DB 成功后才替换；旧目录保留至替换成功；失败恢复前快照 |
| 磁盘不足 | 恢复前预检（解压大小 + 快照 + 20% 余量）；导入前预检（zip + 解压 + 1GB 快照余量，P5b RT-12） |
| 快照不可恢复/损坏 | 快照固定结构 + `manifest.json`/`checksums.sha256` 校验；`restore-snapshot` 失败即中止、不触碰数据（P5b RT-06） |
| 5GB 导出慢 | 流式 zip；V-03 测量；必要时提示后台执行 |
| `pg_dump` 版本不匹配 | 运行镜像固定 `postgres:16-bookworm`，与 db 同大版本 |

## Follow-up Actions

- [x] 项目发起人确认 D-05（2026-11-07）。
- [ ] V-03/V-04 验证导出、恢复与中断。
- [ ] 编写空环境恢复演练脚本与记录模板（唯一入口为 compose `cli restore`/`restore-snapshot`，P5b RT-18）。
- [ ] 集成测试覆盖含/不含回收站两种导出包。
- [ ] 集成测试覆盖快照结构校验与 `restore-snapshot`（P5b RT-06）。

## Decision / Change Log

| 日期 | 版本 | Finding ID | 决策 | 变更摘要 |
|---|---|---|---|---|
| 2026-11-07 | Rev.1 | D-05 | `pg_dump -Fc` + 图片 + manifest；CLI 恢复；恢复前快照 | 初始批准 |
| 2026-11-07 | Rev.2 | RT-06 | 快照目录固定含 `app.dump`/`images/`/`snapshot.json`/`manifest.json`/`checksums.sha256`；新增 `restore-snapshot`；写 `last_restore_at` | 快照原先无恢复入口，FR-55 无法验收 |
| 2026-11-07 | Rev.2 | RT-17 | 快照触发对齐 FR-55：补永久删除，回收站单项恢复不生成，手动快照仅 CLI | 修正“仅 CLI 全量恢复生成快照”字面 |
| 2026-11-07 | Rev.2 | RT-18 | 恢复唯一入口 `docker compose run --rm cli restore <zip>`；`scripts/restore.sh` 非权威 | 消除双恢复路径歧义 |
| 2026-11-07 | Rev.2 | RT-22 | CLI 恢复/迁移与 app 共用 advisory lock，操作前停止 app | 防止恢复/迁移与在线写并发 |
