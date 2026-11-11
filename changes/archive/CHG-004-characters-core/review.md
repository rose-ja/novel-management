# CHG-004 人物档案核心 — 独立代码审查报告

| 项目 | 内容 |
|---|---|
| 变更 | CHG-004「人物档案核心」 |
| 审查分支 | `feat/chg-004-characters-core` |
| 审查基线 | `2852a35`（diff 范围：`git diff 2852a35...HEAD`） |
| 审查日期 | 2026-11-09 |
| 审查方式 | 独立静态审查（作者非审查者）；未修改任何源码、未提交 git |
| 门禁依据 | 引用 `verification.md` 记录（gofmt/vet/build、64 集成用例、43 前端单测、63 E2E、F3 UI 冒烟），未重复执行全量测试；对可疑点做了代码级复核 |
| 结论 | **FAIL**（1 项 S1 未修复/未豁免，14 项 S2） |

## 审查范围

- 后端：迁移 `0004_characters.sql`、`service/characters.go`、`service/trash.go`、`repository/characters.go`、`repository/works.go`、`repository/system.go`、`httpapi/characters.go`、`httpapi/request.go`。
- 前端：`components/characters/*`、`components/ui/{table,filter-bar,toast}.tsx`、`lib/api/characters.ts`、`lib/messages.ts`、`app/characters`、`e2e/characters.spec.ts`。
- 需求与规范：`changes/CHG-004-characters-core/{proposal,design,tasks,checklist,verification}.md`、`project/specs/characters/specs.md`、`project/api-contract.md` §3.3、`project/data-model.md` §3.2、`project/page-specs/PG-02-character-list.md`、`project/page-specs/PG-03-character-detail.md`、`project/testing-strategy.md`、`project/design-system.md`。
- 范围裁决：F1 已批准的 D1（势力/地点占位）、D2（校对学生端点延后 CHG-011）、D3（`root_type=character`）、D4（列表隐藏势力/地点筛选、榜单列固定「—」）不作为缺陷重复报告。

## 问题清单

