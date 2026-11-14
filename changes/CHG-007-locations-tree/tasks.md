# CHG-007 任务拆解

- Change ID：`CHG-007-locations-tree`
- 依据：`proposal.md`、`design.md`
- 规则：任务尽量为可验证的垂直切片；每项完成后提交一次；门禁命令见 `design.md` §6.5。

## 1. 任务列表

### T1 迁移 0007 与模型

- **目标**：建立 `locations`、`character_locations`、`location_factions` 三表与索引；补 `Location` 模型、规范化函数与错误。
- **文件**：`internal/migrations/0007_locations.sql`、`internal/model/location.go`、`internal/model/{character,faction}.go`。
- **验收**：`go build ./...` 通过；迁移在空库 Up/Down 成功；`locations` 名称/类型/简介 CHECK 与软删除一致性约束存在；关系表唯一约束与索引存在。
- **依赖**：无。
- **验证**：`go build ./...`、`go vet ./...`、集成 `TestMigrationRoundTrip`。

### T2 地点仓储

- **目标**：实现 `repository/locations.go` 的读取、写入、计数、祖先/后代判断、关系读写与生命周期删除。
- **文件**：`internal/repository/locations.go`。
- **验收**：`Get/ListActive/Insert/UpdateWithVersion/SoftDelete/Restore` 可用；`IsDescendant` 对自身/子/深层后代返回 true；`BatchRelationCounts` 一次返回每地点人物/势力计数；`ReplaceCharacterLocations` 全量替换且去重；`PurgeByWork` 按深度倒序不触发 FK 违约；`InboundReferenceItems` 返回 `in_trash`。
- **依赖**：T1。
- **验证**：T3/T5/T8 集成用例间接覆盖；`go vet ./...`。

### T3 地点服务：CRUD、树与筛选

- **目标**：实现 `service/locations.go` 的 List/Get/Create/Update，含作品/父级校验、名称规范化与唯一、移动环检测、乐观锁、搜索/类型/待校对筛选补祖先、计数装配。
- **文件**：`internal/service/locations.go`、`internal/model/errors.go`。
- **验收**：三级创建成功；`Sword`/`sword` → `NAME_CONFLICT`；跨作品父级 → `CROSS_WORK_REFERENCE`；移动到自身/后代 → `CYCLE_DETECTED`；旧 `If-Match` → `VERSION_CONFLICT`；`q`/类型/待校对返回命中 + 祖先；计数正确。
- **依赖**：T2。
- **验证**：`internal/integration/locations_test.go` 前 5 组用例 PASS。

### T4 地点 HTTP 与路由

- **目标**：注册 `GET/POST /locations`、`GET/PATCH/DELETE /locations/{id}`，补齐错误映射与依赖注入。
- **文件**：`internal/httpapi/locations.go`、`internal/httpapi/{request,router}.go`、`cmd/server/main.go`、`internal/integration/suite_test.go`。
- **验收**：创建 201；PATCH 需 `If-Match`（缺失 428）；DELETE 返回 `undo_token`；错误码与 `design.md` §5 一致。
- **依赖**：T3。
- **验证**：集成 API 用例；`go test ./...`。

### T5 关系服务与端点

- **目标**：实现地点详情关系装配、`GET /locations/{id}/characters`、`GET/POST/PATCH/DELETE /locations/{id}/factions`。
- **文件**：`internal/service/locations.go`、`internal/repository/locations.go`、`internal/httpapi/locations.go`。
- **验收**：详情返回 parent/children/characters/factions；新增势力关系 201，重复 409 `DUPLICATE_RELATION`；备注 PATCH 生效；DELETE 只删关系不删势力；跨作品势力拒绝。
- **依赖**：T2、T4。
- **验证**：`internal/integration/location_relations_test.go` PASS。

### T6 人物接入真实地点

- **目标**：`location_links` 由占位改为全量替换的真实关系；人物详情返回地点名与备注；`location_id` 筛选启用；人物删除不阻止。
- **文件**：`internal/service/characters.go`、`internal/repository/characters.go`、`internal/model/character.go`。
- **验收**：创建/更新带地点关系成功；重复地点去重；跨作品/已删地点被拒绝；详情 `location_links` 名称与备注正确；人物删除后地点详情不再显示该关系；`location_id` 筛选只返回关联人物。
- **依赖**：T2、T5。
- **验证**：`internal/integration/location_relations_test.go` 人物分组 PASS；不回归 CHG-004 人物用例。

### T7 势力接入地点关系

- **目标**：势力详情 `relations.locations` 返回真实地点与备注；势力删除不阻止；永久删除前清理 `location_factions`。
- **文件**：`internal/service/factions.go`、`internal/repository/factions.go`、`internal/model/faction.go`。
- **验收**：势力详情显示关联地点与备注；地点详情显示该势力；势力软删除后地点详情不显示；势力永久删除后关系行无残留；删除势力不被地点关系阻止。
- **依赖**：T5。
- **验证**：`internal/integration/location_relations_test.go` 势力分组 PASS；不回归 CHG-006 势力用例。

### T8 生命周期与回收站

