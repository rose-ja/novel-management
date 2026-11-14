# CHG-007 验收清单

格式：`- [x] AC-编号 | 关联 Requirement/Spec/Page ID | 验证方式 | 预期结果 | 证据`（勾选表示已验收）

> 勾选规则：F3/F6 实测通过后勾选并写证据（命令、用例名、截图名）；F7 用户 ACCEPT 后统一勾选。
> 验收：2026-11-14 ACCEPT（详见 `acceptance.md`），AC-001～AC-404 全部勾选。

## Functional

- [x] AC-001 | SPEC-LOC-001 / FR-13 / PG-06 | E2E 内联新建三级地点 | 「东荒 → 青丘山 → 狐族秘境」层级正确，默认展开两级 | `e2e/locations.spec.ts#内联新建三级地点、默认展开两级与搜索/类型筛选`
- [x] AC-002 | SPEC-LOC-001 / FR-13 | 集成 + E2E 名称校验 | 名称必填、1–200、同作品唯一（大小写等价）、类型 ≤200、简介 ≤20,000 | `model/location_test.go` 规范化；`TestLocationTreeCRUD`（重名 409）；E2E 重名提示
- [x] AC-003 | SPEC-LOC-001 / FR-66 / PG-06 | E2E 搜索/筛选 | 名称搜索命中并展开祖先；类型/待校对筛选保留祖先链；无匹配提示 | `e2e/locations.spec.ts#1`；`TestLocationTreeCRUD` 搜索/类型补祖先
- [x] AC-004 | SPEC-LOC-001 / FR-16 / PG-06 | E2E 列表行 | 显示名称、类型、下级数、关联人物数、关联势力数、待校对 Badge | `locationSummary`；`e2e/locations.spec.ts#1`
- [x] AC-005 | SPEC-LOC-002 / FR-14 / AC-05 | 集成 + E2E 人物关系 | 人物关联 0..N 地点并带备注；人物与地点两侧一致；修改后刷新保留 | `TestLocationCharacterRelation`；`e2e/locations.spec.ts#详情编辑、人物与势力关系双向可见`
- [x] AC-006 | SPEC-LOC-003 / FR-14 / AC-05、AC-09 | 集成 + E2E 势力关系 | 地点关联 0..N 势力并带备注；两侧双向显示并可跳转 | `TestLocationFactionRelation`；`e2e/locations.spec.ts#2`
- [x] AC-007 | SPEC-LOC-003 / FR-15 / AC-14 | 集成 + E2E 删除阻断 | 有下级 → `HAS_CHILDREN`；有人物/势力关系 → `REFERENCE_CONFLICT` 并列出引用；不级联解除 | `TestLocationDeleteBlockedByChildren`；`TestLocationCharacterRelation`/`TestLocationFactionRelation`；`e2e/locations.spec.ts#3/#4`（分组 + 主操作禁用）
- [x] AC-008 | SPEC-LOC-003 / FR-57~FR-60 / PG-12 | 集成 + E2E 生命周期 | 无引用删除进回收站 + 8 秒撤销；恢复父级已删 → `PARENT_DELETED`；永久删除无残留 | `TestLocationTrashUndoRestorePurge`；`TestLocationRestoreBlockedByDeletedParent`；`e2e/locations.spec.ts#4`
- [x] AC-009 | SPEC-LOC-001 / FR-47~FR-51 / EX-18 | 集成 移动环 | 移动到自身/后代 → `CYCLE_DETECTED`；`If-Match` 缺失 428、过期 409 | `TestLocationParentValidation`
- [x] AC-010 | SPEC-WORK-001 / AC-01 | 集成 作品隔离 | 跨作品父级/关系 `CROSS_WORK_REFERENCE`；两本作品地点互不可见 | `TestLocationParentValidation`；`TestLocationCharacterRelation`（跨作品 422）
- [x] AC-011 | SPEC-LOC-002 / FR-07 | 集成 人物删除 | 人物删除不被地点关系阻止，关系随删隐式隐藏 | `TestLocationCharacterRelation`
- [x] AC-012 | SPEC-LOC-003 / FR-12 | 集成 势力删除 | 势力删除不被地点关系阻止；永久删除清理 `location_factions` 无残留 | `TestLocationFactionRelation`；`TestWorkLifecycleCoversLocations`
- [x] AC-013 | SPEC-LOC-002 / PG-07 | E2E 地点详情 | 首屏显示名称、类型、上级、下级数、关联人物数、关联势力数 | `e2e/locations.spec.ts#2`；视觉基线 `location-detail-1280.png`
- [x] AC-014 | SPEC-LOC-001 / PG-06 | E2E 类型建议 | 已有类型作为输入建议，允许新值；类型可空 | `locationTypeOptions` + datalist；`e2e/locations.spec.ts#1`

