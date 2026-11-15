# CHG-008 验证记录（F3）

- Change ID：`CHG-008-rankings-seats`
- 日期：2026-11-14
- 范围：T1–T9 全部实现（迁移 v8、后端榜单/席位/生命周期/人物引用、前端列表/详情/席位交互、集成/单测/E2E）
- 依据：`proposal.md`、`design.md`、`tasks.md`、`checklist.md`

## 1. 交付清单

| 任务 | 结果 | 关键文件 |
|---|---|---|
| T1 迁移与模型 | 完成 | `backend/internal/migrations/0008_rankings.sql`、`model/ranking.go`、`model/errors.go`、`model/character.go` |
| T2 仓储 | 完成 | `repository/rankings.go`、`repository/characters.go`（ListByIDs） |
| T3 服务 | 完成 | `service/rankings.go` |
| T4 HTTP | 完成 | `httpapi/rankings.go`、`httpapi/request.go`、`httpapi/router.go`、`cmd/server/main.go` |
| T5 生命周期/回收站/人物引用 | 完成 | `repository/works.go`、`service/trash.go`、`service/characters.go` |
| T6 前端 API 与列表 | 完成 | `lib/api/rankings.ts`、`lib/rankings.ts`、`app/rankings/page.tsx`、`components/rankings/*` |
| T7 详情与席位交互 | 完成 | `ranking-detail-page.tsx`、`seat-table.tsx`、`seat-row.tsx`、`jump-to-seat.tsx`、4 个席位对话框 |
| T8 人物/势力/回收站/导航 | 完成 | `character-ranking-links.tsx`、`character-table.tsx`、`member-table.tsx`、`global-nav.tsx`、`app/trash/page.tsx` |
| T9 测试与基线 | 完成 | 4 个集成文件、前端单测、`e2e/rankings.spec.ts`、视觉基线、a11y/键盘更新 |

## 2. 门禁结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| 后端构建与静态检查 | `go build ./...`、`go vet ./...`、`gofmt -l .` | PASS |
| 后端单元测试 | `go test -count=1 ./...` | PASS |
| 后端集成测试 | `TEST_DATABASE_URL=... go test -count=1 -tags=integration ./internal/integration/...` | **104 PASS / 0 FAIL**（10.9s） |
| 迁移往返 | 同上，`TestMigrationRoundTrip` | PASS（版本 8；`rankings`/`ranking_seats` Up/Down） |
| 前端类型检查 | `npm run typecheck` | PASS |
| 前端 lint | `npm run lint` | PASS |
| 前端单测 | `npm run test` | **76 PASS（12 文件）** |
| 前端构建 | `npx -y node@22 node_modules/next/dist/bin/next build` | PASS |
| E2E 全量 | `npx playwright test --reporter=list` | **93 PASS**（含新增 3 条榜单用例；F5 修复后复跑） |
| 榜单相关回归 | `playwright test rankings/a11y/keyboard` | **19 PASS**（样式修复后复跑） |
| 视觉基线 | `playwright test visual -g 榜单` | 3 张基线（列表 1280/1024、详情 1280）一致 |

## 3. 验收点证据

