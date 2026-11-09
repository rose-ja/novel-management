# Proposal — CHG-003 体系类别与节点

- Change ID: CHG-003
- Status: Draft（F1 待批准）
- Owner: 作者（单用户）
- Created at: 2026-11-08
- 关联 Spec: SPEC-SYS-001、SPEC-SYS-002、SPEC-SYS-003、SPEC-SYS-004、SPEC-SYS-005（均为 `planned`）
- 关联页面: PG-10 体系配置（`/system`）
- 依赖: CHG-002（作品上下文、软删除/撤销/回收站、快照基座）
- 后续依赖方: CHG-004（人物动态体系字段）、CHG-009（检索同步）、CHG-011（导入类别协议）

## 1. Why

CHG-002 只交付了「创建作品时自动写入内置『境界』」的最小数据，`/system` 没有任何入口、接口或节点结构；用户无法新增类别、维护两级节点，也无法验证引用阻断与整组回收站。CHG-004 的人物动态表单又直接依赖「启用的类别 + 可选节点」，因此本变更必须先补齐体系域，否则人物档案无法开工。

本变更交付：类别生命周期（新增/改名/停用/启用/删除、上限 10、类型不可变、内置保护）、两级节点（新增/改名/删除/排序/纪元文本）、引用计数与停用/删除阻断、类别整组与节点单项的回收站/撤销/恢复，以及 `/system` 页面。

## 2. 现状与冲突

### 2.1 已核实的当前代码事实

- `backend/internal/migrations/0002_works_trash_snapshot.sql` 已建 `system_categories` 最小结构：`category_type` 默认值为 `'realm'`、无 CHECK、无 `sort_order`；`system_nodes`、`character_system_refs` 尚不存在。
- `backend/internal/service/works.go` 创建作品时写入 `category_type='normal'` 的内置境界；`repository/works.go` 的 `SoftDeleteGroup`/`RestoreGroup` 只处理 `works` 与 `system_categories`，`PurgeGroup`/`PurgeAllDeleted` 只删 `works`。
- `backend/internal/service/trash.go` 仅支持 `root_type=work`；其他类型返回空列表或 `ErrInvalidRootType`。
- 前端 `frontend/components/shell/global-nav.tsx` 只有「概览 / 回收站 / 设置」，无 `/system`。
- 前端设计系统已有 `Badge`、`Dialog`、`FormField`、`IconButton`、`Skeleton`、`EmptyState`、`Toast` 等，无 Tree 组件。

### 2.2 冲突与处理

| ID | 冲突 | 涉及文档/代码 | 处理 |
|---|---|---|---|
| C-01 | `system_categories.category_type` 数据库默认 `'realm'` 且无 CHECK；data-model §3.8/§4.3 要求 `normal` / `era` 且 CHECK | `0002`、`data-model.md` | 0003 将默认改为 `normal`，把历史 `realm` 回填为 `normal`，加 CHECK；以 data-model 为准 |
| C-02 | `works.name_key` 实际为 `lower(trim)`；data-model §4.5 要求 NFKC → casefold → trim → 空白折叠 | `model/validate.go`、`data-model.md` | 本轮只对体系类别/节点使用完整规则（D-02）；works 偏差记录，留 CHG-012 统一 |
| C-03 | 引用阻断依赖 `characters` 与 `character_system_refs`，但人物域属 CHG-004 | `specs/systems`、roadmap | 0003 先建 `character_system_refs` 骨架（无人物 FK），本轮可真实验收引用计数/阻断；CHG-004 补人物 FK 与名称详情（D-01） |
| C-04 | PG-10 含「从固定模板导入」「确认已校对」，但导入与 `/reviews` 属 CHG-011 | PG-10、roadmap | 本轮不提供导入入口；节点 DTO 返回 `review_status`/`reviewed_at` 并支持 `review_status=pending` 筛选，确认接口延后 CHG-011 |
| C-05 | PG-10 类别详情区的「说明」文案在 data-model 中无对应列 | PG-10、data-model | 作为页面静态说明，不新增数据库字段 |
| C-06 | 删除作品/永久删除在加入 `system_nodes` 后会遇到 FK `RESTRICT` | `repository/works.go`、data-model §4.4 | 本变更同步收口：作品软删除/恢复包含节点；永久删除按「引用 → 节点 → 类别 → 作品」顺序物理删除 |

## 3. What Changes

### 3.1 用户可感知

- 导航新增「体系」，打开 `/system`：
  - 左栏类别列表（280px；1024–1279 为 240px），内置「境界」置顶并标记「内置 · 必开」，显示类型、启用状态、节点数、被引用人数。
  - 右栏类别详情与两级节点树；新增主层级/子层级、改名、删除、上移/下移、纪元起止文本。
  - 新增类别（名称 + 类型，最多 10 类）；类别改名、停用、启用、删除；删除类别时说明节点一并进入回收站。
  - 被引用时停用/删除被阻止，弹出引用清单；停用类别只读且可重新启用。
  - 空节点、无自定义类别、加载、错误、10/10 禁用、停用只读等状态完整。