- **目标**：地点删除阻断矩阵、软删除/撤销/恢复/永久删除、回收站 `root_type=location`、作品整组联动。
- **文件**：`internal/service/{locations,trash}.go`、`internal/repository/{locations,works}.go`。
- **验收**：有下级（含软删）删除 `HAS_CHILDREN`；有人物/势力关系 `REFERENCE_CONFLICT` 且 `items[]` 带 `in_trash`；无引用删除返回 `undo_token`；8 秒内撤销恢复；恢复父级已删 `PARENT_DELETED`；作品组删除/恢复/清空覆盖地点；永久删除后无残留。
- **依赖**：T3、T4、T5、T6、T7。
- **验证**：`internal/integration/location_lifecycle_test.go` + 迁移带数据往返 PASS。

### T9 前端 API 与纯函数

- **目标**：`lib/api/locations.ts`、人物/势力/回收站类型扩展、树构建与过滤纯函数。
- **文件**：`lib/api/{locations,characters,factions,trash}.ts`、`lib/api/locations.test.ts`、`lib/locations-tree.ts`、`lib/locations-tree.test.ts`、`lib/messages.ts`。
- **验收**：类型检查通过；错误码映射（`CYCLE_DETECTED`/`HAS_CHILDREN`/`REFERENCE_CONFLICT`/`DUPLICATE_RELATION`）；组树/搜索/筛选/路径/类型选项/后代集合测试通过。
- **依赖**：契约冻结（T4/T5 接口形态确认）。
- **验证**：`npm run typecheck`、`npm run test`。

### T10 地点列表树与内联新建

- **目标**：`/locations` 页面：树、默认展开两级、搜索定位、类型筛选、待校对筛选、展开/收起、内联新建。
- **文件**：`app/locations/page.tsx`、`components/locations/{location-route,location-list-page,location-tree,location-inline-create}.tsx`、`styles/components.css`。
- **验收**：三级节点正确挂载；默认展开两级；搜索命中高亮并展开祖先；无匹配提示；类型筛选保留祖先链；新建后选中新节点；取消有输入需确认；空/加载/错误态完整。
- **依赖**：T9。
- **验证**：`npm run typecheck/lint`；E2E 1/2 通过。

### T11 地点详情、关系编辑与删除

- **目标**：`/locations?id=` 详情：面包屑、摘要条、简介、下级、关联人物/势力、内联编辑、移动禁用、删除对话框与引用阻断。
- **文件**：`components/locations/{location-detail-page,location-edit-form,location-relations,location-delete-dialog}.tsx`、`app/locations/page.tsx`。
- **验收**：首屏显示名称/类型/上级/下级数/人物数/势力数；编辑保存后树与详情同步；非法上级不可选且 422 提示；人物关系编辑通过人物 PATCH 保存且备注保留；势力关系新增/备注/移除生效；删除阻断分组列出下级/人物/势力并可跳转；成功后返回列表并可撤销。
- **依赖**：T5、T6、T7、T10。
- **验证**：E2E 3/4/5 通过。

### T12 人物/势力侧接入、导航与回收站

- **目标**：人物详情关联地点编辑与展示；势力详情关联地点区；导航「资料 > 地点」；回收站地点根与恢复失败提示。
- **文件**：`components/characters/{character-detail-page,character-form-fields,character-filters}.tsx`、`components/factions/faction-detail-page.tsx`、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`lib/api/*`。
- **验收**：人物详情可增删地点关系与备注，保存后地点详情同步；势力详情显示地点链接与备注；导航可达 `/locations`；回收站可筛选地点、恢复失败提示先恢复上级。
- **依赖**：T8、T11。
- **验证**：E2E 3/6 通过；`npm run test`。

### T13 视觉、a11y 与全量回归

- **目标**：1280/1024 视觉基线、a11y/键盘适配、空/错误/冲突状态覆盖、全量回归。
- **文件**：`e2e/{locations,visual,a11y,keyboard}.spec.ts`、视觉基线。
- **验收**：1280/1024 无横向溢出；axe 无 serious/critical；树节点键盘可达；全量 E2E 通过。
- **依赖**：T12。
- **验证**：`npx playwright test` 全量 PASS。

## 2. 依赖图

```text
T1 ── T2 ── T3 ── T4 ── T5 ── T6 ── T8
                │         └── T7 ──┘
                └── T9 ── T10 ── T11 ── T12 ── T13
```

## 3. 并行边界

- 后端 T2/T3 与前端 T9 可在 T4/T5 契约冻结后并行。
- T6 与 T7 可并行（不同服务文件），合并前统一跑集成。
- T10 依赖 T9；T11 依赖 T10 与 T5/T6/T7；T12 依赖 T8/T11；T13 最后执行。

## 4. 门禁

- 每个后端任务：`gofmt -l .`、`go vet ./...`、`go build ./...`、相关 `go test`。
- T4/T5/T8 后：全量集成。
- 每个前端任务：`npm run typecheck`、`npm run lint`、`npm run test`。
- T11/T13 后：Node 22 构建 + 全量 Playwright。
- F3 统一回填 `verification.md`；F4 独立审查；F6 全量回归。
