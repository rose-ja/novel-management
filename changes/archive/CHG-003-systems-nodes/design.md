# Design — CHG-003 体系类别与节点

## 1. 现状

- `system_categories` 由 `0002_works_trash_snapshot.sql` 建立：`category_type` 默认 `'realm'`、无 CHECK、无 `sort_order`；无 `system_nodes`、`character_system_refs`。
- `service/works.go` 创建作品时写入 `category_type='normal'` 的内置境界；`repository/works.go` 的作品软删除/恢复只覆盖 `system_categories`，永久删除只删 `works`。
- `service/trash.go` 只支持 `root_type=work`；`Undo`、`PurgeAll` 同样只处理作品。
- 前端无 `/system` 路由、无体系 API 封装、无 Tree 组件；设计系统组件在 `frontend/components/ui/`。
- 无人物表，引用只能通过本轮新建的 `character_system_refs` 骨架验证。

## 2. 方案总览

沿用 CHG-002 的 handler → service → repository 三层；体系域独立成包内文件，不新建子包。

```
httpapi/system.go
   ├→ service/system.go ──→ repository/system.go ──→ system_categories / system_nodes / character_system_refs
   └→ service/trash.go（扩展 system_category / system_node 分支）
                              └→ repository/works.go（作品级联节点收口）
model/system.go：SystemNode、NormalizeKey、review 映射、树与排序纯函数
```

## 3. 迁移 `0003_system_nodes.sql`

### 3.1 `system_categories` 修正

```sql
ALTER TABLE system_categories ALTER COLUMN category_type SET DEFAULT 'normal';
UPDATE system_categories SET category_type = 'normal'
 WHERE category_type NOT IN ('normal', 'era');
ALTER TABLE system_categories
  ADD CONSTRAINT system_categories_type_check
  CHECK (category_type IN ('normal', 'era'));
ALTER TABLE system_categories
  ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE system_categories
  ADD CONSTRAINT system_categories_deleted_consistent
  CHECK ((deleted_at IS NULL) = (deleted_root_id IS NULL));
```

- `sort_order` 仅记录创建顺序；页面不提供类别拖拽（D-04）。
- 0002 的 `system_categories_work_name_uq` 保持不动。

### 3.2 `system_nodes`

```sql
CREATE TABLE system_nodes (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    category_id        TEXT NOT NULL REFERENCES system_categories (id) ON DELETE RESTRICT,
    parent_id          TEXT REFERENCES system_nodes (id) ON DELETE RESTRICT,
    name               TEXT NOT NULL,
    name_key           TEXT NOT NULL,
    sort_order         INTEGER NOT NULL DEFAULT 0,
    era_start_text     TEXT,
    era_end_text       TEXT,
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
    CONSTRAINT system_nodes_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT system_nodes_era_len CHECK (
        (era_start_text IS NULL OR char_length(era_start_text) <= 100)
        AND (era_end_text IS NULL OR char_length(era_end_text) <= 100)
    ),
    CONSTRAINT system_nodes_deleted_consistent
        CHECK ((deleted_at IS NULL) = (deleted_root_id IS NULL))
);

CREATE UNIQUE INDEX system_nodes_sibling_name_uq
    ON system_nodes (category_id, COALESCE(parent_id, ''), name_key)
    WHERE deleted_at IS NULL;
CREATE INDEX system_nodes_tree_idx
    ON system_nodes (category_id, parent_id, sort_order)
    WHERE deleted_at IS NULL;
CREATE INDEX system_nodes_deleted_root_idx
    ON system_nodes (deleted_root_id)
    WHERE deleted_root_id IS NOT NULL;
```

- 两级上限、父级同类别、era 字段仅 `era` 类别由 service 校验（跨表规则无法用简单 CHECK 表达）。
- `review_status` 沿用 DB 值；API 输出 `confirmed` 时映射 DB `reviewed`（P5b RT-04）。

### 3.3 `character_system_refs` 骨架

```sql
CREATE TABLE character_system_refs (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    character_id       TEXT NOT NULL,
    category_id        TEXT NOT NULL REFERENCES system_categories (id) ON DELETE RESTRICT,
    node_id            TEXT NOT NULL REFERENCES system_nodes (id) ON DELETE RESTRICT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX char_sys_uq ON character_system_refs (character_id, category_id);
CREATE INDEX char_sys_node_idx ON character_system_refs (node_id);
CREATE INDEX char_sys_category_idx ON character_system_refs (category_id);
```

