# CHG-004 任务拆分

- 参考：`proposal.md`、`design.md`（决策 D1～D9）
- 分支：`feat/chg-004-characters-core`；每个任务独立提交，提交信息使用中文前缀（`feat:`/`test:`/`docs:`/`fix:`）
- 完成标志：勾选本文件全部任务 → 每个验收点在 `checklist.md` 勾选 → 进入 F3 验证

## 依赖图

```
T1 迁移/模型
 ├─ T2 仓储（characters）
 │   ├─ T3 服务（CRUD/校验/乐观锁）
 │   │   ├─ T4 HTTP 接口
 │   │   │   ├─ T5 列表查询与索引验证
 │   │   │   ├─ T6 删除/撤销/回收站根类型
 │   │   │   └─ T8 体系引用收口（name/in_trash）
 │   │   └─ T7 作品组联动（软删/恢复/永久删除）
 │   └─ T9 后端集成测试（覆盖 T3～T8）
 └─ T10 前端 API 层 + 单测
     ├─ T11 列表页
     ├─ T12 详情页 + 内联编辑 + 守卫
     └─ T13 回收站/导航适配 + E2E + 视觉基线
T14 文档与契约回写（F10 前完成草稿）
```

## T1 迁移 0004 与领域模型

- 目标：建立 `characters`、`character_names`，补齐 `character_system_refs` 外键；定义 Go 模型与错误类型。
- 文件：
  - `backend/internal/migrations/0004_characters.sql`（新增；Up 首行 `DELETE FROM character_system_refs;` 见 D5）
  - `backend/internal/model/character.go`（新增：`Character`、`CharacterName`、`CharacterSystemRef`、`CharacterSummary`、`SystemRefView`）
  - `backend/internal/model/errors.go`（`ReferenceConflictError`、`CrossWorkReferenceError`；复用 `ValidationError`、`RestoreConflictError`、`NameConflictError`）
  - `backend/internal/migrations/migrations_test.go`（如存在，补 0004 往返）
- 验收：`go test ./internal/integration -run TestMigrationRoundTrip`（或等价）通过；`goose_db_version` 到 4；表/索引/约束与 `design.md` §2 一致。
- 依赖：无。

## T2 人物仓储

- 目标：`repository/characters.go` 提供 CRUD、列表、名称占用、引用替换与回收站操作。
- 关键方法：
  - `Create(ctx, char, names, refs)`
  - `Get(ctx, id)` / `GetDeleted(ctx, id)` / `GetActiveByWork(ctx, workID, id)`
  - `Updated(ctx, id, tx *gorm.DB)`、`NameTaken(ctx, workID, key, excludeID)`
  - `List(ctx, filter)`（分页、`q`、`system_node_id`、`review_status`、排序）、`Count(ctx, filter)`
  - `ReplaceNames`、`ReplaceSystemRefs`、`ListNames`、`ListSystemRefs`
  - `SoftDelete(ctx, id, rootID, action)`、`Restore(ctx, id)`、`RestoreGroupByWork(ctx, workID)`
  - `ListDeletedRoots(ctx)`、`Purge(ctx, id)`（含 refs/names）
- 验收：仓储层编译通过；`go vet` 0；方法签名与设计一致。
- 依赖：T1。

## T3 人物服务：创建/更新/详情

- 目标：`service/characters.go` 实现规范化、校验矩阵、事务与乐观锁。
- 文件：`backend/internal/service/characters.go`（新增）。
- 要点：`works.GetActive` 父级校验、`NormalizeName`/`NormalizeKey`、`names` 去重与排序、`system_refs` 校验（作品/类别启用/节点归属/重复）、`If-Match` 版本比较、`version+1`。
- 验收：单测覆盖校验矩阵（`service/characters_test.go`）；`go test ./internal/service` 通过。
- 依赖：T2。

## T4 HTTP 接口

- 目标：注册 5 个端点并映射错误。
- 文件：`backend/internal/httpapi/characters.go`（新增）、`router.go`、`request.go`。
- 端点：`GET/POST /characters`、`GET/PATCH/DELETE /characters/{id}`。
- 验收：`GET /api/v1/meta` 不变；curl 手工冒烟（创建/详情/更新/删除/撤销）；错误码与 `design.md` §3 一致。
- 依赖：T3。

## T5 列表查询与性能

- 目标：完成筛选/排序/分页 SQL 与 `system_summary` 批量装配。
- 要点：单条 SQL 分页 + 一次批量取节点路径（避免 N+1）；`page_size` 夹取；`q` 转义 `%`/`_`。
- 验收：集成用例断言分页/排序/`q` 命中别名/境界筛选；1000 人 P95 ≤500ms（采样 20 次）。
- 依赖：T4。

## T6 删除、撤销与回收站根类型

