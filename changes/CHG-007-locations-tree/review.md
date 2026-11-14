# CHG-007 独立审查与复审

- Change ID：`CHG-007-locations-tree`
- 审查范围：`git diff d8cce2a..HEAD`（F2 实现至 F3）
- 审查者：独立子 Agent（不参与实现）
- 复审日期：2026-11-14

## 1. F4 审查结论

结论：FAIL（存在 2 项 S1 验收缺口）。核心后端正确性（迁移、环检测、`If-Match`、三类入向引用、撤销/恢复/永久删除、作品组生命周期、FK 清理顺序）经代码核对与集成测试通过，未发现可复现的 S0。

| ID | 级别 | 发现 | 关联 | 处置 |
|---|---|---|---|---|
| F4-01 | S1 | 删除成功后停留在已删实体详情，未返回列表，刷新即 404 | T11、PG-07 §6.3.5 | 已修：`onDeleted` 中 `router.push("/locations")`；E2E 断言 URL |
| F4-02 | S1 | 删除阻断对话框未分组、冲突后主操作仍可点击；有下级时未同时展示人物/势力引用 | AC-105、PG-07 §6.3.3、T11 | 已修：后端 `HAS_CHILDREN` 一次返回三类引用；前端按 `location_child`/`character_location`/`location_faction` 分组，冲突后禁用删除；E2E 覆盖 |
| F4-03 | S2 | 地点编辑与关系编辑缺少未保存守卫 | PG-07 §6.2.6/§6.2.8 | 已修：编辑表单与关系面板接入 `useUnsavedGuard` |
| F4-04 | S2 | 回收站引用被详情隐藏却阻止删除，关系接口与 `in_trash` 分支不可达 | D4、SPEC-LOC-003 场景 3 | 已修：详情保留回收站引用并标记 `in_trash`，活跃计数不变；集成断言更新 |
| F4-05 | S2 | 父级/地点校验未加锁，存在「活跃节点挂到回收站父级」或「关系指向已删地点」的 TOCTOU | 并发一致性 | 已修：父级、地点-势力新增、人物地点关系写入改用 `FOR SHARE`；`AddFaction` 改为事务。势力侧未加共享锁，作为残余风险记录 |
| F4-06 | S2 | 缺少 `locations` API 单测、并发移动用例与有数据 a11y 扫描 | T9、AC-204、AC-104 | 部分修复：新增 `lib/api/locations.test.ts`（6 条）、有数据地点树/详情/Dialog axe 用例、Tree 键盘用例；AC-204 并发移动以代码级 advisory lock 保障，未加自动化并发用例（见 §4） |
| F4-07 | S2 | `role=tree/treeitem` 但无 roving tabindex 与方向键模型，语义不实 | 可访问性 | 已修：改为原生 `ul/li` 列表语义，链接使用 `aria-current="page"`，键盘依赖原生 Tab/Enter |

## 2. F5 修复清单

| 文件 | 修复 |
|---|---|
| `backend/internal/service/locations.go` | `Delete` 在有下级时合并三类引用；`characterViews`/`factionViews` 保留回收站引用；父级校验与 `AddFaction` 使用共享锁；`AddFaction` 事务化 |
| `backend/internal/repository/locations.go` | 新增 `GetForShare`、`GetActiveByWorkForShare` |
| `backend/internal/service/characters.go` | 人物地点关系写入使用共享锁 |
| `backend/internal/integration/location_relations_test.go` | 更新「人物删除后保留关系并标记 `in_trash`、活跃计数为 0」 |
| `frontend/components/locations/location-delete-dialog.tsx` | 兼容 `{items}` 与数组两种 details，按类型分组，冲突后禁用删除 |
| `frontend/components/locations/location-detail-page.tsx` | 删除后返回列表；编辑未保存守卫 |
| `frontend/components/locations/location-edit-form.tsx` | 上报脏状态 |
| `frontend/components/locations/location-relations.tsx` | 关系编辑脏状态守卫；回收站引用只读展示、不可编辑 |
| `frontend/components/locations/location-tree.tsx` | 列表语义与 `aria-current` |
| `frontend/lib/api/locations.test.ts` | 新增 API 单测 |
| `frontend/e2e/{locations,a11y,keyboard}.spec.ts` | 覆盖删除后返回、分组阻断、有数据 axe、树键盘 |

## 3. 修复后验证

| 命令 | 结果 | Evidence |
|---|---|---|
| `go build ./...` / `go vet ./...` | PASS | 无输出 |
| `go test -count=1 ./...` | PASS | 全部包 |
| `go test -count=1 -tags=integration ./internal/integration/...` | PASS | 15.3s |
| `npm run typecheck` / `npm run lint` | PASS | 无错误 |
| `npm run test` | PASS | 10 文件 / 70 用例 |
| `npx -y node@22 next build` | PASS | 10 页面静态导出 |
| `npx playwright test e2e/locations.spec.ts` | PASS | 4/4 |
| 地点相关 a11y/键盘/视觉子集 | PASS | 10/10 |

## 4. 残余风险与豁免

- **AC-204 并发移动自动化**：代码使用 `pg_advisory_xact_lock(hashtext(work_id))` 串行化移动，并以 `IsAncestor` 递归 CTE 判断环；现有测试覆盖自我/直接子/深层后代。真实双连接并发用例未纳入自动化（集成 harness 为单事务回滚模型），作为残余风险接受，后续在 CHG-012 一致性收口时补充。
- **势力侧共享锁**：`AddFaction` 对地点加共享锁，对势力仍为普通活跃读取；极端并发下「势力删除 + 添加关系」可能留下指向回收站势力的关系行，删除地点时该行仍会阻断并标记 `in_trash`，不会静默丢失。接受为 S2。
- **删除阻断清单上限**：单次最多 50 条，超出部分不展示但阻断仍生效。

## 5. 复审结论

PASS。两项 S1 已修复并由 E2E/集成覆盖；S2 中可修项已修复，残余风险已记录并接受。进入 F6 全量回归。

### 5.1 独立复审（2026-11-14）

独立复审者确认：

- S1-1 PASS：删除成功后 `router.push("/locations")`，E2E 断言 URL。
- S1-2 PASS：`HAS_CHILDREN` 一次返回三类引用；对话框按类型分组、冲突后禁用删除；E2E 覆盖下级+人物分组与按钮禁用。
- 无新增 S0/S1。
- 残余 S2：跳转列表后点击撤销不会自动刷新列表（已补 `locations:changed` 事件，列表监听后重新取数）；E2E 未覆盖势力分组（由集成用例 `TestLocationFactionRelation` 覆盖）。