- D-01：`character_id` 暂不加 FK（人物表在 CHG-004）；CHG-004 建表后追加 `REFERENCES characters(id) ON DELETE CASCADE`，并在追加前清理/校验孤儿。
- 关系表不设软删除列；人物删除后引用随人物不可见（CHG-004 接入）。

### 3.4 Down

```sql
DROP TABLE IF EXISTS character_system_refs;
DROP TABLE IF EXISTS system_nodes;
ALTER TABLE system_categories DROP CONSTRAINT IF EXISTS system_categories_deleted_consistent;
ALTER TABLE system_categories DROP COLUMN IF EXISTS sort_order;
ALTER TABLE system_categories DROP CONSTRAINT IF EXISTS system_categories_type_check;
ALTER TABLE system_categories ALTER COLUMN category_type SET DEFAULT 'realm';
```

- Down 回到 0002 的可执行结构；已有 `normal`/`era` 数据保留（无 CHECK 不阻塞）。

## 4. 领域模型

### 4.1 名称规范化

`model.NormalizeKey(raw string) (display, key string, err error)`：

```
display = strings.TrimSpace(raw)
key     = strings.ToLower(strings.Join(strings.Fields(unicode.NFKC(display)), " "))
```

- 空值/超 200 rune → `ValidationError{Field:"name"}`。
- 仅体系类别/节点使用；`works` 保持 CHG-002 规则（D-02）。
- 响应返回 `display`，唯一索引使用 `key`。

### 4.2 SystemCategory

在 `model/work.go` 现有 `SystemCategory` 增加：

- `SortOrder int`（DB 列）
- `NodeCount int64`（`gorm:"-"`，列表/详情计算）
- `ReferenceCount int64`（`gorm:"-"`）

JSON：`id`、`work_id`、`name`、`category_type`、`is_builtin`、`is_enabled`、`sort_order`、`node_count`、`reference_count`、`version`、`created_at`、`updated_at`、`deleted_at`。不输出 `name_key`、`deleted_root_id`、`deleted_by_action`。

### 4.3 SystemNode

`model/system.go`：

```go
type SystemNode struct {
    ID, WorkID, CategoryID string
    ParentID *string
    Name, NameKey string
    SortOrder int
    EraStartText, EraEndText *string
    ReviewStatus string   // DB reviewed/pending
    ReviewedAt *time.Time
    ReviewedBy *string
    Version int
    CreatedAt, UpdatedAt time.Time
    DeletedAt *time.Time
    DeletedRootID, DeletedByAction *string
    ReferenceCount int64        `gorm:"-"`
    Children []SystemNode       `gorm:"-"`
}
```

- JSON 的 `review_status` 输出 `confirmed`（DB `reviewed`）或 `pending`；`reviewed_at`、`reviewed_by` 原样输出。
- `children` 仅在树形响应出现；`flat=true` 时省略/为空数组。

### 4.4 纯函数（可单测）

- `AssembleNodeTree(nodes []SystemNode) []SystemNode`：按 `parent_id` 挂载，根与子层各按 `(sort_order, id)` 排序；超过两级视为数据异常（返回错误）。
- `ValidateSiblingOrder(siblings []string, ordered []string) error`：集合必须不重复且完全一致，否则 `ValidationError`。

## 5. API 映射

### 5.1 类别

| 端点 | 行为 | 错误 |
|---|---|---|
| `GET /api/v1/system-categories?work_id=&include_disabled=` | 未删除类别；内置置顶，其余 `created_at ASC, id ASC`；带 `node_count`、`reference_count` | `VALIDATION_ERROR`（缺 `work_id`） |
| `POST /api/v1/system-categories` | `work_id`、`name`、`category_type=normal|era`、`is_enabled` 默认 true；`is_builtin=false`；`sort_order=max+1`；事务内计数 ≤10 | `CATEGORY_LIMIT`、`NAME_CONFLICT`、`VALIDATION_ERROR` |
| `PATCH /api/v1/system-categories/{id}` | `name?`、`is_enabled?`；`If-Match` 必填；内置：出现 `name` 或 `is_enabled=false` → immutable，`is_enabled=true` 视为无操作；停用前引用检查；类型不可改 | `CATEGORY_BUILTIN_IMMUTABLE`、`REFERENCE_CONFLICT`、`NAME_CONFLICT`、`VERSION_CONFLICT` |
| `DELETE /api/v1/system-categories/{id}` | `If-Match` 必填；内置拒绝；引用检查；整组软删除（类别+全部未删节点，`deleted_root_id=id`），写 `undo_tokens`；返回 `deleted/undo_token/affected_nodes` | `CATEGORY_BUILTIN_IMMUTABLE`、`REFERENCE_CONFLICT`、`VERSION_CONFLICT` |

