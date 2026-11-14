# CHG-007 设计

- Change ID：`CHG-007-locations-tree`
- 依据：`proposal.md`、SPEC-LOC-001~003、PG-06、PG-07、`api-contract.md` §3.5、`data-model.md` §3.5/§3.11/§3.12
- 决策：D1~D4（见 proposal §Human Approval）；技术模式沿用 CHG-006（软删除三件套、扁平全量、`If-Match`、移动串行化、8 秒撤销）

## 1. 总体方案

1. 迁移 `0007_locations.sql` 新建 `locations`、`character_locations`、`location_factions`，补齐索引与 CHECK。
2. 后端新增 `model/location.go`、`repository/locations.go`、`service/locations.go`、`httpapi/locations.go`，实现树 CRUD、筛选、关系读写、删除阻断与生命周期。
3. 人物接入：`location_links` 由占位改为真实关系（全量替换），人物详情返回地点名与备注；人物删除不阻止，关系随删隐式隐藏。
4. 势力接入：详情 `relations.locations` 返回真实地点与备注；势力删除不阻止（出向），永久删除前清理 `location_factions`。
5. 回收站新增 `root_type=location`，并纳入作品整组删除/恢复/清空。
6. 前端新增 `/locations`（树 + 内联新建）与 `/locations?id=`（详情 + 内联编辑 + 关系 + 删除），接入导航、人物详情、势力详情与回收站。
7. 迁移可逆；F10 回写 Spec、契约、数据模型与 roadmap。

## 2. 数据模型与迁移

### 2.1 `0007_locations.sql`（Up）

```sql
CREATE TABLE locations (
    id                TEXT PRIMARY KEY,
    work_id           TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    name_key          TEXT NOT NULL,
    parent_id         TEXT REFERENCES locations (id) ON DELETE RESTRICT,
    type_text         TEXT,
    description       TEXT,
    review_status     TEXT NOT NULL DEFAULT 'reviewed'
                      CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at       TIMESTAMPTZ,
    reviewed_by       TEXT,
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    deleted_root_id   TEXT,
    deleted_by_action TEXT,
    CONSTRAINT locations_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT locations_type_len CHECK (type_text IS NULL OR char_length(type_text) <= 200),
    CONSTRAINT locations_desc_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT locations_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX locations_active_name_uq
    ON locations (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX locations_active_parent_idx
    ON locations (work_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX locations_deleted_root_idx
    ON locations (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE character_locations (
    id           TEXT PRIMARY KEY,
    work_id      TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    character_id TEXT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    location_id  TEXT NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    note         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT character_locations_note_len CHECK (note IS NULL OR char_length(note) <= 500),
    CONSTRAINT char_loc_uq UNIQUE (character_id, location_id)
);
CREATE INDEX character_locations_location_idx ON character_locations (location_id);

CREATE TABLE location_factions (
    id          TEXT PRIMARY KEY,
    work_id     TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    location_id TEXT NOT NULL REFERENCES locations (id) ON DELETE CASCADE,
    faction_id  TEXT NOT NULL REFERENCES factions (id) ON DELETE RESTRICT,
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT location_factions_note_len CHECK (note IS NULL OR char_length(note) <= 500),
    CONSTRAINT loc_fac_uq UNIQUE (location_id, faction_id)
);
CREATE INDEX location_factions_faction_idx ON location_factions (faction_id);
```

### 2.2 Down

```sql
DROP TABLE IF EXISTS location_factions;
DROP TABLE IF EXISTS character_locations;
DROP TABLE IF EXISTS locations;
```

> Down 丢弃地点与两类关系（数据损失），已在 proposal §Risks 记录；迁移前应生成快照。

### 2.3 约束与删除顺序

- `locations.parent_id` 为自引用 `RESTRICT`，禁止环；物理删除必须自叶向根，或先删子节点。
- `character_locations.character_id` 为 `CASCADE`（归属集合）：人物永久删除时关系自动清理。
- `location_factions.location_id` 为 `CASCADE`（归属集合），`faction_id` 为 `RESTRICT`（跨实体引用）：势力永久删除前必须先删 `location_factions` 行。
- 关系表不加删除列；可见性由父实体 `deleted_at` 过滤（ADR-004）。
- 名称唯一只对活跃行生效；地点进回收站后可新建同名。

