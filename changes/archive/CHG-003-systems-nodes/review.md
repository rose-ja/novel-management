# CHG-003 体系类别与节点 — 代码审查报告

- 分支：`feat/chg-003-systems-nodes`
- 范围：`git diff 3c2c10a^..HEAD`（12 个提交）
- 审查日期：2026-11-09
- 审查方式：只读静态审查，未修改代码；未本地重跑测试，测试结论引用 `verification.md` 与现有用例源码。

## 1. 审查范围与方法

### 1.1 命令

```powershell
git log --oneline 3c2c10a^..HEAD
git rev-list --count 3c2c10a^..HEAD
git diff --stat 3c2c10a^..HEAD
git diff --name-only 3c2c10a^..HEAD
git diff 3c2c10a^..HEAD -- backend/internal/migrations backend/internal/model backend/internal/repository backend/internal/service backend/internal/httpapi
git diff 3c2c10a^..HEAD -- frontend/lib frontend/components/system frontend/app/system frontend/e2e
```

随后按 `文件:行号` 直读当前文件，交叉核对实现与测试。

### 1.2 基线文档

- `changes/archive/CHG-003-systems-nodes/proposal.md`、`design.md`、`tasks.md`、`checklist.md`、`verification.md`
- `project/specs/systems/specs.md`（SPEC-SYS-001～005）
- `project/page-specs/PG-10-system-config.md`
- `project/architecture/api.md`、`project/architecture/data-model.md`（体系章节）
- `project/requirements/requirements.md`、`project/user-stories/user-stories.md`（体系相关条目）

### 1.3 重点代码

- 迁移：`backend/internal/migrations/0003_system_nodes.sql`
- 模型：`backend/internal/model/system.go`、`model/errors.go`、`model/work.go`
- 仓储：`repository/system.go`、`repository/works.go`
- 服务：`service/system.go`、`service/trash.go`
- HTTP：`httpapi/system.go`、`httpapi/trash.go`、`httpapi/request.go`
- 前端：`frontend/lib/api/system.ts`、`frontend/components/system/*`、`frontend/app/system/page.tsx`、`frontend/e2e/system.spec.ts`
- 回收站前端（未在本次 diff 中，但被本次新增根类型影响）：`frontend/app/trash/page.tsx`、`frontend/components/trash/trash-dialogs.tsx`

## 2. 结论

- F4 初审：**FAIL**。S0：0；S1：3；S2：6。
- F5 复审（独立 Reviewer，2026-11-09）：**PASS**。S1 清零，无新增回归；CR-07 接受为已知性能项。详见第 6 节。

## 3. 问题清单