补充规则：

- 创建类别 `sort_order` 取当前最大 `sort_order+1`；不提供类别排序接口。
- 停用只改 `is_enabled`，不动节点与引用。
- `PATCH` 出现 `category_type` → `422 VALIDATION_ERROR`（字段只读）。
- 类别列表的 `reference_count` = 该类别下任一节点被多少个不同人物引用（`DISTINCT character_id`）。
- 详情复用列表项，不新增端点。

### 5.2 节点

| 端点 | 行为 | 错误 |
|---|---|---|
| `GET /api/v1/system-categories/{id}/nodes?flat=&review_status=` | 默认树形 + `children`；`flat=true` 平铺；`review_status=pending` 过滤；按 `(sort_order,id)`；节点带 `reference_count` 与 review 字段 | `NOT_FOUND`（类别不存在/已删） |
| `POST /api/v1/system-nodes` | `category_id`、`parent_id?`、`name`、`era_start_text?`、`era_end_text?`；类别必须启用；父级同类别且为根；era 字段仅 `era` 类别；同级唯一；`sort_order` 追加末尾 | `CATEGORY_DISABLED`、`NODE_LEVEL_LIMIT`、`NODE_PARENT_MISMATCH`、`NAME_CONFLICT`、`VALIDATION_ERROR` |
| `PATCH /api/v1/system-nodes/{id}` | `name?`、`sort_order?`、`era_start_text?`、`era_end_text?`；`If-Match`；`parent_id` 不可改；era 字段仅 `era` 类别；改名同级唯一 | `NAME_CONFLICT`、`VERSION_CONFLICT`、`VALIDATION_ERROR` |
| `DELETE /api/v1/system-nodes/{id}` | `If-Match`；有未删子节点 → `HAS_CHILDREN`；有引用 → `REFERENCE_CONFLICT`；软删除节点自身，写 `undo_tokens` | `HAS_CHILDREN`、`REFERENCE_CONFLICT`、`VERSION_CONFLICT` |
| `POST /api/v1/system-nodes/reorder` | `category_id`、`parent_id?`、`ordered_ids[]`；集合必须完整、无重复、同父级；事务内逐项写 `sort_order`；`204` | `VALIDATION_ERROR`、`NOT_FOUND` |

补充规则：

- `parent_id` 非空时，父节点必须 `parent_id IS NULL` 且同 `category_id`、同 `work_id`。
- 节点名称唯一索引为 `(category_id, COALESCE(parent_id,''), name_key) WHERE deleted_at IS NULL`。
- `era_start_text`/`era_end_text` 只保存文本，不解析、不校验日期；普通类别提交非空值 → `422 VALIDATION_ERROR`（字段 `era_start_text`/`era_end_text`）。
- 停用类别下节点保持可读；新增/改名/删除/排序由服务端与 UI 一并禁止（服务端以「类别必须启用」拒绝写操作）。
- 节点 `review_status` 转换：DB `reviewed` → API `confirmed`，DB `pending` → API `pending`；筛选参数使用 API 值，仓储查询转换回 DB 值。

### 5.3 `If-Match`

复用 `service.ParseIfMatch`；缺失、非数字、与实体 `version` 不等 → `409 VERSION_CONFLICT`。删除类别/节点同样要求 `If-Match`。

## 6. 引用检查

- 仓储 `ReferenceCounts`：
  - 类别：`SELECT category_id, COUNT(DISTINCT character_id) FROM character_system_refs WHERE work_id=? AND category_id IN (...) GROUP BY category_id`
  - 节点：`SELECT node_id, COUNT(DISTINCT character_id) ... GROUP BY node_id`
