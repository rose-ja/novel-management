# Verification Report

- Change ID: CHG-004-characters-core
- Commit: 功能分支 `feat/chg-004-characters-core`（T1～T14）
- Environment: Windows + PowerShell 5.1；Go 1.27.3（交叉编译 linux/amd64）；Node.js 22；PostgreSQL 16（容器 `novel-management-db-1`）；应用容器 `novel-management-app-1` @ `http://127.0.0.1:17860`
- Tester: Agent（F2 实施；F3 自测）
- Date: 2026-11-09

## 1. Static Checks

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd frontend; npm run typecheck` | 0 | 无输出 | `tsc --noEmit` |
| `cd frontend; npm run lint` | 0 | 无告警 | ESLint |
| `cd backend; gofmt -l`（LF 副本，65 个 `.go` 文件） | 0 | 无文件需格式化 | 逐文件写入临时 LF 副本后 `gofmt -l` |
| `cd backend; go vet ./...` | 0 | 无输出 | — |
| `cd backend; go build ./...` | 0 | 无输出 | — |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 0 | 8 个静态页面导出成功 | 本机默认 Node 24.22.1 在静态生成阶段报 `spawn UNKNOWN`，改用 Node 22 通过 |

## 2. Unit / Integration Tests

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd frontend; npm run test` | 0 | 6 个文件 / 43 个用例通过 | 含 `lib/api/characters.test.ts` 13 个用例 |
| `cd backend; go test ./...` | 0 | 全部包 ok（config/httpapi/model/service） | — |
| `TEST_DATABASE_URL=... go test -count=1 -v -tags=integration ./internal/integration/...` | 0 | 65 个顶层用例 PASS / 0 FAIL（5.982s） | 含人物 CRUD、列表、回收站、作品联动、引用、迁移往返、Purge 竞态 |
| 其中 `TestMigrationRoundTrip` | 0 | 播种人物+引用后回滚 0004，断言 refs=0，再迁移到版本 4 | 覆盖 DB-05「数据不残留」 |
| 其中 `TestCharacterPurgeSkipsRestoredRoot` | 0 | 恢复后调用 Purge 返回 0 行，人物/别名/引用完整 | C-01 竞态防护 |
| 其中 `TestCharacterListPerformance` | 0 | 1000 人列表 P95=4.844ms（断言 ≤500ms） | `character_list_test.go:211` |

## 3. E2E Scenarios

| Scenario | Steps | Expected | Actual | Pass/Fail | Evidence |
|---|---|---|---|---|---|
| 容器内迁移 | `docker compose up -d app`；查看日志 | `goose` 迁移到 0004 | `successfully migrated database to version: 4` | PASS | `novel-management-app-1` 启动日志 |
| 人物列表 | `characters.spec.ts` 列表 2 例 | 新建后进入详情、搜索别名、境界筛选、待校对、清除筛选、分页 | 全部通过（新建跳详情为 F5 修复后行为） | PASS | 7 passed |
| 人物详情 | `characters.spec.ts` 详情 5 例 | 内联保存、409 冲突固定文案、未保存守卫、删除撤销、回收站恢复、a11y、1024 溢出 | 全部通过 | PASS | 7 passed |
| 视觉基线 · 人物 | `npx playwright test e2e/visual.spec.ts -g "人物"` | 列表 1280/1024、详情 1280 基线生成 | 通过 | PASS | `characters-1280/1024`、`character-detail-1280` PNG |
| 无障碍 | `a11y.spec.ts` 含 `/characters` | axe 无 serious/critical | 6 个页面全部通过 | PASS | — |
| 键盘可达 | `keyboard.spec.ts` 侧栏遍历 | 概览 → 体系 → 人物 → 回收站 → 设置 | 通过 | PASS | 导航新增「人物」项 |
| 体系引用阻断 | `system.spec.ts`「引用阻断」 | 人物引用类别 → 停用/删除 409，details 含人物名与 `in_trash` | 通过 | PASS | 改为经 API 创建真实人物引用 |
| F3 UI 冒烟（临时 Playwright 用例，已删除） | 1280/1024 打开 `/characters` 列表、详情，进入编辑并 Esc 退出；监听 console/pageerror/response | 无横向溢出、无异常控制台或失败请求 | 两档宽度均通过；`responses=[]`、`pageerrors=[]`；仅 `/favicon.ico` 404（全站既有问题，见 §6） | PASS | Playwright 1 passed（1.1s）；临时用例运行后删除 |
| F6 QA 场景（临时 Playwright 用例，已删除） | EX-04 连点保存、EX-06 500 保留输入、DT-02 保存后列表同步、DT-04 人物 404、DT-05 空态、UI-02 无作品引导、UI-03 列表错误态、UI-06 Esc 关闭 Drawer 并归还焦点 | 每项符合预期 | 全部通过；列表错误态仅显示 Alert+重试，Drawer 关闭后焦点回到「新建人物」 | PASS | F6 QA 1 passed（3.2s），用后删除 |
| 全量 E2E · F6 回归 | `cd frontend; npx playwright test` | 全部通过 | 63 passed（47.2s） | PASS | 含人物 7 例、a11y、键盘、错误态、视觉基线、体系/回收站/作品 |

