# CHG-008 设计

- Change ID：`CHG-008-rankings-seats`
- 依据：`proposal.md`、SPEC-RANK-001~004、PG-08、PG-09、`api-contract.md` §3.6、`data-model.md` §3.6/§3.7
- 决策：D1~D6（见 proposal §Human Approval）；技术模式沿用 CHG-006/007（软删除三件套、`If-Match`、8 秒撤销、回收站 `root_type`、事务内 advisory lock）

## 1. 总体方案

1. 迁移 `0008_rankings.sql` 新建 `rankings`、`ranking_seats` 及索引、CHECK、部分唯一索引。
2. 后端新增 `model/ranking.go`、`repository/rankings.go`、`service/rankings.go`、`httpapi/rankings.go`：榜单 CRUD、固定席位、指派/移动/交换/移出、备注、扩容/缩容、软删除与撤销。
3. 生命周期：回收站新增 `root_type=ranking`，作品整组删除/恢复/永久删除/清空覆盖榜单与席位；永久删除先删席位再删榜单，避免 `ranking_seats.character_id RESTRICT` 阻断人物删除。
4. 人物接入：`characterInboundReferences` 追加席位引用（含回收站榜单）；`CharacterSummary`/`CharacterDetail` 增加榜单摘要，人物列表、人物详情、势力成员表三处回显。
5. 前端新增 `/rankings`（列表 + 新建）与 `/rankings?id=`（详情 + 席位表格 + 四个对话框），接入导航、人物、势力与回收站。
6. 迁移可逆；F10 回写 Spec、契约、数据模型与 roadmap。

## 2. 数据模型与迁移

### 2.1 `0008_rankings.sql`（Up）

```sql
CREATE TABLE rankings (
    id                TEXT PRIMARY KEY,
    work_id           TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    name_key          TEXT NOT NULL,
    seat_count        INTEGER NOT NULL DEFAULT 100,
    review_status     TEXT NOT NULL DEFAULT 'reviewed'
                      CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at       TIMESTAMPTZ,
    reviewed_by       TEXT DEFAULT 'local-author',
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    deleted_root_id   TEXT,
    deleted_by_action TEXT,
    CONSTRAINT rankings_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT rankings_seat_count_range CHECK (seat_count BETWEEN 1 AND 200),
    CONSTRAINT rankings_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX rankings_active_name_uq
    ON rankings (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX rankings_active_work_idx
    ON rankings (work_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX rankings_deleted_root_idx
    ON rankings (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE ranking_seats (
    id           TEXT PRIMARY KEY,
    work_id      TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    ranking_id   TEXT NOT NULL REFERENCES rankings (id) ON DELETE CASCADE,
    seat_no      INTEGER NOT NULL,
    character_id TEXT REFERENCES characters (id) ON DELETE RESTRICT,
    note         TEXT,
    version      INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ranking_seats_no_range CHECK (seat_no BETWEEN 1 AND 200),
    CONSTRAINT ranking_seats_note_len CHECK (note IS NULL OR char_length(note) <= 500),
    CONSTRAINT seats_no_uq UNIQUE (ranking_id, seat_no)
);
CREATE UNIQUE INDEX seats_character_uq
    ON ranking_seats (ranking_id, character_id) WHERE character_id IS NOT NULL;
CREATE INDEX ranking_seats_character_idx
    ON ranking_seats (character_id) WHERE character_id IS NOT NULL;
CREATE INDEX ranking_seats_work_idx ON ranking_seats (work_id);
```

### 2.2 Down

```sql
DROP TABLE IF EXISTS ranking_seats;
DROP TABLE IF EXISTS rankings;
```

> Down 丢弃榜单与席位（数据损失），已在 proposal §Risks 记录；Down 仅用于集成往返，迁移前应生成快照。

### 2.3 约束与语义

