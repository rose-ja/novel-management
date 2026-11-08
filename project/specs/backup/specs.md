# backup — 导出、快照与 CLI 恢复

- 域状态：`planned`（P6，2026-11-07）
- 目标行为：本地单机数据可全量导出、可回滚到快照、可在空环境用 CLI 恢复；破坏性操作前自动快照。
- 关联页面：PG-13；关联验收：AC-12、AC-22。
- 说明：本文件中的 Spec 均为待实现目标，不代表当前系统能力。

## SPEC-BACKUP-001 全量导出

- 状态：`planned`
- 来源：FR-52、FR-53、FR-63；US-010；PG-13 §3；API `POST /api/v1/exports`、`GET /api/v1/exports/{id}`；Data `export_runs` + 全部实体；AC-12、EX-25
- 行为：
  - 导出全部作品的数据为一个 zip：`manifest.json`、`data.json`、`media/`、`meta.json`、`README.txt`，可选 `trash/`（含回收站与 `renames`）。
  - 导出文件名含日期时间（如 `novel-backup-YYYYMMDD-HHmmss.zip`）；导出前弹窗询问是否包含回收站，默认包含。
  - 导出在后台执行，完成后提供下载；失败提供原因与重试，不提供残缺 zip。
  - manifest 记录 schema_version、works、实体计数、生成时间与版本。
- 边界：
  - 不做增量/按类型导出、不做自动上传云存储。
  - 不导出快照历史。
- 验收场景：
  1. Given 作品含图片与回收站数据，When 导出（含回收站），Then zip 含五件套 + `trash/`，manifest 计数正确（E2E-10 / AC-12）。
  2. Given 关闭「包含回收站」，When 导出，Then zip 不含 `trash/`，manifest 标注（FR-53）。
  3. Given 导出中断，When 重新打开页面，Then 记录为 failed 并可重试（EX-23）。
- 变更历史：2026-11-07 — P6 新建，状态 `planned`。

## SPEC-BACKUP-002 CLI 恢复、校验与恢复前快照

- 状态：`planned`
- 来源：FR-54、FR-56；US-010；PG-13 §6；API 无（CLI）；Data 全部实体；AC-22、EX-24
- 行为：
  - 命令：`docker compose run --rm cli restore /data/backup/<file>.zip`（按 RT-17）。
  - 恢复前校验 manifest/schema_version/checksum；zip 损坏、版本不兼容、校验失败或磁盘不足时拒绝恢复，不覆盖现有数据。
  - 恢复前自动生成当前数据快照；恢复过程停止在线写入并持有 advisory lock。
  - 恢复后重建检索文档、校验引用完整性，并输出摘要；失败则保留快照并给出回滚命令。
  - 恢复完成后的回收站状态与导出时一致（导出含 `trash/` 时保留，未含时为空）。
- 边界：
  - 不做在线恢复、不做部分恢复、不做跨 schema 迁移。
  - 不提供 GUI 恢复按钮。
- 验收场景：
  1. Given 空环境与合法 zip，When 执行 CLI 恢复，Then 数据、图片、关系与检索可用，摘要与 manifest 一致（AC-22）。
  2. Given schema_version 不兼容，When 恢复，Then 拒绝且不写入（FR-54）。
  3. Given 恢复中途失败，When 查看实例，Then 原数据可由恢复前快照回退（FR-56）。
  4. Given 实例非空，When 恢复，Then 要求确认清空或换实例，不静默覆盖（FR-54）。
- 变更历史：2026-11-07 — P6 新建，状态 `planned`。

## SPEC-BACKUP-003 快照触发、五件套、保留与恢复

- 状态：`implemented-partial`（CHG-002 交付自动快照基座与设置页只读列表；CLI `restore-snapshot` 见 CHG-010）
- 来源：FR-55；US-010、US-011；PG-13 §4；API `GET /api/v1/snapshots`（本阶段无页面内恢复端点）；Data `snapshots`；AC-12
- 行为：
  - 触发：删除作品/永久删除（含清空回收站）/导入/CLI 全量恢复四类破坏性操作前自动触发；失败阻止操作。回收站单项恢复不生成快照。
  - 每次快照产出五件套（`app.dump`（pg_dump custom 格式）、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`）到 `{DATA_DIR}/snapshots/{YYYYMMDDTHHMMSSZ}-{reason}`；同一秒目录冲突时追加 ULID 后 6 位区分。
  - 保留最近 3 份；新快照成功后清理更早的成功记录与文件（失败记录不参与裁剪）。
  - 设置页显示最近快照列表（时间、触发原因、数据库大小、状态），只读；不提供页面手动创建/下载/恢复。
  - `restore-snapshot <dir>` CLI 恢复属 CHG-010；恢复前再生成当前快照，失败可回退。
- 边界：
  - 不做异地备份、不做定时快照、不做用户自定义保留数。
  - 不做快照差异备份。
  - 本阶段页面内不提供恢复入口。
- 验收场景：
  1. Given 删除作品前，When 快照失败，Then 删除被阻止且不产生数据变更（EX-23）。
  2. Given 第 4 次快照成功，When 查看目录，Then 只保留最近 3 份且记录一致（FR-55）。
  3. Given 快照存在，When 执行 `restore-snapshot`，Then 数据回到快照点，恢复前生成新快照（AC-12）。
  4. Given 快照损坏，When 恢复，Then 校验失败并保留当前数据（EX-23）。
- 变更历史：2026-11-07 — P6 新建，状态 `planned`；2026-11-08 — CHG-002 实现快照基座（五件套命名、目录命名、保留 3 份、设置页只读列表），状态转 `implemented-partial`。

## SPEC-BACKUP-004 设置/备份页展示与空环境演练

- 状态：`planned`
- 来源：FR-56；US-010、US-012；PG-13 §2/§6；API `GET /api/v1/export/status`、`/snapshots`；Data `app_settings`；AC-12、AC-22
- 行为：
  - 页面展示访问地址 `http://127.0.0.1:17860`、当前版本、数据目录、快照列表、导出入口与恢复命令。
  - 恢复命令可一键复制，包含 `docker compose run --rm cli restore /data/backup/<file>.zip`。
  - 空环境演练验收：全新容器 → CLI 恢复 → 抽查人物/图片/检索，全部通过。
  - 页面不提供恢复按钮；避免误操作覆盖在线数据。
- 边界：
  - 不做云备份、账号、权限；单用户本地。
  - 不做快照内容预览。
- 验收场景：
  1. Given 设置页，When 查看，Then 显示固定地址、版本、数据目录、快照与复制命令（AC-12）。
  2. Given 全新空容器，When 按页面命令恢复，Then 抽查数据完整（AC-22）。
  3. Given 无快照，When 查看，Then 空状态说明「破坏性操作前自动生成」（PG-13）。
  4. Given 复制命令，When 粘贴执行，Then 命令可直接运行、无需手工替换路径（RT-17）。
- 变更历史：2026-11-07 — P6 新建，状态 `planned`。
