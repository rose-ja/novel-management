# CHG-004 人物档案核心 — 设计

- 关联：`proposal.md`（范围与验收）、`specs/characters/specs.md`、PG-02、PG-03、`api-contract.md` §3.3、`data-model.md` §3.2
- 目标：在不引入势力/地点/榜单真实数据的前提下，交付可验收的人物档案闭环，并为后续变更预留引用与展示结构。

## 1. 分层与文件清单

### 1.1 后端

| 层 | 文件 | 变更 |
|---|---|---|
| 迁移 | `backend/internal/migrations/0004_characters.sql`（新增） | `characters`、`character_names`、`character_system_refs` 补外键 |
| 模型 | `backend/internal/model/character.go`（新增） | `Character`、`CharacterName`、`CharacterSystemRef`、DTO/汇总结构、错误类型 |
| 模型 | `backend/internal/model/errors.go` | 新增 `ReferenceConflictError`（入向引用阻断）、`CrossWorkReferenceError` |
| 仓储 | `backend/internal/repository/characters.go`（新增） | CRUD、列表查询、名称占用、引用写入/读取、软删除三件套、回收站查询与永久删除 |
| 仓储 | `backend/internal/repository/system.go` | 引用清单补充人物名与 `in_trash`（`ListReferencesBy*`、`referenceCountsBy*` 的 join） |
| 仓储 | `backend/internal/repository/works.go` | `SoftDeleteGroup`/`RestoreGroup`/`purgeWorkChildren`/`PurgeAllDeleted` 覆盖 `characters`、`character_names` |
| 服务 | `backend/internal/service/characters.go`（新增） | 创建/更新/列表/详情/删除/撤销、动态体系校验、乐观锁、入向引用检查框架 |
| 服务 | `backend/internal/service/trash.go` | 新增 `RootTypeCharacter`：列表合并、恢复、永久删除、清空分派 |
| HTTP | `backend/internal/httpapi/characters.go`（新增） | 5 个 handler |
| HTTP | `backend/internal/httpapi/router.go` | 注册 `/api/v1/characters*` |
| HTTP | `backend/internal/httpapi/trash.go` | `root_type` 白名单加 `character` |
| HTTP | `backend/internal/httpapi/request.go` | 错误映射（`REFERENCE_CONFLICT`、`CROSS_WORK_REFERENCE`、`VERSION_CONFLICT` 已存在） |

### 1.2 前端

| 文件 | 变更 |
|---|---|
| `frontend/lib/api/characters.ts`（新增） | 类型 + `listCharacters/getCharacter/createCharacter/updateCharacter/deleteCharacter` |
| `frontend/app/characters/page.tsx`（新增） | 读 `searchParams.id`，无 `id` 渲染列表，有 `id` 渲染详情 |
| `frontend/components/characters/character-list-page.tsx`（新增） | 列表容器：筛选、分页、刷新、Toast |
| `frontend/components/characters/character-table.tsx`（新增） | 表格列：姓名/别名称号/境界/势力/榜单/待校对/更新时间/操作 |
| `frontend/components/characters/character-filters.tsx`（新增） | 搜索、境界（`system_node_id`）、待校对、排序、每页 |
| `frontend/components/characters/character-create-drawer.tsx`（新增） | 新建表单（复用 Detail 的共享字段组件） |
| `frontend/components/characters/character-form-fields.tsx`（新增） | 名称、别名/称号编辑器、简介、体系动态字段 |
| `frontend/components/characters/character-detail-page.tsx`（新增） | 首屏、关系区、侧栏档案、内联编辑与保存 |
| `frontend/components/characters/character-delete-dialog.tsx`（新增） | 删除确认、引用阻断列表、撤销入口 |
| `frontend/components/characters/use-unsaved-guard.ts`（新增） | 未保存离开守卫 |
| `frontend/lib/messages.ts` | 人物文案与错误码文案追加 |
| `frontend/components/shell/global-nav.tsx` | 「资料 > 人物」入口 |
| `frontend/app/trash/page.tsx` | 类型筛选与 Badge 增加「人物」 |

## 2. 数据模型（迁移 0004）

