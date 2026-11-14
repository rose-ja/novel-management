# CHG-006 势力树、首领与成员

- Change ID：`CHG-006-factions-tree`
- 分支：`feat/chg-006-factions-tree`
- 目标：用单棵树维护势力层级，人物归属自动推导成员，首领为可选人物引用；删除遵守入向阻止矩阵，势力大事记复用现有协议。
- 关联 Spec：SPEC-FACTION-001～004；SPEC-CHAR-001（势力字段）、SPEC-CHAR-008（首领入向阻断）、SPEC-TRASH-004（永久删除）
- 关联需求：FR-09～FR-12、FR-24、FR-34～FR-38、FR-44、FR-47～FR-51、FR-66、FR-67
- 关联页面：PG-04 势力树/列表、PG-05 势力详情、PG-12 回收站（势力根）
- 关联契约：`api-contract.md` §2.3、§3.4、§3.8 大事记、§6.10 删除矩阵；`data-model.md` §3.4 `factions`、§3.2 `characters.faction_id`、§3.13 `chronicles`
- 依赖：CHG-004（人物）、CHG-005（大事记与缩略图）
- 复杂度：M
- 状态：`accepted`（F7 人工验收 ACCEPT，2026-11-13；待 F9 合并）

## 1. 用户价值

1. 一眼看清势力结构：单棵树表达「万族 → 族 → 分支」，默认展开两级，搜索可定位并展开祖先。
2. 关系不重复录入：成员由人物 `faction_id` 自动推导，可按「仅直属 / 含下级」切换；首领是 0..1 人物引用。
3. 安全维护：移动节点不能成环；删除被下级或成员阻止并列出引用位置；无引用时进回收站，8 秒可撤销。
4. 与人物档案闭环：人物详情/列表显示真实势力并可跳转；人物换势力后两侧成员即时更新。

## 2. 范围

### 2.1 In Scope

- 迁移 `0006_factions_chronicles`：新建 `factions`（父级、首领、软删除三件套、`version`、名称唯一、CHECK）；`characters.faction_id` 补外键与索引；放宽 `chronicles.owner_type` CHECK 允许 `faction`。
- 势力 CRUD：`GET/POST /api/v1/factions`、`GET/PATCH/DELETE /api/v1/factions/{id}`；名称同作品唯一（`lower(trim)`）、1–200 字、简介 ≤20,000。
- 树与列表：扁平全量返回（≤1000 节点），含 `parent`、`leader`、`children_count`、`member_count`、`review_status`、`version`；支持 `q`（命中 + 祖先）、`review_status`、`parent_id` 过滤。
- 首领：0..1 人物，必须是当前作品活跃人物；人物改名自动同步；首领人物在回收站时标注不可用。
- 成员推导：`GET /api/v1/factions/{id}/members?scope=direct|subtree&page&page_size`，子树递归去重、分页。
- 移动与并发：`PATCH` 携带 `If-Match`；移动到自身或后代返回 `CYCLE_DETECTED`；版本不一致 `VERSION_CONFLICT`。
- 删除与恢复：先检查下级（`HAS_CHILDREN`），再检查人物归属（`REFERENCE_CONFLICT`）；首领、势力大事记为出向随删隐藏；返回 8 秒 `undo_token`；回收站新增 `root_type=faction`（列表/恢复/永久删除/清空）；恢复时父级已删则 `PARENT_DELETED`。
- 势力大事记：复用 `chronicles` 与 `ChronicleSection`，`owner_type=faction`，CRUD/排序/2,000 字/`If-Match`/8 秒撤销与人物一致。
- 人物接入：`characters.faction_id` 接受真实势力（同作品、活跃）；人物列表/详情返回真实 `faction`；人物删除在“担任首领”时被阻断并列出势力。
- 生命周期：作品整组软删除/恢复/永久删除覆盖 `factions`；势力永久删除清理势力大事记；快照格式不变。
- 前端：`/factions` 树与内联新建、`/factions?id=` 详情（摘要条、下级、成员表、势力大事记、内联编辑、删除对话框）、人物势力选择与链接、回收站势力根、导航「资料 > 势力」。
- 测试：后端单元 + 集成（含迁移往返）、前端单测、E2E-03、视觉基线与 a11y。

