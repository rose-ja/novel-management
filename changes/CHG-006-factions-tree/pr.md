# PR/MR 描述 — CHG-006 势力树、首领与成员

> 状态：已本地合并（2026-11-14）。分支 `feat/chg-006-factions-tree` → `main` 快进合并至 `c0777b8`；提交范围 `9467d11..c0777b8`（15 个提交，70 文件，+7083/−74）。**未推送远程、未创建远程 PR**。Smoke Test 通过（见 `verification.md` §8）。
> 门禁：F3 自测 PASS、F4 独立审查 FAIL（3×S1 + 4×S2）、F5 修复复审 **PASS**、F6 全量回归/E2E PASS、F7 人工验收 **ACCEPT**（2026-11-13，项目负责人）。

## 1. 标题与目标

**feat: 势力树、首领与成员（CHG-006）**

用单棵树维护势力层级（万族 → 族 → 分支），成员由人物 `faction_id` 自动推导，首领为 0..1 人物引用。补齐后用户可在一页看清势力结构、按名称搜索并展开祖先、用「仅直属/含下级」核对成员，并在删除/恢复/永久删除全链路遵守入向阻止矩阵，避免半棵树、孤儿大事记与 FK 500。

## 2. 关联

- Requirement：FR-09～FR-12（势力）、FR-24（不做势力引用）、FR-34～FR-38（成员）、FR-44（首领）、FR-47～FR-51（大事记）、FR-66/FR-67（回收站与永久删除）
- 验收：AC-01、AC-26、AC-33（性能）；EX-04、EX-07、EX-21
- Spec：SPEC-FACTION-001～004、SPEC-CHAR-001（势力字段）、SPEC-CHAR-008（首领入向阻断）、SPEC-TRASH-004（永久删除）
- Page Spec：PG-04 势力树/列表、PG-05 势力详情、PG-12 回收站（势力根）
- 契约：`project/api-contract.md` §2.3、§3.4、§3.8、§6.10；`project/data-model.md` §3.2/§3.4/§3.13（F10 回写实际形态）
- 变更文档：`changes/CHG-006-factions-tree/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 迁移 `0006_factions_chronicles`：`factions` 表（自引用父级、首领、软删三件套、`version`、名称唯一、CHECK）；`characters.faction_id` 补 FK 与部分索引；`chronicles.owner_type` 放宽为 `character/faction`。
- 势力 CRUD：`GET/POST /factions`、`GET/PATCH/DELETE /factions/{id}`；名称 `lower(trim)` 同作品唯一、1–200 字、简介 ≤20,000、`If-Match` 乐观锁。
- 树与列表：扁平全量（≤1000）+ 前端组树；`q` 返回命中 + 全部祖先；`review_status`、`parent_id` 过滤；默认展开两级。
- 首领：当前作品活跃人物、可清空、改名同步；回收站首领标 `in_trash` 并提供回收站定位，保存时省略未变更字段。
- 成员：`GET /factions/{id}/members?scope=direct|subtree`，递归 CTE 去重、分页 20；摘要条直属/含下级/下级分支计数。
- 移动与环：移动到自身/直接子/深层后代 `CYCLE_DETECTED`；同作品移动以事务级 advisory lock 串行化。
- 删除矩阵：下级（含软删子节点）`HAS_CHILDREN`、活跃成员 `REFERENCE_CONFLICT`；首领与势力大事记为出向随删隐藏；返回 8 秒 `undo_token`。
- 回收站：`root_type=faction` 列表/恢复/永久删除/清空；父级缺失恢复 `PARENT_DELETED`；永久删除前复核活跃成员，永不 500。
- 势力大事记：复用 `chronicles` 与通用 `ChronicleSection`，CRUD/排序/2,000 字/版本/8 秒撤销与人物一致。
- 人物接入：人物新建/编辑设置或清空势力；列表 `faction` 显示与筛选；详情链接；人物删除在「担任首领」时被阻断。
- 生命周期：作品整组软删/恢复/永久删除覆盖 `factions` 与势力大事记；双 RESTRICT FK 按「清首领引用 → 删人物 → 势力深度倒序」处理。

**非目标（已登记延后）**

| 项目 | 承接 |
|---|---|
| 地点-势力关系与地点页（`relations.locations` 恒空） | CHG-007 |
| 校对确认端点 `POST /factions/{id}/review`（默认 `reviewed`） | CHG-011 |
| 榜单席位 / 势力引用体系 | CHG-008 / FR-24 不做 |
| 全局检索命中势力 | CHG-009 |
| 多棵树、拖拽排序、副首领、批量成员、势力图片 | 需求边界 |
| 树虚拟滚动、1000 节点压测、慢网专项 | 性能专项（FAC-09/UI-11） |

## 4. 主要文件/模块变更

**后端**

- 迁移：`backend/internal/migrations/0006_factions_chronicles.sql`（Up/Down 可逆；Up 先置空历史脏 `faction_id` 再建 FK）。
- 模型：`model/faction.go`（`Faction`/`FactionSummary`/`FactionDetail`/`FactionRelations`、名称与简介规范化、上限常量）、`model/errors.go`（`CYCLE_DETECTED`、`ConflictItem.in_trash`）、`model/character.go`（`FactionBrief.in_trash`）、`model/media.go`（`ChronicleOwnerFaction`）。
- 仓储：`repository/factions.go`（递归祖先/子树 CTE、计数、锁、软删/恢复/永久删除、`LeaderBriefs`、`GetDeletedForUpdate`）；`repository/{characters,chronicles,works}.go`（聚合读取、按根/作品计数与清理、作品组与 PurgeAll 的 FK 顺序）。
- 服务：`service/factions.go`（列表搜索 + 祖先补齐、CRUD、环检测、成员、删除矩阵、advisory lock）；`service/characters.go`（真实势力校验、首领入向阻断、`faction` 批量装配）；`service/chronicles.go`（faction owner）；`service/trash.go`（势力根列表/恢复/撤销/永久删除、清空容错、活跃成员复核）。
- HTTP：`httpapi/factions.go`（RawMessage 处理 `null` 清空、`Location`、`If-Match`）、`httpapi/{request,router}.go`、`cmd/server/main.go`。
- 测试：`integration/{factions,faction_members,faction_character,faction_chronicle,faction_lifecycle,faction_review_fix,migrations}_test.go`。

**前端**

- API/纯函数：`lib/api/factions.ts`、`lib/api/characters.ts`（`faction.faction_id`/`in_trash`）、`lib/api/chronicles.ts`、`lib/factions-tree.ts`（组树、命中 + 祖先、路径、选项中禁用后代、默认展开）及单测。
- 组件：`components/factions/{faction-route,faction-list-page,faction-tree,faction-inline-create,faction-detail-page,faction-edit-form,faction-delete-dialog,member-table}.tsx`、通用 `components/chronicles/chronicle-section.tsx`（替换人物专用实现）。
- 人物接入：`components/characters/{character-form-fields,character-create-drawer,character-detail-page,character-table,character-filters,character-delete-dialog}.tsx`。
- 页面/导航：`app/factions/page.tsx`、`app/trash/page.tsx`、`components/trash/trash-dialogs.tsx`、`components/shell/global-nav.tsx`。
- 样式：`styles/components.css`（`.faction-tree*` 行高 36px、内联表单、面包屑、摘要、成员表与 1024 媒体查询）。
- 测试：`e2e/factions.spec.ts`（6 例）、`visual.spec.ts`（3 张新基线）、`a11y.spec.ts`、`keyboard.spec.ts`。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/factions-1280-chromium-win32.png`、`factions-1024-chromium-win32.png`、`faction-detail-1280-chromium-win32.png`。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; go build ./...` | 通过 |
| `cd backend; go vet ./...` | 0 |
| `cd backend; go test -count=1 ./...` | 全部 `ok`（config/httpapi/model/service） |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./...` | 全部 `ok`（`internal/integration` 14.3s，含新增 4 项 F5 用例） |
| `cd frontend; npm run typecheck` / `npm run lint` | 通过 / 0 warning |
| `cd frontend; npm run test` | **54 passed**（`factions-tree` 8 项） |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 静态导出成功（Node 24 下 Next 构建失败，沿用既定 Node 22 方式） |
| `cd frontend; npx playwright test` | **76 passed**（全量，含势力 6 例与视觉 13 项） |
| `docker compose build app && docker compose up -d app` | 通过；容器迁移 `successfully migrated database to version: 6` |
| `TestMigrationRoundTrip` | 0006 Down/Up 往返通过（v6） |

