# Tasks — CHG-003 体系类别与节点

- 分支: `feat/chg-003-systems-nodes`
- 执行原则: 按 T1 → T12 顺序；每个任务完成即跑该任务 Tests；F2 结束跑全量 `scripts/ci.ps1`。
- 所有文档、注释、提交信息使用中文；`scripts/ci.ps1` 保持纯 ASCII 英文。
- 参考: `changes/CHG-003-systems-nodes/proposal.md`、`design.md`。

## T1 迁移 0003 与领域模型

- [x] 已完成：迁移往返、模型单测通过（`TestMigrationRoundTrip`、`go test ./backend/internal/model/...`）

**Files**
- `backend/internal/migrations/0003_system_nodes.sql`（新增）
- `backend/internal/migrations/migrations.go`（如迁移清单为显式列表则登记；`go:embed` 则不改）
- `backend/internal/model/system.go`（新增：`SystemNode`、`NormalizeKey`、`AssembleNodeTree`、`ValidateSiblingOrder`、`ReviewStatusToAPI/ToDB`）
- `backend/internal/model/work.go`（`SystemCategory` 增加 `SortOrder`、`NodeCount`、`ReferenceCount`）
- `backend/internal/model/system_test.go`（新增）

**Steps**
1. 按 `design.md §3` 写 Up/Down：修正 `category_type` 默认值与 CHECK、补 `sort_order`、建 `system_nodes` 与 `character_system_refs`、索引与 CHECK。
2. 实现 `NormalizeKey`：NFKC → 小写 → trim → 连续空白折叠；空值/超 200 rune 返回字段级校验错误。
3. 实现树组装与同级排序集合校验纯函数；实现 review 状态双向映射。
4. 同步 `SystemCategory` 模型字段与 JSON 标签（计算字段 `gorm:"-"`）。

**Expected**
- `goose up`/`down` 在空库与已有 0002 数据库上均可往返；`realm` 历史值被回填。
- 两级树按 `(sort_order,id)` 稳定排序；重复/缺失的排序集合返回校验错误。
- `NormalizeKey("  Ｓｗｏｒｄ  ")` 与 `"sword"` 生成同一 key。

**Tests**
- `go test ./backend/internal/model/...`
- `go test ./backend/internal/integration/... -run TestMigrations`

**Depends** 无

## T2 仓储层：类别、节点、引用计数

- [x] 已完成：`TestSystemRepo` 7 个用例通过（唯一索引、停用/筛选、引用去重、排序、回收站组）

**Files**
- `backend/internal/repository/system.go`（新增）
- `backend/internal/repository/works.go`（仅补作品级联所需的节点删除/恢复入口，具体在 T6）
- `backend/internal/integration/system_repo_test.go`（新增，如与 service 测试合并则并入 `system_test.go`）

**Steps**
1. 类别：列表（内置置顶、`created_at ASC,id ASC`）、按 ID 取、插入、带版本更新、计数、名称占用（`name_key`）、软删除组、恢复组、删除根查询。
2. 节点：列表（树/平铺、`review_status` 过滤）、按 ID 取、插入、带版本更新、同级名称占用、子节点计数、软删除、恢复、删除根查询、排序批量更新。
3. 引用：按类别/节点聚合 `COUNT(DISTINCT character_id)`；按节点列引用明细（`character_id`、`node_id`、节点名）。
4. 所有查询带 `work_id` 隔离；未删条件统一 `deleted_at IS NULL`。

**Expected**
- 同名大小写/全半角/多空格在同类同级被唯一索引拦截。
- 停用类别仍可列出节点；`review_status=pending` 只返回待校对节点。
- 引用计数按人物去重，不按引用行计数。

**Tests**
- `go test ./backend/internal/integration/... -run 'TestSystemRepo'`

**Depends** T1

## T3 类别服务与 HTTP 接口

- [x] 已完成：类别 CRUD、内置保护、10 类上限、引用阻断；`TestSystemCategories` 6 个用例通过

