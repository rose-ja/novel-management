# CHG-008 任务拆解

- Change ID：`CHG-008-rankings-seats`
- 依据：`proposal.md`、`design.md`
- 规则：任务尽量为可验证的垂直切片；每项完成后提交一次；门禁命令见 `design.md` §6.5。

## 1. 任务列表

### T1 迁移 0008 与模型

- **目标**：建立 `rankings`、`ranking_seats` 两表与索引；补 `Ranking`/`RankingSeat` 模型、DTO、规范化函数与四类席位错误。
- **文件**：`backend/internal/migrations/0008_rankings.sql`、`backend/internal/model/{ranking,errors,character}.go`。
- **验收**：`go build ./...` 通过；空库 Up/Down 成功；`rankings_name_len`/`rankings_seat_count_range`/`rankings_deleted_consistent`/`seats_no_uq`/`seats_character_uq` 存在；DTO 字段与 `design.md` §3.1 一致。
- **依赖**：无。
- **验证**：`go build ./...`、`go vet ./...`、集成 `TestMigrationRoundTrip`（v8）。

### T2 榜单与席位仓储

- **目标**：实现 `repository/rankings.go` 的榜单读写、席位分页/写入、批量摘要与生命周期方法。
- **文件**：`backend/internal/repository/rankings.go`。
- **验收**：`Get/GetForUpdate/GetDeleted/ListActive/CountActive/Insert/UpdateWithVersion/SoftDelete/Restore/RenameDeleted/NameTaken/ListDeletedRoots` 可用；`SeatsPage` 按 `seat_no` 升序且支持 `occupied`；`InsertSeatsRange` 幂等；`ConflictSeatNo` 返回冲突席位；`OccupiedAbove` 返回占用人；`RankingBriefsByCharacters` 一次返回每人物最多 N 条与总数；`Purge` 先删席位再删榜单。
- **依赖**：T1。
- **验证**：T3/T9 集成用例间接覆盖；`go vet ./...`。

### T3 榜单服务：CRUD、席位事务与容量

- **目标**：实现 `service/rankings.go` 的 List/Create/Get/Update/Delete/PutSeat/MoveSeat/ListSeats；统一锁顺序；扩容/缩容校验。
- **文件**：`backend/internal/service/rankings.go`。
- **验收**：创建 100 席生成 1..100；同名（大小写）`NAME_CONFLICT`；0/201 `VALIDATION_ERROR`；重复指派 `SEAT_CONFLICT`；move empty/swap 原子；越界 `SEAT_OUT_OF_RANGE`；缩容占用 `SEAT_SHRINK_OCCUPIED` 且 details 含占用人；旧 `If-Match` `VERSION_CONFLICT`；删除返回 8 秒 `undo_token`。
- **依赖**：T2。
- **验证**：`backend/internal/integration/{ranking,ranking_seat}_test.go` PASS。

### T4 榜单 HTTP 与路由

- **目标**：注册 8 个端点，补齐请求解析、`If-Match`、错误映射与依赖注入。
- **文件**：`backend/internal/httpapi/rankings.go`、`backend/internal/httpapi/{request,router}.go`、`backend/cmd/server/main.go`、`backend/internal/integration/suite_test.go`。
- **验收**：创建 201 + `Location`；PATCH/PUT/move 200；DELETE 200；`seat_no` 非数字 400 `SEAT_OUT_OF_RANGE`；四类新错误码映射正确；无路由冲突。
- **依赖**：T3。
- **验证**：集成 API 用例；`go test ./...`。

### T5 生命周期、回收站与人物席位引用

- **目标**：作品整组联动；回收站 `root_type=ranking`；恢复校验人物存在；人物删除被席位阻断；人物摘要/详情回显榜单。
- **文件**：`backend/internal/repository/{works,characters}.go`、`backend/internal/service/{characters,trash}.go`、`backend/internal/model/character.go`。
- **验收**：作品删除/恢复/永久删除/清空无残留且无 FK 报错；删除榜单可撤销/恢复；恢复时人物已删 `REFERENCE_MISSING`；同名恢复 `RESTORE_CONFLICT`；人物删除 `REFERENCE_CONFLICT` 含 `ranking_seat`/`in_trash`；人物列表/详情/成员表回显一致。
- **依赖**：T3、T4。
- **验证**：`backend/internal/integration/{ranking_trash,character_ranking}_test.go` PASS；全量集成。