- 席位无软删除列：可见性由 `rankings.deleted_at` 过滤（ADR-004/data-model §3.7）；删除榜单保留席位行，恢复原样呈现。
- `seats_no_uq` 保证 1..N 唯一；`seats_character_uq` 保证同一人物同榜仅 1 席。
- `ranking_seats.character_id` 为 `RESTRICT`：人物永久删除前必须先删除其席位（正常入口会被引用检查阻止）。
- `ranking_id` 为 `CASCADE`：榜单永久删除自动清理席位。
- 席位数与席位行必须一致：建榜/扩容生成行，缩容删行，均在榜单行锁事务内完成。

## 3. 后端模块

### 3.1 `model/ranking.go`

- 常量：`MaxRankingName = 200`、`MaxRankingSeats = 200`、`DefaultRankingSeats = 100`、`MaxSeatNote = 500`、`RankingActionDelete = "ranking_delete"`。
- `Ranking`（GORM，表 `rankings`）：`ID/WorkID/Name/NameKey/SeatCount/ReviewStatus/ReviewedAt/ReviewedBy/Version/时间/软删三件套`。
- `RankingSeat`（GORM，表 `ranking_seats`）：`ID/WorkID/RankingID/SeatNo/CharacterID/Note/Version/时间`。
- DTO：
  - `RankingTopSeat{SeatNo,Character *CharacterBrief}`；
  - `RankingSummary{ID,WorkID,Name,SeatCount,OccupiedCount,EmptyCount,TopSeats []RankingTopSeat,ReviewStatus,ReviewedAt,UpdatedAt,Version}`；
  - `SeatView{SeatNo,Character *CharacterSummary,Note *string,Version int}`（空席 `character=null`）；
  - `RankingSummaryCounts{OccupiedCount,EmptyCount,TopSeats}`；
  - `RankingDetail{RankingSummary, Seats SeatPage}`；
  - `SeatPage{Items []SeatView,Page,PageSize,Total int}`；
  - `RankingRefView{RankingID,Name,SeatNo,Note *string}`（人物侧回显）。
- `NormalizeRankingName`：`lower(trim)` 生成 `name_key`；空名/超 200 → `ValidationError{field:"name"}`。
- `NormalizeSeatNote`：`trim` 后空串 → nil；>500 → `ValidationError{field:"note"}`。
- `model/errors.go` 新增：
  - `SeatConflictError{SeatNo, ExistingSeatNo int, CharacterID string}`；
  - `SeatOutOfRangeError{SeatNo, SeatCount int}`；
  - `SeatShrinkOccupiedError{FromSeatCount, ToSeatCount int, Items []SeatOccupant}`，`SeatOccupant{SeatNo int, CharacterID, Name string}`；
  - `ReferenceMissingError{Items []ReferenceItem}`。
- `model/character.go`：`CharacterSummary` 增 `Rankings []RankingRefView`、`RankingCount int`；`CharacterDetail` 增 `Rankings []RankingRefView`。

### 3.2 `repository/rankings.go`

- 榜单读：`Get`、`GetForUpdate`、`GetDeleted`、`ListActive(workID, q, reviewStatus, sort, page, pageSize)`、`CountActive`、`NameTaken`、`ListDeletedRoots`。
- 榜单写：`Insert`、`UpdateWithVersion`（名称/席位数，`version+1`）、`SoftDelete(id, action, version, now)`、`Restore`、`RenameDeleted`。
- 席位读：`SeatsPage(rankingID, page, pageSize, occupied *bool)`（`seat_no` 升序）、`SeatByNo(rankingID, seatNo)`、`SeatByNoForUpdate`、`SeatsForCharacter(characterID)`（联表榜单，含回收站，用于删除阻断）、`OccupiedAbove(rankingID, target)`（缩容占用明细）。
- 席位写：`InsertSeatsRange(rankingID, workID, from, to)`（`generate_series` + `ON CONFLICT (ranking_id, seat_no) DO NOTHING`）、`UpdateSeatWithVersion`、`ClearSeatCharacter`、`SetSeatCharacter`、`DeleteSeatsAbove(rankingID, target)`、`ConflictSeatNo(rankingID, characterID, excludeSeat)`。
- 批量：`RankingBriefsByCharacters(ids, limit)`（窗口函数 `row_number()/count(*)`，只取活跃榜单，供 `Characters.Summaries`）。
- 生命周期：`MarkWorkDeleted`、`RestoreByWork`、`Purge(rankingID)`（先删 seats 再删 ranking）、`PurgeByWork`、`PurgeByDeletedWorks`。