```sql
-- +goose Up
CREATE TABLE characters (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    name_key           TEXT NOT NULL,
    description        TEXT,
    faction_id         TEXT,                    -- 占位，CHG-006 建 factions 后补 FK
    image_path         TEXT,                    -- 占位，CHG-005 使用
    review_status      TEXT NOT NULL DEFAULT 'reviewed'
                       CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at        TIMESTAMPTZ,
    reviewed_by        TEXT,
    version            INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    deleted_root_id    TEXT,
    deleted_by_action  TEXT,
    CONSTRAINT characters_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT characters_desc_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT characters_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX characters_active_name_uq
    ON characters (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX characters_list_idx
    ON characters (work_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX characters_deleted_root_idx
    ON characters (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE character_names (
    id           TEXT PRIMARY KEY,
    character_id TEXT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    work_id      TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('alias', 'title')),
    value        TEXT NOT NULL,
    value_key    TEXT NOT NULL,
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT character_names_value_len CHECK (char_length(value) BETWEEN 1 AND 100)
);

CREATE UNIQUE INDEX character_names_uq
    ON character_names (character_id, kind, value_key);
CREATE INDEX character_names_search_idx ON character_names (work_id, value_key);

ALTER TABLE character_system_refs
    ADD CONSTRAINT character_system_refs_character_fk
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE character_system_refs DROP CONSTRAINT IF EXISTS character_system_refs_character_fk;
DROP TABLE IF EXISTS character_names;
DROP TABLE IF EXISTS characters;
```

设计要点：

1. **名称键**：`name_key = lower(trim(name))`（与 works 一致；不做 NFKC/空白折叠，人物名称不做全半角归并，见 §11 D6）。
2. **别名/称号**：整组替换（PATCH 提供完整 `names[]`），按 `(kind, value_key)` 去重；`value_key` 同 `lower(trim(value))`；`sort_order` 按提交顺序写入。
3. **软删除**：`delete`（人物自身）时 `deleted_root_id = characters.id`；作品组删除时 `deleted_root_id = works.id`，`deleted_by_action='work_delete'`。
4. **无 `deleted_at` 于 `character_names`**：随人物可见性联动（读取一律 join 活跃人物），永久删除时物理删除。
5. **`character_system_refs` 保留**：人物软删除不删除引用行，避免恢复需重建；引用计数/清单按「是否阻断」区分（见 §5.3）。

## 3. API 设计

前缀 `/api/v1`；写请求需 `X-Requested-With: XMLHttpRequest`；所有时间 ISO-8601 UTC。

### 3.1 列表 `GET /characters`

| 参数 | 规则 |
|---|---|
| `work_id` | 必填；缺失/为空 → 422 `VALIDATION_ERROR` |
| `q` | 可选，1–100 字符；匹配主名称或别名/称号（`LIKE '%'||key||'%'`，大小写不敏感） |
| `system_node_id` | 可选；匹配任意体系类别下选中的节点（子节点匹配其自身） |
| `faction_id` | 可选；按 `characters.faction_id` 过滤（当前恒空，返回空列表） |
| `location_id` | 可选；**未开放** → 422 `VALIDATION_ERROR`（field=`location_id`） |
| `review_status` | 可选；`pending` / `confirmed`（映射库列 `pending` / `reviewed`） |
| `sort` | `-updated_at`（默认）/ `updated_at` / `name` / `-name` |
| `page` / `page_size` | 默认 1 / 20；`page_size` 上限 100 |

响应 `CharacterSummary[]` + 分页封装：

```json
{
  "items": [{
    "id": "01H...", "name": "陆沉", "names": [{"kind": "alias", "value": "小圣"}],
    "faction": null, "system_summary": "帝境·五重天",
    "image_thumb_url": null, "review_status": "confirmed", "reviewed_at": null,
    "updated_at": "2026-11-09T03:00:00Z", "version": 3
  }],
  "page": 1, "page_size": 20, "total": 42
}
```

> 相对 `api-contract.md` 的差异：`CharacterSummary` 增加 `names[]` 与 `version`（列表页需展示别名/称号 chip 与乐观锁基准），F10 回写契约。

### 3.2 详情 `GET /characters/{id}`

