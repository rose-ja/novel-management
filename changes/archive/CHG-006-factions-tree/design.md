# CHG-006 设计

- Change ID：`CHG-006-factions-tree`
- 依据：`proposal.md`、SPEC-FACTION-001～004、PG-04、PG-05、`api-contract.md` §3.4、`data-model.md` §3.4
- 决策：D1～D10（见 proposal §3）；势力大事记复用 `chronicles`、树接口扁平全量、校对端点延后 CHG-011（已获用户确认）

## 1. 总体方案

1. 新增 `factions` 真实树表与 `characters.faction_id` 外键，补齐 CHG-004 的势力占位。
2. 后端新增势力仓储/服务/HTTP：CRUD、移动环校验、首领校验、成员推导（直属/子树）、删除矩阵。
3. 复用 `chronicles` 承载势力大事记（放宽 CHECK）；`ChronicleSection` 抽为通用组件。
4. 回收站新增 `root_type=faction`，并纳入作品整组生命周期与永久删除顺序。
5. 前端新增 `/factions` 树与 `/factions?id=` 详情；人物侧接入真实势力；导航与回收站适配。
6. 迁移 0006 可逆；F10 回写 SPEC-FACTION-001～004、契约与数据模型。

## 2. 数据模型与迁移

### 2.1 `0006_factions_chronicles.sql`（Up）

```sql
CREATE TABLE factions (
    id                  TEXT PRIMARY KEY,
    work_id             TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    name_key            TEXT NOT NULL,
    parent_id           TEXT REFERENCES factions (id) ON DELETE RESTRICT,
    leader_character_id TEXT REFERENCES characters (id) ON DELETE RESTRICT,
    description         TEXT,
    review_status       TEXT NOT NULL DEFAULT 'reviewed'
                        CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at         TIMESTAMPTZ,
    reviewed_by         TEXT,
    version             INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,
    deleted_root_id     TEXT,
    deleted_by_action   TEXT,
    CONSTRAINT factions_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT factions_desc_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT factions_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX factions_active_name_uq
    ON factions (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX factions_active_parent_idx
    ON factions (work_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX factions_deleted_root_idx
    ON factions (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

-- CHG-004 的 faction_id 仅为占位；历史脏值先置空，避免建 FK 失败。
UPDATE characters SET faction_id = NULL
 WHERE faction_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM factions f WHERE f.id = characters.faction_id);

ALTER TABLE characters
    ADD CONSTRAINT characters_faction_fk
    FOREIGN KEY (faction_id) REFERENCES factions (id) ON DELETE RESTRICT;
CREATE INDEX characters_faction_idx
    ON characters (work_id, faction_id) WHERE deleted_at IS NULL;

ALTER TABLE chronicles DROP CONSTRAINT IF EXISTS chronicles_owner_type_check;
ALTER TABLE chronicles
    ADD CONSTRAINT chronicles_owner_type_check CHECK (owner_type IN ('character', 'faction'));
```

### 2.2 Down

```sql
DELETE FROM chronicles WHERE owner_type = 'faction';
ALTER TABLE chronicles DROP CONSTRAINT IF EXISTS chronicles_owner_type_check;
ALTER TABLE chronicles
    ADD CONSTRAINT chronicles_owner_type_check CHECK (owner_type IN ('character'));

ALTER TABLE characters DROP CONSTRAINT IF EXISTS characters_faction_fk;
DROP INDEX IF EXISTS characters_faction_idx;
DROP TABLE IF EXISTS factions;
```

> Down 会丢弃势力与势力大事记（数据损失），但保留人物 `faction_id` 列值；已在 proposal §8 记录。

### 2.3 约束与顺序

- 自引用 `parent_id` 与 `leader_character_id` 均 `RESTRICT`；物理删除时必须「人物/大事记 → 势力深度倒序 → 作品」。
- 软删除不受 FK 影响；恢复只校验父级是否活跃。
- 势力名称唯一只对活跃行生效，进入回收站后可新建同名。