## 3. 后端模块

### 3.1 `model/location.go`

- 常量：`MaxLocationName = 200`、`MaxLocationType = 200`、`MaxLocationDescription = 20000`、`MaxRelationNote = 500`、`LocationActionDelete = "location_delete"`。
- `Location`（GORM，表 `locations`）：`ID/WorkID/Name/NameKey/ParentID/TypeText/Description/ReviewStatus/ReviewedAt/ReviewedBy/Version/时间/软删三件套`。
- DTO：
  - `LocationBrief{ID,Name}`；
  - `LocationSummary{ID,WorkID,Name,Parent *LocationBrief,TypeText *string,Description *string,ReviewStatus,ReviewedAt,ChildrenCount,CharactersCount,FactionsCount,Version,CreatedAt,UpdatedAt,DeletedAt}`；
  - `LocationChildView{ID,Name,TypeText,CharactersCount,FactionsCount}`；
  - `LocationCharacterView{CharacterID,Name,Faction *FactionBrief,SystemSummary *string,ImageThumbURL *string,Note *string,InTrash bool}`；
  - `LocationFactionView{FactionID,Name,Note *string,InTrash bool}`；
  - `LocationRelations{Children []LocationChildView, Characters []LocationCharacterView, Factions []LocationFactionView}`（`parent` 位于 `LocationSummary`）；
  - `LocationDetail{LocationSummary, Relations}`。
- `NormalizeLocationName`：`lower(trim)` 生成 `name_key`；`NormalizeLocationType` 空串 → nil；`NormalizeLocationDescription` 空串 → nil；`NormalizeRelationNote` 空串 → nil 且 ≤500。
- `model/character.go`：`LocationLinkView` 注释更新为真实关系（结构不变）。
- `model/faction.go`：新增 `LocationRelationView{ID,Name,Note,InTrash}` 替换占位 `LocationBrief`。

### 3.2 `repository/locations.go`

- 读：`Get`、`GetForUpdate`、`GetDeleted`、`ListActive(workID, q, reviewStatus, typeText)`（扁平全量，含全部祖先补齐由 service 完成）、`ListActiveChildren(parentID)`、`CountActiveChildren`、`BatchRelationCounts(workID)`（一次返回每地点的活跃人物数/势力数）、`BatchChildCounts(workID)`。
- 写：`Insert`、`UpdateWithVersion`（`version+1`）、`SoftDelete`、`Restore`、`RenameDeleted`、`MarkWorkDeleted`、`RestoreByWork`。
- 树：`IsDescendant(rootID, candidateID)`（递归 CTE）、`ParentsFor(ids)`。
- 关系（人物）：`CharacterLocations(characterID)`、`ReplaceCharacterLocations(characterID, []RelationInput)`（事务内全量替换）、`CharactersForLocation(locationID)`（联表人物/势力/体系摘要/缩略图，过滤人物活跃，返回备注）。
- 关系（势力）：`FactionsForLocation(locationID, includeTrashed bool)`、`InsertLocationFaction`、`UpdateLocationFactionNote`、`DeleteLocationFaction`、`LocationsForFaction(factionID)`（过滤地点活跃）。
- 删除引用：`CountInboundReferences(locationID)` 返回下级/人物关系/势力关系计数；`InboundReferenceItems(locationID, limit)` 返回 `model.ConflictItem`（`entity_type=character|faction|location`、`name`、`in_trash`）。
- 生命周期：`Purge(locationID)`（先删 `location_factions`，再删地点行）、`PurgeByWork(workID)`（地点按深度倒序）。
- 排他：`HasActiveName`、`ActiveSiblingCount`（排序追加末尾用）。

### 3.3 `service/locations.go`