| ID | 级别 | 文件:行 | 关联 Spec/需求 | 触发条件 | 实际影响 | 修复建议 | 验证建议 |
|---|---|---|---|---|---|---|---|
| CR-01 | S1 | `frontend/app/trash/page.tsx:29-32`、`:88-93`、`:102`、`:111`；对照 `backend/internal/service/trash.go:112-148` | proposal §3.1；SPEC-SYS-005（回收站恢复/永久删除）；tasks T6「回收站可分别列出三类根」 | 删除任一体系类别或节点后打开 `/trash` | 后端已返回 `system_category`/`system_node` 根，但页面把类型固定渲染为「作品」、名称固定加《》、父作品列固定「—」，类型筛选只有 `all/work`。用户无法区分/筛选类别与节点，且恢复/永久删除入口的语义提示错误；proposal 的回收站能力未真正交付 | 在回收站页按 `root_type` 映射类型 Badge 与文案（`messages.systemCategory/systemNode` 已存在），展示 `parent_name`，筛选增加 `system_category`/`system_node`；恢复/删除对话框按根类型组织文案 | 新增 E2E：删除类别/节点 → `/trash` 断言类型 Badge、名称、父级、筛选；对类别/节点分别执行恢复与永久删除 |
| CR-02 | S1 | `backend/internal/service/trash.go:262`、`:605`；`backend/internal/repository/works.go:241-250`；design §7.2（`design.md:263`） | design §7.2「system_category 恢复要求作品必须在回收站外，否则 PARENT_DELETED」；SPEC-SYS-005 | 先单独删除类别，再把其作品移入回收站，然后在回收站恢复该类别（或 8s 内撤销） | `Works.Name` 查询不过滤 `deleted_at`，只判断「作品行存在」，因此恢复校验恒通过。类别会在作品仍在回收站时被恢复为 active，形成「active 类别挂在已删作品下」；随后永久删除该作品时 `PurgeGroup` 会按 `work_id` 物理删除该 active 类别及其节点，用户在界面上看不到却遭永久删除 | 恢复/撤销校验改用只返回 active 作品的查询（如 `Works.Get` 或显式 `deleted_at IS NULL`），失败返回 `PARENT_DELETED`；恢复事务内再校验一次 | 集成测试：删类别→删作品→恢复类别应 409 `PARENT_DELETED`；作品恢复后再恢复类别成功；补充「作品在回收站时不允许恢复类别」的 E2E |
| CR-03 | S1 | `backend/internal/repository/system.go:543-546`；`backend/internal/migrations/0003_system_nodes.sql:28`；`backend/internal/service/system.go:509-515`；`backend/internal/service/trash.go:494-563` | design §7.3 节点永久删除；SPEC-SYS-005 | 先删除子节点（进回收站），再删除其父节点（父节点删除时子节点已软删，`CountNodeChildren` 只统计未删子节点而放行），然后永久删除父节点，或在回收站「清空」 | `PurgeNode` 只 `DELETE FROM system_nodes WHERE id=?`，但已软删子节点行仍带 `parent_id`，命中 `ON DELETE RESTRICT`。单删父节点返回 500；清空回收站时节点根按 `deleted_at DESC` 先删父后删子，整笔事务回滚，导致「清空回收站」不可用 | 二选一：① `PurgeNode` 前检查 `parent_id=?`（含已软删行），存在则 409 `HAS_CHILDREN`；② 同一事务内先物理删除该节点的已软删后代（并按后代做引用检查）；`PurgeAll` 将节点根改为 `deleted_at ASC`（先子后父）或在检测到父子根时拒绝并提示 | 集成测试：子删→父删→永久删父，断言不出现 500；再测清空回收站含父子根可成功且无 FK 残留 |
| CR-04 | S2 | `backend/internal/service/trash.go:298-332`（尤其 `NodeNameTaken` 与 `renameKeys` 预检）；`repository/system.go:242-260`；`migrations/0003_system_nodes.sql:55-57` | design §7.2 / AC-11「冲突节点提供 `renames[]` 后整组原子恢复」；SPEC-SYS-005 | 整组恢复类别时，`renames[]` 将某节点改成**同组未被改名兄弟节点**的名字 | 预检只对 active 行查 `NodeNameTaken`，并只比较「被改名的节点之间」；同组其他节点均已软删，不在检查范围。写入阶段 `RestoreCategoryGroup` 一次性清除全部 `deleted_at`，部分唯一索引触发唯一冲突，事务回滚并返回 500 `INTERNAL_ERROR`，而不是 422 `RESTORE_CONFLICT` 及冲突清单 | 在写入前构建「最终名称」映射：对每个 sibling group 合并改名结果并检测重复；或对每个改名节点同时比对同组未改名节点的 `name_key`。保持现有事务原子性 | 集成测试：类别含根 A/B，`renames=[{B→A}]` 应返回 422 `RESTORE_CONFLICT`（`details.items` 含 A/B），且类别/节点均未恢复 |
| CR-05 | S2 | `backend/internal/httpapi/request.go:92-93`；`backend/internal/service/system.go:513-514`；`frontend/components/system/system-dialogs.tsx:416-433`、`:796-800`；SPEC-SYS-003:56 | SPEC-SYS-003 / AC-10「409 HAS_CHILDREN 并列出子节点」 | 在「待校对」筛选（`flat` 平铺，非 pending 子节点未返回）下删除有子节点的父节点；或 API 客户端直接消费 409 | 后端 `abortError` 不携带 `details`，前端只能回退到本地 `node.children`。默认树视图可列出；筛选视图子节点被过滤时回退为空，对话框无法列出实际子节点（仅提示文案），未满足「列出子节点」。API 消费方也拿不到子节点清单 | 后端用 `abortErrorWithDetails` 返回子节点摘要（至少 id/name），或增加按 `parent_id` 查询子节点的接口供回退；前端两种 details 形状保留 | 集成测试断言 `details.items` 含子节点名；E2E 在待校对筛选下删除父节点并断言列出非 pending 子节点 |
| CR-06 | S2 | `backend/internal/service/system.go:231-241`、`:516-523`；对照 `design.md:294` | design §9「类别删除：事务内引用检查」「节点删除：事务内引用检查」 | 删除类别/节点或停用类别时，引用检查完成后、软删除事务提交前发生并发引用写入 | 当前 `assertCategoryUnreferenced`/`assertNodeUnreferenced` 使用 `s.db` 在事务外执行，存在 TOCTOU 窗口；CHG-004 引入引用写入后，可能在检查通过后写入引用并成功删除被引用类别/节点。`PurgeAll` 的引用预检同样在事务外 | 把引用检查移入删除事务内（使用 `tx` 仓储），或对相关行加锁；`PurgeAll` 在事务内复核 | 并发测试/代码审查：引用检查与写操作共用同一 transaction；CHG-004 写引用后回归引用阻断 |
| CR-07 | S2 | `backend/internal/service/trash.go:103-110`、`:122-148` | 性能（N+1、全表聚合）；design §7.1 | 打开回收站（任意 `root_type` 或分页） | 对每个类别根执行一次 `Works.Name`、对每个节点根执行一次 `CategoryName`（N+1）；且无论 `root_type` 是否限定，都执行 `CountDeletedByRoot` 与 `CountDeletedNodesByCategoryRoot` 两条全表聚合；`PurgeAll` 引用检查也按根逐个查询 | 批量收集 work/category id 后各查一次名称；仅在需要的工作/类别根集合上做聚合；`PurgeAll` 引用查询按根类型批量查询 | 在回收站含较多类别/节点时比对 SQL 次数；为 `root_type` 过滤路径断言不执行无关聚合计 |
| CR-08 | S2 | 测试缺口：`backend/internal/integration/system_trash_test.go:247-306`（永久删除仅叶子节点）、`:147-205`（恢复路径，未覆盖作品在回收站时恢复类别）、`:180-205`（重名仅覆盖 active 冲突与「两个改名互相冲突」）；`frontend/e2e/system.spec.ts`（无回收站系统根用例、无父节点删除阻断用例） | verification.md「E2E 14/14、集成全绿」；AC-10/AC-11/AC-12 | 运行现有测试 | 现有用例无法暴露 CR-02/CR-03/CR-04/CR-05：父节点永久删除、作品在回收站时恢复类别、改名撞未改名兄弟、HAS_CHILDREN details 均无断言；回收站前端对系统类型无 E2E | 补齐上述 4 类集成测试与 1 条回收站 UI E2E；断言错误码与 `details`，而非只断言状态 | `go test ./...`、Playwright 全量；先以新增用例复现问题，再修复 |
| CR-09 | S2 | `checklist.md:27`、`proposal.md` AC-6（若写 409）；实现 `backend/internal/httpapi/request.go:84-87` | 需求/验收文档一致性；SPEC-SYS-003 边界 | 创建第三级节点 | 验收条目写「409 NODE_LEVEL_LIMIT」，但实现与同一条 checklist 证据、E2E 均为 422 `UnprocessableEntity`。文档自相矛盾，易导致后续客户端按 409 处理 | 统一为 422 并修订 proposal/checklist 文案，或代码改为 409；建议保持 422（校验错误）并同步文档 | 文档一致性检查；接口契约测试锁定状态码 |