## 3. 后端模块

### 3.1 `model/faction.go`

- 常量：`MaxFactionName = 200`、`MaxFactionDescription = 20000`、`FactionActionDelete = "faction_delete"`。
- `Faction`（GORM，表 `factions`）：`ID`、`WorkID`、`Name`、`NameKey`、`ParentID`、`LeaderCharacterID`、`Description`、`ReviewStatus`、`ReviewedAt`、`ReviewedBy`、`Version`、时间与软删三件套。
- DTO：`FactionBrief{ID,Name}`、`FactionSummary{…, Parent, Leader, ChildrenCount, MemberCount}`、`FactionDetail{FactionSummary, Relations{Children []FactionBrief, Locations []LocationBrief, ChronicleCount, SubtreeMemberCount}}`、`MemberPage{Items []CharacterSummary, Page, PageSize, Total}`。
- `model/media.go` 增加 `ChronicleOwnerFaction = "faction"`。
- `model/errors.go` 增加 `ErrCycleDetected`；`CycleDetectedError{Field:"parent_id"}`。

### 3.2 `repository/factions.go`

- 读：`Get`、`GetForUpdate`、`GetDeleted`、`ListActive(workID, q, reviewStatus)`、`ListActiveChildren(parentID)`、`CountActiveChildren`、`BatchCounts(workID)`、`BatchLeaderNames`.
- 写：`Insert`、`UpdateWithVersion`（`version+1`）、`SoftDelete`（自身 + 同根大事记）、`Restore`、`RenameDeleted`.
- 成员：`DirectMembers(workID, factionID, page, size)`、`SubtreeMembers(workID, factionID, page, size)`（递归 CTE，`UNION` 去重）、`CountSubtreeMembers`.
- 生命周期：`MarkWorkDeleted`、`RestoreByWork`、`Purge`（先删该势力大事记再删行）、`PurgeByWork`（按深度倒序）、`PurgeByDeletedWorks`.
- 排他：`HasActiveName(workID, nameKey, excludeID)`、`IsDescendant(rootID, candidateID)`（递归 CTE 或祖先链向上遍历）。
- 删除引用：`ActiveMembers(factionID)` 返回人物 `id/name` 用于 `ReferenceConflictError`；`ActiveChildren(factionID)` 含软删子节点用于 `HasChildrenError`.

### 3.3 `service/factions.go`

- `List`：一次取全量节点，批量装配父级、首领、子节点计数与直属成员计数；`q` 命中后补充全部祖先（保持树结构）。
- `Create`：作品必须活跃（`ErrCharacterWorkUnavailable` 复用或新增 `ErrWorkUnavailable`）；`parent_id` 同作品活跃；`leader_character_id` 同作品活跃；名称规范化与唯一。
- `Update`：`If-Match` 必填；名称、父级、首领、简介子集；移动时目标不能等于自身或其后代（`ErrCycleDetected`）；版本冲突返回当前版本。
- `Delete`：先查活跃下级（含软删子节点，避免半棵树）→ `HasChildrenError`；再查活跃成员 → `ReferenceConflictError`（`type=character_faction`）；事务内软删除势力并把势力大事记标记为同根；创建 8 秒撤销令牌。
- `Members`：`scope=direct|subtree`，默认 20/页；子树结果按 `name_key, id` 稳定排序。
- 校验复用 `model.ValidateNameLength` 等既有工具；错误字段使用 `name`、`parent_id`、`leader_character_id`。

### 3.4 人物接入（`service/characters.go`、`repository/characters.go`）

- `faction_id`：非 `null` 时必须为当前作品活跃势力；跨作品 → `CROSS_WORK_REFERENCE`，不存在/已删 → `PARENT_DELETED` 或 422 字段错误（统一取 `faction_id` 字段）。
- DTO：列表/详情批量组装 `FactionBrief`；列表 `faction_id` 过滤改为真实查询。
- 删除：在现有引用检查器中追加 `factions.leader_character_id = ?` 查询，命中返回 `ReferenceConflictError`（`type=faction_leader`、`field=leader_character_id`、`in_trash` 取势力软删状态）。
- 恢复人物不再需要清空 `faction_id`（软删势力行仍存在，FK 满足）。