返回 `Character` DTO：`id, work_id, name, names[], description, faction, system_refs[], location_links[], image, review_status, reviewed_at, reviewed_by, created_at, updated_at, version, deleted_at`。

- `faction`：`null`（CHG-006 填充）。
- `system_refs[]`：`{category_id, category_name, category_type, node: {id,name,parent:{id,name}|null}}`；选中子节点时 `node.parent` 为父层级。
- `location_links`：`[]`（CHG-007）。
- `image`：`null`（CHG-005）。
- 不存在/已删除 → 404 `NOT_FOUND`。

### 3.3 创建 `POST /characters`

```json
{ "work_id": "01H...", "name": "陆沉", "names": [{"kind": "alias", "value": "小圣"}],
  "description": "…", "faction_id": null,
  "system_refs": [{"category_id": "01H...", "node_id": "01H..."}], "location_links": [] }
```

校验与错误：

| 场景 | 状态 | 错误码 / details |
|---|---|---|
| 缺 `work_id` / 作品不存在或已删 | 422 / 422 | `VALIDATION_ERROR` / `PARENT_DELETED` |
| `name` 空/超 200 | 422 | `VALIDATION_ERROR`，`field=name` |
| 同作品主名称冲突 | 409 | `NAME_CONFLICT`，`details.items[{id,name}]` |
| `names` 含非法 `kind`、空值、超 100 | 422 | `VALIDATION_ERROR` |
| `names` 超过硬上限 100 条 | 422 | `VALIDATION_ERROR`（软上限 20 由前端提示） |
| `faction_id` 非空 | 422 | `VALIDATION_ERROR`（field=`faction_id`，势力功能未开放） |
| `location_links` 非空 | 422 | `VALIDATION_ERROR`（field=`location_links`） |
| `system_refs` 类别不属于该作品/不存在 | 422 | `CROSS_WORK_REFERENCE` / `NOT_FOUND` |
| 类别未启用 | 422 | `CATEGORY_DISABLED`，details 含类别名 |
| 节点不属于该类别 / 不属于该作品 | 422 | `NODE_PARENT_MISMATCH` / `CROSS_WORK_REFERENCE` |
| 同一 `category_id` 重复 | 422 | `VALIDATION_ERROR` |

成功 201 + `Location: /api/v1/characters/{id}`；`review_status` 初始 `confirmed`（映射库 `reviewed`）。

### 3.4 更新 `PATCH /characters/{id}`

- Header `If-Match: <version>` 必填；缺失 → 428 `PRECONDITION_REQUIRED`；不匹配 → 409 `VERSION_CONFLICT`（`details.current_version`）。
- Body 允许：`name`、`names`（完整替换）、`description`、`system_refs`（完整替换）、`faction_id`（仅 `null`）、`location_links`（仅空数组）。
- 未知字段 → 422（沿用 `abortUnknownFields` 风格）；`work_id` 不可改。
- 名称冲突 409 `NAME_CONFLICT`；体系校验同创建；成功 200 返回更新后 DTO（`version+1`）。

### 3.5 删除 `DELETE /characters/{id}`

- `If-Match` 必填（PG-03 §6.3 要求删除同样做版本冲突）。
- 入向引用检查：当前检查器返回空（势力首领/榜单席位表尚未创建，CHG-006/008 接入）；命中 → 409 `REFERENCE_CONFLICT`，`details.items[{entity_type,id,name}]`。
- 成功 200：`{"id":"…","deleted_at":"…","undo_token":"…","expires_at":"…"}`；8 秒内 `POST /api/v1/undo/{token}` 恢复。
- 已删除 → 409 `ALREADY_DELETED`。

### 3.6 回收站

- `GET /trash?root_type=character`：`TrashItem{root_type:"character", root_id, name, parent_name: works.name, affected_count:1, deleted_at}`。
- `POST /trash/character/{root_id}/restore`：body `{ "new_name": "…" }` 可选；作品在回收站 → 409 `PARENT_DELETED`；同名活跃人物 → 422 `RESTORE_CONFLICT` + `details.items`；成功 200 `{root_type, root_id, name, restored_count:1}`。
- `DELETE /trash/character/{root_id}`：先物理删除 `character_system_refs`、`character_names`，再删 `characters`；失败整体回滚。
- `POST /trash/purge` 清空：人物根参与，按「引用 → 名称 → 人物」顺序；与作品/体系根互不阻塞（体系根删除前先校验真实引用，含回收站人物）。

