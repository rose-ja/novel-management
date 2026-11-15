# PR/MR 描述 — CHG-008 榜单与席位

> 状态：待推送（2026-11-14）。本次变更直接在 `main` 上按阶段提交，提交范围 `64921e1..e2ba647`（11 个提交，67 文件，+6542/−24）。**未创建远程 PR**（单人本地仓库流程）。
> 门禁：F3 自测 PASS、F4 独立审查 0×S0 / 5×S1 / 4×S2、F5 修复复审 **PASS**、F6 全量回归 **93 passed**、F7 人工验收 **ACCEPT**（用户跳过验收弹窗，按持续开发指令继续）。

## 1. 标题与目标

**feat: 榜单与席位（CHG-008）**

交付一条完整可走通的榜单垂直切片：创建榜单并生成 1..N 固定席位；指派、移动到空席、交换、移出人物；席位备注；扩容/缩容；人物详情、人物列表与势力成员表同步名次；整榜软删除、8 秒撤销、回收站恢复（恢复时校验原席位人物仍存在）；人物删除被「榜单席位」阻止（含回收站中的榜单）。删除榜单、恢复榜单与作品生命周期与 CHG-002/004/006/007 保持一致。

## 2. 关联

- Requirement：FR-17～FR-20、FR-34、FR-37、FR-47～FR-51、FR-57～FR-60、FR-66
- 验收：AC-06、AC-09、AC-13、AC-14；EX-17、EX-19、EX-20
- Spec：SPEC-RANK-001～004
- Page Spec：PG-08 榜单列表、PG-09 榜单详情、PG-12 回收站（榜单根）
- 契约：`project/api-contract.md` §3.6；`project/data-model.md` §3.6/§3.7（F10 回写实际形态）
- 变更文档：`changes/archive/CHG-008-rankings-seats/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 迁移 `0008_rankings`：`rankings`（软删三件套、`version`、席位数 1–200、活跃名称唯一部分索引、`deleted_root_id` 索引、CHECK）、`ranking_seats`（席位号 1–200、`UNIQUE(ranking_id,seat_no)`、部分唯一索引 `(ranking_id,character_id) WHERE character_id IS NOT NULL`、人物 FK RESTRICT、榜单 FK CASCADE）；Down 全部可逆。
- 榜单 CRUD：`GET/POST /rankings`、`GET/PATCH/DELETE /rankings/:id`；名称 `lower(trim)` 同作品唯一、1–200 字；默认 100 席；`If-Match` 乐观锁；删除返回 8 秒 `undo_token`。
- 席位：`GET /rankings/:id/seats`、`PUT /rankings/:id/seats/:seat_no`、`POST /rankings/:id/seats/:seat_no/move`；空席可见、备注 ≤500、同榜一人一席、移动到空席/交换、越界与缩容阻断 details。
- 容量：扩容追加空席；缩容仅删除尾部空席，占用时返回 `SEAT_SHRINK_OCCUPIED` 并列出占用人。
- 并发：`If-Match` 使用席位/榜单 `version`；锁顺序「作品行锁 → `pg_advisory_xact_lock(hashtext(ranking_id))` → 榜单行锁 → 席位行锁」；交换先双席置空再回写。
- 生命周期：作品整组删除/恢复/永久删除/清空覆盖榜单与席位，永久删除先删席位再删榜单；回收站 `root_type=ranking` 列表/恢复/撤销/永久删除。
- 人物引用：`characterInboundReferences` 追加 `ranking_seat`（不过滤回收站榜单，`in_trash` 标注）；人物永久删除前复核。
- 名次回显：人物详情、人物列表、势力成员表统一 `榜单名·第 N 席`（最多 2 条 + `+N`）。
- 前端：`/rankings` 列表与 `/rankings?id=` 详情、新建/删除/指派/调整/容量/移出对话框、跳席高亮、撤销 Toast、导航「资料 > 榜单」、回收站榜单筛选。
- 响应式/可访问性：1280/1024、axe 无 serious/critical、空席指派与对话框键盘可达、视觉基线。
- 文档：F10 回写 SPEC-RANK、`api-contract.md` §3.6、`data-model.md`、roadmap 进度。

**非目标（已登记延后）**

| 项目 | 承接 |
|---|---|
| 校对确认端点 `POST /rankings/{id}/review`（本轮只读 `review_status`） | CHG-011 |
| 榜单图片、榜单大事记、榜单引用体系 | 需求边界 |
| 自动排名、并列名次、拖拽排序、批量导入 | 需求边界 |
| 无限滚动/席位虚拟列表 | 性能专项 |
| 真实双连接席位并发自动化用例 | CHG-012 一致性收口 |
| 检索索引同步榜单 | CHG-009 |

## 4. 主要文件/模块变更

**后端**

- 迁移：`backend/internal/migrations/0008_rankings.sql`。
- 模型：`model/ranking.go`（实体、DTO、规范化与上限常量）、`model/errors.go`（席位冲突/越界/缩容/引用缺失）、`model/character.go`（`rankings`、`ranking_count`）。
- 仓储：`repository/rankings.go`（榜单读写、席位分页与写入、批量摘要、生命周期）、`repository/characters.go`（`ListByIDs`）、`repository/works.go`（组删除/恢复/永久删除/清空覆盖榜单）。
- 服务：`service/rankings.go`（CRUD、席位事务、扩容缩容、锁顺序）、`service/trash.go`（榜单根、撤销、恢复校验、永久删除、清空）、`service/characters.go`（名次批量装配与引用阻断）。
- HTTP：`httpapi/rankings.go`（8 个端点、`If-Match`、省略 vs null）、`httpapi/{request,router}.go`、`cmd/server/main.go`。
- 测试：`integration/{ranking,ranking_seat,ranking_trash,character_ranking,migrations}_test.go`。

**前端**

- API/纯函数：`lib/api/rankings.ts`、`lib/rankings.ts`（跳席页码、前三席格式化、名次标签、容量差）及单测。
- 组件：`components/rankings/{ranking-route,ranking-list-page,ranking-table,ranking-detail-page,seat-table,seat-row,jump-to-seat,create-ranking-dialog,delete-ranking-dialog,assign-seat-dialog,adjust-seat-dialog,capacity-dialog,remove-seat-dialog}.tsx`。
- 人物/势力接入：`components/characters/{character-ranking-links,character-table,character-detail-page,character-delete-dialog}.tsx`、`components/factions/member-table.tsx`。
- 页面/导航/回收站：`app/rankings/page.tsx`、`components/shell/global-nav.tsx`、`app/trash/page.tsx`、`components/trash/trash-dialogs.tsx`。
- 样式与文案：`styles/components.css`（榜单摘要/席位/对话框与 1024 媒体查询）、`lib/messages.ts`。
- 测试：`e2e/rankings.spec.ts`（3 例）、`visual.spec.ts`（榜单列表 1280/1024、详情 1280 基线）、`a11y.spec.ts`、`keyboard.spec.ts`、`e2e/helpers.ts`。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/rankings-1280-chromium-win32.png`、`rankings-1024-chromium-win32.png`、`ranking-detail-1280-chromium-win32.png`。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; go build ./...` | 通过 |
| `cd backend; go vet ./...` | 0 |
| `cd backend; go test -count=1 ./...` | 全部 `ok` |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | **104 passed**（10.7s） |
| `cd frontend; npm run typecheck` / `npm run lint` | 通过 / 0 warning |
| `cd frontend; npm run test` | **76 passed**（12 文件，含 `rankings` 4 项、`api/rankings` 3 项） |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 静态导出成功，新增 `/rankings` |
| `docker compose build app; docker compose up -d app` | 镜像重建、容器 healthy，迁移至 **v8** |
| `cd frontend; npx playwright test --reporter=list` | **93 passed** |
| `TestMigrationRoundTrip` | 0008 Down/Up 往返通过（v8） |

## 7. E2E 与人工验收证据

- E2E 榜单 3 例：新建→详情→指派→跳席/交换→移出撤销；尾席占用缩容阻断→清空→缩容→删除→回收站恢复；名称必填、0/201 与重名校验。
- 集成覆盖：迁移 v8、创建/唯一/校验/列表前三席、席位指派/重复/移动/交换/移出/备注、扩容/缩容阻断与 details、`If-Match` 428/409、榜单删除/撤销/回收站恢复、同名恢复冲突、席位人物缺失 `REFERENCE_MISSING`、人物删除阻断（含回收站榜单）、三处名次回显、作品删除/恢复/永久删除/清空无 FK 报错。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-14，用户跳过验收弹窗并继续）。
- 审查：`review.md` F4 发现 1 个存疑 S0（复核不成立并补回归用例）、5 个 S1、4 个 S2；F5 全部修复后复审 **PASS**。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0008 新建两张表与索引，不改动既有表；无数据回填。
- **兼容性**：`GET /characters` 与 `GET /characters/{id}` 新增 `rankings`/`ranking_count`（空数组/0）；`GET /factions/{id}/members` 成员对象同步新增；`/rankings` 为新增路由。
- **FK 约束**：`ranking_seats.character_id` 为 RESTRICT、`ranking_id` 为 CASCADE；所有 purge 路径先删 seats 再删 rankings。
- **并发**：统一锁顺序；交换先置空规避部分唯一索引；席位按号升序加锁避免交叉等待。
- **性能**：列表批量统计占用量与前三席；人物名次批量查询（窗口函数），无 N+1。
- **回滚方案**：回退本变更提交并重新部署；若已迁移执行 0008 Down（仅删两表，不影响人物/势力/地点）；或从操作前快照恢复。

## 9. Reviewer 重点关注

1. `service/rankings.go` 的锁顺序、交换置空顺序、扩容/缩容事务与唯一索引错误映射。
2. `repository/works.go` 的 `purgeWorkChildren` / `PurgeAllDeleted` 清理顺序（seats → rankings → characters）。
3. `service/trash.go` 的 `restoreRanking`（作品在线、同名冲突、席位人物缺失）与 `PurgeAll` 榜单循环对 0 行的容忍。
4. `service/characters.go` 的 `characterInboundReferences` 是否覆盖回收站榜单、Purge 前复核。
5. `httpapi/rankings.go` 的 `character_id`/`note` 省略 vs null、`seat_no` 解析错误、四类错误码 details。
6. 前端跨页调整的占用查询、跨页移出撤销、`VERSION_CONFLICT` 顶部提示。
7. 文档一致性：`api-contract.md` §3.6、`data-model.md`、SPEC-RANK 由 F10 回写。

## 10. 已知遗留（不阻断合并）

| ID | 内容 | 承接 |
|---|---|---|
| — | 真实双连接席位并发未做自动化用例（逻辑由行锁 + advisory 串行化） | CHG-012 |
| — | `POST /rankings/{id}/review` 未开放，仅展示待校对 Badge/筛选 | CHG-011 |
| — | 未保存离开确认不适用：本页无多字段表单，改名/备注均显式保存 | 接受 |
| — | 待办：榜单不做无限滚动/虚拟列表 | 性能专项 |
