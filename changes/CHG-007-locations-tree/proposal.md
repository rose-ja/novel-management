# Change Proposal

- ID: CHG-007-locations-tree
- Title: 地点树与人物/势力关系
- Status: Approved（2026-11-14，F1 按推荐方案批准）
- Owner: 主 Agent
- Related Spec: SPEC-LOC-001、SPEC-LOC-002、SPEC-LOC-003
- Related Issue: roadmap §CHG-007；FR-13~FR-16、FR-34~FR-38、FR-47~FR-51、FR-66、FR-67；PG-06、PG-07；AC-05、AC-09、AC-14

## Problem / Opportunity

CHG-004 为人物保留了 `location_links` 占位（恒空数组），CHG-006 为势力详情保留了 `relations.locations` 占位（恒空数组），但系统中尚不存在地点实体与关系表。用户无法建立「星域 → 星城 → 秘境」的地理层级，也无法把人物与势力挂到具体地点上，导致世界观中的空间信息只能写在简介里，关系页也无法双向跳转。

## User Value

- 用户可以用单棵树维护地点层级与自由文本类型，并用名称搜索、类型筛选、待校对视图快速定位。
- 用户可以在人物详情与地点详情两侧维护同一份「人物-地点」关系及备注，在地点详情维护「地点-势力」关系及备注。
- 用户可以从人物、势力、地点任意一侧跳到另一侧，不存在死链。
- 删除地点时，下级地点、人物关系、势力关系全部显式阻止并列出引用位置，避免误删造成世界观断层。

## Goal

交付一条完整可走通的地点垂直切片：迁移 `0007` 建立地点与关系表；后端提供地点树 CRUD、筛选、关系读写、删除阻断、软删除/撤销/恢复/回收站；前端提供 `/locations` 树页与 `/locations?id=` 详情页，并接入人物与势力详情；全量门禁与 E2E 通过。

## In Scope

- 迁移 `0007_locations`：`locations`、`character_locations`、`location_factions` 三表与索引。
- 地点 CRUD、父子树、移动环校验、名称同作品唯一（大小写等价）、类型自由文本、简介。
- 列表：扁平全量 + 前端组树、默认展开两级、名称搜索命中并补祖先、类型筛选（含祖先链）、`review_status=pending` 筛选、节点计数（下级/人物/势力）。
- 人物-地点关系：通过 `PATCH /characters/{id}` 的 `location_links` 全量替换维护；人物详情展示地点与备注；地点详情展示人物与备注。
- 地点-势力关系：`GET/POST/PATCH/DELETE /locations/{id}/factions`；地点详情与势力详情双向展示及备注。
- 删除地点：下级 `HAS_CHILDREN`；人物/势力关系 `REFERENCE_CONFLICT` 并列出引用位置（含 `in_trash`）；无引用软删除 + 8 秒撤销。
- 回收站 `root_type=location`：列表、恢复（父级已删阻止）、永久删除；作品整组删除/恢复/清空覆盖地点。
- 前端导航「资料 > 地点」、空/加载/错误/冲突/引用阻断状态、1024/1280 响应式、a11y 与键盘可达。
- 文档：F10 回写 SPEC-LOC-001~003、`api-contract.md` §3.5、`data-model.md`、roadmap 进度。

## Out of Scope

- 地点图片、地点大事记、地点引用体系（FR-16 明确不做）。
- 地图、坐标、关系图谱、多棵树、拖拽排序、级联删除、强制删除。
- 校对确认端点 `POST /locations/{id}/review`（与 CHG-006 一致，延后 CHG-011；本轮只读 `review_status` 并支持筛选）。
- 检索索引同步（CHG-009）、导入解析地点（CHG-011）、导出格式扩展（CHG-010）。
- 剧情/势力/榜单等其他域的无关改动。

## Acceptance Criteria

