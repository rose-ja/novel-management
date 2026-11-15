# backup — 导出、快照与 CLI 恢复

- 域状态：`implemented`（CHG-010，2026-11-15）
- 目标行为：本地单机数据可全量导出、可回滚到快照、可在空环境用 CLI 恢复；破坏性操作前自动快照。
- 关联页面：PG-13；关联验收：AC-12、AC-22。
- 说明：CHG-002 交付快照基座，CHG-010 补齐导出、CLI 恢复与设置页入口。

## SPEC-BACKUP-001 全量导出

- 状态：`implemented`（CHG-010）
- 来源：FR-52、FR-53、FR-63；US-010；PG-13 §3；API `POST /api/v1/exports`、`GET /api/v1/exports`、`GET /api/v1/exports/{id}`、`GET /api/v1/exports/{id}/download`、`GET /api/v1/export/status`；Data `export_runs` + 全部实体；AC-12、EX-25
- 行为：
  - 导出全部作品的数据为一个 zip（ADR-006）：`manifest.json`、`checksums.sha256`、`data/app.dump`（pg_dump -Fc）、`images/`、`README.txt`。
  - `include_trash=true` 直接导出当前库；`false` 时复制到临时库，清理软删除根与隐藏关系后再导出，包内不含回收站数据。
  - 导出文件名含日期时间（`novel-settings-YYYYMMDD-HHmmss.zip`）；导出前弹窗询问是否包含回收站，默认包含。
  - 导出在后台执行，完成后提供下载；失败提供错误码与重试，不提供残缺 zip；进程重启后遗留任务标记为 `EXPORT_INTERRUPTED`。
  - manifest 记录 `export_version`、`schema_version`、`app_version`、`generated_at`、`includes_trash`、works、活跃实体计数、图片列表与文件大小。
- 边界：
  - 不做增量/按类型导出、不做自动上传云存储。
  - 不导出快照目录历史；数据库转储中的快照/导出记录不构成快照文件备份。
- 验收场景：
  1. Given 作品含图片与回收站数据，When 导出（含回收站），Then zip 含五件套，manifest 计数正确（E2E-10 / AC-12）。
  2. Given 关闭「包含回收站」，When 导出，Then zip 不含软删除数据，manifest `includes_trash=false`（FR-53）。
  3. Given 导出中断，When 重新打开页面，Then 记录为 failed/中断并可重试（EX-23）。
- 变更历史：2026-11-07 — P6 新建，状态 `planned`；2026-11-15 — CHG-010 实现；按 ADR-006 将包结构由 `data.json/media/meta.json` 修正为 `data/app.dump + images/ + manifest + checksums + README`（用户裁决），文件名改为 `novel-settings-*`，状态转 `implemented`。

## SPEC-BACKUP-002 CLI 恢复、校验与恢复前快照

- 状态：`implemented`（CHG-010）
- 来源：FR-54、FR-56；US-010；PG-13 §6；API 无（CLI）；Data 全部实体；AC-22、EX-24
- 行为：
  - 命令：`docker compose run --rm cli restore /data/exports/<file>.zip --yes`（非空实例必须 `--yes`）。
  - 恢复前校验 `export_version=1`、`schema_version<=当前版本`、checksums、必需 `data/app.dump` 与磁盘空间；zip 损坏、版本不兼容、校验失败或空间不足时拒绝恢复，不覆盖现有数据。
  - 恢复前自动生成当前数据快照（`reason=cli_restore`）；恢复过程停止在线写入并持有 advisory lock（需先 `docker compose stop app`）。
  - `pg_restore --clean --if-exists --single-transaction` 恢复数据库；成功后原子替换 `images/`，执行 `goose up`、重建检索文档、写 `last_restore_at`、核对 manifest 计数并输出摘要。
  - 失败时保留恢复前快照并输出 `restore-snapshot` 回滚命令。
  - 恢复完成后的回收站状态与导出时一致（`include_trash` 决定）。
- 边界：
  - 不做在线恢复、不做部分恢复、不做跨 schema 迁移（旧包恢复后自动补齐迁移）。
  - 不提供 GUI 恢复按钮。
