# Checklist — CHG-003 体系类别与节点

- 状态: F1 待批准（全部未勾选）
- 勾选规则: 只有拿到可复核证据（测试名、命令输出、截图、API 往返）才能勾选；不得凭印象勾选。
- 证据格式: `命令/测试名 → 结果`，或 `文件:行` + 关键断言。
- 对应规格: SPEC-SYS-001~005、PG-10、AC-07/AC-13/AC-14、EX-16/EX-21。

## 1. 类别

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| AC-01 | 新建作品存在唯一内置「境界」：`is_builtin=true`、`category_type=normal`、`is_enabled=true`，列表置顶 | `TestMigrations` / `TestSystemCategories_Builtin` | ⬜ |
| AC-02 | 内置类别改名、停用、删除均被 `CATEGORY_BUILTIN_IMMUTABLE` 拒绝；`is_enabled=true` 为无操作 | `TestSystemCategories_Builtin` | ⬜ |
| AC-03 | 自定义类别新增成功，`sort_order` 递增，`is_builtin=false` | `TestSystemCategories_Create` | ⬜ |
| AC-04 | 类别上限 10（含内置）；第 11 个返回 `409 CATEGORY_LIMIT`；页面新增按钮禁用 | `TestSystemCategories_Limit` + E2E `system.spec.ts` | ⬜ |
| AC-05 | 类别名称唯一且不区分大小写、全半角、连续空白；冲突返回 `409 NAME_CONFLICT` | `TestSystemCategories_NameConflict` | ⬜ |
| AC-06 | 类别类型仅 `normal`/`era`，创建后不可修改；PATCH 带 `category_type` 返回 422 | `TestSystemCategories_TypeImmutable` | ⬜ |
| AC-07 | 无引用类别可停用，节点保留；可重新启用 | `TestSystemCategories_Disable` | ⬜ |
| AC-08 | 有引用类别停用/删除返回 `409 REFERENCE_CONFLICT`，`details` 数量=去重人物数且含节点位置 | `TestSystemRefs_Category` | ⬜ |
| AC-09 | 类别列表返回 `node_count`、`reference_count`、`is_enabled`、`category_type`、`is_builtin`，内置置顶 | `TestSystemCategories_List` | ⬜ |

## 2. 节点

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| ND-01 | 可创建主层级；可在根下创建子层级，形成两级 | `TestSystemNodes_CreateTwoLevels` | ⬜ |
| ND-02 | 在子层级下再建节点返回 `409 NODE_LEVEL_LIMIT`；页面无第三级入口 | `TestSystemNodes_LevelLimit` + E2E | ⬜ |
| ND-03 | `parent_id` 与类别不匹配返回 `409 NODE_PARENT_MISMATCH` | `TestSystemNodes_ParentMismatch` | ⬜ |
| ND-04 | 同类别同父级名称唯一（含大小写/全半角/空白变体），冲突 `409 NAME_CONFLICT`；不同父级可同名 | `TestSystemNodes_SiblingName` | ⬜ |
| ND-05 | 上移/下移调用 `reorder` 返回 204，刷新后顺序保持；集合缺失/重复/跨父级 422 | `TestSystemNodes_Reorder` + E2E 刷新 | ⬜ |
| ND-06 | 纪元类别可保存起止文本（可空、不校验日期）；普通类别提交非空返回 422 | `TestSystemNodes_Era` | ⬜ |
| ND-07 | 有子节点的父节点删除返回 `409 HAS_CHILDREN` 并列出子节点 | `TestSystemNodes_HasChildren` | ⬜ |
| ND-08 | 被引用节点删除返回 `409 REFERENCE_CONFLICT`；无引用节点删除成功并返回 `undo_token` | `TestSystemRefs_Node` / `TestSystemNodes_DeleteUndo` | ⬜ |
| ND-09 | 节点 DTO 返回 `reference_count`、API `review_status`（DB `reviewed`→`confirmed`）、`reviewed_at`、`reviewed_by`；`review_status=pending` 可筛选 | `TestSystemNodes_ReviewMapping` | ⬜ |
| ND-10 | 停用类别下新增/改名/删除/排序被 `CATEGORY_DISABLED` 拒绝；节点仍可读 | `TestSystemNodes_CategoryDisabled` | ⬜ |

## 3. 回收站与恢复

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| TR-01 | 删除类别后类别与全部节点整组软删除（`deleted_root_id=category_id`），返回 `affected_nodes` 与 `undo_token` | `TestSystemCategories_DeleteGroup` | ⬜ |
| TR-02 | 回收站按 `root_type=system_category` 只显示一条类别根；节点不单独出现；`parent_name` 为作品名 | `TestTrashSystem_List` | ⬜ |
| TR-03 | 类别整组恢复原子：任一节点名称冲突且未提供 `renames[]` → 422，无半恢复 | `TestTrashSystem_RestoreAtomic` | ⬜ |
| TR-04 | 提供 `new_name`（类别）与 `renames[]`（节点）后整组恢复成功，名称生效 | `TestTrashSystem_RestoreRenames` | ⬜ |
| TR-05 | 节点单独软删除后回收站出现一条节点根，可恢复；类别/父级缺失返回 `409 PARENT_DELETED` | `TestTrashSystem_NodeRestore` | ⬜ |
| TR-06 | 8 秒内撤销类别整组/节点单项成功；超时返回 `UNDO_EXPIRED` | `TestTrashSystem_Undo` | ⬜ |
| TR-07 | 永久删除类别/节点：有引用被阻止；无引用按「引用→节点→类别」物理删除；快照先落盘 | `TestTrashSystem_Purge` | ⬜ |
| TR-08 | 清空回收站包含三类根；存在被引用的类别/节点时整体拒绝，不落快照、不清空 | `TestTrashSystem_PurgeAll` | ⬜ |