### 3.3 `service/rankings.go`

- `NewRankings(db, characters)`；`RootTypeRanking = "ranking"`。
- `List`：`work_id` 必填；`q` 300ms 由前端 debounce；`review_status`；`sort` ∈ `updated_at:desc`（默认）/`name:asc`；分页默认 20、最大 100；装配 `occupied_count/empty_count/top_seats`。
- `Create`：作品必须活跃；名称规范化与唯一；`seat_count` 1–200；同事务 `Insert` + `InsertSeatsRange(1..N)`；返回 `RankingDetail` 首页 50 席。
- `Get`：`page_size` 仅 50/100（其他 422 `VALIDATION_ERROR.field=page_size`），默认 50；席位人物批量走 `Characters.Summaries`；人物记录异常缺失时 `character=null` 且 `note` 保留。
- `Update`：`If-Match` 必填；名称/`seat_count` 子集；锁顺序：作品行锁 → `pg_advisory_xact_lock(hashtext(ranking_id))` → 榜单行锁；扩容追加；缩容先 `OccupiedAbove`，命中 `SEAT_SHRINK_OCCUPIED`，否则删行并更新。
- `Delete`：`If-Match` 可选（携带则校验）；锁后软删，返回 8 秒 `undo_token`，席位保留。
- `PutSeat`：`If-Match` 为席位 version；`character_id` 省略不变、`null` 清空并清备注、字符串则校验人物同作品活跃；重复指派 → `SEAT_CONFLICT`（带 `existing_seat_no`）；备注与人物同一条 UPDATE；返回新席位。
- `MoveSeat`：`mode` ∈ `empty|swap`；源席位 version 校验；目标越界 `SEAT_OUT_OF_RANGE`；`empty` 且目标已占 `SEAT_CONFLICT`；交换先两席 `character_id=NULL`，再分别回写，原子提交。
- `ListSeats`：`page/page_size`（默认 50、允许 50/100）、`occupied` 透传。
- 锁顺序统一：作品行锁 → ranking advisory → 榜单行锁 → 席位 `FOR UPDATE` → 人物 `FOR SHARE`（校验活跃与作品）。

### 3.4 生命周期与回收站（`service/trash.go`、`repository/works.go`）

- `works.go`：
  - `SoftDeleteGroup` 在 locations 块后追加 `rankings` 软删（`deleted_root_id=work_id`、`version+1`）。
  - `RestoreGroup` 在 locations 后按 `deleted_root_id=work_id` 恢复。
  - `purgeWorkChildren`：在 `character_names` 删除后、`characters` 删除前执行 `DELETE FROM ranking_seats WHERE work_id=?` 与 `DELETE FROM rankings WHERE work_id=?`。
  - `PurgeAllDeleted`：解除首领后、删 characters 前用同一 scope 删除 seats/rankings。
- `trash.go`：
  - `List` 白名单加 `ranking`；`TrashItem{root_type:"ranking",parent_name:null,affected_count:1}`。
  - `Restore` 加 `restoreRanking`：作品在线（否则 `PARENT_DELETED`）、同名冲突（`RESTORE_CONFLICT`，支持 `new_name`）、席位人物已删 → `REFERENCE_MISSING`（`items` 含人物与席位）；成功清榜单三件套。
  - `Purge` 加 ranking 分支：先快照，事务内复核引用（人物是否已删）后 `Purge`。
  - `Undo` 加 ranking 分支：8 秒内恢复榜单三件套，席位原样。
  - `PurgeAll`：回收站榜单先于人物清理；统计与计数纳入。