| ID | 严重度 | 文件与行号 | 关联需求或 Spec | 触发条件 | 实际影响 | 修复建议 | 验证建议 |
|---|---|---|---|---|---|---|---|
| C-01 | **S1** | `backend/internal/repository/characters.go:375-388`（删除语句在 384 行）；调用方 `backend/internal/service/trash.go:601-617`、`692-697` | design §3.6「永久删除先物理删除…失败整体回滚」；回收站永久删除 `DELETE /trash/character/{id}` | 回收站永久删除与撤销（`POST /undo/{token}`）或恢复并发：Purge 事务外的 `GetDeleted` 预检通过并落快照后，另一事务先提交 Restore；随后 Purge 执行 `DELETE ... WHERE id = ?` | `Purge` 无 `deleted_at IS NOT NULL` 守卫，会把已恢复的**活跃人物物理删除**且不可逆（只能靠快照手工恢复）；与单条 Purge 期望返回 `ALREADY_DELETED` 的语义不一致；PurgeAll 的人物循环同样受影响 | 在 `Purge` 删除条件中增加 `AND deleted_at IS NOT NULL`；事务内先 `SELECT ... FOR UPDATE` 复核根状态（`deleted_at IS NOT NULL AND deleted_root_id = id`）；`rows == 0` 返回 `ErrAlreadyDeleted`/冲突；PurgeAll 对 0 行按「已被作品组清理」容错而不是静默忽略 | 新增集成用例：事务 A 进入 Purge 预检并落快照后，事务 B 先 Restore 提交，再执行 Purge，断言人物仍存在且返回冲突；并覆盖 `PurgeAll` 同场景 |
| C-02 | S2 | `backend/internal/migrations/0004_characters.sql:60-63`；证据 `backend/internal/integration/migrations_test.go:51-61`；`checklist.md:13`（DB-05） | DB-05「0004 Down 可回滚且数据不残留」；data-model §7 | 数据库中存在 `character_system_refs` 行时执行 `goose down` 0004 | Down 只删除外键并 DROP `character_names`/`characters`，未清理 `character_system_refs`，回滚到 0003 后残留指向已不存在人物的引用行；`TestMigrationRoundTrip` 未插入任何数据，无法证明「数据不残留」，DB-05 勾选证据不实 | Down 中先 `DELETE FROM character_system_refs;`（或 `DELETE ... WHERE character_id NOT IN (SELECT id FROM characters)` 已不可行时直接清空），再删表；在迁移说明中明确回滚会丢失人物数据 | 迁移测试先插入 1 人物 + 1 体系引用，再 down 0004，断言 `character_system_refs` 为 0 且 0003 表结构可用；同步修正 DB-05 证据描述 |
| C-03 | S2 | `frontend/components/characters/character-detail-page.tsx:214-221`；`frontend/lib/messages.ts:439-442`；`frontend/e2e/characters.spec.ts:175-181` | PG-03 §6.2 第 7 条（第 179 行：不展示 version，ConflictDialog「数据已被其他页面修改，请刷新后重试」）；SPEC-CHAR-005 验收场景 1（第 93 行）；design §5.2（第 270 行）；proposal EX-07 | 两个标签页/两个客户端先后保存，后保存者收到 409 `VERSION_CONFLICT` | 提示文案输出「内容已被修改（最新版本 N）」，向用户暴露内部 `version`；未使用 `ConflictDialog`，仅顶部 Alert + 刷新按钮；E2E 把违规文案固化为断言，修正文案会直接破坏测试 | 删除文案中的版本号，使用规范指定文案；按 design 实现 `ConflictDialog`（刷新/关闭，刷新仍走未保存守卫）；将 E2E 断言改为固定文案并断言不出现版本数字 | 组件测试断言冲突文案不含数字；E2E 双标签页并发保存分支断言对话框标题与按钮；更新 `characters.spec.ts:178` |
| C-04 | S2 | `frontend/components/characters/character-table.tsx:13-22`、`46-78`；`frontend/components/ui/table.tsx:26-61`（32 行表头）；`frontend/components/ui/filter-bar.tsx:12-22`；`checklist.md:96`（UI-05） | PG-02:44（操作列固定右侧）、61（可排序列 + 箭头）、70/88（行操作：查看/删除，删除仅在行菜单）、102/168（活跃筛选 chip 可单独移除）、158/161（1024–1279 隐藏别名列、操作列 sticky） | 1280/1024 宽度浏览列表；使用筛选或删除 | 操作列仅有「编辑」链接（实际进入详情），无「删除」与行菜单，用户无法从列表删除；无 sticky 操作列；1024–1279 未隐藏别名/称号列；表头不可排序、无 `aria-sort`；筛选栏仅「重置（N）」，无活跃筛选 chip，无法单独移除条件；UI-05 勾选「表格操作列固定」与实现不符 | 补齐行菜单（查看/删除，删除走引用检查与确认）、sticky 操作列、窄屏隐藏别名列、表头排序与 `aria-sort`、活跃筛选 chip（含移除按钮与 `aria-label`）；若本轮有意简化，需按 D 级裁决记录并在 checklist 取消勾选 | E2E 在 1024 断言别名列隐藏、操作列 sticky（计算样式）与删除入口；组件测试断言 chip 渲染与移除；更新 UI-05 证据 |
| C-05 | S2 | `frontend/components/characters/character-list-page.tsx:237-245` | PG-02:121、185、195（PG02-Q1 已裁决「进入详情」）；design §6.1 | 在列表 Drawer 中新建人物并保存成功 | `onCreated` 只重置页码并刷新列表，未跳转详情；用户需再点行才能补全关系/图片，与已裁决的验收标准不符；`onCreated` 仅回传名称，缺少 ID 无法导航 | 让创建接口返回的 `CharacterDetail`（含 `id`）回传给列表，保存成功后 `router.push('/characters?id=<id>')`；如坚持刷新列表，需变更 PG-02/design 裁决并记录 | E2E 新建后断言 URL 含 `?id=` 且详情标题可见；组件测试断言 `onCreated` 携带 id |
| C-06 | S2 | `frontend/components/characters/character-detail-page.tsx:309-324`、`472-487`；`frontend/components/characters/character-delete-dialog.tsx:62-81`、`102-111` | PG-03:154/158（删除在「更多」菜单，不与编辑并列）、183/186（引用 ReferenceDialog 720px、分组、可跳转、主操作禁用）、188（成功后返回 PG-02） | 查看态删除人物；或未来出现 `REFERENCE_CONFLICT` | 删除按钮与编辑并列，误触风险；删除成功后停留在详情页「已移入回收站」空态，而非返回 PG-02；引用冲突仅内联 `ul`（无 720px `ReferenceDialog`、无跳转、主按钮未禁用）。当前 `characterInboundReferences` 返回空，引用分支暂不可达，但 UI 已按错误形态实现 | 将删除收进「更多」菜单；删除成功（或撤销窗口开始）后返回 `/characters` 并保留 Toast；引用冲突复用 `ReferenceDialog`，禁用主按钮并提供定位链接 | E2E 断言删除后回到 `/characters`、撤销后条目恢复；在 CHG-006/008 启用引用后补充 409 用例 |
| C-07 | S2 | `frontend/components/ui/toast.tsx:44-53` | design §6.2（第 304 行：8 秒倒计时、超时 Toast「已移入回收站」）；PG-03 §6.3 第 6 条 | 删除人物后出现撤销 Toast | 仅设置 8 秒定时关闭，无倒计时显示；超时关闭后无「已移入回收站」提示，用户无法感知撤销窗口何时结束；`showToast` 也没有超时回调能力 | 为 undo Toast 增加剩余秒数展示与 `onExpire` 回调，超时后追加「已移入回收站」提示；或明确记录该简化并获得豁免 | 组件测试用 fake timers 断言倒计时与超时回调；E2E 等待 8 秒断言二次提示 |
| C-08 | S2 | `frontend/components/characters/character-detail-page.tsx:186-221`、`328-340`；`frontend/components/characters/character-create-drawer.tsx:46-74` | PG-03 §6.2 第 3 条（第 175 行：首个错误滚动并聚焦、字段错误贴近字段）、§7 Error（保存）第 199 行；SPEC-CHAR-001:15/23；checklist UI-04（☐） | 新建/保存时名称为空、别名空、重名、体系引用非法（`VALIDATION_ERROR.details.field`） | 所有错误只显示在顶部 Alert，未定位到具体字段，未设置 `aria-invalid`、未聚焦首个错误；后端 `details.field` 被丢弃；与本变更自身未勾选的 UI-04 一致 | 建立字段错误状态，按 `details.field`/本地校验映射到 `FormField`，显示字段级文案并聚焦首个错误；重名冲突定位到「姓名」 | 组件测试覆盖空名称、空别名、重名、体系引用错误；E2E 断言焦点与字段错误文案 |
| C-09 | S2 | `frontend/components/characters/character-form-fields.tsx:68`、`243-267`；`frontend/components/characters/character-detail-page.tsx:47-55` | design §5.1（第 265 行：已停用类别在编辑态灰显并提示「类别已停用，请先启用或清除该字段」）；checklist DT-03（☐） | 编辑一个引用了「已停用类别」的人物（正常情况下被 `assertCategoryUnreferenced` 阻止，仅并发绕过停用检查、历史数据或直接改库时可达） | `useCharacterSystemFields` 只拉取启用类别，停用类别不会渲染控件；但 `detailToForm` 仍把它放进 `systemRefs`，提交时原样带回，后端必定返回 `CATEGORY_DISABLED`，而界面上没有可清除该引用的入口，形成无法保存的死胡同 | 编辑态额外拉取已停用类别（或从详情回填），以禁用 Select + 提示展示并允许清除；提交前剔除已清除项 | 组件测试构造含停用类别引用的详情，断言灰显、提示与可清除；集成测试验证清除后可保存 |
| C-10 | S2 | `backend/internal/service/characters.go:285-287`、`354-356`；`backend/internal/httpapi/characters.go:52-54`、`101-139` | design §3.3 第 175 行、§3.4 第 187 行（`faction_id` 仅允许 `null`）；api-contract §3.3（`faction_id: id\|null`） | 请求体传 `"faction_id": ""`（空字符串） | Create/Update 只拒绝 `TrimSpace != ""`，空字符串静默通过；Create 从不写入 `FactionID`，Update 保留旧值，客户端会误以为已设置/清除势力；空串不是合法 ID 也不是 null | 显式拒绝空字符串（`in.FactionID != nil` 即 422），或先把空串归一化为 `nil` 并按 null 处理并在文档中说明；建议沿用「只允许 null」 | 集成测试 POST/PATCH `faction_id: ""` 断言 422 `VALIDATION_ERROR` 且 `field=faction_id` |
| C-11 | S2 | `backend/internal/service/characters.go:309-316`；`backend/internal/service/trash.go:487-498`；`backend/internal/repository/works.go:91-136` | design §4.1 第 1 步「`works.GetActive` → 不存在 PARENT_DELETED」；FR 跨作品/回收站一致性 | 人物创建或回收站恢复与「作品软删除（整组级联）」并发执行 | 事务内只用 `works.Get` 读取且未 `FOR UPDATE`；作品软删除的级联 UPDATE 可能先于人物 INSERT/恢复提交，导致产生「挂在已软删作品下、`deleted_at IS NULL`」的活跃人物：列表不可见、作品整组恢复/清理也不会再处理它；恢复路径同样存在 | 在人物创建/恢复事务内对作品行加锁（`SELECT ... FOR UPDATE`），或在写入后再校验作品仍 `deleted_at IS NULL` 并不满足则回滚；恢复时同样锁定作品行 | 并发集成测试（`-race`）交错执行作品软删除与人物创建/恢复，断言最终不存在「活跃人物 + 已删作品」组合 |
| C-12 | S2 | `backend/internal/service/trash.go:100-108`、`126-166`（N+1 在 157 行） | 性能（P-01 列表 P95 ≤500ms 的同类期望）；FR-57~63 回收站；design §3.6 | 回收站存在大量人物根并请求 `GET /trash?root_type=character` | 每个人物根单独调用 `worksRepo.Name`，N 个人物产生 N 次查询；即使 `root_type=character` 也始终执行作品/节点计数查询，属无效开销 | 先收集人物 `work_id` 去重，批量查询作品名后内存映射；按 `root_type` 条件执行对应计数查询 | 用 GORM 查询计数回调统计 SQL 条数，断言与根数量无关（常数级）；或用大 fixture 做 P95 |
| C-13 | S2 | `frontend/components/characters/character-list-page.tsx:112-126`、`220-226` | PG-02 §6.5（第 125 行：失败显示 Alert + 重试，保留筛选）；列表数据正确性 | 列表已有数据后刷新/翻页/筛选请求失败（网络中断、5xx） | `.catch` 只设置错误状态，未清空 `data`；表格仍渲染上一次结果，错误 Alert 与旧数据并存，用户可能误认为旧数据就是当前筛选结果 | 错误态下清空 `data`（或给旧数据加「可能已过期」标识并禁用行操作），保留筛选条件与重试入口 | 组件测试模拟首次成功、二次失败，断言不再展示旧行或明确标记过期 |
| C-14 | S2 | `backend/internal/migrations/0004_characters.sql:8`、`17`、`49`；`project/data-model.md:90`、`97`、`110`；`changes/CHG-004-characters-core/tasks.md` DOC-03 | data-model §3.2；DOC-03 仅计划补 `faction_id` | 后续开发者按 data-model 理解 schema，或做迁移/删除逻辑设计 | 三处漂移：`characters.work_id` 实现为 `ON DELETE CASCADE`（data-model 为 RESTRICT）、`reviewed_by` 无默认值（data-model 为默认 `local-author`）、`character_names.value` 为 1–100（data-model 为 1–200）。实现与 design §2 一致，但与 data-model 冲突；DOC-03 范围未覆盖，后续可能据错误文档实现级联/长度逻辑 | 裁决以 design 为准后，在 DOC-03 中同步 data-model 三处定义（或反向修正设计/迁移）；在变更记录中注明裁决 | 文档评审对照 `information_schema`/迁移 SQL；把 schema 关键约束加入集成断言 |
| C-15 | S2 | `frontend/e2e/characters.spec.ts:272-285`；`backend/internal/integration/migrations_test.go:51-61`；`checklist.md:13`、`96`；`project/testing-strategy.md` §4 E2E-02 | testing-strategy §3（迁移需覆盖可逆）、§4 E2E-02「删除被阻止」；checklist DB-05/UI-05 | 依赖现有门禁判断质量 | 1024 用例只断言 `documentElement.scrollWidth`，未验证别名列隐藏与操作列 sticky；迁移往返测试未造数据；E2E-02 的「删除被阻止」未覆盖（当前入向引用检查器为空，实际不可达，但未记录豁免）；DB-05/UI-05 却已勾选，导致真实缺陷未被发现 | 补充：迁移带回滚数据用例、1024 列策略与 sticky 断言、字段级错误用例；对「删除被阻止」显式记录 `N/A（CHG-006/008 接入后启用）` 或在该变更补测；修正不实勾选 | 见各用例；CI 中把新增断言纳入 E2E/集成门禁，并在 verification.md 更新证据 |