### 2.2 Out of Scope（明确不做）

| 项目 | 承接/说明 |
|---|---|
| 地点-势力关系（`location_factions`）与地点页 | CHG-007；本轮详情 `relations.locations` 恒空数组 |
| 校对确认端点 `POST /factions/{id}/review` | CHG-011（已确认）；本轮只展示 `review_status`，势力默认 `reviewed` |
| 榜单席位、人物入向引用（榜单） | CHG-008 |
| 检索文档与全局检索命中势力 | CHG-009 |
| 势力引用体系 | FR-24 明确不做 |
| 多棵树、拖拽排序、职位/副首领、批量成员操作、势力图片 | 需求边界 |
| 导入/导出格式变更 | CHG-010/011；快照已按作品全量包含 |
| 树虚拟滚动优化 | 1000 节点先全量渲染；若实测卡顿再单独立项 |

## 3. 已确认决策

| ID | 决策 | 结论 | 理由 |
|---|---|---|---|
| D1 | 势力存储 | 新建 `factions` 表，自引用父级 FK，`characters.faction_id` 补 FK | FR-09 要求真实树；CHG-004 的 `faction_id` 无 FK 仅为占位 |
| D2 | 势力大事记 | 复用 `chronicles`，放宽 `owner_type` CHECK 为 `character/faction` | 已获用户确认；同一 CRUD、版本与撤销语义，避免重复实现 |
| D3 | 树接口形态 | `GET /factions` 返回扁平全量（≤1000），前端组树 | 已获用户确认；搜索/展开在前端即时完成，无分页复杂度 |
| D4 | 校对端点 | `POST /factions/{id}/review` 延后 CHG-011 | 已获用户确认；与人物保持一致 |
| D5 | 同级排序 | 不新增 `sort_order`，按 `name_key, id` 稳定排序 | PG-04 无排序控件；减少列与迁移面 |
| D6 | 名称规范化 | `name_key = lower(trim(name))`，活跃唯一索引 `(work_id, name_key)` | 与作品/人物一致；英文大小写视为重名 |
| D7 | 删除矩阵 | 下级（含软删子节点）+ 活跃成员人物阻止；首领、势力大事记出向随删 | requirements §6.10；软删子节点也阻止，避免恢复半棵树 |
| D8 | 人物删除新增阻断 | 人物担任任意势力首领时 `REFERENCE_CONFLICT` | FR-07 入向矩阵；CHG-004 检查器为空，本轮补齐 |
| D9 | 永久删除顺序 | 先在事务内删人物/大事记，再按深度倒序删势力行 | 自引用 FK `RESTRICT` 与 `characters.faction_id` FK 的约束顺序 |
| D10 | 恢复语义 | 父级不在线 → `PARENT_DELETED`；首领人物在回收站 → 恢复成功但标记不可用 | SPEC-FACTION-004、PG-05 RT-13 |

## 4. 允许与禁止的文件

### 4.1 允许修改

- 后端：`internal/migrations/0006_*.sql`、`internal/model/{faction,errors,media}.go`、`internal/repository/{factions,characters,chronicles,works}.go`、`internal/service/{factions,characters,chronicles,trash}.go`、`internal/httpapi/{factions,characters,chronicles,request,router}.go`、`cmd/server/main.go`、`internal/integration/*_test.go`。
- 前端：`app/factions/page.tsx`、`components/factions/*`、`components/characters/{character-detail-page,character-form-fields,character-filters,character-table}.tsx`、`components/characters/chronicle-section.tsx`（抽为通用 `components/chronicles/*`）、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`lib/api/{factions,characters,chronicles,trash}.ts`、`lib/messages.ts`、`styles/components.css`、`e2e/*`、相关单测。
- 文档：`changes/archive/CHG-006-factions-tree/*`（F1～F10）。

### 4.2 禁止修改

- 不改地点/榜单/检索/导入/校对/导出/快照格式与页面。
- 不改 CHG-004/005 已验收的人物文字字段语义、图片协议与大事记字段语义。
- 不新增第三方依赖、远程服务、搜索索引或后台任务。
- 不修改 `project/` 上游基线；F10 才回写 Spec/契约/数据模型。
- 不重构无关组件；`character-detail-page.tsx` 只做势力选择/展示与通用大事记接入。