**Files**
- `backend/internal/service/system.go`（新增：类别部分）
- `backend/internal/httpapi/system.go`（新增：类别 handler + DTO）
- `backend/internal/httpapi/router.go`（注册路由）
- `backend/internal/integration/system_test.go`（新增）

**Steps**
1. `GET /system-categories`：`work_id` 必填；`include_disabled` 默认 true；组装 `node_count`、`reference_count`。
2. `POST /system-categories`：名称规范化、类型 `normal|era`、上限 10、内置标记固定 false、`sort_order=max+1`。
3. `PATCH /system-categories/{id}`：`If-Match`；内置保护；停用前引用检查；改名唯一；`category_type` 出现即 422。
4. `DELETE /system-categories/{id}`：`If-Match`；内置保护；引用检查；整组软删除（节点 + 类别）、`undo_tokens`、返回 `deleted/undo_token/affected_nodes`。
5. 统一 `details` 错误结构与 `X-Requested-With` 校验。

**Expected**
- 内置保护、10 类上限、`NAME_CONFLICT`、`REFERENCE_CONFLICT`、`VERSION_CONFLICT` 均可通过 API 复现。
- 删除类别后类别与节点同时 `deleted_at` 非空且 `deleted_root_id=category_id`。

**Tests**
- `go test ./backend/internal/integration/... -run 'TestSystemCategories'`

**Depends** T2

## T4 节点服务与 HTTP 接口

- [x] 已完成：树/平铺、两级、同级唯一、纪元、排序、停用只读、review 映射；`TestSystemNodes` 6 个用例通过

**Files**
- `backend/internal/service/system.go`（节点部分）
- `backend/internal/httpapi/system.go`（节点 handler + DTO）
- `backend/internal/httpapi/router.go`
- `backend/internal/integration/system_test.go`

**Steps**
1. `GET /system-categories/{id}/nodes`：默认树形、`flat=true` 平铺、`review_status` 过滤；带 `reference_count` 与 review 字段（API `confirmed`/`pending`）。
2. `POST /system-nodes`：类别启用校验、父级同类别且为根（两级）、era 字段仅 era 类别、同级唯一、`sort_order` 追加。
3. `PATCH /system-nodes/{id}`：`If-Match`；名称/`sort_order`/era 文本；`parent_id` 与 `category_type` 不可改。
4. `DELETE /system-nodes/{id}`：`If-Match`；子节点存在 → `HAS_CHILDREN`；引用 → `REFERENCE_CONFLICT`；软删除 + undo token。
5. `POST /system-nodes/reorder`：同父级完整集合、无重复；事务内逐项更新；`204`。

**Expected**
- 两级限制、同父重名、纪元字段规则、排序持久、版本冲突全部可复现。
- 停用类别下写操作被 `CATEGORY_DISABLED` 拒绝。

**Tests**
- `go test ./backend/internal/integration/... -run 'TestSystemNodes'`

**Depends** T3

## T5 引用检查接线

- [x] 已完成：停用/删除/永久删除统一走引用检查；作品永久删除按人物去重阻断；`TestSystemRefs` 2 个用例通过

**Files**
- `backend/internal/repository/system.go`
- `backend/internal/service/system.go`
- `backend/internal/model/errors.go`（如需补 `REFERENCE_CONFLICT` details 结构）
- `backend/internal/integration/system_refs_test.go`（新增）

**Steps**
1. 停用类别、删除类别、删除节点、永久删除类别/节点统一调用引用检查。
2. 生成 `details`：`type=character_system_ref`、`entity_id=character_id`、`field=node|category`、`value=节点/类别名`、`name=null`、`in_trash=false`（0003 限制，CHG-004 补真实值）。
3. 集成测试直接插入 `character_system_refs`（不建人物），事务回滚。

**Expected**
- 有引用时停用/删除/永久删除均 `409 REFERENCE_CONFLICT`，`details` 条数等于去重人物数。
- 清理引用后同一操作成功。

**Tests**
- `go test ./backend/internal/integration/... -run 'TestSystemRefs'`

**Depends** T4

