# Checklist — CHG-003 体系类别与节点

- 状态: F3 已验收（全部有证据勾选；延后项见 §7）
- 勾选规则: 只有拿到可复核证据（测试名、命令输出、截图、API 往返）才能勾选；不得凭印象勾选。
- 证据格式: `命令/测试名 → 结果`，或 `文件:行` + 关键断言。
- 对应规格: SPEC-SYS-001~005、PG-10、AC-07/AC-13/AC-14、EX-16/EX-21。

## 1. 类别

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| AC-01 | 新建作品存在唯一内置「境界」：`is_builtin=true`、`category_type=normal`、`is_enabled=true`，列表置顶 | `TestCreateWorkCreatesBuiltinCategoryAndActivates` → 通过；容器 psql 唯一性查询 0 异常行 | ✅ |
| AC-02 | 内置类别改名、停用、删除均被 `CATEGORY_BUILTIN_IMMUTABLE` 拒绝；`is_enabled=true` 为无操作 | `TestSystemCategoriesBuiltinProtection` → 409×3、无操作 version 不变 | ✅ |
| AC-03 | 自定义类别新增成功，`sort_order` 递增，`is_builtin=false` | `TestSystemCategoriesListAndCreate`、`TestSystemRepoCategories` → 通过 | ✅ |
| AC-04 | 类别上限 10（含内置）；第 11 个返回 `409 CATEGORY_LIMIT`；页面新增按钮禁用 | `TestSystemCategoriesLimit` → 409；E2E `system.spec.ts`（上限用例）→ 通过 | ✅ |
| AC-05 | 类别名称唯一且不区分大小写、全半角、连续空白；冲突返回 `409 NAME_CONFLICT` | `TestSystemCategoriesListAndCreate` → `NAME_CONFLICT`；恢复冲突 `TestSystemTrashRestoreCategory` → 422 | ✅ |
| AC-06 | 类别类型仅 `normal`/`era`，创建后不可修改；PATCH 带 `category_type` 返回 422 | `TestSystemCategoriesBuiltinProtection` → details.field=`category_type` | ✅ |
| AC-07 | 无引用类别可停用，节点保留；可重新启用 | `TestSystemCategoriesRenameAndDisable` → 停用/启用 version 递增 | ✅ |
| AC-08 | 有引用类别停用/删除返回 `409 REFERENCE_CONFLICT`，`details` 数量=去重人物数且含节点位置 | `TestSystemCategoriesReferenceConflict`、`TestSystemRefsWorkPurge` → details 1 条、value=节点名 | ✅ |
| AC-09 | 类别列表返回 `node_count`、`reference_count`、`is_enabled`、`category_type`、`is_builtin`，内置置顶 | `TestSystemRepoCategories`、`TestSystemRepoReferenceCounts` → 通过 | ✅ |