- 引用清单：按节点列出 `character_id`、`node_id`、节点名；按 `node_id, character_id` 排序。
- 阻断触发点：停用类别、删除类别、删除节点、永久删除类别/节点（D-09/D-10）。
- 错误响应：

```json
{
  "error": {
    "code": "REFERENCE_CONFLICT",
    "message": "该类别仍有 3 人引用，无法停用",
    "details": [
      { "type": "character_system_ref", "entity_id": "01H...", "name": null,
        "field": "node", "value": "金丹", "in_trash": false }
    ]
  }
}
```

- 0003 无人物表：`name` 暂为 `null`，`in_trash` 暂为 `false`；CHG-004 补 join 后返回真实人物名与回收站标记。`details` 的条目数即引用数量，前端 ReferenceDialog 按 `value`（节点名）分组展示。
- 引用计数展示与阻断都统计 `character_system_refs` 全部行（0003 无人物软删除可判定）。CHG-004 接入后：展示排除已删人物，阻断仍包含回收站人物（C-07）。

## 7. 回收站扩展

### 7.1 列表

- `GET /api/v1/trash?root_type=work|system_category|system_node`；不传或空 = 全部类型。
- 数据来源：
  - work：`works WHERE deleted_at IS NOT NULL AND deleted_root_id=id`
  - system_category：`system_categories WHERE deleted_at IS NOT NULL AND deleted_root_id=id`
  - system_node：`system_nodes WHERE deleted_at IS NOT NULL AND deleted_root_id=id`
- 排序：`deleted_at DESC, id DESC`；分页在服务层合并后切片（沿用 CHG-002 内存分页）。
- 展示字段：
  - `parent_name`：类别 → 作品名；节点 → 类别名；作品 → `null`。
  - `affected_count`：作品 → 其下类别数 + 节点数；类别 → 其下节点数；节点 → 0。

### 7.2 恢复

- `POST /api/v1/trash/{root_type}/{id}/restore`：
  - `system_category`：作品必须在回收站外（否则 `PARENT_DELETED`）；类别名称冲突可用 `new_name`；节点名称冲突用 `renames[]`（`entity_type=system_node`、`entity_id`、`new_name`）；全部预检通过后单事务改名并恢复类别组；任一冲突未解决 → `422 RESTORE_CONFLICT`，不部分恢复。
  - `system_node`：类别与父级必须存在且未删（否则 `PARENT_DELETED`）；节点同级名称冲突用 `new_name` 或 `renames[]`（本节点自身）；恢复节点自身。
- 恢复后 `version=version+1`、`updated_at=now`；不重建检索（CHG-009）。

### 7.3 永久删除与清空

- `DELETE /api/v1/trash/{root_type}/{id}`：先落 `trash_purge` 快照；引用检查（D-10）；物理删除顺序：
  - `system_node`：`character_system_refs WHERE node_id` → `system_nodes`
  - `system_category`：引用 → 节点 → 类别
  - `work`：引用（`work_id`）→ 节点 → 类别 → 作品
- `DELETE /api/v1/trash`：先聚合全部回收站根；若任一类别/节点仍有引用 → `409 REFERENCE_CONFLICT`，不落快照、不清空；否则落 `purge_all` 快照后按上述顺序物理删除全部根。

### 7.4 撤销

- `Undo` 扩展 `root_type=system_category|system_node`：
  - 类别：整组恢复；名称冲突不自动改名，返回 `ALREADY_DELETED` 之外的冲突错误（8 秒窗口内正常入口不会产生冲突）。
  - 节点：父级/类别必须在回收站外，否则 `PARENT_DELETED`。
- 撤销成功补回当前作品不受影响（仅作品删除会清空 `current_work_id`）。

## 8. 作品级联收口

`repository/works.go`：

- `SoftDeleteGroup`：额外 `UPDATE system_nodes SET deleted_at/root/action WHERE work_id=? AND deleted_at IS NULL`。
- `RestoreGroup`：额外恢复 `system_nodes WHERE work_id=? AND deleted_root_id=?`。
- `CountDeletedByRoot`：作品根影响条数改为「类别数 + 节点数」。
- `PurgeGroup` / `PurgeAllDeleted`：改为事务内按 `character_system_refs(work_id) → system_nodes(work_id) → system_categories(work_id) → works` 物理删除；返回作品删除数量。