## T6 回收站扩展与作品级联收口

- [x] 已完成：三类根列表/恢复/撤销/永久删除/清空与作品级联；`TestSystemTrash` 7 个用例与 `TestTrash` 回归通过

**Files**
- `backend/internal/service/trash.go`
- `backend/internal/httpapi/trash.go`（如需放宽 root_type 校验）
- `backend/internal/repository/works.go`
- `backend/internal/integration/trash_system_test.go`（新增）
- `backend/internal/integration/trash_test.go`（更新受影响断言）

**Steps**
1. 列表支持 `root_type=system_category|system_node` 与空值聚合；`parent_name`、`affected_count`、时间排序、分页。
2. 恢复：类别整组（`new_name`/`renames[]`、单事务、全量预检）、节点（类别/父级缺失 → `PARENT_DELETED`、`new_name`）。
3. 永久删除：快照 → 引用检查 → 引用/节点/类别/作品顺序物理删除；清空回收站先全量引用检查再快照。
4. 撤销：类别整组、节点单项；保留 8 秒窗口。
5. `repository/works.go`：作品软删除/恢复包含 `system_nodes`；影响条数 = 类别 + 节点；永久删除先删引用与节点再删类别/作品。

**Expected**
- 回收站可分别列出三类根；类别只出现一条；节点单独一条。
- 整组恢复原子；命名冲突 422 且无半恢复；节点父级缺失 409。
- 删除作品的节点随组进入回收站，恢复/永久删除均无 FK 残留。

**Tests**
- `go test ./backend/internal/integration/... -run 'TestTrashSystem|TestTrash'`

**Depends** T5

## T7 前端 API 层与文案

- [x] 已完成：`lib/api/system.ts` 9 个函数 + 11 个单测；`npm run test`/`typecheck`/`lint` 通过

**Files**
- `frontend/lib/api/system.ts`（新增）
- `frontend/lib/messages.ts`
- `frontend/lib/api/system.test.ts`（新增）

**Steps**
1. 定义 `SystemCategory`、`SystemNode`、引用明细、回收站 DTO 类型。
2. 封装类别/节点的 list/create/update/delete/reorder；更新与删除传 `If-Match`；写请求经现有 `request` 带 `X-Requested-With`。
3. 补错误码文案与页面标签（类别、节点、内置、必开、停用、纪元、待校对等）。
4. 单测 URL、请求体、版本头与错误码透传。

**Expected**
- `npm run test -- lib/api/system.test.ts` 通过；错误对象保留 `code`/`details`。

**Tests**
- `npm run test`（前端单测）

**Depends** T4（接口形状定稿即可并行）

## T8 页面骨架、类别列表与详情

- [x] 已完成：`/system` 路由、左栏 280/240px、内置置顶、10/10 禁用、详情与状态、导航入口；`test`/`lint`/`build` 通过

**Files**
- `frontend/app/system/page.tsx`（新增）
- `frontend/components/system/system-page.tsx`（新增）
- `frontend/components/system/category-list.tsx`（新增）
- `frontend/components/system/category-detail.tsx`（新增）
- `frontend/components/shell/global-nav.tsx`
- 相关单测/快照（如现有组件测试目录）

**Steps**
1. 路由 + 当前作品读取；无作品空状态。
2. 左栏 280px/240px；内置置顶与 Badge；类型/状态/节点数/引用数；选中态；10/10 禁用与说明。
3. 右栏头部、静态说明、启停状态、待校对筛选占位、节点区容器、新增主层级按钮。
4. 加载/空/错误/网络状态；导航加入「体系」。
5. 1024 布局与 `listbox` 语义。

**Expected**
- 内置境界不可见改名/停用/删除；10/10 时新增按钮禁用；1024 无横向溢出。

**Tests**
- `npm run test`
- `npm run lint`

**Depends** T7

## T9 类别操作与引用弹窗

- [x] 已完成：新增/改名/停用/启用/删除整组弹窗、引用阻断按节点名分组、撤销 Toast；`test`/`typecheck`/`lint`/`build` 通过

