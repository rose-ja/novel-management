# PR/MR 描述 — CHG-004 人物档案核心

> 状态：本地分支已就绪，**未推送、未创建远程 PR、未合并**（按 F8 规则等待用户明确授权）。
> 分支：`feat/chg-004-characters-core` → 目标 `main`；提交范围 `2852a35..HEAD`（18 个提交）。
> 门禁：F3 自测、F4 独立审查（FAIL：S1=1/S2=15）、F5 修复+复审 PASS、F6 回归/E2E PASS、F7 人工验收 **ACCEPT**。

## 1. 标题与目标

**feat: 人物档案核心（CHG-004）**

让用户在 `/characters` 完成人物档案的首个完整业务闭环：新建 → 列表浏览/搜索/筛选/排序/分页 → 详情查看 → 内联编辑（动态体系字段、别名/称号）→ 乐观锁冲突保护 → 软删除/8 秒撤销/回收站恢复/永久删除。完成后 CHG-001～004 里程碑闭环，系统可开始真实试用。

## 2. 关联

- Requirement：FR-04～FR-08、FR-44、FR-47、FR-48、FR-66、FR-67
- 验收：AC-01、AC-02、AC-03、AC-14、AC-16、AC-17、AC-24；EX-04、EX-06、EX-07、EX-08、EX-21、EX-22
- Spec：SPEC-CHAR-001～005、SPEC-CHAR-008、SPEC-WORK-001、SPEC-SYS-002/005、SPEC-TRASH-002/003/004
- Page Spec：PG-02 人物列表、PG-03 人物详情、PG-12 回收站
- 契约：`project/api-contract.md` §3.3、`project/data-model.md` §3.2（F10 回写差异）
- 变更文档：`changes/CHG-004-characters-core/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 迁移 `0004_characters`：`characters`、`character_names`、`character_system_refs.character_id` 外键、部分唯一索引、软删除一致性 CHECK。
- 人物 CRUD：创建、列表（`q` 命中主名称/别名/称号、境界筛选、待校对筛选、排序、分页）、详情、PATCH 全量替换 `names`/`system_refs`、软删除 + 撤销。
- 动态体系字段：按启用类别生成 0..1 节点；类别/节点必须属于当前作品且启用；停用类别拒写。
- 别名/称号：多值、同人物 `kind + value_key` 去重、稳定排序、软上限 20（前端提示）。
- 乐观锁：`version` + `If-Match`；PATCH/DELETE 409 `VERSION_CONFLICT`。
- 回收站：新增 `root_type=character`（列表/筛选/恢复/永久删除/清空），父级显示作品名。
- 体系引用收口：引用清单补人物名与 `in_trash`；回收站人物仍阻断类别停用/删除。
- 作品组联动：作品软删除/恢复/永久删除覆盖人物（`deleted_root_id=work_id`）。
- 前端：`/characters` 列表与详情、新建 Drawer、内联编辑、未保存守卫、冲突提示、删除/引用对话框、导航「资料 > 人物」。

**非目标（已登记延后）**

| 项目 | 承接 |
|---|---|
| 主图上传/缩略图、大事记 | CHG-005 |
| 势力引用真实数据、首领引用阻断 | CHG-006 |
| 地点关联与地点筛选 | CHG-007 |
| 榜单席位/名次、入向引用阻断 | CHG-008 |
| 全局检索命中别名/称号 | CHG-009 |
| `POST /{resource}/{id}/review`、导入 `pending` 人物 | CHG-011 |
| 人物-人物关系、关系图谱、批量操作、富文本简介 | 不做（需求边界） |

## 4. 主要文件/模块变更

**后端**

- 迁移：`backend/internal/migrations/0004_characters.sql`（含 Down 清空 `character_system_refs`）。
- 模型：`model/character.go`、`model/errors.go`（`NAME_CONFLICT`、`CATEGORY_DISABLED`、`CROSS_WORK_REFERENCE`、`NODE_PARENT_MISMATCH`、`PARENT_DELETED` 等）。
- 仓储：`repository/characters.go`（参数化搜索 + `LIKE` 转义、稳定排序白名单、批量装配）、`repository/works.go`（`GetForUpdate` 行锁）、`repository/system.go`（引用清单人物名/`in_trash`）。
- 服务：`service/characters.go`（校验矩阵、体系引用事务校验、乐观锁）、`service/trash.go`（人物根、撤销/恢复、锁序统一的永久删除）、`service/system.go`。
- HTTP：`httpapi/characters.go`、`httpapi/request.go`、`httpapi/router.go`、`cmd/server/main.go`。
- 集成测试：`internal/integration/character_{test,list_test,reference_test,trash_test}.go`、`migrations_test.go`（65 个顶层用例）。

**前端**

- API：`frontend/lib/api/characters.ts`（+ 13 单测）、`lib/messages.ts`。
- 页面/组件：`app/characters/page.tsx`、`components/characters/*`（列表、详情、Drawer、表单、删除/引用对话框、路由）、`components/ui/{table,drawer}.tsx`、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`styles/components.css`。
- E2E：`frontend/e2e/characters.spec.ts`（7）、`a11y/keyboard/visual/system` 适配；视觉基线 `characters-1280/1024`、`character-detail-1280`。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/characters-1280-chromium-win32.png`、`characters-1024-chromium-win32.png`、`character-detail-1280-chromium-win32.png`。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; gofmt -l`（LF 副本，65 个 `.go`） | 无文件需格式化 |
| `cd backend; go vet ./...` / `go build ./...` | 0，无输出 |
| `cd backend; go test ./...` | 全部 ok（config/httpapi/model/service） |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | 65 个顶层用例 PASS（5.982s） |
| 其中 `TestCharacterListPerformance` | 1000 人 P95=4.844ms（≤500ms） |
| 其中 `TestMigrationRoundTrip` / `TestCharacterPurgeSkipsRestoredRoot` | PASS（0004 往返、Purge 竞态防护） |
| `cd frontend; npm run typecheck` / `npm run lint` | 0 |
| `cd frontend; npm run test` | 6 files / 43 tests 通过 |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 8 个静态页面导出成功（本机 Node 24 报 `spawn UNKNOWN`，Node 22 通过） |
| `cd frontend; npx playwright test` | **63 passed（47.2s）** |
| 容器迁移 | `successfully migrated database to version: 4` |

## 7. E2E 与人工验收证据

- E2E 人物 7 例：新建跳详情、搜索别名、境界筛选、待校对/清除筛选、分页、内联保存、409 固定文案、未保存守卫、删除撤销、回收站恢复、1024 无溢出、axe 无 serious/critical。
- 体系引用阻断（`system.spec.ts`）：人物引用类别 → 停用/删除 409，details 含人物名与 `in_trash`。
- F6 QA 8 场景：EX-04 连点保存、EX-06 500 保留输入、DT-02 列表同步、DT-04 人物 404、DT-05 空态、UI-02 无作品引导、UI-03 列表错误态、UI-06 Drawer 焦点归还，全部通过。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-11），已确认接受当前交付与 S2 遗留。
- 审查：`review.md` F4 初审 FAIL（S1=1/S2=15）→ F5 修复 → 独立复审 **PASS**；F6 修复 R-01 锁序、UI-03、UI-06。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0004 新增 2 表 + 1 外键；Down 先清 `character_system_refs` 再删表，`TestMigrationRoundTrip` 覆盖带数据往返。0003 表与数据保留。
- **兼容性**：既有作品/体系契约不变；`CharacterSummary` 新增 `names[]`/`version`、DELETE 要求 `If-Match` 并返回 `undo_token`，已在 F10 回写 `api-contract.md`。
- **数据一致性**：人物写入/恢复在事务内对作品行加锁（C-11）；永久删除统一「先锁作品下人物行」的锁序（R-01），并在事务内复核 `deleted_at`（C-01）。
- **性能**：列表批量装配无 N+1；回收站人物根父名查询仍为 N+1（C-12，已登记技术债）。
- **安全**：写请求保持 `X-Requested-With` 守卫；按 `work_id` 隔离；搜索参数化并转义 `%`/`_`；无新增依赖与密钥。
- **回滚方案**：回退本分支提交；若已迁移，执行 0004 Down（先清引用），或从删除前快照恢复。

## 9. Reviewer 重点关注

1. `service/trash.go` 人物永久删除的 `FOR UPDATE` 复核与 `PurgeAll` 锁序（C-01、R-01）。
2. `service/characters.go` 创建/恢复时作品行锁与 `PARENT_DELETED` 并发语义（C-11）。
3. `replaceSystemRefs` 的跨作品/停用类别/节点归属校验（CH-07～CH-10）。
4. `faction_id` 空串显式拒绝、`location_links` 非空拒绝（C-10、D1）。
5. 列表搜索转义与排序白名单、批量装配是否引入 N+1。
6. 文档一致性：`api-contract.md`、`data-model.md` 三处漂移由 F10 回写（C-14、DOC-02/03）。
7. 合并方式：建议 squash merge 或保留 18 个提交；需用户授权后再合并/推送。

## 10. 已知遗留（不阻断合并）

| ID | 内容 | 承接 |
|---|---|---|
| C-03 | 冲突使用顶部 Alert，未实现独立 ConflictDialog | 后续 UI 变更 |
| C-04 | 列表行菜单、sticky 操作列、1024 隐藏别名列、表头排序、筛选 chips | 后续 UI 变更 |
| C-06 | 删除入口未收进「更多」菜单、删除成功未自动返回列表 | CHG-006/008 接入引用后统一 |
| C-07 | 撤销 Toast 无倒计时与超时提示 | 后续 UI 变更 |
| C-08 | 字段级错误未定位到具体字段 | 后续 UI 变更 |
| C-09 | 停用类别在编辑态未灰显/可清除 | 后续 UI 变更 |
| C-12 | 回收站人物根父名查询 N+1 | 性能专项 |
| C-01/C-11 | 真实并发交错用例待补（防护已实现） | 测试专项 |
