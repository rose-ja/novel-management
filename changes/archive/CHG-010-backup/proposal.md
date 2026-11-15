# Change Proposal

- ID: CHG-010-backup
- Title: 导出、CLI 恢复与快照收口
- Status: Approved（2026-11-15，按实施计划批准）
- Owner: 主 Agent
- Related Spec: SPEC-BACKUP-001、SPEC-BACKUP-002、SPEC-BACKUP-003、SPEC-BACKUP-004
- Related Issue: roadmap §CHG-010；FR-52~FR-56、FR-63；US-010、US-012；PG-13；AC-12、AC-22；EX-23~EX-25

## Problem / Opportunity

CHG-002 已交付删除前自动快照与设置页只读快照列表，但用户仍无法在产品内导出全量数据；没有 `restore` 与 `restore-snapshot` 命令，已有快照无法恢复；设置页缺少导出入口、恢复命令与完整系统信息。SPEC-BACKUP-001~004 全部为 `planned`，数据无法离线备份与迁移，AC-12/AC-22 无法验收。

## User Value

- 设置页一键导出全量 zip（可选包含回收站），后台执行、进度可见、完成后下载或复制路径。
- 导出包可在空环境通过 CLI 恢复，数据、关系、图片与检索可用。
- 删除/清空等破坏性操作前的快照不仅能看，还能用 `restore-snapshot` 恢复。
- 恢复失败不覆盖现有数据，恢复前自动快照并提供回滚命令。

## Goal

交付完整备份垂直切片：迁移 `0010` 建立 `export_runs`；导出服务生成 `manifest.json + checksums.sha256 + data/app.dump + images/ + README.txt`（含/不含回收站两种模式）；HTTP 导出任务与下载；CLI `restore`/`restore-snapshot`（校验 manifest/checksum/schema、恢复前快照、单事务恢复）；设置页 PG-13；单测、集成、E2E、空环境演练与文档回写。

## In Scope

- 迁移 `0010_export_runs`：状态、模式、文件、错误码、时间。
- 导出：后台任务、两种模式、五种产物、manifest 计数、checksums、失败清理与重试、启动中断标记。
- API：`POST /api/v1/exports`、`GET /api/v1/exports`、`GET /api/v1/exports/{id}`、`GET /api/v1/exports/{id}/download`、`GET /api/v1/export/status`。
- CLI：`restore <zip> [--yes]`、`restore-snapshot <dir> [--yes]`；advisory lock；版本/checkums/空间/空实例校验；恢复前快照；`pg_restore --single-transaction`；图片原子替换；迁移与检索重建；写 `last_restore_at`；摘要与回滚命令。
- 设置页：导出区块（Dialog/进度/结果/失败重试/无作品禁用）、快照路径与复制、CLI 命令复制、系统信息（版本/数据目录/访问地址/最近导出恢复）。
- 文档：SPEC-BACKUP-001~004、PG-13、api-contract §3.12、data-model §3.17、roadmap 进度。

## Out of Scope

- 增量/按作品/按类型导出、云存储上传、定时导出、导出历史保留策略。
- GUI 恢复按钮、页面上传 zip、在线恢复、部分恢复、跨 schema 迁移。
- 快照手动创建/下载、快照内容预览、定时快照、超过 3 份保留。
- 导入（CHG-011）与跨域一致性收口（CHG-012）。

## Acceptance Criteria

1. 设置页可导出全量 zip；Dialog 询问是否包含回收站，默认勾选；无作品时禁用并说明。
2. 导出在后台执行；完成后展示文件名、大小、时间，可下载或复制路径；失败展示中文原因并可重试，不产生残缺 zip。
3. 含回收站与不含回收站两种包结构正确；`manifest.json.includes_trash` 与包内容一致；不含回收站的包不包含软删除根。
4. 导出包包含 `manifest.json`、`checksums.sha256`、`data/app.dump`、`images/`、`README.txt`；manifest 记录 schema/app/export 版本、works、counts、images。
5. 空环境执行 `docker compose run --rm cli restore <zip>` 可恢复数据、图片、关系与检索；输出摘要与 manifest 一致。
6. schema/export 版本不兼容、checksum 失败、zip 损坏、磁盘不足时拒绝恢复且不覆盖现有数据。
7. 实例非空且未传 `--yes` 时拒绝恢复并提示；恢复前自动生成 `cli_restore` 快照；`pg_restore` 失败保留原数据与图片，并输出回滚命令。
8. `restore-snapshot <dir>` 可恢复快照五件套；旧 manifest 缺 schema_version 时兼容并告警；恢复成功写 `last_restore_at`。
9. 设置页快照列表显示最近 3 份的时间、大小、来源、状态、路径；路径可复制；提供两条固定恢复命令与注意事项。
10. 系统信息显示版本、数据目录、访问地址 `http://127.0.0.1:17860`、最近导出/恢复时间。
11. 导出/快照/恢复失败均有中文提示与可重试边界；无堆栈或内部路径泄露。
12. 全量门禁、集成、前端单测、Docker 构建、Playwright、空环境演练通过并记录。

## Constraints and Assumptions

- 技术栈不变：Go + Gin + GORM + PostgreSQL 16 + goose；Next.js 静态导出；不新增第三方依赖。
- 导出包采用 ADR-006/api-contract 结构（用户 2026-11-15 裁决），不实现 SPEC-BACKUP-001 的 `data.json/media/`；F10 修正 Spec。
- 运行时镜像已包含 `pg_dump/pg_restore`；CLI 与 app 共用 advisory lock，恢复前必须停止 app。
- 快照 manifest 从本变更起增加 `schema_version/app_version`；旧快照按兼容处理。
- 不含回收站导出通过临时库过滤，确保包内容真实不含软删除数据。

## Dependencies

- CHG-002（快照基座、advisory lock、app_settings）。
- CHG-008（榜单/席位，用于回收站过滤的完整表集）。
- CHG-009（检索重建，用于恢复后校验）。

## Risks

| 风险 | 影响 | 缓解 |
|---|---|---|
| 临时库过滤遗漏隐藏关系 | 不含回收站的包仍有残留 | 固定删除顺序；集成覆盖两种包；恢复后计数核对 |
| `pg_restore` 覆盖快照表 | 恢复前快照记录丢失 | 恢复后回填恢复前记录并清理失效路径 |
| 大图片包耗时/磁盘不足 | 导出慢或失败 | 后台任务、流式下载、失败清理与重试；磁盘预检 |
| 恢复中途失败 | 数据/图片不一致 | 单事务回滚；图片 DB 成功后才替换；快照回退 |
| 导出文件名碰撞 | 覆盖旧包 | 时间戳 + ID 后缀 |