## 已检查且判定无问题的高风险点

- **SQL 注入与 LIKE 转义**：`repository/characters.go:60-85` 全部参数化，排序走 `characterOrder` 白名单；`likePattern` 转义 `\`、`%`、`_` 并配合 `ESCAPE '\'`。
- **列表 N+1（人物列表）**：`service/characters.go:149-205` 使用批量查询装配别名/体系摘要，未按行查询。
- **跨作品隔离**：`replaceSystemRefs` 校验类别 `work_id`、类别启用、节点属于类别且未删（`service/characters.go:502-540`）；列表强制 `work_id`；`GetActiveByWork` 提供作品内读取。
- **软删除与唯一约束**：活跃主名称部分唯一索引 `(work_id, name_key) WHERE deleted_at IS NULL`；`NameTaken` 过滤软删；`UpdateWithVersion` 识别 `characters_active_name_uq`；恢复要求 `deleted_root_id = id`。
- **软删除级联与恢复**：作品整组删除/恢复/物理删除对人物、别名、体系引用的处理与 design 一致（`repository/works.go:91-190`、`331-345`）；体系引用在人物软删除后保留，支持撤销（design D7）。
- **`name_key` 规范化**：`NormalizeCharacterName` 为 `TrimSpace + ToLower`，别名按 `(kind, value_key)` 去重并限制 100 条，符合 design §3.3。
- **`review_status` 只读**：API 返回 `confirmed/pending` 映射库内 `reviewed/pending`，无未授权的校对写入入口（D2 延后 CHG-011）。
- **`location_id` 筛选/`location_links` 非空**：均返回 422（D1 占位裁决），未静默忽略。
- **错误码映射**：`ErrPreconditionRequired → 428`（design §3.4 明确），`VERSION_CONFLICT details.current_version` 与前端读取一致；`REFERENCE_CONFLICT` 以数组作为 `details` 与 `api-contract.md` §1.3 示例一致（design §3.5 写作 `details.items`，属文档表述冲突，代码与既有契约、前端 `referenceConflictItems` 一致，不构成代码缺陷，建议随文档修订）。
- **安全渲染**：前端未使用 `dangerouslySetInnerHTML`/`innerHTML`，名称、简介、标签均走 React 文本转义；错误文案未泄露堆栈。
- **权限**：本版无认证（EX-05 = N/A），未发现新增越权入口。
- **迁移 Up 的 `DELETE FROM character_system_refs;`**：D5 已批准（0003 骨架无生产数据），不作为缺陷。
- **F1 范围裁决 D1–D4**：势力/地点占位、校对端点延后、`root_type=character`、列表隐藏势力/地点筛选与榜单「—」均按裁决实现，未重复计为问题。

