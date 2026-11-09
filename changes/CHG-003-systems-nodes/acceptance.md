# Acceptance — CHG-003 体系类别与节点

- 变更目录：`changes/CHG-003-systems-nodes/`
- 分支：`feat/chg-003-systems-nodes`
- 验收人：用户（本机验收）
- 验收日期：2026-11-09
- 结论：**ACCEPT**
- 关联：`proposal.md`、`checklist.md`、`verification.md`（§8 F5、§9 F6）、`review.md`

## 1. 前置条件

- `docker compose up -d`，`app`/`db` 均 healthy；应用入口 `http://127.0.0.1:17860`。
- 数据库已应用迁移 0003（`goose_db_version` version=3，`system_categories`/`system_nodes`/`character_system_refs` 存在）。
- 当前无作品、回收站为空（F6 验证后状态），从零开始验收。
- 测试数据统一使用「验收-」前缀；验收结束后删除作品并清空回收站。
- 如需重跑自动化门禁：`cd frontend; npx playwright test`（52 条）；后端集成见 `verification.md` §8.2。

## 2. 验收脚本（按顺序执行）

| # | 操作步骤 | 预期结果 | 对应 AC/UI | 证据位置 |
|---|---|---|---|---|
| 1 | 顶栏新建作品「验收-体系」并进入「资料 > 体系」 | 自动成为当前作品；`/system` 左栏仅内置「境界」，标记「内置 · 必开」 | AC-01、UI-01、UI-02 | `verification.md` §9.2；`system.spec.ts` 内置用例 |
| 2 | 对「境界」尝试改名/停用/删除（页面应无入口，可用 API/curl 验证） | 页面无入口；API 均返回 409 `CATEGORY_BUILTIN_IMMUTABLE` | AC-02 | `checklist.md` AC-02 |
| 3 | 新增类别「验收-圣体」类型「普通」 | 创建成功，`sort_order` 递增，显示「普通/启用/0 节点」 | AC-03、AC-09、UI-04 | `system.spec.ts` 类别 CRUD |
| 4 | 再新增「验收-圣体」及大小写/全半角变体（如「验收-ＳＡＮＴＩ」对应同名变体） | 返回/提示 `NAME_CONFLICT`，不产生重复类别 | AC-05 | 同上 |
| 5 | 停用「验收-圣体」，再重新启用 | 停用后节点只读、显示启用按钮；节点保留；启用后恢复可写 | AC-07、UI-03 | `system.spec.ts` 停用只读用例 |
| 6 | 在「验收-圣体」下新增主层级「验收-父」与子层级「验收-子」 | 两层 treeitem 正确渲染；子节点行无「新增子层级」入口 | ND-01、ND-02、UI-06 | `system.spec.ts` 两级用例 |
| 7 | 直接调用 API 在「验收-子」下建节点 | 422 `NODE_LEVEL_LIMIT` | ND-02 | `verification.md` §9.2 |
| 8 | 同级新增同名节点（含大小写/空白变体）；再在不同父级下建同名节点 | 同级冲突 `NAME_CONFLICT`；不同父级可同名 | ND-04 | `system.spec.ts` 同级重名用例 |
| 9 | 调整主层级顺序（上移/下移），刷新页面 | 顺序保持；首末项按钮禁用 | ND-05、UI-06 | `system.spec.ts` 排序用例 |
| 10 | 新增「纪元」类型类别，给节点填写起止文本；普通类别尝试填起止 | 纪元类别保存文本可空；普通类别提交非空 → 422 | ND-06 | `system.spec.ts` 纪元用例 |
| 11 | 删除仍有子节点的父节点「验收-父」 | 409 `HAS_CHILDREN`，弹窗列出子节点「验收-子」 | ND-07、UI-05 | `system.spec.ts` 删除阻断用例 |
| 12 | 删除类别「验收-圣体」 | 提示整组移入回收站；打开 `/trash` 显示「体系类别」Badge、名称不加《》、父级为「验收-体系」、影响条数 | TR-01、TR-02、UI-04 | `trash-system.spec.ts` 类别用例 |
| 13 | 在回收站按类型筛选「体系类别」，再恢复该类别 | 筛选只剩类别根；恢复成功且类别/节点回到 `/system` | TR-04、UI-04 | `trash-system.spec.ts` 类别用例 |
| 14 | 单独删除一个节点，打开回收站 | 出现「体系节点」Badge，父级为类别名；可筛选；可永久删除 | TR-05、TR-07 | `trash-system.spec.ts` 节点用例 |
| 15 | 删除类别后在 8 秒内点「撤销」；另建类别删除后等待超过 8 秒再撤销 | 8 秒内恢复成功；超时返回 `UNDO_EXPIRED` | TR-06 | `checklist.md` TR-06 |
| 16 | 恢复类别时制造节点重名（先建同名活跃节点），按提示提供 `new_name`/`renames[]`；不提供时先观察 | 不提供 → 422 `RESTORE_CONFLICT` 且无半恢复；提供后整组恢复 | TR-03、TR-04 | `verification.md` §8/§9 |
| 17 | 删除作品「验收-体系」，再打开 `/trash` 恢复作品 | 类别与节点随作品整组进出回收站，恢复后无孤儿 | WS-01、WS-02 | `checklist.md` WS-01/02 |
| 18 | 构造 10 类（含境界）后查看「新增类别」 | 10/10 时按钮禁用并提示「最多 10 类（含内置境界）」；API 第 11 类 409 `CATEGORY_LIMIT` | AC-04、UI-07 | `system.spec.ts` 上限用例 |
| 19 | 1024 宽度查看 `/system`、`/trash`；仅键盘遍历类别列表与节点树 | 无横向溢出；焦点可见、操作可达；axe 无 serious/critical | UI-08、UI-09 | `visual.spec.ts`、`keyboard.spec.ts`、`a11y.spec.ts` |
| 20 | 临时停止 app 后刷新 `/system`，再恢复 | 提示无法连接并可重试，无堆栈；恢复后重试成功 | UI-07 | `error-states.spec.ts` |
| 21 | 验收后清理：删除「验收-」作品并清空回收站 | works=0、trash=0、`current_work_id=null` | — | F6 清理记录 |