- `List`：校验作品活跃；取当前作品全部活跃地点（≤1000）；`q` 命中后补全部祖先；`review_status=pending` 与 `type_text` 同理补祖先；批量装配父级、下级数、人物数、势力数。
- `Create`：作品活跃；父级同作品活跃（跨作品 `CROSS_WORK_REFERENCE`，已删 `PARENT_DELETED`）；名称规范化与唯一；类型/简介规范化；同级默认追加末尾（无 `sort_order`，按 `name_key` 稳定排序，与 CHG-006 一致）。
- `Update`：`If-Match` 必填；名称/父级/类型/简介子集；移动时 `pg_advisory_xact_lock(hashtext(work_id))` 串行化，目标不能等于自身或后代 → `CYCLE_DETECTED`；版本冲突返回当前版本。
- `Get`：返回 `LocationDetail`，装配 parent、children（活跃）、characters（含回收站标记）、factions（含回收站标记）；回收站引用保留展示，用于解释删除阻断，但不计入活跃计数。
- `Delete`：事务内：活跃下级（含软删子节点）→ `HasChildrenError`（`items[]` 同时包含下级与人物/势力关系，一次展示全部阻断）；无下级但存在关系行（含引用方在回收站）→ `ReferenceConflictError`（`items[]` 带 `in_trash`）；无引用 → 软删除 + 8 秒 `undo_token`。
- `Relation` 方法：
  - `AddFaction(locationID, factionID, note)`：地点活跃、势力同作品活跃；重复 → `DUPLICATE_RELATION`。
  - `UpdateFactionNote(locationID, factionID, note)`：关系不存在 → `NOT_FOUND`。
  - `RemoveFaction(locationID, factionID)`：不存在 → `NOT_FOUND`；只删关系。
- `CharacterLocations`：读取供人物详情使用；人物侧写入由人物服务完成（见 3.4）。

### 3.4 人物接入（`service/characters.go`、`repository/characters.go`）

- `location_links` 入参：`[]CharacterLocationLinkInput{LocationID, Note}`；`PATCH` 全量替换，`POST` 创建时写入。
- 校验：去重（重复 `location_id` 去重保留最后一个备注）；地点必须同作品活跃（跨作品 `CROSS_WORK_REFERENCE`，已删 `PARENT_DELETED`，`field=location_links`）；`note` ≤500。
- `CharacterDetail.location_links`：批量装配地点名与备注（活跃地点；若地点在回收站则 `in_trash: true`，正常不应出现）。
- 列表 `location_id` 筛选启用：`EXISTS (SELECT 1 FROM character_locations ...)` 且地点活跃。
- 人物删除：地点关系是出向引用，不加入阻断；关系行保留并由人物软删除隐式隐藏。

### 3.5 势力接入（`service/factions.go`、`repository/factions.go`）

- `FactionDetail.relations.locations`：批量查询 `location_factions` 联表活跃地点，返回 `{id,name,note,in_trash}`。
- 势力删除：地点关联为出向引用，不阻止；关系行保留并随势力软删除隐式隐藏。
- 势力恢复：关系原样呈现（地点需活跃；若地点在回收站则 `in_trash=true`，正常不应出现，因为地点删除会被关系阻止）。
- 永久删除：`Purge` / `PurgeByWork` / `PurgeByDeletedWorks` 先 `DELETE FROM location_factions WHERE faction_id = ?` 再删势力行。
- 势力删除引用检查不变（下级/成员）；地点关系不加入。

### 3.6 回收站与生命周期（`service/trash.go`、`repository/works.go`）

- 新增 `RootTypeLocation = "location"`，加入 List/Restore/Purge/Undo 白名单。
- 回收站列表：名称、父级名（`parent_name`）、影响条数（地点自身）；`deleted_at` 取根。
- 恢复：父级不在线 → `PARENT_DELETED`；成功恢复地点。
- 永久删除：事务内先删 `location_factions`，再删地点；若仍存在 `character_locations`（理论不可达，防御）→ `REFERENCE_CONFLICT` 并列出人物。
- 作品组：`SoftDeleteGroup`/`RestoreGroup` 追加 `locations`；`PurgeGroup`/`PurgeAllDeleted` 在人物与势力之后删除地点（清空 `location_factions` 后按深度倒序）；清空回收站计数纳入地点根。

### 3.7 `httpapi/locations.go`

