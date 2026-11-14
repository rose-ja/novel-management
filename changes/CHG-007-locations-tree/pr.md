# PR/MR 描述 — CHG-007 地点树与人物/势力关系

> 状态：已本地合并（2026-11-14）。本次变更直接在 `main` 上按阶段提交，提交范围 `f977bbd..fb86b99`（13 个提交，67 文件，+7371/−60）。**未创建远程 PR**；F9 仅做本地 Smoke 与推送 `origin/main`。
> 门禁：F3 自测 PASS、F4 独立审查 FAIL（2×S1 + 5×S2）、F5 修复复审 **PASS**、F6 全量回归 **86 passed**、F7 人工验收 **ACCEPT**（2026-11-14，项目负责人）。

## 1. 标题与目标

**feat: 地点树与人物/势力关系（CHG-007）**

补齐地点实体，让「星域 → 星城 → 秘境」可以用单棵树维护；人物可关联 0..N 地点并带备注，地点可关联 0..N 势力并带备注，人物/势力/地点三侧双向可见与跳转。删除地点时，下级地点与两类关系显式阻止并分组列出引用（含回收站引用），避免世界观断层与死链；软删除、8 秒撤销、回收站与作品组生命周期与 CHG-002/004/006 保持一致。

## 2. 关联

- Requirement：FR-13～FR-16（地点）、FR-34～FR-38（关系）、FR-47～FR-51（`If-Match`）、FR-57～FR-60（回收站）、FR-66/FR-67（永久删除）
- 验收：AC-05、AC-09、AC-14
- Spec：SPEC-LOC-001（地点树）、SPEC-LOC-002（人物-地点）、SPEC-LOC-003（地点-势力与删除阻断）、SPEC-WORK-001（作品隔离）、SPEC-PLAT-005（输入安全）
- Page Spec：PG-06 地点树/列表、PG-07 地点详情、PG-12 回收站（地点根）
- 契约：`project/api-contract.md` §3.5；`project/data-model.md` §3.5（F10 回写实际形态）
- 变更文档：`changes/CHG-007-locations-tree/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 迁移 `0007_locations`：`locations`（自引用父级、软删三件套、`version`、活跃名称唯一部分索引、CHECK）、`character_locations`、`location_factions` 与索引；Down 全部可逆。
- 地点 CRUD：`GET/POST /locations`、`GET/PATCH/DELETE /locations/{id}`；名称 `lower(trim)` 同作品唯一、1–200 字、类型 ≤200、简介 ≤20,000、`If-Match` 乐观锁。
- 树与列表：扁平全量（≤1,000）+ 前端组树；`q` 命中补全部祖先；`type_text`/`review_status`/`parent_id` 过滤；默认展开两级；节点计数（下级/人物/势力）。
- 移动与环：移动到自身/后代 `CYCLE_DETECTED`；同作品移动以 `pg_advisory_xact_lock(hashtext(work_id))` 串行化，`IsAncestor` 递归 CTE 校验。
- 人物-地点关系：统一由 `PATCH /characters/{id}` 的 `location_links` 全量替换（去重保留最后备注、备注 ≤500）；地点页通过该接口改写，避免双入口。
- 地点-势力关系：`GET/POST /locations/{id}/factions`、`PATCH/DELETE /locations/{id}/factions/{faction_id}`；重复 `DUPLICATE_RELATION`。
- 删除阻断：有下级 `HAS_CHILDREN`（一次返回下级地点 + 人物关系 + 势力关系，标注 `in_trash`）；有人物/势力关系 `REFERENCE_CONFLICT`；无引用软删除 + 8 秒撤销。
- 回收站：`root_type=location` 列表（含父级名）/恢复/永久删除/清空；父级已删恢复 `PARENT_DELETED`；作品整组删除/恢复/清空覆盖地点与两张关系表。
- 前端：`/locations` 树页与 `/locations?id=` 详情页、内联新建、编辑、关系编辑、删除阻断 Dialog；人物表单地点关联与列表地点筛选；势力详情关联地点；导航「资料 > 地点」；回收站地点类型。
- 响应式/可访问性：1280/1024、原生 `ul/li` 列表语义 + `aria-current`、axe 无 serious/critical、键盘可达。
- 文档：F10 回写 SPEC-LOC-001~003、`api-contract.md` §3.5、`data-model.md`、roadmap 进度。

**非目标（已登记延后）**

| 项目 | 承接 |
|---|---|
| 校对确认端点 `POST /locations/{id}/review`（默认 `pending` 只读） | CHG-011 |
| 地点图片、地点大事记、地点引用体系 | FR-16 不做 |
| 地图、坐标、关系图谱、多棵树、拖拽排序、级联/强制删除 | 需求边界 |
| 检索索引同步地点 | CHG-009 |
| 真实双连接并发移动自动化用例 | CHG-012 一致性收口 |

## 4. 主要文件/模块变更

**后端**

- 迁移：`backend/internal/migrations/0007_locations.sql`。
- 模型：`model/location.go`（实体、DTO、规范化与上限常量）、`model/faction.go`（`LocationRelationView`）、`model/errors.go`（`ErrDuplicateRelation`）。
- 仓储：`repository/locations.go`（CRUD、`GetForShare`/`GetActiveByWorkForShare`、`IsAncestor`、计数、关系读写、软删/恢复/永久删除、按作品清理）；`repository/{characters,factions,works}.go`（`location_id` 过滤、关系装配与 FK 清理顺序）。
- 服务：`service/locations.go`（列表筛选 + 祖先补齐、CRUD、移动环、删除矩阵合并三类引用、关系维护、事务与共享锁）；`service/characters.go`（`normalizeCharacterLocationLinks`/`replaceLocationLinks`、详情装配）；`service/factions.go`（`relations.locations`）；`service/trash.go`（地点根、恢复、撤销、永久删除、清空）。
- HTTP：`httpapi/locations.go`（10 个端点、`parent_id`/`type_text`/`description` 显式 `null` 清空、`If-Match`）、`httpapi/{request,router}.go`。
- 测试：`integration/{locations,location_relations,location_lifecycle,migrations}_test.go`、`model/location_test.go`、`service/characters_test.go`。

**前端**

- API/纯函数：`lib/api/locations.ts`；`lib/locations-tree.ts`（组树、搜索补祖先、路径、选项、后代集合、类型建议、默认展开）及单测；`lib/api/locations.test.ts`。
- 组件：`components/locations/{location-route,location-list-page,location-tree,location-inline-create,location-detail-page,location-edit-form,location-relations,location-delete-dialog}.tsx`。
- 人物/势力接入：`components/characters/{character-form-fields,character-create-drawer,character-detail-page,character-filters,character-list-page}.tsx`、`components/factions/faction-detail-page.tsx`。
- 页面/导航/回收站：`app/locations/page.tsx`、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`components/trash/trash-dialogs.tsx`。
- 样式与文案：`styles/components.css`（树/内联表单/关系行与 1024 媒体查询）、`lib/messages.ts`。
- 测试：`e2e/locations.spec.ts`（4 例）、`visual.spec.ts`（地点树 1280/1024、详情 1280 基线）、`a11y.spec.ts`、`keyboard.spec.ts`、`e2e/helpers.ts`。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/locations-1280-chromium-win32.png`、`locations-1024-chromium-win32.png`、`location-detail-1280-chromium-win32.png`。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; go build ./...` | 通过 |
| `cd backend; go vet ./...` | 0 |
| `cd backend; go test -count=1 ./...` | 全部 `ok` |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | 全部 `ok`（10.3s） |
| `cd frontend; npm run typecheck` / `npm run lint` | 通过 / 0 warning |
| `cd frontend; npm run test` | **70 passed**（10 文件，含 `locations-tree` 10 项、`api/locations` 6 项） |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 静态导出 10 页面，`/locations` 9.47 kB |
| `docker compose build app; docker compose up -d app` | 镜像重建、容器 healthy，迁移至 **v7** |
| `cd frontend; npx playwright test --reporter=list` | **86 passed**（1.2m） |
| `TestMigrationRoundTrip` | 0007 Down/Up 往返通过（v7） |

## 7. E2E 与人工验收证据

- E2E 地点 4 例：三级内联新建/默认展开与搜索、类型筛选；详情编辑与人物/势力关系双向；删除被下级+人物分组阻断且主操作禁用；删除被人物阻断 → 移除 → 删除 → 返回列表 → 撤销恢复。
- 集成覆盖：迁移 v7、三级 CRUD 与判重、跨作品父级、自我/后代移动、`If-Match` 428/409、搜索与类型补祖先、删除矩阵、人物关系双向与 `location_id` 筛选、人物删除后关系保留且 `in_trash`、势力关系 CRUD 与重复 409、回收站撤销/恢复/永久删除、父级已删恢复阻断、作品组生命周期、清空回收站。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-14，项目负责人），接受 §4 遗留。
- 审查：`review.md` F4 **FAIL**（F4-01/F4-02 S1，F4-03～07 S2）；F5 全部修复后复审 **PASS**。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0007 新建三张表与部分唯一索引，不改动既有表；历史 `location_links`/`relations.locations` 为占位恒空，无需数据回填。
- **兼容性**：`GET /characters/{id}` 的 `location_links` 由恒空占位变为真实数组；`FactionDetail.relations.locations` 由空数组变为真实对象数组；`/locations` 为新增路由。DELETE 地点同样要求 `If-Match`（缺失 428）。
- **FK 约束**：`character_locations.location_id` 与 `location_factions.faction_id` 为 RESTRICT，`location_factions.location_id` 为 CASCADE；永久删除势力前先清 `location_factions`，作品清空顺序为「人物 → 势力（清关系）→ 地点深度倒序」。
- **并发**：`If-Match` 保护文字与结构；移动加同作品 advisory lock；父级校验、人物地点写入、地点-势力新增使用 `FOR SHARE` 避免 TOCTOU。
- **性能**：树单请求全量（≤1,000）+ 前端本地过滤；计数由聚合查询一次取回，避免 N+1；不做虚拟滚动。
- **回滚方案**：回退本变更提交并重新部署；若已迁移执行 0007 Down（仅删三表，不影响人物/势力本体）；或从操作前快照恢复。

## 9. Reviewer 重点关注

1. `service/locations.go` 删除矩阵：有下级时合并三类引用；`HAS_CHILDREN` 与 `REFERENCE_CONFLICT` 的口径与 `in_trash` 语义。
2. `service/characters.go` 的 `replaceLocationLinks`：全量替换、去重保最后备注、`FOR SHARE` 校验同作品活跃。
3. `repository/locations.go` 的 `IsAncestor` 递归 CTE、`GetForShare`、软删/恢复/永久删除的 `deleted_root_id` 处理。
4. `repository/{factions,works}.go` 的 `location_factions` 清理时机与顺序。
5. `httpapi/locations.go` 对 `parent_id`/`type_text`/`description` 的 `null` 清空语义与 `If-Match`。
6. 前端 `lib/locations-tree.ts`（命中补祖先、后代禁用、默认展开）与 `location-delete-dialog.tsx`（两种 `details` 形状、分组、禁用主操作）。
7. 文档一致性：`api-contract.md` §3.5、`data-model.md`、SPEC-LOC 由 F10 回写。

## 10. 已知遗留（不阻断合并）

| ID | 内容 | 承接 |
|---|---|---|
| F4-06 / AC-204 | 并发移动未做真实双连接自动化用例 | CHG-012 |
| — | 势力侧共享锁未加，极端并发可能留下指向回收站势力的关系行（仍阻断删除并标 `in_trash`） | 接受为 S2 |
| — | 删除阻断清单单次最多 50 条，超出不展示但阻断生效 | 后续评估 |
| — | `POST /locations/{id}/review` 未开放，地点默认 `pending` | CHG-011 |
| — | 地点树不做虚拟滚动，>1,000 节点未压测 | 性能专项 |