**Files**
- `frontend/components/system/system-dialogs.tsx`（新增）
- `frontend/components/system/reference-dialog.tsx`（新增）
- `frontend/components/system/system-page.tsx`
- `frontend/lib/messages.ts`

**Steps**
1. 新增类别（名称 + 普通/纪元）、改名、停用确认、启用、删除整组确认。
2. 409 引用 → ReferenceDialog 按节点名分组；409 名称冲突 → 字段级错误；版本冲突 → 刷新提示。
3. 成功 Toast 与列表/详情刷新；保留表单输入直到成功或取消。

**Expected**
- 类别全生命周期在 UI 可完成；错误可定位、无英文堆栈。

**Tests**
- `npm run test`

**Depends** T8

## T10 节点树与节点操作

**Files**
- `frontend/components/system/node-tree.tsx`（新增）
- `frontend/components/system/system-dialogs.tsx`
- `frontend/components/system/category-detail.tsx`
- `frontend/lib/messages.ts`

**Steps**
1. `role=tree/treeitem`、两层展开、引用数、待校对 Badge、上移/下移（首末禁用）。
2. 新增主层级/子层级、改名、删除；子节点不提供新增子层级；父节点删除 → HAS_CHILDREN 清单。
3. 纪元类别显示起止文本输入（可空、无格式校验）；普通类别隐藏。
4. 停用类别只读；删除节点成功 Toast 与撤销入口（对接现有 Toast 撤销组件）。
5. `review_status=pending` 筛选与空结果文案。

**Expected**
- 两级结构、排序即时持久、纪元字段、停用只读、键盘操作符合 PG-10。

**Tests**
- `npm run test`
- `npm run lint`

**Depends** T9

## T11 E2E、视觉基线、键盘与可访问性

**Files**
- `frontend/e2e/system.spec.ts`（新增）
- `frontend/e2e/shell.spec.ts`（更新导航断言）
- `frontend/e2e/a11y.spec.ts`（新增 `/system`）
- `frontend/e2e/keyboard.spec.ts`（新增体系键盘用例）
- `frontend/e2e/visual.spec.ts`（新增 system 1280/1024 基线）
- 基线图文件

**Steps**
1. 覆盖：内置保护、类别 CRUD、10 上限、两级、同级重名、排序持久、停用只读、引用阻断（API 播种 `character_system_refs` 后清理）、整组删除/恢复/`new_name`、节点删除/撤销/父级缺失、1024、axe。
2. 数据使用时间戳前缀；结束精确删除回收站根，不清空全局回收站。
3. 更新导航与既有基线；如布局变化导致既有截图差异，仅更新受影响文件。

**Expected**
- 新增用例全部通过；axe 无 serious/critical；视觉差异 ≤ 0.02。

**Tests**
- `npx playwright test e2e/system.spec.ts e2e/shell.spec.ts e2e/a11y.spec.ts e2e/keyboard.spec.ts e2e/visual.spec.ts`

**Depends** T10

## T12 全量门禁与文档同步

**Files**
- `changes/CHG-003-systems-nodes/verification.md`（F3 输出，F2 结束时建骨架）
- `project/PROJECT-SETUP.md`（§11 迁移与测试）
- `project/page-specs/PG-10-system-config.md`（实施差异）
- `project/specs/systems/specs.md`（F10 再改状态）

**Steps**
1. 运行 `scripts/ci.ps1`：格式化、lint、Go 单测/集成、前端单测、构建、E2E、视觉。
2. 容器内核对 `0003` 已应用、三表存在、内置境界唯一。
3. 记录真实命令/结果/差异到 `verification.md`。
4. 同步实施差异（导入与校对确认延后 CHG-011；人物联动延后 CHG-004）。
5. 不提前把 Spec 标为 verified；留 F10。

**Expected**
- 全量门禁通过；文档无冲突；工作区干净准备 F3/F4。

**Tests**
- `powershell -NoProfile -File scripts/ci.ps1`

**Depends** T11