- `registerLocations`：
  - `GET /locations`、`POST /locations`；
  - `GET /locations/:id`、`PATCH /locations/:id`、`DELETE /locations/:id`；
  - `GET /locations/:id/characters`（只读）；
  - `GET /locations/:id/factions`、`POST /locations/:id/factions`、`PATCH /locations/:id/factions/:faction_id`、`DELETE /locations/:id/factions/:faction_id`。
- 请求体：创建 `{work_id,name,parent_id,type_text,description}`；更新任意子集 + `If-Match`；新增势力关系 `{faction_id,note}`；备注更新 `{note}`。
- 删除返回 `{deleted:true,undo_token}`；恢复走既有 `/trash/location/{id}/restore`。
- `request.go` 复用 `CYCLE_DETECTED → 422`、`HAS_CHILDREN`、`REFERENCE_CONFLICT`、`DUPLICATE_RELATION` 映射；`router.go`/`main.go` 注入 `Locations` 服务。

## 4. 前端设计

### 4.1 API 层与纯函数

- `lib/api/locations.ts`：`LocationSummary/LocationDetail/LocationChild/LocationCharacter/LocationFaction` 类型；`listLocations`、`getLocation`、`createLocation`、`updateLocation(ifMatch)`、`deleteLocation`、`listLocationCharacters`、`listLocationFactions`、`addLocationFaction`、`updateLocationFactionNote`、`removeLocationFaction`。
- `lib/api/characters.ts`：`location_links` 类型改为真实 `{location_id, name?, note}`，创建/更新请求携带 `location_links`；`location_id` 筛选参数启用。
- `lib/api/factions.ts`：`relations.locations` 类型增加 `note`、`in_trash`。
- `lib/api/trash.ts`：根类型增加 `location`。
- `lib/locations-tree.ts`：纯函数 `buildLocationTree`、`filterLocations(nodes, {q,type,reviewStatus})`（命中 + 祖先）、`locationPath(nodes,id)`、`locationTypeOptions(nodes)`、`descendantIDs(nodes,id)`；配 `locations-tree.test.ts`。
- `lib/messages.ts`：地点列表/详情、创建/保存/删除/撤销、`CYCLE_DETECTED`、`HAS_CHILDREN`、`REFERENCE_CONFLICT`、空态与引用分组文案。

### 4.2 列表树（PG-06）

- `app/locations/page.tsx` + `LocationRoute`：读取 `?id=`；无作品显示 PG-00 无作品态。
- `LocationListPage`：页头（当前作品 + 节点总数 + 新建地点节点）、FilterBar（搜索/类型筛选/待校对/展开全部/收起全部/重置）、`LocationTree`、`LocationInlineCreate`。
- 树：一次 `listLocations(work_id)` 取扁平全量，`useMemo` 组树；默认展开两层；搜索 300ms debounce，命中 `<mark>` 高亮并展开祖先、滚动到首个命中；类型选项来自当前节点已有 `type_text` 去重；筛选保留祖先链。
- 行：名称链接 `/locations?id=`、类型、下级数、关联人物数、关联势力数、待校对 Badge；展开按钮阻止冒泡。
- 内联新建：上级 TreeSelect（可空）+ 名称 + 类型（datalist 建议）+ 简介；保存后展开父级、选中新节点、Toast；有输入取消需确认；保存失败保留输入。

### 4.3 详情（PG-07）

- `LocationDetailPage`：面包屑（由扁平列表计算路径）、页头（名称/待校对/编辑/更多）、摘要条（下级/关联人物/关联势力）、简介、下级地点、关联人物表、关联势力列表、侧栏档案。
- 内联编辑：名称、上级 TreeSelect（禁用自身与后代，清空=顶层）、类型（datalist）、简介；保存先 PATCH 再刷新；失败保留输入；`VERSION_CONFLICT` 顶部 Alert。
- 关联人物（地点页）：只读表格 + 「编辑关联」进入关系编辑面板：多选当前作品人物，每项备注；保存时对发生变化的每个人物调用 `PATCH /characters/{id}`（携带该人物完整 `location_links`），不删除人物。
- 关联势力（地点页）：关系编辑面板：多选当前作品势力 + 备注；新增 `POST`，备注变更 `PATCH`，移除 `DELETE`。
- 删除：`LocationDeleteDialog` 先调 DELETE；`HAS_CHILDREN` / `REFERENCE_CONFLICT` 按 `location_child` / `character_location` / `location_faction` 分组列出引用，提供跳转，回收站中的引用显示徽标；冲突后禁用删除主操作；成功后 `router.push("/locations")` 并在列表页 Toast 撤销。
- 并发冲突与未保存守卫复用人物/势力实现模式。