- 人物引用：`characterInboundReferences` 追加 `SeatsForCharacter`（不过滤榜单 `deleted_at`），`ReferenceItem{Type:"ranking_seat",EntityID:榜单ID,Name:榜单名,Field:"seat_no",Value:strconv.Itoa(seatNo),InTrash:榜单已删}`。
- 人物 `Purge` 前复核同一引用，避免 FK RESTRICT 触发 500。

### 3.5 `httpapi/rankings.go`

- 注册：`GET/POST /rankings`、`GET/PATCH/DELETE /rankings/:id`、`GET /rankings/:id/seats`、`PUT /rankings/:id/seats/:seat_no`、`POST /rankings/:id/seats/:seat_no/move`。
- 请求体：创建 `{work_id,name,seat_count}`；更新 `{name?,seat_count?}` + `If-Match`；席位 `{character_id?,note?}` + `If-Match`；移动 `{target_seat_no,mode}` + `If-Match`。
- `seat_no` 解析失败 → 400 `SEAT_OUT_OF_RANGE`；`If-Match` 复用 `ParseIfMatch`。
- `request.go` 新增错误映射（见 §5）。

## 4. 前端设计

### 4.1 API 层与纯函数

- `lib/api/rankings.ts`：类型 `RankingSummary/RankingDetail/SeatView/SeatPage/RankingTopSeat`；`listRankings/getRanking/createRanking/updateRanking/deleteRanking/listSeats/putSeat/moveSeat`；错误码映射 `SEAT_CONFLICT/SEAT_OUT_OF_RANGE/SEAT_SHRINK_OCCUPIED/REFERENCE_MISSING/NAME_CONFLICT/VERSION_CONFLICT`。
- `lib/rankings.ts`：纯函数 `seatPage(seatNo, pageSize)`、`formatTopSeats(topSeats)`、`rankingRefLabel(ref)`、`capacityDelta`；配 `rankings.test.ts`。
- `lib/api/characters.ts`：`CharacterSummary`/`CharacterDetail` 增加 `rankings`、`ranking_count`。
- `lib/messages.ts`：榜单列表/详情、四个对话框、错误码、空态与撤销文案。

### 4.2 列表（PG-08）

- `app/rankings/page.tsx` + `RankingRoute`：`?id=` 决定列表或详情；无作品显示 PG-00 态。
- `RankingListPage`：页头（当前作品 + 总数 + 新建榜单）、FilterBar（搜索/待校对/排序/重置）、`RankingTable`、分页。
- 列：名称 + 待校对、席位数、已占、空席、前三席（`1 姓名 · 2 姓名 · 3 —`）、更新时间、行操作（查看/删除）。
- 新建 Dialog：名称 + 席位数（1–200，默认 100）；成功 Toast + 进入详情；失败保留输入。
- 删除：确认后整榜进回收站 + Toast 撤销；`?id=` 返回后保留筛选/排序/页码。

### 4.3 详情（PG-09）

- `RankingDetailPage`：面包屑、页头（名称 + 待校对 + 扩容 + 更多）、摘要条、工具栏（跳至席位 + 页码）、`SeatTable`、分页；内联改名。
- `SeatTable` 每页 50 行：席位号等宽、已占行（头像/姓名链接/境界·势力/备注/行菜单）、空席行（「虚位以待」+ 指派）。
- `JumpToSeat`：1..N 校验，`page=floor((n-1)/50)+1`，滚动并高亮 2s，焦点入行。
- `AssignSeatDialog`：搜索当前作品人物；已在本榜其他席时阻止并给出跳席；成功 Toast。
- `AdjustSeatDialog`：输入目标席位，实时提示「移动到空席」/「与 X 交换」；目标为当前席禁用。
- `CapacityDialog`：扩容/缩容；缩容占用时列出「席位号 + 姓名 + 链接」。
- `RemoveSeatDialog`：确认后清空席位；Toast 撤销用前端快照 + 再 PUT（409 时提示席位已变化）。
- `DeleteRankingDialog`：整榜删除；成功后回列表并 Toast 撤销。
- `VERSION_CONFLICT` 顶部 Alert；未保存守卫复用人物/地点模式。