## 4. 已验证无问题的关键点

1. **软删除三件套与 `deleted_root_id` 主干路径**：单独删除节点（root=node）后再整组删除类别（root=category）时，组删除通过 `deleted_at IS NULL` 跳过已删节点，回收站显示类别根 + 独立节点根；恢复类别只恢复 `deleted_root=category` 的节点，独立节点保留，随后可恢复（集成 `TestSystemTrashRestoreNode` 覆盖）。除 CR-03 的永久删除 FK 与 CR-04 的改名预检外，恢复顺序自洽。
2. **PurgeAll 顺序/事务/计数语义**：引用检查在快照前、`Purged` 为「根数量」（works+categories+nodes），与列表行语义一致；每个根在独立事务内删除，外层事务保证整体回滚。CR-03 是节点父子的删除顺序问题，不是计数错误。
3. **恢复冲突预检先于写入**：`restoreCategory`/`restoreNode` 的冲突检查均在事务内、写入前完成；任一失败整体回滚，未发现「部分恢复」。CR-04 是预检覆盖不全，不是写入顺序问题。
4. **引用 SQL 的作品隔离**：按 `category_id`/`node_id` 查询，二者全局唯一；`referenceCountsByCategory` 带 `work_id` 条件。未发现跨作品误判。`name`/`in_trash` 恒为 `null/false` 属 design §5 明确延后 CHG-004，不重复报缺陷。
5. **前端契约一致**：`updateNode` 纪元文本清空发送 `era_start_text:null`，后端 `parseOptionalString` 正确识别为 clear；`reorderNodes` 根层省略 `parent_id`，后端 nil 即根层；停用类别 `readOnly` 隐藏所有写入口；待校对筛选下仅禁用排序按钮，与 PG-10 一致。
6. **树/排序算法**：`AssembleNodeTree` 按 `parent_id` 组装，空集合返回非 nil 空切片；`SortSystemNodes` 稳定排序并以 id 兜底；`ValidateSiblingOrder` 对空集合、缺失、重复、未知 id 均有处理；待校对走平铺路径，不会因缺失父节点触发树组装错误。
7. **迁移 0003**：`category_type` CHECK、类别/节点 sibling 部分唯一索引与代码 `name_key` 计算一致；删除一致性 CHECK（`0003_system_nodes.sql:19-22,49-52`）实际只约束「active 时三列全空 / deleted 时 `deleted_root_id` 非空」，代码写入 `deleted_by_action` 不违反约束；Down 删除新表/新索引/新 CHECK 并保留 0002 表与数据，`TestMigrationRoundTrip` 覆盖。
8. **HTTP 字段存在性**：`parent_id`（含显式 `null`）与 `category_type` 一律 422；名称空串/null 拒绝；`sort_order` 非整数拒绝；错误码整体映射与 design 一致（CR-09 单独列出状态码文档冲突）。
9. **前端 details 解析**：`ReferenceItem` 兼容数组与 `{items}` 两种形状；`HAS_CHILDREN` 的本地树回退在默认树视图可用（筛选视图缺口见 CR-05）；E2E/键盘/axe 用例未发现假通过断言或测试间数据依赖。
10. **E2E 清理**：`cleanupE2E` 按 E2E 前缀清理作品与引用，单 worker 串行执行，未发现测试互相污染；缺口仅是覆盖范围（CR-08）。

