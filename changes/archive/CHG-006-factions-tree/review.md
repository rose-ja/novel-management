# CHG-006 独立代码审查（F4）

- 审查对象：`feat/chg-006-factions-tree`（`9467d11..496d1b0`）
- 依据：proposal / design / tasks / checklist / verification、`project/specs/factions/specs.md`、PG-04、PG-05、`project/api-contract.md`、`project/data-model.md`
- 审查者：独立审查（作者之外）
- 结论：**FAIL**（3×S1 未修复/未豁免）
- 日期：2026-11-13

## 问题清单

### RV-01（S1）永久删除势力未复核活跃成员，可导致 500

- 位置：`backend/internal/service/trash.go:749-765`、`backend/internal/repository/factions.go:519-524`
- 关联：SPEC-FACTION-004（入向引用阻止删除）、PG-05 §6.10、设计 D9
- 触发条件：人物 C 引用势力 F → 删除 C → 删除 F（此时无活跃成员，允许）→ 回收站恢复 C（`restoreCharacter` 只校验作品与重名，不校验势力）→ 永久删除 F。
- 实际影响：`Purge` 只清理 `deleted_at IS NOT NULL` 的人物引用，恢复后的活跃 C 仍指向 F，`characters_faction_fk`（RESTRICT）报错 → 永久删除返回 500，且该势力永远无法从回收站清除。并发场景（删势力与人物改属势力同时发生）同样可能以 500 收场。
- 修复建议：Purge 事务内锁定回收站根并复核活跃成员，存在则 `REFERENCE_CONFLICT` 列出人物；清空回收站同样复核。补充「恢复人物后永久删除势力」用例。
- 验证建议：集成用例覆盖上述序列，断言 409 `REFERENCE_CONFLICT` 且势力行仍存在；解除成员引用后可成功永久删除。

### RV-02（S1）清空回收站与「作品内已单独删除势力根」组合可能失败

- 位置：`backend/internal/service/trash.go:819-863`
- 关联：SPEC-FACTION-004、设计 D9/LIFE-05
- 触发条件：先单独删除势力 F（回收站根 F），再删除其所属作品 W，随后清空回收站。
- 实际影响：`PurgeGroup(W)` 已物理删除作品内全部势力（含 F），后续循环再次处理 F 根。若该路径返回 `ErrAlreadyDeleted`，整个 `PurgeAll` 回滚并返回 409，回收站永远无法清空。
- 修复建议：清空回收站对「根已不存在」视为已处理；或先去重/先删子根。补充「单独删除势力根 + 删除作品 + 清空回收站」用例。
- 验证建议：集成用例断言清空成功（200）且无残留行。

### RV-03（S1）首领选择不支持搜索，超过一页人物时无法选中

- 位置：`frontend/components/factions/faction-detail-page.tsx:81-87`、`frontend/components/factions/faction-edit-form.tsx:117-126`
- 关联：PG-05 §6.2.3「首领选择：搜索当前作品人物；显示势力/境界摘要；可清空」、SPEC-FACTION-002
- 触发条件：当前作品人物超过 100（`pageSize:100` 单页加载）或需要按名称定位。
- 实际影响：只能在下拉中浏览前 100 位人物，无法通过搜索选到目标首领，核心编辑能力受限。
- 修复建议：首领选择接入服务端 `q` 搜索（300ms 防抖），并保留可清空能力。
- 验证建议：E2E 覆盖「输入关键字 → 选择首领 → 详情摘要更新」。

### RV-04（S2）首领在回收站时详情未标注不可用，编辑保存返回 409

- 位置：`frontend/components/factions/faction-detail-page.tsx:262-270`、`backend/internal/service/factions.go:414-423`
- 关联：SPEC-FACTION-002 验收场景 4（首领在回收站 → 标注不可用并提供引用定位，RT-13）
- 触发条件：首领人物移入回收站后打开势力详情，或在该状态下点击保存。
- 实际影响：首领链接指向已删除人物（404），页面无「不可用」标记；保存时提交回收站首领 ID 触发 `PARENT_DELETED` 409，用户无法仅修改名称/简介。
- 修复建议：详情 DTO 暴露首领可用性（`in_trash`），页面标注并提供回收站定位；编辑表单在首领未变更时省略该字段。
- 验证建议：集成断言 `leader.in_trash=true`；E2E 断言标注文案与保存成功。

### RV-05（S2）被软删下级阻断父级删除时，引用列表缺回收站标识且链接 404

- 位置：`backend/internal/service/factions.go:490-495`、`frontend/components/factions/faction-delete-dialog.tsx:26-37`
- 关联：SPEC-FACTION-004、设计 D7
- 触发条件：下级势力已移入回收站后删除父级。
- 实际影响：阻断弹窗列出下级并链接 `/factions?id=`，但该势力已不在活跃树，跳转 404，且未提示「回收站」。
- 修复建议：`ConflictItem` 增加 `in_trash`，前端对回收站项改为回收站定位并标注。
- 验证建议：E2E 覆盖「删除父级 → 下级带（回收站）标识」。