### 4.4 人物、势力、回收站与导航

- 人物详情：替换「所在榜单」占位为 `rankings` 列表，链接 `/rankings?id=`；空态「未上榜」。
- 人物列表与势力成员表：`rankings` 最多 2 条，格式 `榜单名·第N席`，超出显示 `+N`。
- 导航「资料 > 榜单」（`ListOrdered` 图标，位于地点之后）。
- 回收站：类型筛选加「榜单」；恢复失败展示「请先恢复原席位人物」；`highlight` 定位复用现有逻辑。

## 5. 错误与边界

| 场景 | 错误码 | HTTP | details |
|---|---|---:|---|
| 名称空/超 200、席位数越界、备注超 500、page_size 非 50/100 | `VALIDATION_ERROR` | 422 | `field` |
| 同作品重名 | `NAME_CONFLICT` | 409 | — |
| 同一人物同榜已有席位 | `SEAT_CONFLICT` | 409 | `{seat_no,character_id,existing_seat_no}` |
| 席位号越界 / 非法 seat_no | `SEAT_OUT_OF_RANGE` | 400 | `{seat_no,seat_count}` |
| 缩容目标尾部有占用 | `SEAT_SHRINK_OCCUPIED` | 409 | `{from_seat_count,to_seat_count,items[{seat_no,character_id,name}]}` |
| 恢复时席位人物已删 | `REFERENCE_MISSING` | 409 | `{items:[{entity_type:"character",entity_id,name,field:"seat_no",value}]}` |
| 人物跨作品/已删 | `CROSS_WORK_REFERENCE` / `PARENT_DELETED` | 422 / 409 | `field=character_id` |
| 版本不一致 / 缺失 `If-Match` | `VERSION_CONFLICT` / `PRECONDITION_REQUIRED` | 409 / 428 | `current_version` |
| 恢复时作品/同名冲突 | `PARENT_DELETED` / `RESTORE_CONFLICT` | 409 / 422 | `items` |
| 人物仍被席位引用 | `REFERENCE_CONFLICT` | 409 | `items[]` 含 `ranking_seat`、`in_trash` |

## 6. 测试设计

### 6.1 后端单元

- 名称规范化与长度；席位数边界 1/200；备注空串转 nil；席位号解析；`seatPage` 等价逻辑在后端只覆盖分页偏移。
- 错误类型 details 序列化形状。

### 6.2 集成（真实 PostgreSQL，事务回滚）

- 迁移：`TestMigrationRoundTrip` v8；0008 Down 后两表消失、其他表保留；重跑 Up 幂等。
- 榜单：创建 100 席行数/序号、同名冲突（大小写）、0/201、列表 top3/占用/空席、排序/筛选/分页。
- 席位：指派/清空/备注；重复指派 `SEAT_CONFLICT`；move empty/swap 原子与唯一约束；越界 400；移出。
- 容量：扩容行连续；缩容纯空尾成功；尾部有人 `SEAT_SHRINK_OCCUPIED` 明细；目标 <1 越界。
- 生命周期：删除 → 回收站列表/撤销/恢复；恢复时人物已删 `REFERENCE_MISSING`；同名恢复冲突；永久删除清理席位；作品组删除/恢复/永久删除/清空无残留。
- 人物：席位阻断删除（含回收站榜单）、`details` 可定位；人物列表/详情/成员表回显一致。
- 并发：两请求同时交换/指派，串行化后仅一个成功或结果一致；缩容与指派不产生非法席位。

### 6.3 前端单测