## 5. 未覆盖/需人工判定项

- **类别永久删除会连带删除「单独删除的节点根」**：`PurgeCategoryGroup` 按 `category_id` 删除全部节点，包含此前单独删除、在回收站单独展示的节点根。按 design §7.3「类别永久删除顺序：节点→类别」这属于预期行为，但用户会在回收站看到两个根，删除类别后节点根一并消失。是否需要单独提示，需产品确认。
- **类别上限竞态**：`CreateCategory` 的计数与插入未在同一事务（design §9 要求事务内 COUNT+插入），但 design/已知限制已将其归为单用户本地竞态，本报告不作为缺陷重复上报。
- **`in_trash` 恒 false、人物名恒 null**：设计明确延后 CHG-004，不作为本轮缺陷。
- 本次未本地执行 `go test ./...` 与 Playwright；如需以运行结果作为门禁，请按 `verification.md` 命令重跑，并补上 CR-08 指出的用例。

## 6. F5 修复记录（作者自述，待独立 Reviewer 复审）

修复范围：CR-01～CR-06、CR-08、CR-09；CR-07 为性能类 S2，本轮按「不阻塞 F6、记录为已知性能项」处理。

| ID | 根因 | 修复 | 影响文件 | 证据 |
|---|---|---|---|---|
| CR-01 | 回收站前端把根类型固定为「作品」，名称固定加《》，父级列固定「—」，筛选仅 `all/work` | 按 `root_type` 渲染 Badge 与名称（系统根不加书名号）、展示 `parent_name`、增加 `system_category`/`system_node` 筛选并走服务端 `root_type`；恢复/永久删除对话框文案按根类型变化 | `frontend/app/trash/page.tsx`、`frontend/components/trash/trash-dialogs.tsx`、`frontend/lib/messages.ts` | 新增 `frontend/e2e/trash-system.spec.ts` 2 用例（类别整组显示/筛选/恢复、节点显示/筛选/永久删除）；全量 Playwright 52 passed |
| CR-02 | `Works.Name` 不过滤 `deleted_at`，作品在回收站时仍判定存在，类别被恢复为 active 挂在已删作品下 | 新增 `repository.Works.ActiveName`（`deleted_at IS NULL`）；`restoreCategory` 与 `Undo` 的 system_category 分支改用它，失败映射 `PARENT_DELETED` | `backend/internal/repository/works.go`、`backend/internal/service/trash.go` | 新增 `TestSystemTrashCategoryRequiresActiveWork` |
| CR-03 | `PurgeNode` 只按 id 删除，已软删子节点行仍带 `parent_id` 命中 FK RESTRICT；`PurgeAll` 按 `deleted_at DESC` 先父后子 | `PurgeNode` 前用 `ListNodeChildren`（含软删）检查并返回 `HAS_CHILDREN`；`PurgeAll` 用 `orderNodeRootsForPurge` 拓扑排序保证子先父后；`PurgeCategoryGroup` 先删 `parent_id IS NOT NULL` 子行、再删根行、最后删类别 | `backend/internal/repository/system.go`、`backend/internal/service/trash.go` | 新增 `TestSystemTrashPurgeNodeWithDeletedChild`、`TestSystemTrashPurgeAllParentChild`、`TestSystemTrashPurgeCategoryWithTwoLevels` |
| CR-04 | 恢复预检只比较「被改名节点之间」，未覆盖改名撞同组未改名兄弟或活跃节点 | `restoreCategory` 构建每个节点的最终名称（改名优先，否则原名），按 `siblingGroupKey` 检测同组重复，并对每个节点用最终 key 调 `NodeNameTaken` 与活跃节点比对；冲突统一返回 `RESTORE_CONFLICT` | `backend/internal/service/trash.go` | 新增 `TestSystemTrashRestoreCategoryRenamedCollidesSibling`（422 `RESTORE_CONFLICT`，类别/节点均未恢复） |
| CR-05 | `abortError` 不携带 `details`，前端在筛选视图无法列出子节点 | 新增 `model.HasChildrenError{Items}`，`httpapi/request.go` 映射为 409 `HAS_CHILDREN`，`details` 为 `{"items":[{id,name,entity_type:"system_node"}]}` | `backend/internal/model/errors.go`、`backend/internal/httpapi/request.go`、`backend/internal/service/system.go` | `TestSystemNodesDelete` 增加 `details.items` 断言 |
| CR-06 | 引用检查在事务外执行，存在 TOCTOU | `UpdateCategory`、`DeleteCategory`、`DeleteNode` 改为 `Transaction` 内用 `repository.NewSystem(tx)` 完成读取、引用检查与写入；`Purge`/`PurgeAll` 引用检查移入事务内复核 | `backend/internal/service/system.go`、`backend/internal/service/trash.go` | `go vet`、单测、集成全绿；并发窗口消除（同一事务） |
| CR-08 | 测试缺口：父节点永久删除、作品在回收站时恢复类别、改名撞未改名兄弟、`HAS_CHILDREN` details、回收站系统根 UI | 补齐 5 个后端集成用例 + 2 个回收站 E2E | `backend/internal/integration/system_trash_test.go`、`frontend/e2e/trash-system.spec.ts` | 集成 `ok 10.123s`；E2E 52 passed |
| CR-09 | 文档写 409 `NODE_LEVEL_LIMIT`，实现与 E2E 为 422 | `checklist.md` ND-02/ND-03、`proposal.md` AC-6 统一为 422 | `checklist.md`、`proposal.md` | 文档一致性检查通过 |