## 2. 节点

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| ND-01 | 可创建主层级；可在根下创建子层级，形成两级 | `TestSystemNodesTreeAndEra` → 根 2、子 1；E2E `system.spec.ts` | ✅ |
| ND-02 | 在子层级下再建节点返回 `422 NODE_LEVEL_LIMIT`；页面无第三级入口 | `TestSystemNodesHierarchyAndUnique` → 422 `NODE_LEVEL_LIMIT`；E2E 断言子行无新增入口 | ✅ |
| ND-03 | `parent_id` 与类别不匹配返回 `422 NODE_PARENT_MISMATCH` | `TestSystemNodesHierarchyAndUnique` → 跨类别/不存在父级各 422 | ✅ |
| ND-04 | 同类别同父级名称唯一（含大小写/全半角/空白变体），冲突 `409 NAME_CONFLICT`；不同父级可同名 | `TestSystemNodesHierarchyAndUnique` → `Ｓｗｏｒｄ` 与 `"  sword  "` 冲突；不同父级同名通过 | ✅ |
| ND-05 | 上移/下移调用 `reorder` 返回 204，刷新后顺序保持；集合缺失/重复/跨父级 422 | `TestSystemNodesReorder` → 204 且顺序 `c,b,a`；E2E 刷新后顺序保持 | ✅ |
| ND-06 | 纪元类别可保存起止文本（可空、不校验日期）；普通类别提交非空返回 422 | `TestSystemNodesTreeAndEra`（修剪回填）、`TestSystemNodesUpdate`（`null` 清空）、`TestSystemNodesHierarchyAndUnique`（普通类别 422） | ✅ |
| ND-07 | 有子节点的父节点删除返回 `409 HAS_CHILDREN` 并列出子节点 | `TestSystemNodesDelete` → 409；E2E 子节点名展示 | ✅ |
| ND-08 | 被引用节点删除返回 `409 REFERENCE_CONFLICT`；无引用节点删除成功并返回 `undo_token` | `TestSystemNodesDelete`、`TestSystemTrashUndoCategoryAndNode` | ✅ |
| ND-09 | 节点 DTO 返回 `reference_count`、API `review_status`（DB `reviewed`→`confirmed`）、`reviewed_at`、`reviewed_by`；`review_status=pending` 可筛选 | `TestSystemRepoNodeListReviewAndDisabled`（pending=2）、`TestSystemNodesTreeAndEra`（confirmed、非法值 422） | ✅ |
| ND-10 | 停用类别下新增/改名/删除/排序被 `CATEGORY_DISABLED` 拒绝；节点仍可读 | `TestSystemNodesDisabledCategory` → 4 个写操作 409、读取 2 条正常 | ✅ |

## 3. 回收站与恢复

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| TR-01 | 删除类别后类别与全部节点整组软删除（`deleted_root_id=category_id`），返回 `affected_nodes` 与 `undo_token` | `TestSystemCategoriesDeleteGroup` → 通过 | ✅ |
| TR-02 | 回收站按 `root_type=system_category` 只显示一条类别根；节点不单独出现；`parent_name` 为作品名 | `TestSystemTrashListsThreeTypes` → 三类根各 1 条 | ✅ |
| TR-03 | 类别整组恢复原子：任一节点名称冲突且未提供 `renames[]` → 422，无半恢复 | `TestSystemTrashRestoreCategory` → 422 后类别仍在回收站 | ✅ |
| TR-04 | 提供 `new_name`（类别）与 `renames[]`（节点）后整组恢复成功，名称生效 | `TestSystemTrashRestoreCategory` → `RestoredCount=4`、树形名称生效 | ✅ |
| TR-05 | 节点单独软删除后回收站出现一条节点根，可恢复；类别/父级缺失返回 `409 PARENT_DELETED` | `TestSystemTrashRestoreNode` → 单独节点仍在回收站、父级缺失 409 `PARENT_DELETED` | ✅ |
| TR-06 | 8 秒内撤销类别整组/节点单项成功；超时返回 `UNDO_EXPIRED` | `TestSystemTrashUndoCategoryAndNode`；`TestDeleteThenUndo` 回归 | ✅ |
| TR-07 | 永久删除类别/节点：有引用被阻止；无引用按「引用→节点→类别」物理删除；快照先落盘 | `TestSystemTrashPurgeCategoryAndNode` → `REFERENCE_CONFLICT` 后清理引用再 204 | ✅ |
| TR-08 | 清空回收站包含三类根；存在被引用的类别/节点时整体拒绝，不落快照、不清空 | `TestSystemTrashPurgeAllTypes`、`TestSystemRefsPurgeAllDedup` | ✅ |

## 4. 作品级联

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| WS-01 | 删除作品时其类别与节点全部随组软删除；影响条数=类别+节点 | `TestSystemTrashWorkCascade` → 通过 | ✅ |
| WS-02 | 恢复作品时类别与节点一并恢复，无孤儿 | `TestSystemTrashWorkCascade`、`TestTrashRestoreWork` 回归 | ✅ |
| WS-03 | 永久删除作品按顺序清理引用/节点/类别/作品，无 FK 阻塞与残留 | `TestSystemRefsWorkPurge` → 通过 | ✅ |