### 3.5 势力大事记（`service/chronicles.go`）

- `requireOwner` 支持 `faction`：查活跃势力，缺失 → `NOT_FOUND`，已删 → `PARENT_DELETED`。
- `List/Create` 落库 `work_id` 取势力 `work_id`；其余校验与人物一致。
- 单条删除/撤销沿用 `RootTypeChronicle`，无需新增分支。

### 3.6 回收站与生命周期（`service/trash.go`、`repository/works.go`）

- `RootTypeFaction = "faction"` 加入 List/Restore/Purge/Undo 白名单。
- 回收站列表：名称、父级名（`parent_name`）、影响条数（势力自身 + 大事记）；`deleted_at` 取根。
- 恢复：父级不在线 → `PARENT_DELETED`；成功恢复势力与大事记；首领在回收站不阻止（详情标记不可用）。
- 永久删除：事务内 `DELETE chronicles(owner_type=faction, owner_id=?)` → 删除势力行。
- 作品组：`SoftDeleteGroup`/`RestoreGroup` 追加 `factions` 与势力大事记；`PurgeGroup`/`PurgeAllDeleted` 在人物之后按深度倒序删除势力。
- 清空回收站计数纳入势力根。

### 3.7 `httpapi/factions.go`

- `registerFactions`：`GET/POST /factions`、`GET/PATCH/DELETE /factions/:id`、`GET /factions/:id/members`。
- 请求体：创建 `{work_id,name,parent_id,leader_character_id,description}`；更新任意子集 + `If-Match`；删除无 body。
- 删除返回 `{deleted:true,undo_token}`；恢复走既有 `/trash/faction/{id}/restore`。
- `request.go` 增加 `ErrCycleDetected → 422 CYCLE_DETECTED`；其余复用既有映射。
- `router.go`/`main.go` 注入 `Factions` 服务。

## 4. 前端设计

### 4.1 API 层

- `lib/api/factions.ts`：`FactionSummary/FactionDetail/MemberPage` 类型、`listFactions`、`getFaction`、`createFaction`、`updateFaction(ifMatch)`、`deleteFaction`、`listMembers`。
- `lib/api/characters.ts`：`faction` 类型改为真实 `FactionBrief`；新增/更新请求携带 `faction_id`。
- `lib/api/chronicles.ts`：`ownerType` 参数化（默认 character）。
- `lib/api/trash.ts`：根类型增加 `faction`。

### 4.2 列表树（PG-04）

- `app/factions/page.tsx` + `FactionRoute`：读取 `?id=`；无作品显示 PG-00 无作品态。
- `FactionListPage`：页头（当前作品 + 节点总数 + 新建）、FilterBar（搜索/待校对/展开全部/收起全部/重置）、`FactionTree`、`InlineCreatePanel`。
- 组树：一次 `listFactions(work_id)` 取扁平列表，`useMemo` 建 `childrenMap`；默认展开两层；搜索本地过滤（命中 + 祖先），命中 `<mark>` 高亮并滚动到首个命中。
- 树行：名称链接 `/factions?id=`、首领名、直属成员数、下级数、待校对 Badge；展开按钮阻止冒泡。
- 内联新建：上级 TreeSelect（可为空）+ 名称 + 简介；保存后展开父级、选中新节点、Toast；有输入取消需确认。

### 4.3 详情（PG-05）