## 9. 事务与并发

- 类别创建：事务内 `COUNT` 未删类别 + 插入；`NAME_CONFLICT` 由部分唯一索引兜底。并发上限竞态记录为已知限制（单用户本地）。
- 类别删除：事务内引用检查 → 软删除节点 → 软删除类别 → 写 undo token。
- 节点创建/改名/删除/排序：单事务；版本通过 `WHERE version=?` 的受影响行数判定。
- 整组恢复：先全量冲突预检，再改名，再恢复；任一失败整体回滚。
- 永久删除：快照在事务外先行；物理删除单事务完成。
- 所有写请求要求 `X-Requested-With`；更新/删除要求 `If-Match`。

## 10. 前端

### 10.1 文件

| 文件 | 内容 |
|---|---|
| `frontend/app/system/page.tsx` | 路由入口；读取当前作品，组装页面 |
| `frontend/components/system/system-page.tsx` | 客户端编排：数据加载、选中类别、错误/加载、Toast |
| `frontend/components/system/category-list.tsx` | 左栏 280px / 240px；内置置顶；类型/状态/节点数/引用数；新增按钮与 10/10 禁用 |
| `frontend/components/system/category-detail.tsx` | 右栏头部、类型/状态 Badge、静态说明、待校对筛选、节点树、新增主层级 |
| `frontend/components/system/node-tree.tsx` | `role=tree/treeitem`、两层缩进、展开、引用数、上移/下移、行菜单 |
| `frontend/components/system/system-dialogs.tsx` | 新增/改名类别，新增/改名节点，停用/删除确认 |
| `frontend/components/system/reference-dialog.tsx` | 引用阻断清单（按节点名分组） |
| `frontend/lib/api/system.ts` | 类型与 API 封装；写请求带 `If-Match` |
| `frontend/lib/messages.ts` | 体系与新增错误码文案 |
| `frontend/components/shell/global-nav.tsx` | 「资料」组新增「体系」→ `/system` |

### 10.2 页面行为

- 无当前作品：复用壳层「未选择作品」引导，不渲染体系操作。
- 类别列表：
  - 内置「境界」置顶，Badge「内置 · 必开」；不显示改名/停用/删除。
  - 其他项显示「普通/纪元」「启用/已停用」「N 节点」「被引用 N 人」。
  - 达到 10 类：新增按钮禁用 + Tooltip「最多 10 类（含内置境界）」。
- 类别详情：
  - 停用类别：显示「该类别已停用」+「启用」按钮；节点树只读，隐藏新增/改名/删除/排序。
  - 普通类别：显示改名、停用、删除入口（更多菜单）。
  - 删除确认文案：「删除类别「X」及其下 N 个节点？整组进入回收站，可整组恢复。」
  - 停用有引用：先请求停用 → 409 → ReferenceDialog。
- 节点树：
  - 新增主层级（描边主按钮）；子层级仅根节点行菜单「新增子层级」；子节点行菜单不提供该入口。
  - 上移/下移：即时调用 reorder；首/末位置按钮禁用并带说明。
  - 纪元类别：节点表单显示可选「起始年份」「结束年份」文本框，提示「仅排序，不校验日期」。
  - 行菜单：改名、删除、新增子层级（仅根）、确认已校对（本变更无 pending 数据，Badge 与筛选保留）。
  - 待校对筛选：Checkbox「仅看待校对」→ `review_status=pending`；无结果显示「没有待校对节点」。
- 空状态：类别无节点 → 「还没有节点」+「新增主层级」；不显示导入按钮（D-08）。
- Toast：已创建类别/已保存/已停用/已启用/已移入回收站/已调整顺序。
- 版本冲突：Dialog 字段级提示「内容已被修改，请刷新后重试」，不展示内部字段。

### 10.3 UI 状态

| 状态 | 表现 |
|---|---|
| Loading | 左栏类别 Skeleton + 右栏节点 Skeleton |
| Empty（类别无节点） | 空状态 + 新增主层级 |
| Empty（无自定义类别） | 仅内置境界 |
| Success | 上述 Toast |
| Error（加载） | 页面错误 + 重试 |
| Error（保存） | Dialog 字段级错误，保留输入 |
| Disabled | 10/10；停用类别；内置保护；有引用操作被阻止 |
| Pending | 节点待校对 Badge + 筛选 |
| Network | 顶部 Alert + 重试，保留输入 |

