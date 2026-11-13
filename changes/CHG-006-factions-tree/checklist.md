# CHG-006 验收清单

> 勾选规则：F3/F6 实测通过后勾选并写证据（命令、用例名、截图名）。未实现项显式标注「延后」及承接变更。

## 1. 数据与迁移

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| DB-01 | `factions` 字段、`parent_id`/`leader_character_id` FK、软删除 CHECK 与设计一致 | `0006_*.sql`；迁移测试 | ☐ |
| DB-02 | 活跃名称唯一 `(work_id, name_key)`；回收站中可新建同名 | 集成用例 | ☐ |
| DB-03 | `characters.faction_id` FK 与 `(work_id, faction_id)` 部分索引生效，历史脏值已置空 | `0006_*.sql`；迁移测试 | ☐ |
| DB-04 | `chronicles.owner_type` 允许 `character/faction`；Down 恢复为仅 character 并清理势力大事记 | `TestMigrationRoundTrip` v6 | ☐ |
| DB-05 | 0006 Down 删除 `factions`、FK 与索引，人物表保留 | `TestMigrationRoundTrip` | ☐ |

## 2. 势力树与 CRUD

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| FAC-01 | 三级创建（万族 → 妖族 → 青丘）与默认同级排序稳定 | 集成 + E2E | ☐ |
| FAC-02 | 名称必填 1–200，同作品唯一，`Sword`/`sword` 判重 | 集成 | ☐ |
| FAC-03 | 父级可空、同作品活跃；跨作品 → `CROSS_WORK_REFERENCE` | 集成 | ☐ |
| FAC-04 | `GET /factions` 返回扁平全量 + 计数；`q` 返回命中 + 全部祖先 | 集成 | ☐ |
| FAC-05 | `PATCH` 必须 `If-Match`；版本不一致 `VERSION_CONFLICT` | 集成 | ☐ |
| FAC-06 | 移动到自身/直接子/深层后代 → `CYCLE_DETECTED` | 集成 + E2E | ☐ |
| FAC-07 | 首领必须当前作品活跃人物；可清空；人物改名后展示同步 | 集成 | ☐ |
| FAC-08 | 详情返回 `parent/children/leader/locations(空)/chronicle_count` | 集成 | ☐ |
| FAC-09 | 1000 节点列表加载不阻塞页面 | F3 手工/基线 | ☐ |

## 3. 成员推导

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| MEM-01 | 直属成员 = `characters.faction_id` 直接归属且活跃 | 集成 | ☐ |
| MEM-02 | `scope=subtree` 递归包含下级并去重 | 集成 + E2E | ☐ |
| MEM-03 | 分页 20、排序稳定、total 正确 | 集成 | ☐ |
| MEM-04 | 人物换势力后原/新势力成员即时变化 | 集成 + E2E | ☐ |
| MEM-05 | 成员为空时区分「暂无直属成员」「暂无下级成员」 | E2E | ☐ |
| MEM-06 | 摘要条直属/含下级/下级分支计数正确 | E2E | ☐ |

## 4. 删除、恢复与生命周期

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| LIFE-01 | 有下级（含软删子节点）删除返回 `HAS_CHILDREN` 并列出 | 集成 | ☐ |
| LIFE-02 | 有活跃成员删除返回 `REFERENCE_CONFLICT` 并列出人物 | 集成 | ☐ |
| LIFE-03 | 首领、势力大事记为出向：不阻止，随势力隐藏 | 集成 | ☐ |
| LIFE-04 | 无引用删除返回 `undo_token`；8 秒内撤销恢复势力与大事记 | 集成 + E2E | ☐ |
| LIFE-05 | 回收站 `root_type=faction` 列表/恢复/永久删除；清空回收站包含势力 | 集成 | ☐ |
| LIFE-06 | 恢复时父级已删 → `PARENT_DELETED` | 集成 | ☐ |
| LIFE-07 | 作品整组软删/恢复/永久删除覆盖势力与势力大事记 | 集成 | ☐ |
| LIFE-08 | 永久删除后无势力行、无势力大事记、无 FK 违约 | 集成 | ☐ |
| LIFE-09 | 人物担任首领时删除人物 → `REFERENCE_CONFLICT`（`faction_leader`） | 集成 | ☐ |
| LIFE-10 | 势力永久删除后，原首领人物可正常删除 | 集成 | ☐ |