- 回收站新增「体系类别」「体系节点」两类根：类别整组显示为一条，节点单独一条；可恢复、永久删除、清空；名称冲突时要求改名。

### 3.2 系统内部

- 迁移 `0003_system_nodes.sql`：修正类别类型约束、补 `sort_order`；新建 `system_nodes` 与 `character_system_refs`。
- 新增 `model/system.go`、`repository/system.go`、`service/system.go`、`httpapi/system.go`；扩展 `service/trash.go` 与 `repository/works.go`。
- 新增共享名称规范化 `model.NormalizeKey`（NFKC + casefold + trim + 空白折叠），仅用于体系类别/节点。
- 前端新增 `app/system/page.tsx`、`components/system/*`、`lib/api/system.ts`，扩展 `lib/messages.ts` 与导航。

## 4. In Scope

- SPEC-SYS-001：内置境界保护、自定义类别新增/改名/停用/启用/删除、名称唯一、类型不可变、列表计数。
- SPEC-SYS-002：10 类上限、`normal`/`era` 类型、停用语义（无引用才可停用、数据保留、重新启用）。
- SPEC-SYS-003：两级节点、同级唯一、同级排序、纪元起止文本、父节点有子节点时删除阻止。
- SPEC-SYS-004（本轮可验收部分）：节点 DTO 引用计数、删除引用阻断；动态人物表单与人物侧改名同步延后 CHG-004。
- SPEC-SYS-005：停用/删除引用阻断、类别整组软删除/恢复、`new_name`/`renames[]`、整组原子、节点单项软删除/撤销/恢复。
- 引用计数与阻断包含回收站中的引用（数据模型 §5 C-07）；错误 `details` 标注「回收站中」。
- 作品软删除/恢复/永久删除与新表的级联收口。

## 5. Out of Scope

- 人物表单按启用类别动态渲染、人物引用写入、改名在人物详情/列表同步（CHG-004）。
- 检索文档维护与检索命中（CHG-009）。
- 导入模板类别解析、待校对确认接口 `/reviews`（CHG-011）。
- 类别排序自定义、类别复制、跨作品共享类别（SPEC-SYS-001/002 边界）。
- 三级及以上节点、跨类别移动、节点批量新建（SPEC-SYS-003 边界）。
- 纪元日期计算/格式校验、类别 `description` 字段、体系图形化树。
- 类别/节点的搜索文档与缓存失效。

## 6. F1 裁决记录

- D-01 引用依赖：0003 先建 `character_system_refs` 骨架（`character_id` 无 FK，`category_id`/`node_id` 有 FK），本轮引用计数与阻断可真实验收；CHG-004 建人物表后补 `characters` FK（ON DELETE CASCADE）与人物名称详情。（用户裁决 2026-11-08）
- D-02 名称规范化：体系类别/节点使用完整规则（NFKC → casefold → trim → 连续空白折叠）；works 保持现状，偏差记录，CHG-012 统一。（用户裁决 2026-11-08）
- D-03 类型收敛：`category_type` 取值 `normal` / `era`；0003 加 CHECK 并回填 `realm → normal`；PATCH 不接受 `category_type`。
- D-04 类别排序：不提供自定义排序；内置置顶，其他按 `created_at ASC, id ASC`；`sort_order` 仅记录创建顺序，不做拖拽。
- D-05 节点回收站：节点可单独软删除/撤销/恢复；类别删除时节点随整组进入回收站，回收站只显示一条类别项；恢复节点时类别或父级缺失返回 `PARENT_DELETED`。
- D-06 停用类别：只读展示 + 可启用；不显示新增/改名/删除/排序节点入口；服务端拒绝向停用类别新增节点（`CATEGORY_DISABLED`）。
- D-07 校对：节点 DTO 返回 API 映射值 `review_status`（DB `reviewed` → API `confirmed`）、`reviewed_at`、`reviewed_by`；支持 `review_status=pending` 过滤；确认动作延后 CHG-011。
- D-08 PG-10 的「从固定模板导入」入口不渲染（CHG-011 再补），避免死按钮。
- D-09 删除/停用引用检查包含回收站引用；`details` 每项带 `in_trash` 标记；无引用才允许操作。
- D-10 永久删除类别/节点同样检查引用（含回收站），不级联删除人物引用；物理删除顺序按 C-06。

## 7. Acceptance Criteria