### 3.7 体系引用收口

`GET /system-categories/{id}/references` 与删除/停用错误 details 的 `items[]` 增加：

```json
{ "id": "char_uuid", "name": "陆沉", "node_name": "五重天", "in_trash": true }
```

- `name`：人物主名称（`characters.name`）。
- `in_trash`：`characters.deleted_at IS NOT NULL`。
- 计数与阻断规则：回收站人物**计入**阻断（避免类别恢复后出现悬挂引用），对话框标注「回收站中」。

## 4. 服务流程

### 4.1 创建

1. `works.GetActive(work_id)` → 不存在 `PARENT_DELETED`。
2. 规范化 `name`/`names`；校验长度、kind、条数。
3. 事务内：`characters.NameTaken(work_id, name_key)` → 冲突；`INSERT characters`；`INSERT character_names`（按顺序）；`ReplaceSystemRefs`。
4. `ReplaceSystemRefs`：逐条校验类别属于该作品、启用、节点属于该类别且未删；`DELETE ... WHERE character_id=?` 后批量插入；唯一键 `(character_id, category_id)`。

### 4.2 更新

1. 事务内 `SELECT ... FOR UPDATE` 读取人物；`version != If-Match` → `VERSION_CONFLICT`。
2. 若改名为新 key：`NameTaken(work_id, key, excludeID)` → `NAME_CONFLICT`。
3. 替换 `names`（先删后插）、`system_refs`（先删后插）。
4. `UPDATE ... SET version=version+1, updated_at=now()`。

### 4.3 删除 / 撤销

1. `If-Match` 校验同更新；`assertNoInboundReferences`（空实现框架）。
2. `UPDATE characters SET deleted_at=now(), deleted_root_id=id, deleted_by_action='delete', version=version+1`。
3. 生成 8 秒一次性撤销令牌（复用 CHG-002 的 `model.UndoToken`/`service.UndoStore`），返回 `undo_token`。
4. 撤销：令牌有效 → 事务内确认人物仍为同一 `deleted_root_id` → 清三件套、`version+1`；体系引用与别名无需重建（未删除）。

### 4.4 列表查询

```sql
SELECT c.*, ... FROM characters c
WHERE c.work_id = ? AND c.deleted_at IS NULL
  AND (? = '' OR c.name_key LIKE '%' || ? || '%'
       OR EXISTS (SELECT 1 FROM character_names n
                  WHERE n.character_id = c.id AND n.value_key LIKE '%' || ? || '%'))
  AND (? = '' OR EXISTS (SELECT 1 FROM character_system_refs r
                         WHERE r.character_id = c.id AND r.node_id = ?))
  AND (? = '' OR c.review_status = ?)
ORDER BY c.updated_at DESC, c.id DESC
LIMIT ? OFFSET ?;
```

- `system_summary`：取内置「境界」类别下选中节点的路径（`父·子`），其余类别不参与摘要。
- 总数：同条件 `COUNT(*)`。
- 索引：`characters_list_idx (work_id, updated_at DESC) WHERE deleted_at IS NULL`；`character_names_search_idx (work_id, value_key)`。

## 5. 关键语义决策

### 5.1 动态体系字段

- 前端按 `GET /system-categories?work_id=` 的启用类别渲染；每类别一个「类别名 + 节点树选择（含“不设置”）」。
- `system_refs` 只提交有值项；未提交的类别即清空。
- 类别停用：列表页仍展示已存引用（只读），编辑时该类别灰显并提示「类别已停用，请先启用或清除该字段」；后端拒绝写入（`CATEGORY_DISABLED`）。

### 5.2 乐观锁

- 前端持有 `version`；保存/删除请求带 `If-Match`。
- 409 `VERSION_CONFLICT` → `ConflictDialog`（「数据已被其他页面修改，请刷新后重试」），提供「刷新」与「关闭」；不自动合并、不覆盖。

