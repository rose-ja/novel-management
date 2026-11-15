# CHG-008 独立审查与修复记录（F4/F5）

- Change ID：`CHG-008-rankings-seats`
- 日期：2026-11-14
- 审查方式：未参与实现的独立审查代理，对照 `design.md`、`tasks.md`、`checklist.md`、SPEC-RANK 与 API 契约阅读全部变更
- 结论：0 个确认的 S0；5 个 S1 与 4 个 S2 已全部处理；F6 全量回归通过

## 1. 审查发现与处置

| 编号 | 级别 | 发现 | 处置 | 状态 |
|---|---|---|---|---|
| R1 | S0（存疑） | 清空回收站时榜单快照可能已被作品组删除，`Purge` 返回 0 行被当作 `ALREADY_DELETED` | 复核 `PurgeAll` 榜单循环已忽略 0 行；补充回归用例「先单独删榜 → 再删作品 → `PURGE_ALL`」，实测 200 通过，确认原判断不成立 | 已澄清并加测试 |
| R2 | S1 | 显式 `seat_count:0` 被当成未传而建成 100 席 | `CreateRankingInput.SeatCount` 改为 `*int`，省略才用默认 100；`0/201` 均 422；补充集成用例 | 已修复 |
| R3 | S1 | 移出撤销只查第 1 页，51 席以后无法撤销 | 撤销时按 `seatPage(seat_no)` 调 `listSeats` 取目标页的最新 version 再回填 | 已修复 |
| R4 | S1 | 调整席位按当前页判断目标占用，跨页目标被误判为空席而提交 `mode=empty` | 目标不在当前页时 250ms 防抖查询目标页占用与姓名，再决定 `empty/swap` | 已修复 |
| R5 | S1 | 人物删除阻断未展示榜单席位号与定位链接 | `character-delete-dialog` 渲染 `ranking_seat` 类型：榜单链接 + 第 N 席 | 已修复 |
| R6 | S2 | 非数字 `seat_no` 的 details 缺 `seat_count` | 补齐 `seat_count: 0` | 已修复 |
| R7 | S2 | 唯一索引兜底分支解引用空指针、缺 `existing_seat_no` | 改为重新查询 `ConflictSeatNo` 再构造错误；无人物时返回哨兵错误 | 已修复 |
| R8 | S2 | 移动目标已占时复用「人物已在其他席位」文案 | 前端 `SEAT_CONFLICT` 文案改为「席位冲突：该人物已在其他席位，或目标席位已被占用」 | 已修复 |
| R9 | S2 | `formatTopSeats` 在席位不足 3 时虚构席位；撤销成功却提示「已调整」 | 仅格式化实际返回席位；新增 `rankingSeatRestored` 文案 | 已修复 |

## 2. 复审结论

- 并发与锁：作品行锁 → 榜单 advisory → 榜单行锁 → 席位行锁顺序一致；交换先清空双席，规避 `seats_character_uq`；席位按号升序加锁避免交叉等待。
- 一致性：`seat_count` 与席位行数在 Create/扩容/缩容同一事务内维护；缩容前 `OccupiedAbove` 校验并返回占用人。
- 生命周期：作品删除/恢复/永久删除、榜单删除/恢复/永久删除、清空回收站均先删 `ranking_seats` 再删 `rankings`，无 FK RESTRICT 阻断。
- 引用：人物入向引用覆盖回收站榜单，Purge 前复核；人物列表批量名次查询无 N+1，空数组统一序列化为 `[]`。
- HTTP：8 个端点、`If-Match` 428、越界 400、四类错误码与 details 形状符合 `design.md` §5。
- 前端：`character_id` 省略/null/字符串语义、If-Match、移出撤销快照、删除撤销与回收站恢复均覆盖。

## 3. 修复后回归证据

- 后端：`go build ./...`、`go vet ./...`、`go test -count=1 ./...` PASS；集成 104 PASS（含新增 `seat_count:0` 与 `PURGE_ALL` 用例）。
- 前端：`npm run typecheck`、`npm run lint`、`npm run test`（76 PASS）PASS；Node 22 `next build` 成功。
- E2E：`npx playwright test --reporter=list` **93 PASS**（含榜单 3 条、a11y 9 页、键盘 6 条、视觉基线 21 张）。
- 视觉基线：榜单列表 1280/1024、详情 1280 与修复后实现一致。