## 4. Review Findings

独立审查（`review.md`）结论 FAIL：1 项 S1（C-01）+ 14 项 S2。F5 处理如下。

| ID | Severity | Finding | Fix / Waiver | Retest |
|---|---|---|---|---|
| C-01 | S1 | Purge 未校验 `deleted_at`，与恢复并发会物理删除活跃人物 | 已修复：事务内 `FOR UPDATE` 复核根状态后删除 | `TestCharacterPurgeSkipsRestoredRoot` PASS；集成 65 passed |
| C-02 | S2 | 0004 Down 残留 `character_system_refs`，DB-05 证据不实 | 已修复：Down 清空 refs；迁移测试播种并断言 refs=0 | `TestMigrationRoundTrip` PASS |
| C-03 | S2 | 冲突提示暴露 version，未用规范对话框 | 部分修复：文案改为固定文案并移除 version；`ConflictDialog` 延后 | `characters.spec.ts` 冲突用例 PASS |
| C-04 | S2 | 列表行菜单/sticky/排序/chips 与 PG-02 不符 | 延后：超出 T11 范围；UI-05 已取消勾选 | — |
| C-05 | S2 | 新建后未进入详情 | 已修复：创建后跳转 `/characters?id=` | E2E 新建用例 PASS |
| C-06 | S2 | 删除入口与成功返回、引用对话框不符 | 延后：引用检查待 CHG-006/008 | — |
| C-07 | S2 | 撤销 Toast 无倒计时/超时提示 | 延后：已知简化 | — |
| C-08 | S2 | 字段级错误未定位 | 延后：UI-04 保持未勾选 | — |
| C-09 | S2 | 停用类别编辑态未灰显/可清除 | 延后：DT-03 保持未勾选 | — |
| C-10 | S2 | `faction_id: ""` 静默通过 | 已修复：非 null 即 422；新增创建/更新用例 | 集成用例 PASS |
| C-11 | S2 | 人物创建/恢复与作品软删 TOCTOU | 已修复（实现）：`Works.GetForUpdate` 行锁；并发测试待补 | 集成回归 PASS |
| C-12 | S2 | 回收站人物根作品名 N+1 | 延后：性能待办 | — |
| C-13 | S2 | 列表错误态仍展示旧数据 | 已修复：失败清空 `data` | 前端门禁 PASS |
| C-14 | S2 | data-model 三处漂移 | 延后至 F10 DOC-03 | — |
| C-15 | S2 | 证据不足（迁移数据、1024 列策略、E2E-02） | 部分修复：迁移已补数据；其余随 C-04 延后并记录 N/A | — |
| R-01 | S2 | 单条 Purge 与作品组 Purge 锁序相反，存在死锁风险 | 已修复：`purgeWorkChildren` 先锁定作品下人物行，统一锁序 | 集成 65 passed（真实交错的并发测试待补） |