## 7. E2E 与人工验收证据

- E2E 势力 6 例：三级树默认展开与搜索祖先补齐；内联新建与详情编辑（首领走服务端搜索）；成员直属/含下级切换与人物跳转；删除被下级阻止；软删下级标注（回收站）；回收站恢复/永久删除。
- 集成覆盖：迁移 v6、三级 CRUD 与判重、跨作品、环检测与版本冲突、成员子树去重与分页、人物势力接入、首领删除阻断（含回收站势力）、势力大事记 CRUD/撤销、删除矩阵、回收站根、父级缺失恢复、作品组生命周期、永久删除活跃成员阻断、清空回收站重复根。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-13，验收人：项目负责人），接受 §4 遗留。
- 审查：`review.md` F4 **FAIL**（RV-01～03 S1、RV-04～07 S2）；F5 全部修复后复审 **PASS**。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0006 新增 `factions` 并给 `characters.faction_id` 加 FK/索引；历史非空脏值在 Up 中先置空。Down 删除势力与 FK、恢复 `chronicles.owner_type` 仅 character，并清理 `owner_type=faction` 大事记（回滚存在数据损失，已记录）。
- **FK 环**：`characters.faction_id → factions` 与 `factions.leader_character_id → characters` 双 RESTRICT；永久删除严格按「清首领引用 → 删人物 → 势力深度倒序」；单势力 Purge 前复核活跃成员，防止恢复人物后 500。
- **兼容性**：`CharacterSummary/Detail.faction` 由占位改为真实对象（新增 `in_trash`）；`GET /factions` 返回 `{items,total}`；DELETE 强制 `If-Match`（缺失 428）；未知 `owner_type` 仍 `VALIDATION_ERROR`。旧数据无需回填。
- **并发**：`If-Match` 保护文字；移动在事务内锁自身 + 同作品 advisory lock 后做环检测；删除/永久删除锁根复核成员。
- **性能**：树单请求全量（≤1000）；成员子树递归 CTE + 分页；列表计数聚合避免 N+1。1000 节点压测与慢网未做（已登记）。
- **回滚方案**：回退本分支提交并重新部署；若已迁移执行 0006 Down；或从操作前快照恢复（快照格式未变，含全部势力数据）。

