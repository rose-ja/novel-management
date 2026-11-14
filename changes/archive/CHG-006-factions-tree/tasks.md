# CHG-006 任务拆解

- Change ID：`CHG-006-factions-tree`
- 依据：`proposal.md`、`design.md`
- 规则：任务尽量为可验证的垂直切片；每项完成后提交一次；门禁命令见 `design.md` §6.5。

## 1. 任务列表

### T1 迁移 0006 与模型

- **目标**：建立 `factions` 表、`characters.faction_id` FK/索引、放宽 `chronicles.owner_type`；补 `Faction` 模型与错误。
- **文件**：`internal/migrations/0006_factions_chronicles.sql`、`internal/model/faction.go`、`internal/model/{errors,media}.go`。
- **验收**：`go build ./...` 通过；迁移在空库 Up/Down 成功；`TestMigrationRoundTrip` 覆盖 v6（先写骨架，T9 补数据）。
- **依赖**：无。
- **验证**：`go build ./...`、集成 `-run TestMigrationRoundTrip`。

### T2 势力仓储

- **目标**：实现 `repository/factions.go` 的读取、写入、计数、祖先/后代判断、子树成员查询与生命周期删除。
- **文件**：`internal/repository/factions.go`、`internal/model/faction.go`。
- **验收**：`Get/ListActive/Insert/UpdateWithVersion/SoftDelete/Restore/Purge` 可用；`IsDescendant` 对自身/子/深层后代返回 true；`SubtreeMembers` 递归去重；`PurgeByWork` 按深度倒序不触发 FK 违约。
- **依赖**：T1。
- **验证**：T3 集成用例间接覆盖；`go vet ./...`。

### T3 势力服务：CRUD、名称与环

- **目标**：实现 `service/factions.go` 的 List/Create/Update，含作品/父级/首领校验、名称规范化与唯一、移动环检测、乐观锁。
- **文件**：`internal/service/factions.go`、`internal/model/errors.go`。
- **验收**：三级创建成功；`Sword`/`sword` → `NAME_CONFLICT`；跨作品父级 → `CROSS_WORK_REFERENCE`；移动到自身/后代 → `CYCLE_DETECTED`；旧 `If-Match` → `VERSION_CONFLICT`；`q` 返回命中 + 祖先。
- **依赖**：T2。
- **验证**：`internal/integration/factions_test.go` 前 4 组用例 PASS。

### T4 势力 HTTP 与路由

- **目标**：注册 `GET/POST /factions`、`GET/PATCH/DELETE /factions/{id}`，补齐错误映射与依赖注入。
- **文件**：`internal/httpapi/factions.go`、`internal/httpapi/{request,router}.go`、`cmd/server/main.go`、`internal/integration/suite_test.go`。
- **验收**：创建 201 + `Location`；PATCH 需 `If-Match`；DELETE 返回 `undo_token`；错误码与 `design.md` §5 一致。
- **依赖**：T3。
- **验证**：集成 API 用例；`go test ./...`。

### T5 成员推导与计数

- **目标**：实现 `/factions/{id}/members?scope=direct|subtree` 与列表/详情的 `children_count`、`member_count`、`subtree_member_count`。
- **文件**：`internal/service/factions.go`、`internal/repository/factions.go`、`internal/httpapi/factions.go`。
- **验收**：直属成员只含直接归属；`subtree` 含整棵子树且去重；分页 20 稳定；人物换势力后原/新势力结果即时变化。
- **依赖**：T2、T4。
- **验证**：`internal/integration/faction_members_test.go` PASS。

### T6 人物接入真实势力

- **目标**：`characters.faction_id` 接受真实势力；列表/详情返回 `FactionBrief`；`faction_id` 筛选生效。
- **文件**：`internal/service/characters.go`、`internal/repository/characters.go`、`internal/model/character.go`。
- **验收**：设置/清空势力成功；跨作品或已删势力被拒绝；列表 `faction` 显示名称；筛选只返回该势力成员。
- **依赖**：T2、T3。
- **验证**：集成人物用例新增分组 PASS。

### T7 人物删除新增首领阻断

- **目标**：删除人物时检查 `factions.leader_character_id`，命中返回 `REFERENCE_CONFLICT`。
- **文件**：`internal/service/characters.go`（或引用检查器）、`internal/repository/factions.go`。
- **验收**：首领人物删除被 409 阻止，`details.items[]` 含势力名与 `in_trash`；清空首领后可删除；首领在回收站的势力也阻断。
- **依赖**：T6。
- **验证**：集成用例 PASS；不回归 CHG-004 删除矩阵。

### T8 势力大事记

- **目标**：`chronicles` 支持 `owner_type=faction`，含列表/创建/版本/撤销与 owner 校验。
- **文件**：`internal/service/chronicles.go`、`internal/model/media.go`、`internal/repository/chronicles.go`、`internal/httpapi/chronicles.go`。
- **验收**：势力大事记 CRUD 与人物一致；`owner_type=faction` 可用；未知类型仍 `CHRONICLE_OWNER_UNSUPPORTED`；2,000 字与版本冲突生效。
- **依赖**：T1、T2。
- **验证**：集成用例 PASS。