## 5. 接口与数据影响

- 新增端点：`GET/POST /api/v1/factions`、`GET/PATCH/DELETE /api/v1/factions/{id}`、`GET /api/v1/factions/{id}/members`。
- 已有端点行为变化：`POST/PATCH /characters` 接受真实 `faction_id`；`GET /characters*` 返回真实 `faction`；`DELETE /characters/{id}` 新增首领入向阻断；`GET/POST /chronicles` 接受 `owner_type=faction`。
- 数据表新增 `factions`；`characters.faction_id` 增加 FK 与索引；`chronicles.owner_type` CHECK 扩为 `character/faction`。
- 错误码新增：`CYCLE_DETECTED`（422）；复用 `HAS_CHILDREN`（409）、`REFERENCE_CONFLICT`（409）、`PARENT_DELETED`（409）、`NAME_CONFLICT`（409）、`CROSS_WORK_REFERENCE`（422）、`VERSION_CONFLICT`（409）。
- 回收站新增 `root_type=faction`；`undo_tokens` 复用；快照与导出格式不变。

## 6. UI 状态

- 列表树：loading（8 行 Skeleton）/ empty（「还没有势力节点」+ 新建）/ no-result（「没有匹配的势力节点」+ 清除）/ error（重试）/ ready（默认展开两级、命中高亮、祖先自动展开）。
- 内联新建：idle / editing（父级 + 名称 + 简介）/ saving（禁用）/ error（字段级 + 保留输入）/ 取消确认。
- 详情：loading / not-found / ready / editing（名称、上级、首领、简介）/ saving / conflict（保留输入）/ 删除阻断（ReferenceDialog 列出下级与成员）。
- 成员区：direct / subtree 切换，分页 20；空态区分「暂无直属成员」「暂无下级成员」。
- 势力大事记：与人物一致（loading / empty / adding / editing / deleting 行内确认 / error）。
- 回收站：势力行 Badge、父级列、恢复失败提示「请先恢复上级势力」。

## 7. DoD

1. 可建立「万族 → 妖族 → 青丘」三级并默认展开两级；名称必填、同作品唯一、大小写不敏感。
2. 搜索节点名称可定位并展开祖先；无匹配有明确提示；1000 节点加载不阻塞页面。
3. 首领为当前作品人物，改名/删除状态同步；成员按直属/含下级正确推导、递归去重、分页稳定。
4. 移动父级到自身或后代返回 `CYCLE_DETECTED`，页面禁用非法选项并给出提示。
5. 删除仅被下级（含软删）与活跃成员阻止并列出引用；首领/大事记出向解除；无引用进回收站并 8 秒可撤销；恢复父级缺失被阻止。
6. 势力大事记可增删改、2,000 字校验、版本冲突与 8 秒撤销；人物换势力后原/新势力成员即时更新。
7. 作品整组删除/恢复/永久删除覆盖势力；永久删除后无势力行与势力大事记残留；迁移 0006 Up/Down 往返通过。
8. 后端 gofmt/vet/build/单测/集成，前端 typecheck/lint/单测/Node 22 构建/全量 E2E 通过；文档与证据齐全。

## 8. 风险与回滚

- **迁移风险**：`characters.faction_id` 历史值可能非空且指向不存在的势力；Up 先置空再建 FK。0006 Down 会删除 `owner_type=faction` 的大事记（回滚数据损失），已记录。
- **约束顺序**：自引用父级与人物归属 FK 均为 `RESTRICT`；永久删除需按「人物/大事记 → 势力深度倒序」执行，避免 FK 违约。
- **性能**：树一次返回 ≤1000 行；成员子树使用递归 CTE + 分页；计数用聚合查询，避免 N+1。
- **并发**：`If-Match` 保护文字编辑；移动与删除在事务内锁定节点/作品行；人物换势力与势力删除的并发以行锁复核。
- **回滚方案**：回退本分支提交；若已迁移，执行 0006 Down（恢复 `chronicles` 仅 character、删除 `factions` 与 FK），或从操作前快照恢复。