### RV-06（S2）并发移动可能形成环

- 位置：`backend/internal/service/factions.go:369-412`
- 关联：SPEC-FACTION-003 验收场景 2、设计 D4
- 触发条件：两请求分别把 A 挂到 B、B 挂到 A；事务各自只锁自身行，环检测读取不到对方未提交变更。
- 实际影响：极小概率写入父子环，树渲染与祖先遍历异常。
- 修复建议：对同作品势力移动加事务级互斥（如 `pg_advisory_xact_lock`），再执行环检测。
- 验证建议：并发集成用例或代码评审确认；至少记录为已知限制。

### RV-07（S2）「仅看待校对」不补祖先，命中节点可能永不显示

- 位置：`frontend/lib/factions-tree.ts:49-63`
- 关联：PG-04 §筛选、SPEC-FACTION-001
- 触发条件：勾选「仅看待校对」，命中的待校对节点存在上级（上级非待校对）。
- 实际影响：`visible` 只含命中节点，`roots` 仅取顶层，导致命中节点既非根也无可见祖先，树中永不渲染；单测固化了错误行为。
- 修复建议：任何过滤（含 `pendingOnly`）命中后都补齐祖先。
- 验证建议：更新单测断言祖先可见、非待校对祖先不标记为命中。

## 契约偏差记录（转入 F10 回写）

1. `GET /api/v1/factions` 返回 `{items,total}`，`api-contract.md` 需同步包装形态与 `total` 语义。
2. `DELETE /api/v1/factions/{id}` 强制 `If-Match`（缺失 428），需写入契约。
3. `GET /factions` 忽略 `flat`/`page`/`page_size`；`GET /factions/{id}/members` 返回 `{items,page,page_size,total}` 而非裸数组。
4. `PATCH /factions/{id}` 支持 `parent_id`/`leader_character_id`/`description` 的 `null` 清空语义。
5. 本轮 `relations.locations` 恒为空（CHG-007 接入），校对端点延后，未知 `owner_type` 统一 `VALIDATION_ERROR`。

## 结论

存在 3 项 S1（RV-01～RV-03）未修复/未豁免，本轮审查 **FAIL**，进入 F5 修复复审。

## F5 修复记录与复审结论（2026-11-13）

| ID | 级别 | 处理 | 证据 |
|---|---|---|---|
| RV-01 | S1 | 已修复：单根永久删除先预检、事务内锁根复核活跃成员；清空回收站逐根复核；冲突返回 `REFERENCE_CONFLICT` 列出人物 | `service/trash.go`；`TestFactionPurgeBlockedByRestoredMember` |
| RV-02 | S1 | 已修复：清空回收站对已被组删除的势力根跳过；单根 `ALREADY_DELETED` 语义保留（E2E 清理辅助同步容忍） | `service/trash.go`；`TestPurgeAllWithIndividuallyDeletedFactionRoot`；`e2e/helpers.ts` |
| RV-03 | S1 | 已修复：首领选择新增服务端 `q` 搜索（300ms 防抖），保留清空与回收站禁用 | `faction-edit-form.tsx`；E2E「内联新建与详情编辑保存后同步」 |
| RV-04 | S2 | 已修复：`leader.in_trash` 暴露到 DTO；详情标注（回收站）并提供回收站定位；未变更首领时 PATCH 省略字段，避免 409 | `TestFactionLeaderInTrashMarked`；`faction-detail-page.tsx` |
| RV-05 | S2 | 已修复：`ConflictItem.in_trash`；软删下级在阻断弹窗标注（回收站）并指向 `/trash` | `TestFactionDeleteConflictMarksTrashedChild`；E2E「软删下级在删除阻断中标出回收站」 |
| RV-06 | S2 | 已修复：上级变更前以 `pg_advisory_xact_lock(hashtext(work_id))` 串行化同作品移动，再执行环检测 | `service/factions.go`；既有环用例回归通过（未新增并发用例，见下） |
| RV-07 | S2 | 已修复：任意过滤命中（含「仅看待校对」）统一补齐祖先；单测更新 | `lib/factions-tree.ts`、`lib/factions-tree.test.ts` |

补充说明：

- RV-06 未新增并发集成用例（测试事务模型为单连接回滚，无法真实并发）；实现层面在锁内完成父链读取，残留风险记录为已知限制。
- RV-01 的「恢复人物后重新引用回收站势力」在正常 API 流程可达（恢复人物不校验势力，设计 D 允许），已由集成用例固化。

复审复跑结果：后端 `go build`/`go vet`/单测/全量集成全绿；前端 typecheck/lint/单测（54）/Node 22 构建通过；Playwright **76 passed**（含新增 1 项 E2E），视觉基线无变更。

**复审结论：PASS**（S1 全部修复，S2 全部处理）。