## 结论

**FAIL**：C-01 为 S1，按审查规则必须修复或取得人工豁免后方可合入；其余 14 项 S2 建议在本变更内修复，或明确记录延期/豁免依据。修复完成后建议补充 C-02、C-04、C-15 对应的迁移数据回滚、1024 列策略与字段级错误测试，再更新 `verification.md` 证据。

---

## F5 修复记录（作者回应，待复审）

| ID | 处理 | 说明与证据 |
|---|---|---|
| C-01 | 已修复 | `Characters.Purge` 改为事务内 `SELECT ... FOR UPDATE` 复核 `deleted_at IS NOT NULL AND deleted_root_id = id`，未命中返回 0 行（服务层映射 `ALREADY_DELETED`），命中后才按 refs → names → characters 删除；新增 `TestCharacterPurgeSkipsRestoredRoot` 覆盖「恢复后清理」，断言人物/别名/引用均保留 |
| C-02 | 已修复 | `0004` Down 增加 `DELETE FROM character_system_refs;`；`TestMigrationRoundTrip` 先播种作品/类别/节点/人物/引用，回滚后断言 refs=0，再重新 Up；DB-05 证据同步修正 |
| C-03 | 部分修复 | 文案改为规范固定文案「数据已被其他页面修改，请刷新后重试」，移除 version 展示；E2E 断言同步。`ConflictDialog` 未落地，仍为顶部 Alert + 刷新按钮，S2 偏差保留待裁决 |
| C-04 | 延后 | 行菜单/sticky 操作列/窄屏隐藏别名列/表头排序/筛选 chips 属 PG-02 完整交互，超出 T11 已声明范围；已将 UI-05 取消勾选并记录偏差，建议独立变更承接 |
| C-05 | 已修复 | 创建成功后回传 `{id,name}` 并 `router.push('/characters?id=…')`；E2E 新建后断言详情 URL 与标题，再返回列表 |
| C-06 | 延后 | 删除入口与「成功返回列表」保持现状；引用 `ReferenceDialog` 依赖入向引用检查（CHG-006/008），本轮不可达，随引用变更统一改造 |
| C-07 | 延后 | 撤销 Toast 仍为 8 秒定时关闭，无倒计时与超时提示；已记录为已知简化 |
| C-08 | 延后 | 字段级错误定位未实现，错误仍集中在顶部 Alert；UI-04 保持未勾选 |
| C-09 | 延后 | 停用类别编辑态灰显与清除未实现；DT-03 保持未勾选 |
| C-10 | 已修复 | Create/Update 改为 `in.FactionID != nil` 即 422；新增创建与更新 `faction_id: ""` 集成用例 |
| C-11 | 已修复（实现） | 新增 `Works.GetForUpdate`（`FOR UPDATE` + `deleted_at IS NULL`），人物创建与回收站恢复在事务内锁定作品；并发交错测试未补，记录为后续测试增强 |
| C-12 | 延后 | 回收站人物根作品名 N+1 未优化；不涉及列表 P95 门禁，记录为性能待办 |
| C-13 | 已修复 | 列表请求失败时 `setData(null)`，不再与旧数据并存；保留筛选与重试 |
| C-14 | 延后至 F10 | data-model 三处漂移（work_id 级联、reviewed_by 默认值、names 长度）随 DOC-03 统一回写 |
| C-15 | 部分修复 | 迁移往返已补真实数据；1024 列策略/sticky 断言随 C-04 延后；E2E-02「删除被阻止」记录为 N/A（入向检查待 CHG-006/008）」 |