### 5.3 引用语义

| 方向 | 处理 |
|---|---|
| 出向（体系引用、别名、未来地点/大事记） | 人物软删除后自然不可见；数据保留，恢复即恢复；永久删除时物理清理 |
| 入向（势力首领、榜单席位） | 删除前检查；当前无数据（框架空实现），命中 `REFERENCE_CONFLICT` |
| 体系侧引用计数 | 计入回收站人物，清单标注 `in_trash` |

### 5.4 作品组联动

- 作品软删除：`characters SET deleted_at, deleted_root_id=work_id, deleted_by_action='work_delete'`（仅未删行）。
- 作品恢复：`RestoreGroup` 增加 `characters` 条件 `deleted_root_id=work_id`。
- 作品永久删除：`purgeWorkChildren` 顺序为 `character_system_refs → character_names → characters → system_nodes → system_categories`。
- 回收站「清空」中人物根与作品根并存：先按根逐个永久删除（人物根可先于作品根），最终清空。

## 6. 前端设计

### 6.1 列表页 `/characters`

- 无当前作品：空态「选择或新建作品后管理人物」+ 前往作品入口。
- 筛选栏：搜索框（300ms 防抖）、境界下拉（按内置境界类别节点平铺）、待校对开关、排序、每页。
- 表格列：姓名（链接到详情）｜别名/称号（最多 3 个 chip，超出 `+N`）｜境界｜势力（`—`）｜榜单（`—`）｜待校对 Badge｜更新时间｜操作（编辑/删除）。
- 状态：骨架行（5 行）、空态（「还没有人物」+ 新建按钮）、错误态（重试）、Toast 成功/失败。
- 新建 Drawer：名称（必填）、别名/称号（可增删，>20 提示）、简介（≤20,000 计数）、体系动态字段；提交期间禁用按钮（EX-04）。

### 6.2 详情页 `/characters?id=<id>`

- 首屏：姓名、别名/称号、境界/势力/榜单/地点摘要（无值显示 `—`）、待校对 Badge、主图占位（灰底 + 「暂无主图」）。
- 关系区：体系引用（按类别分组）、势力（空态）、地点（空态）、榜单（空态）。
- 侧栏档案：简介、创建/更新时间。
- 内联编辑：姓名、别名/称号、简介、体系字段；底部「保存」「取消」；保存成功 Toast + 刷新 DTO/version。
- 未保存离开：`beforeunload` + 站内链接拦截，确认「放弃未保存的修改？」。
- 删除：确认弹窗 → 成功 Toast + 「撤销」按钮（8 秒倒计时）→ 超时 Toast「已移入回收站」；409 `REFERENCE_CONFLICT` 列出引用项。

### 6.3 回收站

- 类型筛选增加「人物」；Badge「人物」；名称不加书名号；父级列显示作品名；恢复/永久删除文案按类型变化。

## 7. UI 状态矩阵

| 区域 | 加载 | 空 | 错误 | 成功 | 禁用 |
|---|---|---|---|---|---|
| 列表 | 骨架行 | 「还没有人物」 | 提示 + 重试 | 表格 | — |
| 详情 | 骨架块 | 404 → 「人物不存在或已删除」 | 提示 + 重试 | 只读展示 | — |
| 表单 | — | — | 字段级错误 / 表单级 Toast | Toast「已保存」 | 保存中禁用；无改动禁用保存 |
| 删除 | — | — | 409 引用清单 | Toast + 撤销 | 删除中禁用 |
| 回收站 | 骨架行 | 「回收站是空的」 | 提示 + 重试 | 类型筛选结果 | — |

## 8. 测试策略