## 5. UI / 交互

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| UI-01 | 导航「资料 > 体系」进入 `/system`；无当前作品显示引导 | E2E `shell.spec.ts`（体系链接）、`system.spec.ts`（无作品空态） | ✅ |
| UI-02 | 左栏类别列表 280px（≥1280）/240px（1024–1279）；内置置顶 + 「内置 · 必开」；不显示改名/停用/删除 | E2E `visual.spec.ts-snapshots/system-1280/1024`；`system.spec.ts` 内置保护 | ✅ |
| UI-03 | 类别详情显示类型/启停 Badge、节点数、引用数、静态说明、待校对筛选；停用类别只读 + 启用按钮 | E2E `system.spec.ts`（停用只读用例） | ✅ |
| UI-04 | 新增类别、改名、停用、启用、删除整组确认均可完成；删除文案说明节点一并进回收站 | E2E `system.spec.ts`（类别 CRUD + 整组删除） | ✅ |
| UI-05 | 引用阻断弹出清单（按节点分组）；名称冲突/版本冲突为字段级或刷新提示，无英文堆栈 | E2E `system.spec.ts`（引用阻断）；`messages.ts` | ✅ |
| UI-06 | 节点树两层渲染；行菜单按层级变化；上移/下移首末禁用；纪元类别显示起止输入 | E2E `system.spec.ts`（两级、排序、纪元） | ✅ |
| UI-07 | 加载/空/错误/网络/10 类上限/停用只读状态完整，保留输入 | E2E `system.spec.ts`；空节点树修复 R-01 | ✅ |
| UI-08 | 1024 无横向溢出；键盘可达类别列表、节点树、行操作；axe 无 serious/critical | E2E `system.spec.ts`（scrollWidth）、`keyboard.spec.ts`、`a11y.spec.ts` | ✅ |
| UI-09 | 视觉基线 system 1280/1024 差异 ≤ 0.02 | `visual.spec.ts` → 通过（`maxDiffPixelRatio 0.02`） | ✅ |

## 6. 安全 / 契约 / 回归

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| SEC-01 | 所有写请求缺 `X-Requested-With` 返回 `403 FORBIDDEN_ORIGIN`；更新/删除缺或错 `If-Match` 返回 `409 VERSION_CONFLICT` | `works_test.go:362`（全局写守卫 `FORBIDDEN_ORIGIN`）；系统用例多处 `VERSION_CONFLICT` | ✅ |
| SEC-02 | 所有类别/节点查询按 `work_id` 隔离，不返回其他作品数据 | `TestSystemCrossWorkIsolation` → 通过 | ✅ |
| CT-01 | 错误码仅使用既有集合；新增语义使用 `CATEGORY_LIMIT/CATEGORY_BUILTIN_IMMUTABLE/CATEGORY_DISABLED/NODE_LEVEL_LIMIT/NODE_PARENT_MISMATCH/HAS_CHILDREN/PARENT_DELETED/REFERENCE_CONFLICT` | `backend/internal/model/errors.go`、`backend/internal/httpapi/request.go`、`frontend/lib/messages.ts` | ✅ |
| RG-01 | CHG-002 既有 E2E（34 条）与回收站/设置/作品切换全部仍通过 | `npx playwright test` → 50 passed / 0 failed | ✅ |
| RG-02 | 迁移 0003 Up/Down 往返成功；容器内三表存在、内置境界唯一 | `TestMigrationRoundTrip`；容器 psql `goose_db_version=3`、三表存在、唯一性 0 异常 | ✅ |
| RG-03 | 全量 `scripts/ci.ps1` 通过（Go 单测/集成、前端单测、构建、E2E、视觉） | 各步骤逐条执行全部通过；本机 gofmt 步骤因 `core.autocrlf=true` 环境性中止（LF 副本复核无输出），详见 `verification.md` §1/§5 | ✅ |
| DOC-01 | `verification.md` 记录真实命令、结果与差异 | `changes/CHG-003-systems-nodes/verification.md` | ✅ |
| DOC-02 | `PROJECT-SETUP.md`、`PG-10` 实施差异已同步 | `project/PROJECT-SETUP.md`、`project/page-specs/PG-10-system-config.md` §14 | ✅ |

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