### 修复后门禁

| 检查 | 结果 |
|---|---|
| `gofmt`（LF 副本） / `go vet ./...` / `go test ./...` / `go build ./...` | 通过 |
| 集成测试 `-tags=integration` | 65 PASS / 0 FAIL（含新增 Purge 竞态用例与迁移数据回滚断言） |
| 前端 `typecheck` / `lint` / `test`（43 用例） | 通过 |
| 前端生产构建 | 通过（本机 Node 24 触发 `spawn UNKNOWN`，改用 `npx -y node@22` 执行；产物已部署容器） |
| `characters.spec.ts` | 7 passed（新建跳详情、冲突新文案均验证） |

结论待复审更新。

## F5 复审结果

复审对象：`feat/chg-004-characters-core` 的 `36f6398`（16 files changed）。本地仓库不存在任务给出的 `6a6f0a6`，故以 `git show 36f6398` 对照当前工作区代码复核；工作区干净，未修改源码、未提交 git。

### 逐 ID 状态

| ID | 原严重度 | 复审状态 | 结论依据 | 遗留/备注 |
|---|---|---|---|---|
| C-01 | S1 | **已修复** | `repository/characters.go:374-400` 在事务内以 `FOR UPDATE` 复核 `deleted_at IS NOT NULL AND deleted_root_id = id`，未命中返回 0 行；`service/trash.go:608-617` 将 0 行映射为 `ALREADY_DELETED`；`PurgeAll` 对 0 行容错（`trash.go:692-698`）。恢复先提交时，Purge 的 `SELECT ... FOR UPDATE` 会重新求值条件并返回 0 行，不再删除活跃人物 | `TestCharacterPurgeSkipsRestoredRoot`（`character_trash_test.go:211-239`）为「先恢复、后 Purge」的顺序等价场景，能证明条件守卫分支，但未起两个并发事务，也未覆盖服务层预检与 PurgeAll 路径；建议 F6 补真实并发用例（非阻断） |
| C-02 | S2 | **已修复** | `migrations/0004_characters.sql` Down 在删表前执行 `DELETE FROM character_system_refs;`；`migrations_test.go:51-81` 播种作品/类别/节点/人物/引用后回滚 0004，断言 refs=0，并完成全量 Down/Up 往返 | DB-05 证据已同步修正，描述与测试一致 |
| C-03 | S2 | **部分修复，接受延后** | `messages.ts:439` 改为固定文案「数据已被其他页面修改，请刷新后重试」，`character-detail-page.tsx:206-212` 不再读取/展示 `current_version`，E2E `characters.spec.ts:183` 已同步 | `ConflictDialog` 仍未落地（顶部 Alert + 刷新按钮）；`review.md`/`verification.md` 已如实标注为部分修复，未虚假勾选 |
| C-04 | S2 | **接受延后** | 行菜单/sticky 操作列/窄屏隐藏别名列/表头排序/筛选 chips 未实现；UI-05 已取消勾选并写入偏差（`checklist.md:96`） | 延期说明「超出 T11 已声明范围」不准确：T11 明确包含「筛选、排序」（`tasks.md:118`），实际只是排序改为筛选栏 Select；建议修正表述并登记承接任务 |
| C-05 | S2 | **已修复** | `character-create-drawer.tsx:67` 回传 `{id,name}`；`character-list-page.tsx:244-246` 创建后 `router.push('/characters?id=…')`；E2E `characters.spec.ts:73-77` 断言详情 URL/标题并返回列表 | 列表在返回时重新挂载拉取，未发现刷新回归 |
| C-06 | S2 | **部分修复，接受延后** | 引用冲突 `ReferenceDialog` 依赖入向引用检查（CHG-006/008），当前不可达，理由成立 | 「删除入口收进更多菜单」「删除成功后返回 PG-02」不依赖引用检查，仍保持现状；延期理由未覆盖这两点，登记为遗留 S2，需在 F6/CHG-006/008 明确承接 |
| C-07 | S2 | **接受延后** | 撤销 Toast 仍为 8 秒自动关闭，无倒计时/超时提示；已作为已知简化记录在 `review.md` 与 `verification.md` | 属 UI 体验 S2，未虚假声明已实现 |
| C-08 | S2 | **接受延后** | 前端字段级错误定位未实现，错误仍在顶部 Alert；UI-04 保持未勾选；后端 `VALIDATION_ERROR.details.field`/`NAME_CONFLICT` 明细已具备 | T12 验收文字含「名称冲突字段错误」（`tasks.md:127`），与当前实现存在张力；建议 F6/F7 收口时明确记为遗留或调整验收 |
| C-09 | S2 | **接受延后** | 停用类别编辑态灰显/清除未实现；DT-03 保持未勾选（`checklist.md:58`） | 正常路径被「被引用类别不可停用」阻断，仅并发绕过/历史数据可达，接受为记录项 |
| C-10 | S2 | **已修复** | `service/characters.go:285、354` 改为 `in.FactionID != nil` 即 422，空串不再静默通过；`character_test.go:69、215` 覆盖 Create/Update 空串 | `faction_id: null` 仍可清空（`ClearFaction` 分支保留）；前端默认发送 `null`，未发现回归 |
| C-11 | S2 | **已修复（实现）** | 新增 `repository/works.go:57-68` `GetForUpdate`（`FOR UPDATE` + `deleted_at IS NULL`）；`service/characters.go:311` 与 `service/trash.go:493` 在事务内先锁作品再写人物；锁序为「作品 → 人物」，与作品组软删除一致 | 未补并发交错测试；属 S2 测试增强项，建议 F6 用 `-race` 补「作品软删除 vs 人物创建/恢复」用例。实现本身可接受 |
| C-12 | S2 | **接受延后** | 回收站人物根父名 N+1 未优化；已记录为性能待办 | 不涉及已声明的列表 P95 门禁，接受为遗留 S2 |
| C-13 | S2 | **已修复** | `character-list-page.tsx:122` 请求失败时 `setData(null)`，不再与旧数据并存；筛选与重试保留 | 与 PG-02 §6.5 一致 |
| C-14 | S2 | **部分修复，接受延后** | data-model 三处漂移延后至 F10 的 DOC-03 | `checklist.md:109` 的 DOC-03 目前只写「`faction_id` 无 FK」，未覆盖 `work_id` 级联、`reviewed_by` 默认值、`names` 长度；需在 F10 前扩大 DOC-03 范围，否则「随 DOC-03 回写」不成立 |
| C-15 | S2 | **部分修复** | 迁移往返已补真实数据；1024 列策略/sticky 断言随 C-04 延后；E2E-02「删除被阻止」在 `review.md`/`verification.md` 记录为 N/A（入向检查待 CHG-006/008） | `characters.spec.ts` 的 1024 用例仍只断言文档无横向溢出；UI-05 已取消勾选，未虚假声明 |

