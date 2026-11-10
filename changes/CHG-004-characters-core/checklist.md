# CHG-004 验收清单

> 勾选规则：每个验收点在 F3/F6 实测通过后勾选，并在证据列写命令、用例名或截图名。未实现项必须显式标注「延后」及承接变更。

## 1. 数据与迁移

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| DB-01 | `characters` 字段、长度 CHECK、软删除一致性 CHECK 与设计一致 | `0004_characters.sql`；`TestMigrationRoundTrip` | ☑ |
| DB-02 | 活跃主名称唯一 `(work_id, name_key)`，`Sword`/`sword` 视为同名 | `TestCharacterDatabaseConstraints`（大小写等价重名被拒） | ☑ |
| DB-03 | `character_names` 唯一 `(character_id, kind, value_key)`，随人物级联删除 | `TestCharacterDatabaseConstraints`（物理删除后 names=0） | ☑ |
| DB-04 | `character_system_refs` 追加人物外键，`ON DELETE CASCADE` | `0004_characters.sql`；`TestCharacterDatabaseConstraints`（refs=0） | ☑ |
| DB-05 | 0004 Down 可回滚且数据不残留 | `TestMigrationRoundTrip`：播种人物+引用，回滚后 refs=0，再迁移至版本 4 | ☑ |

## 2. 创建与字段

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| AC-01 | 新建 Drawer 按启用类别生成体系字段；停用/未启用类别不出现 | E2E + 集成 | ☐ |
| AC-02 | 详情首屏显示姓名、别名、称号、境界（`父·子`）；势力/地点/榜单空态 | `characters.spec.ts`「内联编辑、未保存守卫与版本冲突」 | ☑ |
| AC-03 | 同名别名跨人物允许；同人物重复别名去重且不报错 | 集成 + E2E | ☐ |
| AC-17 | 姓名空/超 200/同作品重名 → 422/409 字段级错误 | `TestCharacterCreateValidationMatrix`（空/超长/NAME_CONFLICT 明细） | ☑ |
| CH-01 | 简介 ≤20,000，超长 422 | `TestCharacterCreateValidationMatrix/简介超长` | ☑ |
| CH-02 | `names` 非法 kind/空值/超 100 → 422；>20 前端提示软上限 | 单测 + E2E | ☐ |
| CH-03 | `faction_id` 非空 → 422（势力未开放） | `TestCharacterCreateValidationMatrix/势力未开放` | ☑ |
| CH-04 | `location_links` 非空 → 422（地点未开放） | `TestCharacterCreateValidationMatrix/地点未开放` | ☑ |
| CH-05 | 创建返回 201 + `Location` 头；初始 `review_status=confirmed` | `TestCharacterCreateValidationMatrix` 首段 | ☑ |
| CH-06 | 作品不存在/已删 → 422 `PARENT_DELETED` | 集成 | ☐ |
| CH-07 | 体系类别不属于该作品 → 422 `CROSS_WORK_REFERENCE` | `TestCharacterCrossWorkAndDisabledCategory` | ☑ |
| CH-08 | 类别停用 → 422 `CATEGORY_DISABLED` | `TestCharacterCrossWorkAndDisabledCategory`、`TestCharacterUpdateOptimisticLockAndFields` | ☑ |
| CH-09 | 节点不属于所选类别 → 422 `NODE_PARENT_MISMATCH` | `TestCharacterCrossWorkAndDisabledCategory` | ☑ |
| CH-10 | 同一 `category_id` 重复提交 → 422 | `TestCharacterCreateValidationMatrix/重复类别` | ☑ |

## 3. 列表与查询

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| LD-01 | 列表仅返回当前作品未删人物；`work_id` 缺失 → 422 | `TestCharacterListFiltersSortAndPaging` | ☑ |
| LD-02 | `q` 命中主名称/别名/称号，大小写不敏感，`%`/`_` 已转义 | `TestCharacterListFiltersSortAndPaging`（shadow/SWORD/%25） | ☑ |
| LD-03 | 境界筛选按选中节点匹配 | `characters.spec.ts` 列表筛选用例；`TestCharacterListFiltersSortAndPaging` | ☑ |
| LD-04 | 待校对筛选 `pending`/`confirmed` 与库列映射正确 | `TestCharacterListFiltersSortAndPaging` | ☑ |
| LD-05 | 排序 `-updated_at`（默认）/`name`/`-name` 生效 | `TestCharacterListFiltersSortAndPaging` | ☑ |
| LD-06 | 分页默认 20、上限 100、越界页码返回空 items 且 total 正确 | `TestCharacterListFiltersSortAndPaging` | ☑ |
| LD-07 | 别名/称号 chip 最多 3 个，超出显示 `+N` | E2E | ☐ |
| LD-08 | 1000 人 P95 ≤500ms（采样 20 次） | `TestCharacterListPerformance`：P95=4.844ms | ☑ |
| LD-09 | 列表无 N+1（节点路径批量装配） | 代码审查 + SQL 计数 | ☐ |

## 4. 详情与编辑

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| AC-16 | 并发保存后到者 409 `VERSION_CONFLICT`，输入保留、不覆盖 | `characters.spec.ts` 版本冲突用例 | ☑ |
| EX-04 | 连续点击保存仅产生一次更新，提交期间按钮禁用 | E2E | ☐ |
| EX-06 | 保存失败（断网/500）输入保留 + 重试入口 | E2E | ☐ |
| EX-08 | 未保存离开弹窗，取消保留编辑 | `characters.spec.ts`「未保存守卫拦截站内链接」 | ☑ |
| DT-01 | 内联编辑姓名/别名/称号/简介/体系字段后保存成功，version+1 | `characters.spec.ts` 内联编辑用例 | ☑ |
| DT-02 | 保存后关系摘要与列表同步刷新 | E2E | ☐ |
| DT-03 | 停用类别在编辑态灰显并提示，保存被拒绝 | E2E + 集成 | ☐ |
| DT-04 | 404 人物显示「不存在或已删除」并可返回列表 | E2E | ☐ |
| DT-05 | 主图占位、势力/地点/榜单空态文案正确 | E2E | ☐ |