| 验收点 | 证据 |
|---|---|
| AC-06-1 100 席固定升序 | `TestRankingCreateListAndCapacity`：100 席、首屏 50、首/空席正确；E2E 详情 10 席全可见 |
| AC-06-2 0/201 阻止 | `TestRankingCreateListAndCapacity`（201 → 422）、E2E「名称必填、席位数 0/201」 |
| AC-06-3 名称唯一（大小写/空格） | `TestRankingCreateListAndCapacity`（`  封神榜  ` → 409 `NAME_CONFLICT`） |
| AC-06-4 指派/移动到空席 | `TestRankingSeatAssignMoveAndSwap`、E2E 第 1 席 → 第 5 席 |
| AC-06-5 交换 | 同用例，交换后第 5 席乙、第 7 席甲；E2E 交换断言 |
| AC-06-6 同榜重复指派 | 同用例 `SEAT_CONFLICT`，details `existing_seat_no=1` |
| AC-06-7 一人多榜 | `RankingBriefsByCharacters` 最多 2 条 + `ranking_count`；列表/详情回显用例 |
| AC-06-8 移出后隐藏 | E2E 移出 + 撤销；`character_ranking_test.go` 移出后可删除人物 |
| AC-06-9/10 扩容/尾部缩容 | `TestRankingCreateListAndCapacity`（100→120）、`TestRankingShrinkBlockedByOccupiedSeat`（清空后 10→4） |
| AC-06-11 缩容阻断 | 同用例，details 含 `from/to` 与 `items[{seat_no,name}]`；E2E 尾席清单可跳转 |
| AC-06-12 50 席分页/跳席 | `lib/rankings.test.ts`（1/50/51/120）、`page_size=20` → 422、`jump-to-seat` E2E |
| AC-09-1 三处名次回显 | `character_ranking_test.go`（列表 + 详情）、`CharacterRankingLinks` 复用成员表；E2E 通过 API 断言 |
| AC-09-2 席位人物信息 | `SeatCharacterCell` 渲染缩略头像、境界摘要与势力 |
| AC-13-1/2 回收站 | `TestRankingDeleteUndoAndRestore`（8 秒 undo、回收站恢复）、E2E 删除→回收站恢复 |
| AC-14-1 人物删除阻断 | `character_ranking_test.go`：活跃榜单 `in_trash=false`，回收站榜单 `in_trash=true` |
| AC-14-2 恢复缺人物 | `TestRankingRestoreConflictAndMissingCharacter`：`REFERENCE_MISSING`；同名 `RESTORE_CONFLICT` |
| AC-14-3/4 生命周期 | `TestRankingPurgeAndWorkLifecycle`：榜单永久删除清席位、作品删除/恢复/永久删除无 FK 报错 |
| 异常与 UI | E2E 空态/失败重试/删除确认/冲突提示；a11y 8 页 + 榜单详情无 serious/critical；键盘导航更新通过 |

## 4. 已确认决策落实

- D1：本轮只读 `review_status` 并显示待校对 Badge/筛选；`POST /rankings/{id}/review` 延后 CHG-011。
- D2：名次在人物详情、人物列表、势力成员表三处回显（最多 2 条 + `+N`）。
- D3：席位为固定行，空席保留可见；缩容仅删除尾部空席。
- D4：`If-Match` 使用席位/榜单 `version`；锁顺序为作品行锁 → 榜单 advisory → 榜单行锁 → 席位行锁；交换先双席置空再回写。
- D5：移出撤销为前端快照 + 再 PUT，席位已变化时提示「席位已发生变化，无法撤销移出」。
- D6：删除榜单为整榜软删除，席位保留，返回 8 秒 `undo_token`。

## 5. 实现说明与偏差

1. 席位移动（`mode=empty`）时备注随人物一并移动；交换（`mode=swap`）时备注留在原席位。与 design 中「交换仅回写人物」一致。
2. 未单独实现「未保存离开确认」：详情页没有多字段编辑表单，改名与备注均需显式保存，弹窗关闭即丢弃输入，不触发路由离开；与地点/人物页的守卫场景不同。
3. `GET /rankings/{id}` 的 `page_size` 仅接受 50/100，其他值返回 422 `VALIDATION_ERROR`（field=`page_size`）。
4. 作品整组删除的榜单不出现在「榜单」回收站筛选中（`deleted_root_id=work_id`），随作品恢复；只有单独删除的榜单作为 `root_type=ranking` 根出现。
5. 迁移 Down 会丢弃榜单与席位数据，仅用于测试往返；生产迁移前应生成快照。

## 6. F5 审查修复

独立审查后修复了 5 个 S1 与 4 个 S2：显式 0 席校验、跨页移出撤销、跨页调整占用判断、人物删除阻断定位、唯一索引兜底 details、`seat_no` 解析 details、目标已占文案、前三席格式化与撤销文案。详细记录见 `review.md`。修复后后端集成、前端门禁、Node 22 构建与全量 E2E（93 PASS）均已复跑。

## 7. 复现命令

```powershell
cd backend
gofmt -l .; go vet ./...; go build ./...; go test -count=1 ./...
$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
go test -count=1 -tags=integration ./internal/integration/...

cd ..\frontend
npm run typecheck; npm run lint; npm run test
npx -y node@22 node_modules/next/dist/bin/next build

cd ..
docker compose build app; docker compose up -d app
cd frontend
npx playwright test --reporter=list
```