1. Given 新建作品，When 打开 `/system`，Then 存在唯一内置「境界」，且页面与 API 均无改名/停用/删除入口；直接调用返回 `CATEGORY_BUILTIN_IMMUTABLE`。
2. Given 已有 8 个自定义类别（含内置共 9 个），When 新增第 9 个自定义类别（共 10 个），Then 成功；再新增第 10 个自定义类别（共 11 个），Then `409 CATEGORY_LIMIT` 且页面「新增类别」禁用并说明上限。
3. Given 新增类别「圣体」类型「普通」，When 再次新增「圣体」或「圣体」的大小写/全半角变体，Then `409 NAME_CONFLICT`；类别类型创建后 PATCH 被拒绝。
4. Given 类别「星命」无引用，When 停用，Then `is_enabled=false`，节点保留；When 重新启用，Then 状态恢复。
5. Given 类别下任一节点被 `character_system_refs` 引用，When 停用或删除该类别，Then `409 REFERENCE_CONFLICT`，`details` 列出引用数量与位置；停用/删除均未发生。
6. Given 类别「境界」已有根「炼气」与子「初期」，When 在「初期」下新增节点，Then `422 NODE_LEVEL_LIMIT`；页面无第三级入口。
7. Given 同类别同父级已有「炼气」，When 新增「炼气」或大小写/全半角变体，Then `409 NAME_CONFLICT`。
8. Given 同级三个节点，When 上移/下移，Then `POST /system-nodes/reorder` 返回 `204`，刷新后顺序保持；集合不完整或跨父级返回 `422 VALIDATION_ERROR`。
9. Given 类型为「纪元」的类别，When 新增节点并填写起止文本，Then 保存成功且仅按 `sort_order` 展示；起止文本可空、不校验格式；普通类别提交纪元字段被拒绝。
10. Given 父节点仍有子节点，When 删除父节点，Then `409 HAS_CHILDREN` 并列出子节点；删除无引用叶子节点成功并返回 `undo_token`，8 秒内撤销恢复。
11. Given 类别无引用，When 删除类别，Then 类别与全部节点整组软删除（`deleted_root_id=category_id`），返回 `affected_nodes` 与 `undo_token`；回收站出现一条类别项。
12. Given 回收站中的类别整组与现存类别重名且未提供 `new_name`，When 恢复，Then `422 RESTORE_CONFLICT`，不产生半恢复；提供 `new_name` 或对冲突节点提供 `renames[]` 后整组原子恢复。
13. Given 节点被删除且其父级/类别仍在回收站，When 恢复节点，Then `409 PARENT_DELETED`；父级与类别恢复后可恢复节点。
14. Given 类别列表，Then 每项返回 `node_count`、`reference_count`、`is_enabled`、`category_type`、`is_builtin`；节点 DTO 返回 `reference_count`、`review_status`、`reviewed_at`、`reviewed_by`。
15. Given 删除作品，Then 其类别与节点全部随作品进入回收站；恢复作品时一并恢复；永久删除作品成功且无 FK 残留。
16. Given `/system`，Then 1280/1024 无横向溢出；加载/空/错误/停用/10 类上限状态完整；类别列表与节点树键盘可达，axe 无 serious/critical。
17. Given 两个标签页同时编辑同一类别或节点，When 后保存者提交旧 `version`，Then `409 VERSION_CONFLICT`，无静默覆盖。

## 8. Risks

| 风险 | 影响 | 处置 |
|---|---|---|
| 引用骨架表与 CHG-004 人物表衔接 | 0004 加 FK 时可能出现孤儿引用 | 0004 迁移前清理/校验孤儿；本变更测试数据均在事务回滚内，不落库 |
| 整组恢复的 `renames[]` 编排复杂 | 半恢复或名称错乱 | 单事务 + 先全量冲突预检，再统一改名，最后恢复；任一冲突返回 422 |
| 作品永久删除顺序遗漏 | FK 阻塞或残留 | 固定顺序：引用 → 节点 → 类别 → 作品；集成测试覆盖 |
| 类别上限并发竞态 | 超过 10 类 | 本地单用户接受；`COUNT` 与插入同事务；记录为已知限制 |
| 停用/删除/回收站语义混淆 | 数据误删 | 停用只置 `is_enabled=false`；删除才软删除；UI 文案与确认弹窗区分 |
| 前端 Tree 与键盘可访问性 | 不达 PG-10 §10 | 使用 `role=tree/treeitem` + `aria-expanded`；E2E 键盘与 axe 覆盖 |

## 9. Rollback

- 迁移 `0003` 提供 `Down`：删 `character_system_refs`、`system_nodes`；移除类别 CHECK 与 `sort_order`；恢复 `category_type` 默认 `'realm'`（与 0002 一致）。
- 代码回退本变更提交；前端静态导出随镜像回退。
- 数据回退：删除前已有作品快照可恢复；体系数据随作品快照回退。

## 10. Dependencies

- 前置：CHG-002 已验收并归档（`changes/archive/CHG-002-works-lifecycle/`）。
- 后续：CHG-004 建人物表并补 `character_system_refs` FK；CHG-009 接入检索；CHG-011 接入导入与校对确认。