### 4.4 人物与势力侧接入

- 人物详情编辑：新增「关联地点」多选 + 每项备注（当前作品地点，可清空），保存并入人物 PATCH；查看态显示地点链接与备注。
- 人物列表：`location_id` 筛选启用（若现有筛选栏有位置，则加「地点」下拉）。
- 人物删除：地点关系不阻止，无需新增阻断提示。
- 势力详情：关联地点区由空态改为真实列表（地点链接 + 备注）；无地点时保留空态。
- 势力删除：地点关联不阻止，无需新增阻断提示。

### 4.5 回收站与导航

- `app/trash/page.tsx`：类型筛选增加「地点」；行显示父级；恢复失败展示「请先恢复上级地点」。
- `global-nav.tsx`：资料下新增「地点」（人物、势力、地点顺序）。

## 5. 错误与边界

| 场景 | 错误码 | HTTP | details |
|---|---|---:|---|
| 名称空/超 200 | `VALIDATION_ERROR` | 422 | `field=name` |
| 类型超 200 / 简介超 20,000 / 备注超 500 | `VALIDATION_ERROR` | 422 | 对应字段 |
| 同作品重名 | `NAME_CONFLICT` | 409 | — |
| 父级跨作品/不存在 | `CROSS_WORK_REFERENCE` / `PARENT_DELETED` | 422 / 409 | `field=parent_id` |
| 移动到自身或后代 | `CYCLE_DETECTED` | 422 | `field=parent_id` |
| 版本不一致 / 缺失 `If-Match` | `VERSION_CONFLICT` / `PRECONDITION_REQUIRED` | 409 / 428 | `current_version` |
| 仍有下级（含软删） | `HAS_CHILDREN` | 409 | `items[]` |
| 仍有人物/势力关系 | `REFERENCE_CONFLICT` | 409 | `items[]` 含 `in_trash` |
| 人物引用跨作品/已删地点 | `CROSS_WORK_REFERENCE` / `PARENT_DELETED` | 422 / 409 | `field=location_links` |
| 重复地点-势力关系 | `DUPLICATE_RELATION` | 409 | — |
| 恢复时父级已删 | `PARENT_DELETED` | 409 | — |

## 6. 测试设计

### 6.1 后端单元

- 名称/类型/简介/备注规范化与长度；`name_key` 大小写等价；`NormalizeRelationNote` 空串转 nil。
- 前端纯函数同构逻辑不在后端重复。

### 6.2 集成（真实 PostgreSQL，事务回滚）

- 迁移：`TestMigrationRoundTrip` v7；0007 Down 删除三表。
- 树与 CRUD：三级创建、名称冲突（大小写）、跨作品父级、`q`/类型/待校对返回命中 + 祖先、计数正确。
- 移动：自身/直接子/深层后代 → `CYCLE_DETECTED`；合法移动后父子与计数更新。
- 人物关系：创建/全量替换/备注保留/去重/跨作品拒绝/已删地点拒绝；人物详情 `location_links` 双向一致；人物删除不阻止并隐式隐藏。
- 势力关系：新增/重复 409/备注更新/移除/跨作品拒绝；地点与势力详情双向一致；势力删除不阻止并隐式隐藏；势力永久删除清理关系行。
- 删除阻断：有下级 → `HAS_CHILDREN`；人物关系/势力关系 → `REFERENCE_CONFLICT`（含引用方在回收站）；无引用 → 删除 + `undo_token`。
- 生命周期：撤销恢复；回收站 `location` 列表/恢复/永久删除；父级已删恢复 `PARENT_DELETED`；作品组删除/恢复/清空覆盖地点与关系。
- 并发：两请求同时移动成环，串行化后仅一个成功。

### 6.3 前端单测