1. 可建立「东荒 → 青丘山 → 狐族秘境」三级地点，默认展开两级；节点类型自由文本且有已有值建议。
2. 名称必填、1–200 字符、同作品唯一（`Sword`/`sword` 视为重复）；类型 ≤200；简介 ≤20,000。
3. 名称搜索命中后自动展开祖先并定位；类型筛选与待校对筛选只过滤命中节点并保留祖先链。
4. 列表行显示名称、类型、上级路径（缩进）、下级数、关联人物数、关联势力数、待校对 Badge。
5. 人物关联 0..N 地点并带备注；人物详情与地点详情两侧显示一致，修改备注后刷新仍保留。
6. 地点关联 0..N 势力并带备注；地点详情与势力详情双向显示并可跳转。
7. 删除地点时：有下级 → `HAS_CHILDREN`；有人物/势力关系 → `REFERENCE_CONFLICT` 并列出引用与跳转；不级联、不静默解除。
8. 无引用删除 → 回收站 + 8 秒撤销；恢复时父级已删 → `PARENT_DELETED`；永久删除无残留。
9. 移动地点到自身或后代 → `CYCLE_DETECTED`；`If-Match` 缺失 428、过期 409。
10. 作品隔离：跨作品父级/关系 `CROSS_WORK_REFERENCE`；两本作品地点互不可见。
11. 1024/1280 无横向溢出；axe 无 serious/critical；树节点键盘可达。

## Constraints and Assumptions

- 技术栈不变：Go + Gin + GORM + PostgreSQL + goose；Next.js 静态导出；不新增第三方依赖。
- 复用 CHG-006 已固化的模式：软删除三件套、`version`/`If-Match`、`name_key = lower(trim(name))` 活跃唯一部分索引、扁平全量 + 前端组树、`pg_advisory_xact_lock(hashtext(work_id))` 串行化移动、8 秒 `undo_token`、回收站 `root_type`。
- `locations` 表在现有迁移中不存在；`character_locations` / `location_factions` 也是新建，无需迁移历史脏数据。
- 人物删除时地点关系为出向引用，随删解除（隐式隐藏，不阻止）；势力删除时地点关联为出向引用，随删解除（隐式隐藏，不阻止），但物理永久删除必须先清理 `location_factions` 行以满足 FK。
- 地点删除阻断按「关系行存在即阻止」处理，引用项返回 `in_trash`；这样可避免恢复后被引用的实体指向已删地点。
- 关系备注上限 500（`/meta` `limits.relation_note`）。

## Dependencies

- CHG-004（人物核心、`location_links` 占位、删除引用矩阵）。
- CHG-006（势力实体、势力详情、回收站与生命周期基座）。

## Risks

| 风险 | 影响 | 缓解 |
|---|---|---|
| 人物/地点双入口写同一关系 | 备注被覆盖 | 人物侧只走 `PATCH /characters/{id}`；地点侧只读人物关系，人物增删改在地点页内部转换为对该人物的 PATCH |
| 删除阻断清单过长 | 大地点详情打不开 | 引用列表最多返回前 50 条 + 总数；前端分组折叠 |
| 物理删除顺序违反 FK | 永久删除/清空回收站失败 | 清空时先清 `location_factions`，人物先于地点；补集成用例 |
| 树搜索 1000 节点 | 卡顿 | 扁平全量 + 前端本地过滤；不做虚拟滚动（与 CHG-006 一致） |
| 迁移 Down 丢数据 | 回滚数据丢失 | 迁移前快照；Down 仅用于集成往返测试 |

## Human Approval

| 决策 | 方案 | 状态 |
|---|---|---|
| D1 校对确认端点 | 延后 CHG-011，本轮只读 `review_status` + 筛选 + Badge | 批准（2026-11-14） |
| D2 人物-地点维护入口 | 仅 `PATCH /characters/{id}` 全量替换，地点页通过该接口编辑 | 批准（2026-11-14） |
| D3 地点-势力备注更新 | 新增 `PATCH /locations/{id}/factions/{faction_id}`（仅改备注） | 批准（2026-11-14） |
| D4 删除阻断口径 | 关系行存在即阻止（含引用方在回收站），返回 `in_trash` | 批准（2026-11-14） |