### T9 生命周期与回收站

- **目标**：势力软删除/撤销/恢复/永久删除、回收站 `root_type=faction`、作品整组联动。
- **文件**：`internal/service/trash.go`、`internal/repository/{factions,works,chronicles}.go`、`internal/repository/works.go`。
- **验收**：有下级（含软删）/成员时删除被阻止；无引用删除返回 `undo_token`；8 秒内撤销恢复势力与大事记；回收站列表/恢复/永久删除可用；父级已删恢复 `PARENT_DELETED`；作品组删除/恢复/清空覆盖势力；永久删除后无残留。
- **依赖**：T4、T8。
- **验证**：`internal/integration/faction_lifecycle_test.go` + 迁移带数据往返 PASS。

### T10 前端 API 与纯函数

- **目标**：`lib/api/factions.ts`、chronicle/trash/characters 类型扩展、树构建与过滤纯函数。
- **文件**：`lib/api/{factions,characters,chronicles,trash}.ts`、`lib/api/factions.test.ts`、`lib/messages.ts`。
- **验收**：类型检查通过；错误码映射（`CYCLE_DETECTED`/`HAS_CHILDREN`/`REFERENCE_CONFLICT`）；树构建/命中 + 祖先测试通过。
- **依赖**：契约冻结（T4 接口形态确认）。
- **验证**：`npm run typecheck`、`npm run test`。

### T11 势力列表树与内联新建

- **目标**：`/factions` 页面：树、默认展开两级、搜索定位、展开/收起、待校对筛选、内联新建。
- **文件**：`app/factions/page.tsx`、`components/factions/{faction-route,faction-list-page,faction-tree,faction-inline-create}.tsx`、`styles/components.css`。
- **验收**：三级节点正确挂载；默认展开两级；搜索命中高亮并展开祖先；无匹配提示；新建后选中新节点；取消有输入需确认；空/加载/错误态完整。
- **依赖**：T10。
- **验证**：`npm run typecheck/lint`；E2E 1/2 通过。

### T12 势力详情与删除

- **目标**：`/factions?id=` 详情：摘要条、简介、下级、内联编辑（名称/上级/首领/简介）、移动禁用、删除对话框与引用阻断。
- **文件**：`components/factions/{faction-detail-page,faction-edit-form,faction-delete-dialog,member-table}.tsx`、`app/factions/page.tsx`。
- **验收**：首屏显示名称/上级/首领/计数；编辑保存后树与详情同步；非法上级不可选且 422 提示；删除阻断列出下级/成员并可跳转；成功后返回列表并可撤销。
- **依赖**：T5、T11。
- **验证**：E2E 3/4/5 通过。

### T13 成员、势力大事记与人物侧

- **目标**：成员表直属/含下级切换与分页；通用 `ChronicleSection`；人物详情势力选择、列表势力列与筛选、人物删除首领阻断提示。
- **文件**：`components/chronicles/chronicle-section.tsx`（迁移）、`components/characters/*`、`components/factions/member-table.tsx`、`lib/api/*`。
- **验收**：成员切换与分页正确；势力大事记增删改 + 撤销；人物设置/清空势力；人物删除被首领阻断时展示势力名并可跳转。
- **依赖**：T5、T8、T12。
- **验证**：E2E 3/6 通过；`npm run test`。

### T14 回收站、导航、视觉与回归

- **目标**：回收站势力根 UI、导航「资料 > 势力」、视觉基线、a11y/键盘适配、全量回归。
- **文件**：`app/trash/page.tsx`、`components/shell/global-nav.tsx`、`e2e/{factions,visual,a11y,keyboard}.spec.ts`、视觉基线。
- **验收**：回收站筛选「势力」、恢复失败提示父级；导航可达；1280/1024 视觉基线更新且无溢出；axe 无 serious/critical；全量 E2E 通过。
- **依赖**：T9、T13。
- **验证**：`npx playwright test` 全量 PASS。

## 2. 依赖图

```text
T1 ── T2 ── T3 ── T4 ── T5 ── T12 ── T13 ── T14
         │         ├── T6 ── T7
         ├── T8 ───┘
         └── T9 ─────────────── T14
T10 ── T11 ── T12
```

## 3. 并行边界

- 后端 T2/T3/T8 与前端 T10 可在 T4 契约冻结后并行。
- T5 与 T6 可并行（不同服务文件），合并前统一跑集成。
- T11 依赖 T10；T12/T13 依赖 T11 与对应后端任务。
- T14 最后执行，避免视觉基线与 E2E 反复变动。

## 4. 门禁

- 每个后端任务：`gofmt -l .`、`go vet ./...`、`go build ./...`、相关 `go test`。
- T4/T5/T7/T9 后：全量集成。
- 每个前端任务：`npm run typecheck`、`npm run lint`、`npm run test`。
- T13/T14 后：Node 22 构建 + 全量 Playwright。
- F3 统一回填 `verification.md`；F4 独立审查；F6 全量回归。