**CR-07 处理决定**：N+1 与全表聚合属性能类 S2，不影响正确性与契约；本轮保留并记录为已知性能项，待后续性能专项（或回收站数据量增长后）再优化。接受理由：修复会改动列表查询结构与分页语义，超出 F5「修复审查缺陷、不打乱已验收契约」的范围。

**复审状态**：已完成。独立 Reviewer 只读检查 F5 Diff 与受影响区域，逐项复核 CR-01～CR-09：

| ID | 复审状态 | 说明 |
|---|---|---|
| CR-01 | 已修复 | 回收站类型/名称/父级/筛选/文案按 `root_type` 渲染；E2E 2 条实跑通过 |
| CR-02 | 已修复 | `ActiveName` 只取 active 作品；恢复与 Undo 两路径均映射 `PARENT_DELETED` |
| CR-03 | 已修复 | 含软删子节点阻断、子先父后、类别组先子行后根行；3 个集成用例通过 |
| CR-04 | 已修复 | 最终名称同级去重 + 活跃节点比对，写入前完成、事务回滚 |
| CR-05 | 已修复 | `details.items` 形状一致，前端兼容 `{items}` 与数组并有本地回退 |
| CR-06 | 已修复 | 引用检查与写入同事务；未加行锁的单用户窗口接受 |
| CR-07 | 接受 | 性能类 S2，不影响正确性 |
| CR-08 | 已修复 | 5 个集成用例 + 2 条 E2E，断言错误码/details/未恢复状态 |
| CR-09 | 已修复 | `checklist.md`、`proposal.md` 已 422；复审发现的 `project/api-contract.md:102-103` 残留已同步为 422 |

**最终结论：PASS**（S1 清零、无新增回归）。遗留非阻塞项：待校对筛选下 `HAS_CHILDREN` 的 UI E2E 未补；工作级 `PurgeGroup` 未纳入父子顺序专项测试。