- `lib/api/rankings.test.ts`：错误映射、`If-Match`、分页参数、请求方法。
- `lib/rankings.test.ts`：跳席页码、前三席格式化、名次摘要与 `+N`。

### 6.4 E2E（`e2e/rankings.spec.ts`）

1. 新建 100 席榜单 → 进入详情 → 列表显示 100/0/100；0 与 201 校验。
2. 指派人物到空席、移动到另一空席、交换、移出并撤销。
3. 同一人物同榜第二席被阻止并提示先移出/交换；同一人物上两榜。
4. 扩容 100→120；尾部第 145 席有人时缩容被阻止并列出占用人；清空后缩容成功。
5. 跳至席位 120 → 第 3 页并高亮；越界提示。
6. 删除榜单 → 回收站 → 恢复；先删榜再删人后恢复被阻止。
7. 人物详情/人物列表/成员表名次同步并可跳转。
8. 1024/1280 无横向溢出；axe 无 serious/critical；空席指派与对话框键盘可达。

### 6.5 门禁命令

```powershell
cd backend; gofmt -l .; go vet ./...; go build ./...; go test ./...
$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
go test -count=1 -tags=integration ./internal/integration/...
cd frontend; npm run typecheck; npm run lint; npm run test
npx -y node@22 node_modules/next/dist/bin/next build
npx playwright test
```

## 7. 兼容性与迁移

- `characters` 响应新增 `rankings`/`ranking_count`，属契约扩展，向后兼容。
- 回收站新增 `ranking` 类型；前端同步升级，旧客户端忽略未知类型。
- 榜单删除可选 `If-Match`，前端始终传；不破坏既有客户端。
- 快照/导出格式不变；导入与检索在后续变更接入。

## 8. 实施顺序与并行边界

1. T1 迁移 + 模型 → T2 仓储 → T3 服务 → T4 HTTP → T5 生命周期/人物引用。
2. T6 前端 API + 列表 → T7 详情与对话框 → T8 人物/势力/回收站/导航。
3. T9 集成测试与前端单测/E2E 可与 T7/T8 并行准备，最终统一跑全量。
4. 每个任务独立提交；T4 后跑一次后端集成，T5 后跑全量集成，T8 后跑前端门禁，T9 全量回归。

## 9. 文件级改动清单

**后端新增**：`internal/migrations/0008_rankings.sql`、`internal/model/ranking.go`、`internal/repository/rankings.go`、`internal/service/rankings.go`、`internal/httpapi/rankings.go`、`internal/integration/{ranking,ranking_seat,ranking_trash,character_ranking}_test.go`。

**后端修改**：`internal/model/{character,errors}.go`、`internal/repository/{works,characters}.go`、`internal/service/{characters,trash}.go`、`internal/httpapi/{request,router}.go`、`cmd/server/main.go`、`internal/integration/{suite,migrations}_test.go`。

**前端新增**：`app/rankings/page.tsx`、`components/rankings/{ranking-route,ranking-list-page,ranking-table,create-ranking-dialog,ranking-detail-page,seat-table,seat-row,jump-to-seat,assign-seat-dialog,adjust-seat-dialog,capacity-dialog,remove-seat-dialog,delete-ranking-dialog}.tsx`、`lib/api/rankings.ts`、`lib/api/rankings.test.ts`、`lib/rankings.ts`、`lib/rankings.test.ts`、`e2e/rankings.spec.ts`。

**前端修改**：`components/characters/{character-detail-page,character-table}.tsx`、`components/factions/member-table.tsx`、`components/shell/global-nav.tsx`、`components/trash/*`、`app/trash/page.tsx`、`lib/api/{characters,trash}.ts`、`lib/messages.ts`、`styles/components.css`、`e2e/{visual,a11y,keyboard}.spec.ts` 与视觉基线。

## 10. 待人工确认（F1）

见 proposal §Human Approval：D1 校对端点延后、D2 三处名次回显、D3 席位模型、D4 席位并发、D5 移出撤销、D6 删除榜单口径。批准后进入 F2。