### 新增问题（本次修复引入/暴露）

| ID | 严重度 | 文件与行号 | 触发条件 | 实际影响 | 修复建议 | 验证建议 |
|---|---|---|---|---|---|---|
| R-01 | S2 | `backend/internal/repository/characters.go:379-396`；`backend/internal/repository/works.go:346-359` | 同一人物的「单条永久删除」与「作品根永久删除/PurgeAll」并发执行（人物先单独删除、作品后删除时，两者会同时出现在回收站） | 单条 Purge 新锁序为「人物行 → refs → names → characters」；`purgeWorkChildren` 为「refs → names → characters」。两事务可形成环：一方持有 refs 等待人物行锁，另一方持有该行锁等待 refs，PostgreSQL 报 `40P01` 并使其中一个请求失败（500），无数据损坏，可重试 | 统一加锁顺序：在 `purgeWorkChildren` 开头先 `SELECT id FROM characters WHERE work_id = ? FOR UPDATE`，或统一改为「先删人物（依赖 FK 级联）再删 refs/names」；也可对死锁错误做有限重试 | 并发集成测试：两个事务分别执行单条 Purge 与 PurgeGroup，断言无 `40P01`、最终状态一致；或对 `purgeWorkChildren` 的锁序做代码级约束测试 |

### 复审结论