## 9. Reviewer 重点关注

1. `service/trash.go` 势力永久删除：预检 + 事务内 `GetDeletedForUpdate` + 活跃成员复核，以及与 `PurgeAll` 的根已消失容错（RV-01/02）。
2. `repository/factions.go` 的递归 CTE（`IsAncestor`/`SubtreeIDs`/成员）与 `Purge` 的 FK 清理顺序；`works.go` 中「清首领引用 → 删人物 → 删势力」。
3. `service/factions.go` 删除矩阵顺序（下级先于成员）、`HAS_CHILDREN.items[].in_trash`、`CYCLE_DETECTED` 与 advisory lock 的锁序。
4. `service/characters.go` 首领入向检查（含回收站势力）与 `factionBriefs` 的 `in_trash` 装配。
5. `httpapi/factions.go` 对 `parent_id`/`leader_character_id`/`description: null` 的显式清空语义与 `If-Match` 处理。
6. 前端 `lib/factions-tree.ts`（命中 + 祖先、自身/后代禁用、默认展开）与 `faction-edit-form.tsx`（首领搜索防抖、回收站首领省略字段）。
7. 文档一致性：`api-contract.md`、`data-model.md`、SPEC-FACTION 由 F10 回写（含 `{items,total}`、DELETE `If-Match`、members 对象形态）。

## 10. 已知遗留（不阻断合并）

| ID | 内容 | 承接 |
|---|---|---|
| RV-06 | 并发移动以 advisory lock 串行化，未新增真实并发集成用例 | 并发测试专项 |
| — | 单根永久删除对已不存在根返回 409 `ALREADY_DELETED`（非幂等） | 后续契约评估 |
| FAC-09 / UI-11 | 1000 节点压测、慢网计数后到未自动化 | 性能专项 |
| UI-04 | 保存失败注入未自动化（手工覆盖） | 后续测试专项 |
| — | `relations.locations` 恒空、校对端点未开放 | CHG-007 / CHG-011 |
| — | 契约偏差待回写 | F10 |
