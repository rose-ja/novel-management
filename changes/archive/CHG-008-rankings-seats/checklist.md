# CHG-008 验收清单

- Change ID：`CHG-008-rankings-seats`
- 依据：`proposal.md` §Acceptance Criteria、SPEC-RANK-001~004、PG-08、PG-09
- 状态：F6 全量回归通过，待 F7 人工验收

## 1. 功能验收

- [x] AC-06-1 新建 100 席榜单：席位 1–100 固定升序、全部为空；列表显示 100/0/100（SPEC-RANK-001）
- [x] AC-06-2 席位数 0 或 201 被阻止并提示 1–200（EX-20）
- [x] AC-06-3 名称同作品唯一，`Sword`/`sword`/首尾空格视为重复（AC-17）
- [x] AC-06-4 指派到空席成功；移动到另一空席后源席变空、目标席占用（SPEC-RANK-002）
- [x] AC-06-5 与已占席位交换后两人互换且都不重复（FR-18）
- [x] AC-06-6 同一人物同榜第二席被阻止（`SEAT_CONFLICT`），提示先移出或交换（EX-19）
- [x] AC-06-7 同一人物可上两个榜单，人物详情显示两条名次（AC-06）
- [x] AC-06-8 移出榜单后席位变空，人物侧不再显示该榜（SPEC-RANK-002）
- [x] AC-06-9 扩容 100→120 后新增 20 个空席可见（SPEC-RANK-003）
- [x] AC-06-10 尾部全空时缩容成功且席位号连续（SPEC-RANK-003）
- [x] AC-06-11 尾部有人时缩容被阻止并列出占用人（EX-17）
- [x] AC-06-12 详情每页 50 席；跳至 120 计算为第 3 页并高亮；`page_size` 仅 50/100（RT-15）

## 2. 一致性与引用

- [x] AC-09-1 人物详情、人物列表、势力成员表显示 `榜单名·第N席`（最多 2 条 + `+N`）并可跳转（AC-09）
- [x] AC-09-2 榜单席位显示人物姓名/头像/境界/势力，与人物详情一致
- [x] AC-13-1 删除榜单进回收站，列表与人物侧隐藏；8 秒内可撤销（AC-13）
- [x] AC-13-2 回收站可筛选榜单、恢复席位与备注
- [x] AC-14-1 人物在榜单（含回收站中的榜单）任职时删除被阻止，可定位榜单与席位（AC-14）
- [x] AC-14-2 先删榜再删人后恢复榜单被阻止并提示先恢复人物（FR-20）
- [x] AC-14-3 永久删除榜单清理席位；作品永久删除/清空回收站无 FK 报错
- [x] AC-14-4 作品删除/恢复整组覆盖榜单与席位；两本作品榜单互不可见

## 3. 异常与 UI

- [x] 空列表、筛选无结果、加载骨架、加载失败重试、删除/提交中禁用均正确（AC-15）
- [x] 空席「虚位以待」文字存在，不只靠颜色/虚线；指派按钮有可读名称
- [x] `VERSION_CONFLICT` 显示顶部冲突提示，不展示内部字段；未保存离开有确认
- [x] 1024/1280 无横向溢出；席位表允许横向滚动且操作列可达
- [x] axe 无 serious/critical；空席指派、跳席、对话框键盘可达；焦点陷阱与 Esc 关闭正常
- [x] 无英文堆栈、无重复记录、无死链

## 4. 门禁与证据

- [x] `go build ./...`、`go vet ./...`、`go test -count=1 ./...` PASS
- [x] 集成 `go test -count=1 -tags=integration ./internal/integration/...` PASS（记录用例数）
- [x] `npm run typecheck`、`npm run lint`、`npm run test` PASS
- [x] Node 22 `next build` 成功
- [x] `npx playwright test --reporter=list` 全量 PASS（记录通过数）
- [x] 迁移 v8；`TestMigrationRoundTrip` 覆盖 Up/Down
- [x] F3 `verification.md`、F4 `review.md`、F6 全量回归证据齐全；F7 人工 ACCEPT

## 5. 文档回写（F10）

- [x] `project/specs/rankings/specs.md` 状态转 `implemented` 并补实现说明
- [x] `project/api-contract.md` §3.6 实现状态 + 人物 `rankings` 字段 + 变更历史
- [x] `project/data-model.md` 确认 `0008` 与席位语义
- [x] `project/roadmap.md` 进度与 CHG-008 状态
- [x] 变更目录归档到 `changes/archive/CHG-008-rankings-seats/`