## UI / UX

- [x] AC-101 | PG-06、PG-07 | 页面布局对照 | 树缩进 20px/层、行高 36px；详情主栏 + 侧栏；Token 与设计系统一致 | 视觉基线 `locations-1280.png`、`locations-1024.png`、`location-detail-1280.png`
- [x] AC-102 | PG-06、PG-07 | Loading/Empty/Error/Success/Conflict | 树 8 行 Skeleton；空态含新建入口；错误可重试；保存失败保留输入；版本冲突 Alert | `location-list-page.tsx`/`location-detail-page.tsx`/`location-edit-form.tsx`；`e2e/error-states.spec.ts`
- [x] AC-103 | PG-06、PG-07 | 1280/1024 响应式 | 无横向溢出；1024 侧栏下移；关系表可横向滚动 | `locations-1024.png`；`components.css` 1024 媒体查询
- [x] AC-104 | PG-06、PG-07 | 键盘与 axe | 树节点/展开按钮 Tab 可达，Enter 打开、空格展开；axe 无 serious/critical | `e2e/a11y.spec.ts#地点：有数据无障碍`；`e2e/keyboard.spec.ts#地点树键盘`
- [x] AC-105 | PG-07 | 删除引用 Dialog | 按下级/人物/势力分组，条目可跳转，回收站引用显示徽标，主操作禁用 | `e2e/locations.spec.ts#3`（「关联人物（1）」+ 按钮 disabled）；`location-delete-dialog.tsx`

## Error / Boundary

- [x] AC-201 | SPEC-LOC-001 | 边界校验 | 空名称、超长字段、重名均字段级定位；无英文堆栈 | `model/location_test.go`；`TestLocationTreeCRUD`；`e2e/error-states.spec.ts`
- [x] AC-202 | SPEC-LOC-003 | 关系边界 | 跨作品人物/势力拒绝；重复关系 `DUPLICATE_RELATION`；备注 ≤500 | `TestLocationFactionRelation`；`normalizeCharacterLocationLinks`；`ErrDuplicateRelation` → 409
- [x] AC-203 | SPEC-LOC-003 | 恢复边界 | 父级在回收站时恢复被阻止并提示先恢复上级 | `TestLocationRestoreBlockedByDeletedParent`
- [x] AC-204 | 通用 | 并发 | 两请求同时移动成环，串行化后仅一个成功，无环数据 | `pg_advisory_xact_lock(hashtext(work_id))` + `IsAncestor` 递归 CTE；双连接并发自动化延后（`review.md` §4 残余风险）
- [x] AC-301 | SPEC-PLAT-005 | 输入与路径 | 名称/简介/备注不接受脚本注入；不暴露内部 ID 或堆栈 | React 文本渲染转义；`e2e/error-states.spec.ts` 无堆栈；axe 扫描通过

## Regression

- [x] AC-401 | CHG-004 | 人物回归 | 人物 CRUD、别名、体系引用、图片、大事记、回收站用例全通过 | F6 全量集成 + E2E（`characters*.spec.ts`）PASS
- [x] AC-402 | CHG-006 | 势力回归 | 势力树、首领、成员、大事记、删除/恢复用例全通过 | F6 全量集成 + E2E（`factions.spec.ts`）PASS
- [x] AC-403 | CHG-002 | 作品/回收站回归 | 作品切换、组删除/恢复/清空、撤销用例全通过 | F6 全量集成 + E2E（`works.spec.ts`、`trash*.spec.ts`）PASS
- [x] AC-404 | 通用 | 全量门禁 | 后端 build/vet/unit/integration、前端 typecheck/lint/test/build、Playwright 全通过 | `verification.md` §4：86 passed（1.2m）