- `lib/api/locations.test.ts`：错误映射、`If-Match` 头、关系请求方法。
- `lib/locations-tree.test.ts`：组树、默认展开、搜索命中 + 祖先、类型/待校对过滤保留祖先、路径、类型选项去重、后代集合。

### 6.4 E2E（`e2e/locations.spec.ts`）

1. 三级内联新建（东荒 → 青丘山 → 狐族秘境）、默认展开两级、重名提示。
2. 搜索定位并展开祖先、无匹配提示、类型筛选保留祖先。
3. 人物关联两个地点并填备注；人物详情与地点详情双向显示、备注修改保留。
4. 地点关联势力并填备注；两侧详情双向显示并可跳转。
5. 删除阻断：下级 / 人物关系 / 势力关系分组列出并可跳转；清空后删除 → Toast 撤销 → 回收站恢复。
6. 移动父级禁用自身/后代 + 422 提示。
7. 1024/1280 无横向溢出；axe 无 serious/critical；树节点键盘可达。

### 6.5 门禁命令

```powershell
cd backend; gofmt -l .; go vet ./...; go build ./...; go test ./...
$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
go test -count=1 -tags=integration ./internal/integration/...
cd frontend; npm run typecheck; npm run lint; npm run test
npx -y node@22 node_modules/next/dist/bin/next build
npx playwright test
```

## 7. 兼容性与迁移

- `characters.location_links` 从「仅空数组」变为真实关系，属于契约扩展；旧客户端传空数组仍可用。
- `factions.relations.locations` 从恒空变为真实数组，属于契约扩展；新增字段 `note`/`in_trash` 向后兼容。
- 回收站新增 `location` 类型；前端同步升级，旧客户端忽略未知类型。
- 新增 `PATCH /locations/{id}/factions/{faction_id}` 为契约扩展（D3），F10 回写。
- 快照/导出格式不变；导入与检索在后续变更接入。

## 8. 实施顺序与并行边界

1. T1 迁移 + 模型 → T2 仓储 → T3 服务 CRUD/树/筛选 → T4 HTTP → T5 关系 → T6 人物接入 → T7 势力接入 → T8 生命周期/回收站。
2. T9 前端 API + 纯函数 → T10 列表树 → T11 详情与关系编辑 → T12 人物/势力侧 + 导航 → T13 回收站/E2E/视觉/a11y/回归。
3. 并行边界：T2/T3 与 T9 可在契约冻结后并行；T10 依赖 T9；T11/T12 依赖 T4/T5/T7；T13 最后执行。
4. 每个任务独立提交；T4 后跑一次后端集成，T8 后跑全量集成，T12 后跑前端门禁，T13 全量回归。

## 9. 文件级改动清单

**后端**

- 新增：`internal/migrations/0007_locations.sql`、`internal/model/location.go`、`internal/repository/locations.go`、`internal/service/locations.go`、`internal/httpapi/locations.go`、`internal/integration/{locations,location_relations,location_lifecycle}_test.go`。
- 修改：`internal/model/{character,faction,errors}.go`、`internal/repository/{characters,factions,works}.go`、`internal/service/{characters,factions,trash}.go`、`internal/httpapi/{request,router}.go`、`cmd/server/main.go`、`internal/integration/{suite,migrations}_test.go`。

**前端**

- 新增：`app/locations/page.tsx`、`components/locations/{location-route,location-list-page,location-tree,location-inline-create,location-detail-page,location-edit-form,location-relations,location-delete-dialog}.tsx`、`lib/api/locations.ts`、`lib/api/locations.test.ts`、`lib/locations-tree.ts`、`lib/locations-tree.test.ts`、`e2e/locations.spec.ts`。
- 修改：`components/characters/{character-detail-page,character-form-fields,character-filters}.tsx`、`components/factions/faction-detail-page.tsx`、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`lib/api/{characters,factions,trash}.ts`、`lib/messages.ts`、`styles/components.css`、`e2e/{visual,a11y,keyboard}.spec.ts` 与视觉基线。

## 10. 待人工确认（F1）

见 proposal §Human Approval：D1 校对端点延后、D2 人物关系单一写入入口、D3 新增势力关系备注 PATCH、D4 关系行存在即阻断。批准后进入 F2。