### 10.4 响应式与可访问性

- ≥1280：左栏 280px；1024–1279：240px；<1024 保持最小 1024 横向滚动，不上下堆叠。
- 类别列表 `listbox` + `aria-selected`；节点树 `role=tree/treeitem` + `aria-expanded`；上移/下移 `aria-label`；Dialog 焦点陷阱；Toast `aria-live`；错误 `role=alert`。

## 11. 错误码与文案

新增映射（`frontend/lib/messages.ts`）：

| code | 文案 |
|---|---|
| `CATEGORY_LIMIT` | 最多 10 类（含内置境界） |
| `CATEGORY_BUILTIN_IMMUTABLE` | 内置「境界」不可改名、停用或删除 |
| `CATEGORY_DISABLED` | 该类别已停用，无法新增或修改节点 |
| `NODE_LEVEL_LIMIT` | 体系节点最多两级 |
| `NODE_PARENT_MISMATCH` | 上级节点与类别不匹配 |
| `HAS_CHILDREN` | 仍有下级节点，无法删除 |
| `PARENT_DELETED` | 上级/类别仍在回收站，请先恢复 |
| `REFERENCE_CONFLICT` | 仍被引用，无法执行该操作 |

## 12. 测试策略

| 层级 | 文件 | 覆盖 |
|---|---|---|
| 单元 | `backend/internal/model/system_test.go` | `NormalizeKey`（NFKC/大小写/空白）、树组装、排序集合校验、review 映射 |
| 集成 | `backend/internal/integration/system_test.go`、`trash_system_test.go`、`migrations_test.go` | 迁移往返；类别 CRUD/上限/唯一/内置/版本；停用引用；节点两级/唯一/纪元/排序；整组删除/恢复/`renames[]`/原子；节点撤销/恢复/父级缺失；回收站列表/永久删除/清空；作品级联 |
| 前端单测 | `frontend/lib/api/system.test.ts` | URL/请求体、`If-Match`、错误码透传 |
| E2E | `frontend/e2e/system.spec.ts`、更新 `shell.spec.ts` / `a11y.spec.ts` / `keyboard.spec.ts` / `visual.spec.ts` | 内置保护、类别 CRUD、10 上限、两级、重名、排序持久、删除阻断、整组回收站/恢复、停用只读、1024、键盘、axe、视觉基线 |
| 门禁 | `scripts/ci.ps1` | 单元 + 集成 + E2E 全量 |

E2E 数据策略：类别/节点名带时间戳前缀；用例结束通过 `DELETE /api/v1/trash/system_category/{id}` 与 `DELETE /api/v1/trash/system_node/{id}` 精确清理，不使用清空回收站，避免影响其他用例。

## 13. 文档同步

- F3/verification 记录真实命令与结果。
- F10 将 `project/specs/systems/specs.md` 五条状态从 `planned` 更新为 `implemented`（验收通过后 `verified`），并在变更历史注明 CHG-003。
- `project/page-specs/PG-10-system-config.md` 记录实施差异：导入按钮与校对确认延后 CHG-011；人物表单联动验收延后 CHG-004。
- `project/PROJECT-SETUP.md` §11 记录迁移 0003 与新增测试命令。
- 不修改 `data-model.md`、`api-contract.md` 的契约内容（本轮实现与契约一致）；若发现新差异，按 00-operating-manual §5 暂停并上报。

## 14. 回滚

- `0003` Down + `git revert` 本变更提交。
- 数据从删除前作品快照恢复；体系数据随快照回退。
- 前端隐藏 `/system` 导航入口即可降级，不影响作品/回收站/设置。

## 15. 延后项（不属于本轮验收）

- 人物动态表单、人物引用写入、人物侧改名同步（CHG-004）。
- 检索文档与检索结果同步（CHG-009）。
- 导入类别协议、待校对确认 `/reviews`（CHG-011）。
- 类别自定义排序、类别复制、跨作品共享、三级节点、节点批量新建、纪元日期计算。