- 目标：软删除 + 8 秒撤销 + `character` 根类型完整接入。
- 文件：`service/characters.go`、`service/trash.go`、`httpapi/trash.go`、`model/errors.go`（如需）。
- 要点：
  - `RootTypeCharacter = "character"`；`List` 合并人物根（`parent_name=work.name`、`affected_count=1`）。
  - `Restore`：作品 active → `PARENT_DELETED`；同名 → `RESTORE_CONFLICT`（支持 `new_name`）。
  - `Purge`：`character_system_refs → character_names → characters`。
  - `PurgeAll`：人物根参与，顺序不与作品/体系冲突。
  - `Undo`：复用撤销令牌；人物分支校验 `deleted_root_id=id`。
- 验收：集成用例覆盖删除/撤销/回收站列表/恢复冲突/永久删除；E2E 恢复用例（T13）。
- 依赖：T4。

## T7 作品组联动

- 目标：作品软删除/恢复/永久删除覆盖人物与别名。
- 文件：`repository/works.go`、`service/works.go`（如组删除逻辑在服务层）。
- 要点：`SoftDeleteGroup` 增 `characters`（未删行）；`RestoreGroup` 增 `deleted_root_id=work_id`；`purgeWorkChildren` 顺序 `refs → names → characters → nodes → categories`。
- 验收：集成用例「删作品 → 人物不可见 → 恢复作品 → 人物回来 → 永久删除作品 → 人物物理删除」。
- 依赖：T2、T6。

## T8 体系引用收口

- 目标：类别/节点引用清单补充人物名与 `in_trash`；计数含回收站人物。
- 文件：`repository/system.go`、`httpapi/request.go`、`frontend/components/system/reference-dialog.tsx`（展示列）。
- 验收：集成用例「人物引用类别 → 停用/删除类别 409，details 含 `name`/`in_trash`」；人物进回收站后仍阻断且 `in_trash=true`；永久删除人物后阻断消失。
- 依赖：T6。

## T9 后端集成测试汇总

- 目标：补齐 `design.md` §8 的 1～8 类用例并全绿。
- 文件：`backend/internal/integration/character_test.go`、`character_repo_test.go`、`character_trash_test.go`（新增）。
- 验收：`TEST_DATABASE_URL=... go test -count=1 -tags=integration ./internal/integration/...` 全部 ok。
- 依赖：T5、T7、T8。

## T10 前端 API 层

- 目标：`frontend/lib/api/characters.ts` + 单测。
- 要点：类型与 DTO 对齐、`ifMatch` 透传、错误码解析、列表参数序列化。
- 验收：`npm run test` 覆盖 8+ 用例；`npm run typecheck` 通过。
- 依赖：T4（接口稳定）。

## T11 人物列表页

- 目标：`/characters` 列表、筛选、排序、分页、新建 Drawer、空/加载/错误态。
- 文件：`app/characters/page.tsx`、`components/characters/*`（列表部分）、`lib/messages.ts`、`components/shell/global-nav.tsx`。
- 验收：E2E 列表用例（新建后出现在首行、搜索别名命中、筛选境界、待校对切换、分页）。
- 依赖：T10。

## T12 人物详情页与内联编辑

- 目标：`/characters?id=` 详情、内联编辑、未保存守卫、版本冲突、删除与撤销。
- 文件：`components/characters/character-detail-page.tsx`、`character-form-fields.tsx`、`character-delete-dialog.tsx`、`use-unsaved-guard.ts`。
- 验收：E2E 详情用例（内联保存、名称冲突字段错误、未保存守卫、删除 → 撤销、409 冲突对话框）。
- 依赖：T11。

## T13 回收站/导航适配、E2E 与视觉基线

- 目标：回收站人物根展示与筛选；`characters.spec.ts` 全量 E2E；视觉基线；a11y。
- 文件：`app/trash/page.tsx`、`components/trash/trash-dialogs.tsx`、`e2e/characters.spec.ts`（新增）、`e2e/trash-system.spec.ts`（增补）、`e2e/visual.spec.ts`（增补）、基线 PNG。
- 验收：全量 Playwright 通过；`characters-1280/1024` 基线生成且差异 ≤0.02。
- 依赖：T12。

## T14 文档与契约回写草稿

- 目标：F3 前记录实施差异，F10 由归档任务回写。
- 文件：`changes/CHG-004-characters-core/verification.md`（骨架）、`project/api-contract.md`（`CharacterSummary.names[]`/`version`、DELETE `If-Match`、`location_id` 未开放）——**在 F10 统一提交**。
- 验收：差异清单与 `design.md` §3.1、D9 一致。
- 依赖：T13。

## 任务完成记录

| 任务 | 状态 | 提交 |
|---|---|---|
| T1 | ☑ | `feat: 人物迁移 0004 与领域模型` |
| T2 | ☑ | `feat: 人物仓储层` |
| T3 | ☐ | |
| T4 | ☐ | |
| T5 | ☐ | |
| T6 | ☐ | |
| T7 | ☐ | |
| T8 | ☐ | |
| T9 | ☐ | |
| T10 | ☐ | |
| T11 | ☐ | |
| T12 | ☐ | |
| T13 | ☐ | |
| T14 | ☐ | |