## 4. 作品级联

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| WS-01 | 删除作品时其类别与节点全部随组软删除；影响条数=类别+节点 | `TestWorks_DeleteIncludesNodes` | ⬜ |
| WS-02 | 恢复作品时类别与节点一并恢复，无孤儿 | `TestWorks_RestoreIncludesNodes` | ⬜ |
| WS-03 | 永久删除作品按顺序清理引用/节点/类别/作品，无 FK 阻塞与残留 | `TestWorks_PurgeWithNodes` | ⬜ |

## 5. UI / 交互

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| UI-01 | 导航「资料 > 体系」进入 `/system`；无当前作品显示引导 | E2E `shell.spec.ts` / `system.spec.ts` | ⬜ |
| UI-02 | 左栏类别列表 280px（≥1280）/240px（1024–1279）；内置置顶 + 「内置 · 必开」；不显示改名/停用/删除 | E2E 截图 + 断言 | ⬜ |
| UI-03 | 类别详情显示类型/启停 Badge、节点数、引用数、静态说明、待校对筛选；停用类别只读 + 启用按钮 | E2E `system.spec.ts` | ⬜ |
| UI-04 | 新增类别、改名、停用、启用、删除整组确认均可完成；删除文案说明节点一并进回收站 | E2E `system.spec.ts` | ⬜ |
| UI-05 | 引用阻断弹出清单（按节点分组）；名称冲突/版本冲突为字段级或刷新提示，无英文堆栈 | E2E + `messages.ts` | ⬜ |
| UI-06 | 节点树两层渲染；行菜单按层级变化；上移/下移首末禁用；纪元类别显示起止输入 | E2E `system.spec.ts` | ⬜ |
| UI-07 | 加载/空/错误/网络/10 类上限/停用只读状态完整，保留输入 | E2E + 视觉基线 | ⬜ |
| UI-08 | 1024 无横向溢出；键盘可达类别列表、节点树、行操作；axe 无 serious/critical | E2E `visual.spec.ts` / `keyboard.spec.ts` / `a11y.spec.ts` | ⬜ |
| UI-09 | 视觉基线 system 1280/1024 差异 ≤ 0.02 | `visual.spec.ts` | ⬜ |

## 6. 安全 / 契约 / 回归

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| SEC-01 | 所有写请求缺 `X-Requested-With` 返回 `403 FORBIDDEN_ORIGIN`；更新/删除缺或错 `If-Match` 返回 `409 VERSION_CONFLICT` | `TestSystem_Guards` | ⬜ |
| SEC-02 | 所有类别/节点查询按 `work_id` 隔离，不返回其他作品数据 | `TestSystem_CrossWorkIsolation` | ⬜ |
| CT-01 | 错误码仅使用既有集合；新增语义使用 `CATEGORY_LIMIT/CATEGORY_BUILTIN_IMMUTABLE/CATEGORY_DISABLED/NODE_LEVEL_LIMIT/NODE_PARENT_MISMATCH/HAS_CHILDREN/PARENT_DELETED/REFERENCE_CONFLICT` | `errors.go` + 前端 `messages.ts` | ⬜ |
| RG-01 | CHG-002 既有 E2E（34 条）与回收站/设置/作品切换全部仍通过 | `scripts/ci.ps1` | ⬜ |
| RG-02 | 迁移 0003 Up/Down 往返成功；容器内三表存在、内置境界唯一 | `TestMigrations` + 容器核对 | ⬜ |
| RG-03 | 全量 `scripts/ci.ps1` 通过（Go 单测/集成、前端单测、构建、E2E、视觉） | CI 输出 | ⬜ |
| DOC-01 | `verification.md` 记录真实命令、结果与差异 | 文件 | ⬜ |
| DOC-02 | `PROJECT-SETUP.md`、`PG-10` 实施差异已同步 | 文件 diff | ⬜ |

## 7. 本轮不验收（延后）

以下条目依赖后续变更，不作为 CHG-003 返工依据：

| 条目 | 延后到 | 原因 |
|---|---|---|
| 人物表单按启用类别动态出现字段 | CHG-004 | 人物表与引用写入属 CHG-004 |
| 类别/节点改名在人物详情与列表同步 | CHG-004 | 无人物页面与引用数据 |
| 检索结果包含类别/节点、改名同步检索 | CHG-009 | 检索文档维护属 CHG-009 |
| 导入模板类别/节点、待校对确认 `/reviews` | CHG-011 | 导入与校对确认接口属 CHG-011 |
| 类别自定义排序、复制、跨作品共享 | 后续路线图 | SPEC-SYS-001/002 边界 |
| 三级及以上节点、跨类移动、批量新建 | 后续路线图 | SPEC-SYS-003 边界 |
| 纪元日期计算/格式校验、类别 `description` 字段 | 后续路线图 | PG-10 与 data-model 未要求 |