- `FactionDetailPage`：面包屑、页头（名称/待校对/编辑/更多）、摘要条（首领/直属/含下级/下级分支）、简介、下级分支、成员区、关联地点（空态）、势力大事记、侧栏档案。
- 内联编辑：名称、上级 TreeSelect（禁用自身与后代；清空=顶层）、首领 CharacterSelect（当前作品人物，可清空）、简介；保存顺序先 PATCH 再刷新；失败保留输入。
- 删除：`FactionDeleteDialog` 先调 DELETE；`HAS_CHILDREN`/`REFERENCE_CONFLICT` 用 ReferenceDialog 列出下级/成员并可跳转；成功后返回 `/factions`，Toast 撤销。
- 成员表：`SegmentedControl` 仅直属/含下级；列缩略图 40px、姓名链接、境界摘要、势力路径（含下级时）、榜单占位；分页 20。
- 势力大事记：复用通用 `ChronicleSection`（`ownerType="faction"`）。
- 并发冲突与未保存守卫复用人物实现模式（顶部 Alert + 保留输入）。

### 4.4 人物侧接入

- 详情编辑：新增「所属势力」TreeSelect（当前作品，可清空）；查看态显示链接。
- 列表：`faction` 列显示真实名称链接；`faction_id` 筛选启用。
- 人物删除：首领阻断时在 `ReferenceDialog` 展示「担任首领：势力名」，可跳转势力详情。

### 4.5 回收站与导航

- `app/trash/page.tsx`：类型筛选增加「势力」；行显示父级；恢复失败展示「请先恢复上级势力」。
- `global-nav.tsx`：资料下新增「势力」。

### 4.6 文案

- `lib/messages.ts` 增加：势力列表/详情、创建/保存/删除/撤销、`CYCLE_DETECTED`、`HAS_CHILDREN`、成员空态、大事记复用文案。

## 5. 错误与边界

| 场景 | 错误码 | HTTP | details |
|---|---|---:|---|
| 名称空/超 200 | `VALIDATION_ERROR` | 422 | `field=name` |
| 同作品重名 | `NAME_CONFLICT` | 409 | — |
| 父级跨作品/不存在 | `CROSS_WORK_REFERENCE` / `PARENT_DELETED` | 422 / 409 | `field=parent_id` |
| 移动到自身或后代 | `CYCLE_DETECTED` | 422 | `field=parent_id` |
| 首领跨作品/已删 | `CROSS_WORK_REFERENCE` / `PARENT_DELETED` | 422 / 409 | `field=leader_character_id` |
| 版本不一致 | `VERSION_CONFLICT` | 409 | `current_version` |
| 仍有下级（含软删） | `HAS_CHILDREN` | 409 | `items[]` |
| 仍有成员人物 | `REFERENCE_CONFLICT` | 409 | `items[]` |
| 人物担任首领时删除人物 | `REFERENCE_CONFLICT` | 409 | `type=faction_leader` |
| 恢复时父级已删 | `PARENT_DELETED` | 409 | — |
| 大事记归属非 character/faction | `CHRONICLE_OWNER_UNSUPPORTED` | 422 | — |

## 6. 测试设计

### 6.1 后端单元

- 名称规范化与长度、父级环检测（自身/后代/深层）、成员排序稳定性、`in_trash` 标记。
- 复用 `validate` 工具测试；不引入数据库。

### 6.2 集成（真实 PostgreSQL，事务回滚）

- 迁移：`TestMigrationRoundTrip` v6；0006 Down 删 `factions`、恢复 chronicles CHECK、删除 faction 大事记。
- 树与 CRUD：三级创建、名称冲突（大小写）、跨作品父级、默认顺序、`q` 返回命中 + 祖先。
- 移动：自身/直接子/深层后代 → `CYCLE_DETECTED`；合法移动后父子关系与计数更新。
- 首领：设置/清空、跨作品拒绝、人物改名同步、删除首领人物被阻断（`type=faction_leader`）。
- 成员：直属/子树、递归去重、分页稳定、人物换势力后原/新势力结果变化。
- 大事记：`owner_type=faction` 创建/列表/版本冲突/2,000 字/撤销；owner 404/已删。
- 生命周期：势力删除（有下级/有成员阻止；无引用成功 + `undo_token`）；撤销恢复；回收站 `faction` 列表/恢复/永久删除；父级已删恢复阻断；作品组删除/恢复/清空覆盖势力与势力大事记。