## 3. 验收范围核对

- 业务行为：类别生命周期（新增/改名/停用/启用/删除、上限、内置保护、类型不可变）、两级节点（新增/改名/删除/排序/纪元文本）、引用阻断骨架、类别整组与节点单项回收站/撤销/恢复。
- UI/UX：`/system` 类别列表与节点树、弹窗与字段级错误、`/trash` 三类根展示与筛选；与 PG-10 §0 阶段范围一致。
- 交互状态：加载/空/错误/成功、停用只读、10 类上限、名称/版本冲突、删除阻断、恢复冲突。
- 响应式：1280/1024 无横向溢出；视觉基线差异 ≤ 0.02。
- 权限与数据边界：写请求 `X-Requested-With` 守卫；类别/节点按 `work_id` 隔离；跨作品父级 422。
- 异常反馈：错误码集合与 `details` 形状符合 `project/api-contract.md`（含 `HAS_CHILDREN` items、`RESTORE_CONFLICT` items）。
- 回归：CHG-002 作品、回收站、设置、壳层、错误态、键盘、a11y 全量 52 条 E2E 通过。
- 测试证据可信度：集成用例事务回滚；E2E 使用 `E2E-` 前缀并自清理；F6 API 断言实跑（24/24）。
- 范围越界：无；人物表单、导入/校对、检索、三级节点、跨类移动按 checklist §7 延后。
- 安全/兼容性风险：无新增依赖；迁移 0003 可 Down；`character_system_refs` 为骨架，`character_id` 无外键（CHG-004 补）。
- 回滚可行性：迁移 `Down` 往返通过（`TestMigrationRoundTrip`）；回退提交即可；快照机制未改动。

## 4. 已知项（不阻塞验收）

| ID | 说明 |
|---|---|
| CR-07 | 回收站列表存在 N+1 与全表聚合，记录为性能类 S2，待后续性能专项优化 |
| F6-03 | `GET /favicon.ico` 404 为既有全局低危项，非本变更引入 |
| 测试缺口 | 待校对筛选下 `HAS_CHILDREN` 的 UI E2E 未补；工作级 `PurgeGroup` 未纳入父子顺序专项测试 |
| 契约观察 | `POST /system-nodes` 显式 `parent_id:null` 视为根节点（201）；`PATCH` 显式 null 返回 422（parent_id 不可改），符合 design §5 |

## 5. 验收人意见

> 验收人选择 **ACCEPT**（2026-11-09）。依据：F5 独立复审 PASS（S1 清零）、F6 全量 52 条 E2E 与 24 项 API 边界断言通过；接受 `acceptance.md` 脚本与 `verification.md` §8/§9 的证据，无返工项。

## 6. 归档摘要

- 已交付：类别生命周期（新增/改名/停用/启用/删除、上限 10 含内置、内置保护、类型不可变、名称唯一）、两级节点（新增/改名/删除/排序/纪元文本）、引用阻断骨架、类别整组与节点单项回收站/撤销/恢复、`/system` 页面与回收站三类根适配。
- 未交付（后续变更）：人物动态表单与引用写入（CHG-004）、导入模板与待校对确认（CHG-011）、检索（CHG-009）、三级节点/跨类移动/类别自定义排序（后续路线图）。
- 已知项：CR-07 回收站 N+1/全表聚合（性能类 S2）；`favicon.ico` 404（既有全局低危）；待校对筛选 `HAS_CHILDREN` UI E2E 与工作级 `PurgeGroup` 专项测试缺口。
- 下一变更依赖：CHG-004 依赖本轮「启用的类别 + 可选节点」与 `character_system_refs` 骨架。