- **单元（Go）**：名称规范化与等价冲突、`names` 去重与排序、`system_refs` 校验矩阵、带逗号/中文的 `q` 转义、`sort` 解析、分页夹取、`review_status` 映射。
- **集成（Go，scratch DB + 事务回滚）**：
  1. 创建/详情/列表（分页、排序、`q` 命中别名、境界筛选）。
  2. 同名冲突（`Sword` vs `sword`）、跨作品隔离、跨作品体系引用 `CROSS_WORK_REFERENCE`。
  3. 停用类别写入 `CATEGORY_DISABLED`；节点/类别不匹配 `NODE_PARENT_MISMATCH`。
  4. PATCH 正常、`If-Match` 缺失/不匹配、名称冲突、`names` 全量替换、`system_refs` 全量替换。
  5. DELETE → 撤销 → 再删；`ALREADY_DELETED`；撤销令牌过期。
  6. 回收站列表（类型/父级/条数）、恢复（作品在回收站 `PARENT_DELETED`、同名 `RESTORE_CONFLICT`、`new_name` 成功）、永久删除（引用先删）。
  7. 作品删除 → 人物隐藏；作品恢复 → 人物回来；作品永久删除 → 人物与别名物理删除。
  8. 体系引用计数：创建人物引用类别 → 停用/删除类别 409 `REFERENCE_CONFLICT` 且 `in_trash` 正确；人物删除后仍阻断并标注 `in_trash=true`。
  9. 1000 人种子数据（事务内批量插入）→ 列表查询 P95 ≤ 500ms（采样 20 次）。
- **前端单测（vitest）**：`lib/api/characters.ts` 参数序列化/错误解析、`messages` 文案键、表单校验函数。
- **E2E（Playwright）**：`characters.spec.ts` — 新建→列表→详情→内联编辑保存→名称冲突→未保存守卫→删除撤销→回收站恢复→1024 视口；`trash-system.spec.ts` 增补人物根用例；a11y 覆盖 `/characters`。
- **视觉基线**：`characters-1280`、`characters-1024`。

## 9. 安全与性能

- 所有写请求沿用 `X-Requested-With` 守卫；无新增外部依赖。
- 查询参数白名单校验，`q` 走参数化 SQL；`sort` 映射为固定列，不做字符串拼接。
- 列表 SQL 分页，无 N+1：节点路径与类别名一次 join（按 `system_refs` 批量查询）。
- 1000 人：组合索引覆盖 `(work_id, updated_at)`；`q` 走 `name_key`/`value_key` LIKE（无 trigram，数据量小）。
- 迁移 0004 在空表上执行，锁表时间可忽略。

## 10. 兼容与迁移

- `character_system_refs.character_id` 在 0003 中无外键，0004 追加；因该表在 CHG-003 仅由测试种子写入，生产/本地库为空或可清空，补外键前不需数据修复。
- 若本地已有测试残留（`character_system_refs` 行指向不存在人物），0004 Up 会失败：迁移前置 `DELETE FROM character_system_refs WHERE character_id NOT IN (SELECT id FROM characters)` 不可行（表刚建）。改为在 Up 开始处 `DELETE FROM character_system_refs;`（骨架表无生产数据，F1 决策 D5）。

## 11. 决策记录

| ID | 决策 | 理由 |
|---|---|---|
| D1 | 不建 `factions`/`locations` 表；`faction_id` 无 FK 占位，`location_links` 仅空数组 | 避免半成品表与双向外键；CHG-006/007 建表后补 FK |
| D2 | 校对端点延后 CHG-011，DTO 只读返回 review 字段 | roadmap 将导入/校对归 CHG-011；避免提前实现无数据入口 |
| D3 | 回收站新增 `character` 根 | roadmap DoD 要求人物回收站恢复可用 |
| D4 | 列表隐藏势力/地点筛选与榜单排序，显示 `—` 占位 | 对应实体尚不存在，避免误导 |
| D5 | 0004 Up 先清空 `character_system_refs` 再补外键 | 该表当前仅为骨架+测试种子，无用户数据 |
| D6 | 人物名称键沿用 `lower(trim)`（不做 NFKC/空白折叠） | 与 works 一致；体系类别的完整规则不适用于人物名称 |
| D7 | 人物软删除保留 `character_system_refs`，引用计数含回收站人物并标注 `in_trash` | 撤销/恢复零重建；避免类别恢复后悬挂引用 |
| D8 | `CharacterSummary` 增加 `names[]` 与 `version` | 列表需展示别名/称号、编辑需乐观锁基准 |
| D9 | 删除 `/characters/{id}` 要求 `If-Match` | PG-03 §6.3 要求删除同样做版本冲突；契约 F10 回写 |
