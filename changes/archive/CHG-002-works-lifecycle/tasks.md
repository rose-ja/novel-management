# Tasks — CHG-002

> 每个任务完成后勾选；`Tests` 为必须新增或修改的测试。

## Execution Tasks

- [x] T1 迁移与领域骨架
  - Files: `backend/internal/migrations/0002_works_trash_snapshot.sql`、`backend/internal/model/{work,settings,snapshot,undo}.go`、`backend/internal/migrations/migrations_test.go`
  - Steps: 建 `works`、`system_categories`（最小结构：id/work_id/name/name_key/is_builtin/is_enabled/category_type/version/软删除三件套）、`app_settings`（单行，初始化）、`snapshots`、`undo_tokens`；索引 `works_name_key_uq`（partial）、`snapshots_created_at_idx`、`undo_tokens_expires_at_idx`；提供完整 `Down`。
  - Expected: `goose up` 后 5 张表存在；`goose down` 可回到 0001；重复 up 幂等。
  - Tests: 迁移往返（集成）。

- [x] T2 仓储层
  - Files: `backend/internal/repository/{works,settings,snapshot,undo}.go`
  - Steps: 全部基于 `gorm` + `tx` 注入；`works` 提供 `List(q,page,size)`、`Get`、`Insert`、`UpdateWithVersion`、`SoftDeleteGroup`、`RestoreGroup`、`PurgeGroup`；`settings` 提供 `Get/SetCurrentWork`；`snapshot` 提供 `Insert/ListRecent/Prune`；`undo` 提供 `Insert/Consume/DeleteOlderThan`。
  - Expected: 版本冲突由 `WHERE version = ?` 的受影响行数判定，不抛数据库错误。
  - Tests: 单测（版本冲突、partial index 冲突映射为 NAME_CONFLICT）。