- 验收场景：
  1. Given 空环境与合法 zip，When 执行 CLI 恢复，Then 数据、图片、关系与检索可用，摘要与 manifest 一致（AC-22）。
  2. Given schema_version 不兼容，When 恢复，Then 拒绝且不写入（FR-54）。
  3. Given 恢复中途失败，When 查看实例，Then 原数据可由恢复前快照回退（FR-56）。
  4. Given 实例非空，When 未传 `--yes`，Then 拒绝并提示清空或换实例（FR-54）。
- 变更历史：2026-11-07 — P6 新建；2026-11-15 — CHG-010 实现 `restore`，状态转 `implemented`。

## SPEC-BACKUP-003 快照触发、五件套、保留与恢复

- 状态：`implemented`（CHG-002 基座 + CHG-010 CLI 恢复）
- 来源：FR-55；US-010、US-011；PG-13 §4；API `GET /api/v1/snapshots`；Data `snapshots`；AC-12
- 行为：
  - 触发：删除作品/永久删除（含清空回收站）/导入/CLI 全量恢复四类破坏性操作前自动触发；失败阻止操作。回收站单项恢复不生成快照。
  - 每次快照产出五件套（`app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`）到 `{DATA_DIR}/snapshots/{YYYYMMDDTHHMMSSZ}-{reason}`；同一秒目录冲突时追加 ULID 后 6 位区分。
  - 保留最近 3 份；新快照成功后清理更早的成功记录与文件（失败记录不参与裁剪）。
  - 设置页显示最近快照列表（时间、触发原因、数据库大小、状态、路径），只读；路径可复制；不提供页面手动创建/下载/恢复。
  - `restore-snapshot <dir>` CLI 恢复快照（CHG-010）；恢复前再生成当前快照，失败可回退；旧 manifest 缺少 `schema_version` 时兼容并告警。
  - 快照 manifest 记录 `schema_version`、`app_version`、文件列表；`snapshot.json` 的计数按活跃记录（`deleted_at IS NULL`）。
- 边界：
  - 不做异地备份、不做定时快照、不做用户自定义保留数。
  - 不做快照差异备份。
  - 页面内不提供恢复入口。
- 验收场景：
  1. Given 删除作品前，When 快照失败，Then 删除被阻止且不产生数据变更（EX-23）。
  2. Given 第 4 次快照成功，When 查看目录，Then 只保留最近 3 份且记录一致（FR-55）。
  3. Given 快照存在，When 执行 `restore-snapshot`，Then 数据回到快照点，恢复前生成新快照（AC-12）。
  4. Given 快照损坏，When 恢复，Then 校验失败并保留当前数据（EX-23）。
- 变更历史：2026-11-07 — P6 新建；2026-11-08 — CHG-002 实现快照基座；2026-11-15 — CHG-010 交付 `restore-snapshot`、路径展示与版本字段，状态转 `implemented`。

## SPEC-BACKUP-004 设置/备份页展示与空环境演练

- 状态：`implemented`（CHG-010）
- 来源：FR-56；US-010、US-012；PG-13 §2/§6；API `GET /api/v1/exports`、`GET /api/v1/export/status`、`GET /api/v1/snapshots`、`GET /api/v1/settings`；Data `app_settings`；AC-12、AC-22
- 行为：
  - 页面展示访问地址 `http://127.0.0.1:17860`、当前版本、数据目录、快照列表、导出入口与恢复命令。
  - 导出区块支持包含/排除回收站、进度、成功下载/复制路径、失败重试；无作品时禁用并说明。
  - 恢复命令可一键复制，包含 `docker compose run --rm cli restore <zip>` 与 `restore-snapshot <dir>`。
  - 空环境演练验收：全新容器 → CLI 恢复 → 抽查人物/图片/检索，全部通过（2026-11-15 已演练）。
  - 页面不提供恢复按钮；避免误操作覆盖在线数据。
- 边界：
  - 不做云备份、账号、权限；单用户本地。
  - 不做快照内容预览。
- 验收场景：
  1. Given 设置页，When 查看，Then 显示固定地址、版本、数据目录、快照与复制命令（AC-12）。
  2. Given 全新空容器，When 按页面命令恢复，Then 抽查数据完整（AC-22）。
  3. Given 无快照，When 查看，Then 空状态说明「破坏性操作前自动生成」（PG-13）。
  4. Given 复制命令，When 粘贴执行，Then 命令可直接运行、无需手工替换路径（RT-17）。
- 变更历史：2026-11-07 — P6 新建；2026-11-15 — CHG-010 实现导出区块、命令复制与系统信息，状态转 `implemented`。