### T6 前端 API、榜单列表与新建

- **目标**：`lib/api/rankings.ts`、纯函数与 `/rankings` 列表页；新建 Dialog；删除 + 撤销。
- **文件**：`frontend/lib/api/rankings.ts`、`frontend/lib/api/rankings.test.ts`、`frontend/lib/rankings.ts`、`frontend/lib/rankings.test.ts`、`frontend/app/rankings/page.tsx`、`frontend/components/rankings/{ranking-route,ranking-list-page,ranking-table,create-ranking-dialog}.tsx`、`frontend/lib/messages.ts`。
- **验收**：列表列（名称/待校对/席位数/已占/空席/前三席/更新时间/操作）正确；搜索/筛选/排序/分页可用；0/201 字段错误；空/加载/错误/筛选空态完整；删除后 Toast 撤销。
- **依赖**：T4 契约冻结。
- **验证**：`npm run typecheck`、`npm run lint`、`npm run test`。

### T7 榜单详情、席位表格与对话框

- **目标**：`/rankings?id=` 详情：摘要、50 席分页、跳席、指派/调整/移出/扩容缩容/删除，内联改名与未保存守卫。
- **文件**：`frontend/components/rankings/{ranking-detail-page,seat-table,seat-row,jump-to-seat,assign-seat-dialog,adjust-seat-dialog,capacity-dialog,remove-seat-dialog,delete-ranking-dialog}.tsx`、`frontend/app/rankings/page.tsx`。
- **验收**：空席可见；指派/移动/交换/移出生效并可撤销；跳席页码与高亮正确；缩容占用列出人物可跳转；冲突/未保存/删除恢复状态完整。
- **依赖**：T6。
- **验证**：E2E 1~6 通过。

### T8 人物、势力、回收站与导航接入

- **目标**：人物详情/列表、势力成员表显示名次；导航「榜单」；回收站类型筛选与恢复失败提示。
- **文件**：`frontend/components/characters/{character-detail-page,character-table}.tsx`、`frontend/components/factions/member-table.tsx`、`frontend/components/shell/global-nav.tsx`、`frontend/app/trash/page.tsx`、`frontend/components/trash/*`、`frontend/lib/api/{characters,trash}.ts`、`frontend/lib/messages.ts`。
- **验收**：三处显示 `榜单名·第N席`（最多 2 条 + `+N`）并可跳转；导航可达 `/rankings`；回收站可筛选榜单、恢复失败提示先恢复人物。
- **依赖**：T5、T7。
- **验证**：E2E 6/7 通过；`npm run test`。

### T9 集成测试、前端单测与 E2E

- **目标**：补齐集成/单测/E2E、视觉基线、a11y 与键盘用例，完成全量回归。
- **文件**：`backend/internal/integration/{ranking,ranking_seat,ranking_trash,character_ranking}_test.go`、`backend/internal/integration/{suite,migrations}_test.go`、`frontend/e2e/rankings.spec.ts`、`frontend/e2e/{visual,a11y,keyboard}.spec.ts` 与视觉基线。
- **验收**：全量集成 PASS；前端单测 PASS；1280/1024 无横向溢出；axe 无 serious/critical；空席指派与对话框键盘可达；全量 Playwright PASS。
- **依赖**：T8。
- **验证**：`design.md` §6.5 全部门禁。

## 2. 依赖图

```text
T1 ── T2 ── T3 ── T4 ── T5 ──┐
                     └── T6 ── T7 ── T8 ── T9
```

## 3. 并行边界

- T5 与 T6 可在 T4 契约冻结后并行（后端生命周期 vs 前端列表）。
- T7 依赖 T6；T8 依赖 T5/T7；T9 最后执行。
- 后端集成用例可与 T6/T7 并行编写，合并前统一跑。

## 4. 门禁

- 每个后端任务：`gofmt -l .`、`go vet ./...`、`go build ./...`、相关 `go test`。
- T4/T5 后：全量集成。
- 每个前端任务：`npm run typecheck`、`npm run lint`、`npm run test`。
- T7/T9 后：Node 22 构建 + 全量 Playwright。
- F3 统一回填 `verification.md`；F4 独立审查；F6 全量回归。