**PASS**（无新增 S0/S1；C-01 已实质修复）。

- 已修复：C-01、C-02、C-05、C-10、C-11、C-13。
- 部分修复并如实记录：C-03、C-06、C-14、C-15。
- 已记录延后且 checklist/verification 未虚假勾选：C-04、C-07、C-08、C-09、C-12。
- 需在 F6/F10 跟踪的遗留项：R-01 死锁风险；C-06 的「删除入口/成功返回列表」部分；C-14 的 DOC-03 范围；C-11 的并发测试与 C-01 的真实交错测试；C-04 的延期表述修正与承接任务。

以上遗留均为 S2，且已明确记录，不阻断本次复审通过；建议在 F6 证据补充阶段关闭 R-01 与 C-06 未覆盖部分，在 F10 前完成 DOC-03 扩项。

---

## F6 回归补充（作者记录）

| 项 | 处理 | 证据 |
|---|---|---|
| R-01 | 已修复：`purgeWorkChildren` 开头先 `SELECT id FROM characters WHERE work_id = ? FOR UPDATE`，与单条 Purge 统一为「人物行 → refs → names」锁序 | `repository/works.go`；集成 65 PASS（未做真实交错并发测试，作为后续增强） |
| UI-03 | 已修复：错误态不再同时渲染表格与空态，仅保留 Alert + 重试 | QA 场景 + `error-states.spec.ts` 通过 |
| UI-06 | 已修复：`Drawer` 在 `onOpenAutoFocus` 记录触发元素、`onCloseAutoFocus` 归还焦点 | QA 场景断言 Esc 后焦点回到「新建人物」通过 |
| C-06 | 仍延后（S2）：删除入口菜单与「成功后返回列表」未改，引用对话框待 CHG-006/008 | `review.md` 保留 |
| C-14 | 仍延后至 F10；DOC-03 范围需包含 `work_id` 级联、`reviewed_by` 默认值、`names` 长度 | 待 F10 扩项 |
| C-11 / C-01 并发测试 | 未补真实交错用例，保留为 S2 测试增强 | 现有顺序等价用例通过 |

回归结果：全量 E2E 63 passed（47.2s）；F6 QA 临时用例 1 passed（8 个场景：EX-04/EX-06/DT-02/DT-04/DT-05/UI-02/UI-03/UI-06），用后删除；后端 gofmt/vet/单测/集成 65 PASS；前端 typecheck/lint/43 单测通过。