## 5. 势力大事记

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| CHR-01 | `owner_type=faction` 可创建/列表/编辑/删除 | 集成 + E2E | ☐ |
| CHR-02 | 文本 1–2,000、时间标签 ≤200、`If-Match` 版本冲突 | 集成 | ☐ |
| CHR-03 | 单条删除 8 秒内可撤销；未知 owner_type 仍 422 | 集成 | ☐ |
| CHR-04 | 人物大事记行为不回归 | 全量集成 | ☐ |

## 6. 人物接入

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| CHAR-01 | 人物新建/编辑可设置与清空势力；跨作品/已删势力被拒绝 | 集成 + E2E | ☐ |
| CHAR-02 | 人物列表 `faction` 显示名称并可筛选 | 集成 + E2E | ☐ |
| CHAR-03 | 人物详情势力链接可跳转势力详情 | E2E | ☐ |

## 7. UI 与交互

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| UI-01 | 树默认展开两级；展开/收起全部；行高 36px 与缩进 | E2E + 视觉 | ☐ |
| UI-02 | 搜索 300ms debounce、命中高亮、自动展开祖先、滚动到首个命中 | E2E | ☐ |
| UI-03 | 新建为页内内联表单；保存后展开父级并选中新节点 | E2E | ☐ |
| UI-04 | 取消有输入时确认放弃；保存失败保留输入 | E2E | ☐ |
| UI-05 | 详情首屏名称/上级/首领/直属/含下级/下级分支 | E2E + 视觉 | ☐ |
| UI-06 | 内联编辑上级禁用自身与后代；清空=顶层 | E2E | ☐ |
| UI-07 | 成员表直属/含下级切换、分页、缩略图 | E2E | ☐ |
| UI-08 | 删除阻断 ReferenceDialog 分组并可跳转；成功后 Toast 撤销 | E2E | ☐ |
| UI-09 | 回收站势力行、父级列、恢复失败提示 | E2E | ☐ |
| UI-10 | 1280/1024 无横向溢出；导航「资料 > 势力」可达 | E2E | ☐ |
| UI-11 | 慢网下树先渲染文字，计数后到无布局抖动 | 未自动化（延后） | ☐ |

## 8. 可访问性

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| A11Y-01 | 树 `role=tree/treeitem` + `aria-expanded`；Tab/Enter/空格可用 | E2E keyboard | ☐ |
| A11Y-02 | 内联表单错误 `aria-describedby`；Dialog 焦点陷阱与归还 | E2E | ☐ |
| A11Y-03 | 搜索 `<mark>` 不只靠颜色；axe 无 serious/critical | axe | ☐ |

## 9. 测试与证据

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| TEST-01 | 后端 gofmt/vet/build/单测全绿 | `verification.md` §1 | ☐ |
| TEST-02 | 集成覆盖迁移/树/成员/首领/大事记/生命周期 | `verification.md` §2 | ☐ |
| TEST-03 | 前端 typecheck/lint/单测/Node 22 构建通过 | `verification.md` §1 | ☐ |
| TEST-04 | 全量 E2E 通过，含新增势力用例 | `verification.md` §3 | ☐ |
| TEST-05 | 视觉基线更新（势力列表/详情、人物势力列） | `visual.spec.ts-snapshots/` | ☐ |

## 10. 文档

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| DOC-01 | SPEC-FACTION-001～004 状态与缺口在 F10 更新 | `project/specs/factions/specs.md` | ☐ |
| DOC-02 | `api-contract.md` 回写势力 CRUD/成员/错误码与人物势力字段 | `project/api-contract.md` | ☐ |
| DOC-03 | `data-model.md` 回写 `factions`、FK、`chronicles.owner_type` | `project/data-model.md` | ☐ |
| DOC-04 | verification/review/acceptance/pr 齐全并含证据 | `changes/archive/CHG-006-.../` | ☐ |