## 5. 删除、撤销与回收站

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| AC-14 | 删除返回 `undo_token`；8 秒内撤销恢复人物与体系引用 | `characters.spec.ts`「删除后可撤销」 | ☑ |
| TR-01 | 回收站 `root_type=character` 列出人物，父级为作品名 | 同上，回收站按人物筛选 | ☑ |
| TR-02 | 重复删除 → 409 `ALREADY_DELETED`；撤销令牌过期 → 409 `UNDO_EXPIRED` | 集成 | ☐ |
| TR-03 | 恢复时作品在回收站 → 409 `PARENT_DELETED` | `TestCharacterRestoreConflictAndParentDeleted` | ☑ |
| TR-04 | 恢复同名冲突 → 422 `RESTORE_CONFLICT` 且未恢复；`new_name` 成功 | `TestCharacterRestoreConflictAndParentDeleted` | ☑ |
| TR-05 | 永久删除清理 refs/names/characters，失败整体回滚 | 集成 | ☐ |
| TR-06 | 清空回收站含人物根 + 作品根 + 体系根时全部成功 | 集成 | ☐ |
| TR-07 | 回收站类型筛选「人物」、Badge、文案正确 | `characters.spec.ts`「删除后可撤销…」 | ☑ |

## 6. 引用与跨作品

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| RF-01 | 人物引用类别 → 停用/删除类别 409 `REFERENCE_CONFLICT`，details 含人物名与 `in_trash` | `system.spec.ts`「引用阻断：停用与删除均弹出引用清单」 | ☑ |
| RF-02 | 人物进入回收站后引用仍阻断，`in_trash=true` | `TestCharacterReferencesIncludeNameAndTrashState` | ☑ |
| RF-03 | 永久删除人物后类别可停用/删除 | `TestCharacterReferencesIncludeNameAndTrashState` | ☑ |
| RF-04 | 入向引用检查框架存在；当前空实现下删除成功（势力/榜单 CHG-006/008 接入） | 代码审查 + 集成 | ☐ |
| EX-21 | 跨作品同名人物互不冲突、列表互不可见、体系引用不可跨作品 | `TestCharacterCrossWorkAndDisabledCategory` | ☑ |
| WS-01 | 作品软删除后人物在列表/详情不可见，回收站显示人物根 | 集成 | ☐ |
| WS-02 | 作品恢复后人物与体系引用原状恢复 | `TestCharacterWorkGroupLifecycle` | ☑ |
| WS-03 | 作品永久删除后人物、别名、引用物理清除 | `TestCharacterWorkGroupLifecycle` | ☑ |

## 7. UI 与响应式

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| UI-01 | 导航「资料 > 人物」进入 `/characters`，当前项高亮 | `keyboard.spec.ts` 导航遍历 + 人物页标题 | ☑ |
| UI-02 | 无当前作品显示引导空态 | E2E | ☐ |
| UI-03 | 列表骨架/空态/错误态/成功态齐备 | E2E | ☐ |
| UI-04 | 新建 Drawer 字段顺序、必填标记、计数与错误定位正确 | E2E | ☐ |
| UI-05 | 1280/1024 无横向溢出；表格操作列固定 | 溢出通过；操作列 sticky 未实现（审查 C-04/C-15，延后） | ☐ |
| UI-06 | 键盘可达：Tab 顺序、Esc 关闭 Drawer/对话框、焦点返回触发元素 | E2E | ☐ |
| UI-07 | axe 无 serious/critical（`/characters` 列表与详情） | `characters.spec.ts` 无障碍用例 + `a11y.spec.ts` | ☑ |

## 8. 回归与文档

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| RG-01 | 既有 E2E 全量通过（含作品/体系/回收站/设置/壳层） | `npx playwright test`：63 passed | ☑ |
| RG-02 | 后端 `go vet`、单测、集成全绿 | gofmt/vet/build 0；`go test ./...` ok；65 集成用例 PASS | ☑ |
| RG-03 | `npm run lint/typecheck/test/build` 全绿 | typecheck/lint 0；43 单测通过；build 成功 | ☑ |
| DOC-01 | Spec 状态与缺口在 F10 更新（CHAR-001～005/008、WORK-001、SYS-002/005、TRASH-002/003/004） | `project/specs/*` | ☐ |
| DOC-02 | `api-contract.md` 回写 `CharacterSummary.names[]/version`、DELETE `If-Match`、`location_id` 未开放 | `project/api-contract.md` | ☐ |
| DOC-03 | `data-model.md` 记录 `faction_id` 无 FK 的实施差异 | `project/data-model.md` | ☐ |
| DOC-04 | `verification.md` 记录门禁、性能与 E2E 证据 | `verification.md` §1～§3、§7 | ☑ |

## 9. 明确延后（不计入本轮失败）

| 项目 | 承接 |
|---|---|
| 主图上传/缩略图、大事记 | CHG-005 |
| 势力引用数据与首领阻断、地点关联与筛选、榜单席位与入向阻断 | CHG-006 / CHG-007 / CHG-008 |
| `POST /{resource}/{id}/review` 与「确认已校对」按钮、导入 pending | CHG-011 |
| 全局检索命中别名/称号 | CHG-009 |
| 关系区双向跳转目标页、回收站 `highlight` 定位 | CHG-006～008 / CHG-012 |
| 人物-人物关系、关系图谱、批量操作、富文本简介 | 不做（需求边界） |