## 5. Implementation Differences（对照 `design.md` §3.1 / D1～D9）

| 项 | 设计 | 实施 |
|---|---|---|
| `faction_id` | 势力引用 | 无外键的 `TEXT`，本轮仅接受 `null`（D1，CHG-006 补 FK） |
| `location_links` | 地点关联 | 仅接受空数组（D1，CHG-007 开放） |
| `review_status` | 校对闭环 | DTO 只读返回；确认端点延后 CHG-011（D2） |
| 列表列 | 势力/地点/榜单 | 隐藏势力/地点筛选；榜单列固定「—」（D4） |
| `CharacterSummary` | 摘要 | 含 `names[]` 与 `version` |
| `DELETE /characters/{id}` | 乐观锁 | 要求 `If-Match`，返回 `undo_token` |

## 6. Known Limitations

- 本机 Docker BuildKit 构建会话反复 `EOF`，本轮镜像由交叉编译 Linux 二进制 + `docker cp` + `docker commit` 生成（布局与 Dockerfile 一致）；`Dockerfile` 未改动，Linux CI 应可正常构建。
- 沙箱内执行 `npm run build` 会在 `next build` 静态导出阶段因 `VirtualAlloc failed` 崩溃（退出码 `-1073740791`）；在沙箱外重跑同一命令成功（8 个静态页面），判定为环境内存限制，非代码问题。
- 本机默认 Node 已升级到 24.22.1，`next build` 在 `Generating static pages` 阶段稳定报 `spawn UNKNOWN`；改用 `npx -y node@22` 执行同一构建脚本可成功。CI 与容器仍使用 Node 22（Dockerfile 未变），该问题不影响产物。
- 全站 `.ico` 请求 `/favicon.ico` 返回 404，导致浏览器控制台出现一条资源错误；该问题在人物功能之前已存在，不属于 CHG-004 范围，本轮记录但不修复。
- Docker Desktop 宿主端口转发偶发失效：`127.0.0.1:5432` 与 `127.0.0.1:17860` 会返回 `unexpected EOF`/`ECONNRESET`，容器内健康检查正常；`docker compose restart db|app` 后恢复。E2E/集成重跑均以重启后结果为准，非代码问题。
- 正文图片、势力/地点/榜单关系、校对确认分别由 CHG-005～008、011 承接。

## 7. Rollback Verification

- `TestMigrationRoundTrip`（0004 Down/Up）：PASS；回滚后重新迁移到版本 4。
- 数据回滚依赖既有快照机制（删除/清空前落盘，保留 3 份），未新增回滚路径；`TestSnapshotRetentionKeepsLatestThree`、`TestSnapshotFailureBlocksDelete` 通过。

## 8. Conclusion

READY FOR HUMAN ACCEPTANCE（F6 完成，等待验收人选择 ACCEPT / REWORK / REPLAN）

- F3/F5/F6 门禁全绿：gofmt / vet / build / 单测 / 65 个集成用例 / 前端 typecheck / lint / 43 单测 / Node 22 生产构建 / 全量 E2E 63 passed。
- F5 修复 S1（Purge 竞态）及 C-02/C-05/C-10/C-11/C-13；F6 修复 R-01（锁序）、UI-03（错误态互斥）、UI-06（Drawer 焦点归还）。
- 遗留 S2：C-03 对话框、C-04 列表完整交互、C-06 删除入口、C-07 倒计时、C-08 字段级错误、C-09 停用类别编辑态、C-12 回收站 N+1、C-14 文档漂移（F10）、C-11/C-01 真实并发测试，均在 `review.md`/`checklist.md` 如实记录。
- 人工验收脚本见 `acceptance.md`。