### 6.3 前端单测

- `lib/api/factions.test.ts`：错误映射、`If-Match` 头、成员查询参数。
- 树构建/过滤纯函数（命中 + 祖先）单元测试。

### 6.4 E2E（`e2e/factions.spec.ts`）

1. 三级内联新建（万族 → 妖族 → 青丘）、默认展开两级、名称重复提示。
2. 搜索定位并展开祖先、无匹配提示。
3. 详情设置首领、成员直属/含下级切换、人物换势力后两侧更新。
4. 移动父级到后代被禁用/422 提示。
5. 删除阻断（下级/成员）→ 清空后删除 → Toast 撤销 → 回收站恢复。
6. 势力大事记增删改 + 撤销。
7. 1024/1280 无横向溢出；axe 无 serious/critical；键盘可达树节点。

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

- `characters.faction_id` 从“仅 null”变为真实引用，属于契约扩展；旧客户端传 `null` 仍可用。
- `chronicles` 表结构不变，仅放宽 CHECK；人物大事记行为不变。
- 回收站新增类型，旧前端不识别 `faction` 时仅显示未知 Badge（本轮同步升级前端）。
- 快照/导出/导入格式不变；迁移前建议生成快照。

## 8. 实施顺序与并行边界

1. T1 迁移 + 模型 → T2 仓储 → T3 服务 CRUD/环/校验 → T4 HTTP → T5 成员 → T6 人物接入 → T7 人物删除首领阻断 → T8 势力大事记 → T9 生命周期/回收站。
2. T10 前端 API 与纯函数 → T11 列表树 → T12 详情 → T13 成员/大事记/人物侧 → T14 E2E/视觉/a11y/回归。
3. 并行边界：T2/T3 与 T10 可在契约冻结后并行；T11 依赖 T10；T12/T13 依赖 T11 与 T4/T5；T14 最后执行。
4. 每个任务独立提交；T4 后跑一次后端集成，T13 后跑前端门禁，T14 全量回归。

## 9. 文件级改动清单

**后端**

- 新增：`internal/migrations/0006_factions_chronicles.sql`、`internal/model/faction.go`、`internal/repository/factions.go`、`internal/service/factions.go`、`internal/httpapi/factions.go`、`internal/integration/{factions,faction_members,faction_lifecycle}_test.go`。
- 修改：`internal/model/{errors,media,character}.go`、`internal/repository/{characters,chronicles,works}.go`、`internal/service/{characters,chronicles,trash}.go`、`internal/httpapi/{characters,chronicles,request,router}.go`、`cmd/server/main.go`、`internal/integration/{suite,migrations}_test.go`。

**前端**

- 新增：`app/factions/page.tsx`、`components/factions/{faction-route,faction-list-page,faction-tree,faction-inline-create,faction-detail-page,faction-edit-form,faction-delete-dialog,member-table}.tsx`、`components/chronicles/chronicle-section.tsx`（由人物目录迁移泛化）、`lib/api/factions.ts`、`lib/api/factions.test.ts`、`e2e/factions.spec.ts`。
- 修改：`components/characters/{character-detail-page,character-form-fields,character-filters,character-table}.tsx`、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`lib/api/{characters,chronicles,trash}.ts`、`lib/messages.ts`、`styles/components.css`、`e2e/{visual,a11y,keyboard}.spec.ts` 与视觉基线。

## 10. 待人工确认（F1）

已确认：D2 复用 chronicles、D3 扁平全量、D4 校对端点延后。

待 F1 批准后进入 F2；若需调整以下项请在批准时说明：

1. 势力同级排序采用 `name_key`（不新增 `sort_order`）。
2. 恢复势力时首领人物在回收站不阻止，详情标记不可用。
3. 1000 节点先全量渲染，不做虚拟滚动。