- [x] T3 快照服务
  - Files: `backend/internal/service/snapshot.go`、`backend/internal/service/snapshot_test.go`、`Dockerfile`
  - Steps: 产出 `app.dump`（`pg_dump --format=custom`）、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`；目录 `{DATA_DIR}/snapshots/{ts}-{reason}`；成功后保留最近 3 份；失败返回 `ErrSnapshotFailed`。镜像运行阶段 `apk add --no-cache postgresql16-client`。
  - Expected: 单测可用假 exec 与临时目录验证五件套与裁剪逻辑；容器内 `pg_dump --version` 可用。
  - Tests: `snapshot_test.go`（五件套、保留 3 份、失败路径）。

- [x] T4 作品服务与接口
  - Files: `backend/internal/service/works.go`、`backend/internal/httpapi/works.go`、`backend/internal/httpapi/works_test.go`
  - Steps: `GET/POST /works`、`GET/PATCH/DELETE /works/{id}`；创建事务含内置「境界」；`If-Match` 解析；删除：快照 → 软删除 → undo token → 清空 `current_work_id`。
  - Expected: 全部错误码与 proposal AC-1~6 一致。
  - Tests: 集成（创建/同名冲突/乐观锁/删除确认名/快照失败阻止）。

- [x] T5 回收站服务与接口
  - Files: `backend/internal/service/trash.go`、`backend/internal/httpapi/trash.go`、`backend/internal/httpapi/trash_test.go`
  - Steps: `GET /trash`（root_type/分页）、`POST /trash/{type}/{id}/restore`（整组原子 + `new_name`/`renames[]`）、`DELETE /trash/{type}/{id}`、`DELETE /trash`、`POST /trash/undo`。
  - Expected: 恢复冲突返回 422 + `details.items`；undo 8 秒窗口精确。
  - Tests: 集成（恢复、冲突、永久删除落快照、undo 过期/重复）。

- [x] T6 设置与快照接口
  - Files: `backend/internal/httpapi/settings.go`、`backend/internal/httpapi/snapshots.go`
  - Steps: `/settings` 改为读 `app_settings`；新增 `GET /snapshots`（倒序最多 10 条，字段 `id/reason/created_at/db_bytes/image_count/path`）。
  - Expected: 契约与 `api-contract.md` 一致，`current_work_id` 真实。
  - Tests: 集成。

- [x] T7 集成测试基建
  - Files: `backend/internal/integration/suite_test.go`、`backend/internal/testdb/*.go`、`scripts/ci.ps1`
  - Steps: `-tags=integration`；读 `TEST_DATABASE_URL`，每个用例在事务中运行并回滚；`ci.ps1` 在 compose 栈健康后对 db 执行。
  - Expected: 本机与 CI 均可运行；无 DB 时明确 SKIP 并打印命令。
  - Tests: 自身即测试。

- [x] T8 前端 API 层与未保存保护
  - Files: `frontend/lib/api/{works,trash,snapshots}.ts`、`frontend/lib/api/client.ts`（补 `If-Match`/`X-Requested-With`）、`frontend/lib/hooks/use-unsaved-guard.ts`、`frontend/lib/messages.ts`
  - Steps: 类型化封装；写请求统一带 `X-Requested-With`；`If-Match` 由调用方传 `version`。
  - Expected: 类型完整，错误码到文案的映射覆盖新 code。
  - Tests: `works.test.ts`（错误映射、If-Match 头）。

- [x] T9 顶栏作品切换器与作品弹窗
  - Files: `frontend/components/shell/{work-switcher,top-bar}.tsx`、`frontend/components/works/{work-dialogs,undo-toast}.tsx`
  - Steps: 下拉列表（加载/空/错误）、新建 Dialog、重命名 Dialog（带 `If-Match`）、删除 Dialog（输入名称一致才可确认）、8 秒撤销 Toast。
  - Expected: 切换后 `current_work_id` 持久化并即时反映到顶栏与设置页。
  - Tests: E2E。

- [x] T10 回收站页
  - Files: `frontend/app/trash/page.tsx`、`frontend/components/trash/*`
  - Steps: 列表（名称/删除时间/影响条数）、作品筛选、恢复（冲突时提示改名）、永久删除（不可撤销确认）、清空、空状态、错误重试。
  - Expected: 与 PG-12 一致；导航新增「回收站」项。
  - Tests: E2E。

- [x] T11 设置页快照卡片
  - Files: `frontend/components/settings/snapshot-list.tsx`、`frontend/app/settings/page.tsx`
  - Steps: 展示快照时间/原因/大小/状态；空状态「暂无快照」；不提供恢复按钮。
  - Expected: 与 SPEC-BACKUP-004 一致（本变更只交付只读列表）。
  - Tests: E2E。

- [x] T12 E2E 增补
  - Files: `frontend/e2e/{works,trash,settings-snapshots}.spec.ts`、`frontend/e2e/shell.spec.ts`（导航新增回收站）
  - Steps: 覆盖 proposal AC-1~12 中用户可见部分；数据自清理（测试作品名带时间戳后缀，结束时删除并清空回收站）。
  - Expected: 全绿；无跨用例污染。
  - Tests: 自身即测试。

- [x] T13 CI 与镜像
  - Files: `scripts/ci.ps1`、`Dockerfile`、`docker-compose.yml`（如需 `DATA_DIR` 挂载）
  - Steps: 加入集成测试阶段；E2E 前置条件说明；确认快照目录可写。
  - Expected: `CI PASSED` 覆盖单元 + 集成 + E2E。
  - Tests: 实际执行一次。

- [x] T14 文档同步
  - Files: `project/specs/backup/specs.md`（快照文件命名按 D-03 修订）、`project/page-specs/PG-00-global-shell.md`（作品位改为可交互）、`project/page-specs/PG-12-*`（实施状态）、`project/PROJECT-SETUP.md` §11
  - Steps: 记录本次交付范围与遗留（导入/恢复/导出仍属 CHG-010）。
  - Expected: 文档无静默偏离。
  - Tests: 人工核对。

- [x] T15 验收与归档（F5~F10）
  - Files: `changes/CHG-002-works-lifecycle/{checklist.md,verification.md}`、`changes/archive/CHG-002-works-lifecycle/`
  - Steps: 逐条勾选、记录真实命令与证据、提交、合并、归档。
  - Expected: 所有验收项有证据。
  - Tests: 按 checklist 执行。

## Definition of Done

- [x] Required checks pass（`scripts/ci.ps1`，含集成与 E2E）
- [x] Acceptance scenarios pass（proposal §4 的 12 条）
- [x] No unresolved P0
- [x] P1 handled or approved waiver
- [x] Documentation synchronized（backup spec 命名、PG-00、PG-12、SETUP §11）
- [x] Rollback understood（迁移 Down + revert 提交）
