# CHG-007 验证报告

- Change ID：`CHG-007-locations-tree`
- Commit：`2c30376`（F2 实现头；F1 文档 `d8cce2a`）
- Environment：Windows 11 / Go 1.26 / Node 22（构建）/ Node 24（测试）/ PostgreSQL 16（Docker）
- Tester：主 Agent（F3 自测）
- Date：2026-11-14

## 1. 静态检查

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `gofmt -l .`（backend） | 0 | PASS | 无输出 |
| `go vet ./...` | 0 | PASS | 无告警 |
| `go build ./...` | 0 | PASS | 全部包编译通过 |
| `npm run typecheck` | 0 | PASS | tsc 无错误 |
| `npm run lint` | 0 | PASS | ESLint 无告警 |
| `npx -y node@22 node_modules/next/dist/bin/next build` | 0 | PASS | 静态导出 10 个页面，含 `/locations` |

## 2. 单元 / 集成测试

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `go test -count=1 ./...` | 0 | PASS | 含新增 `model/location_test.go`（名称/类型/简介/备注规范化） |
| `$env:TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | 0 | PASS | 全量集成通过，含迁移 v7 往返与 CHG-004/006 回归 |
| `npm run test` | 0 | PASS | 含新增 `locations-tree.test.ts`（组树/搜索补祖先/类型/待校对/路径/下拉/后代） |

新增集成用例：

- `TestLocationTreeCRUD`：三级创建、重名 409、搜索补祖先、类型筛选补祖先。
- `TestLocationParentValidation`：跨作品父级 422、自我/后代移动 422、`If-Match` 428/409、改名与清空上级。
- `TestLocationDeleteBlockedByChildren`：有下级 409、叶子删除返回 `undo_token`。
- `TestLocationCharacterRelation`：创建带地点关系、重复去重、备注同步、`location_id` 筛选、跨作品 422、人物删除不阻止且关系隐式隐藏、地点删除阻断并标记 `in_trash`。
- `TestLocationFactionRelation`：新增/重复 409/跨作品 422/备注 PATCH、地点与势力详情双向一致、地点删除阻断、移除关系后可删除。
- `TestLocationTrashUndoRestorePurge`：回收站地点根、父级显示作品名、撤销、恢复、永久删除无残留。
- `TestLocationRestoreBlockedByDeletedParent`：父级在回收站 → `PARENT_DELETED`。
- `TestWorkLifecycleCoversLocations`：作品组删除隐藏地点、恢复后关系完整、永久删除后 `locations`/`location_factions`/`character_locations` 均无残留。
- `TestPurgeAllIncludesLocations`：清空回收站包含地点根。

迁移：`TestMigrationRoundTrip` 升级到 v7，验证 `0007` Up/Down 三表可逆。

## 3. E2E 场景

| Scenario | Steps | Expected | Actual | Pass/Fail | Evidence |
|---|---|---|---|---|---|
| 三级内联新建 | `/locations` 依次新建东荒、青丘山、狐族秘境 | 层级正确，默认展开两级，新建后自动展开 | 符合 | PASS | `e2e/locations.spec.ts` 用例 1 |
| 搜索与类型筛选 | 搜索「狐族秘境」、类型筛选「秘境」、重置 | 命中高亮、展开祖先、保留祖先链 | 符合 | PASS | 同上 |
| 重名提示 | 新建同名地点 | 字段级提示「名称已存在，请换一个名称」 | 符合 | PASS | 同上 |
| 人物关系双向 | 地点页改备注「祖地」，跳人物详情 | 人物详情显示地点链接与备注 | 符合 | PASS | `locations.spec.ts` 用例 2 |
| 势力关系双向 | 地点页新增势力关系「宗门驻地」，跳势力详情 | 势力详情显示地点链接与备注 | 符合 | PASS | 同上 |
| 编辑与非法上级 | 编辑地点，检查自身选项禁用，保存简介 | 自身/后代不可选，保存后详情同步 | 符合 | PASS | 同上 |
| 删除被下级阻断 | 删除有下级的地点 | Dialog 提示「仍有下级地点」，列出下级链接 | 符合 | PASS | `locations.spec.ts` 用例 3 |
| 删除被人物阻断并恢复 | 删除有人物关系地点，移除关系后删除，回收站恢复 | 引用清单可跳转，删除成功，恢复成功 | 符合 | PASS | `locations.spec.ts` 用例 4 |
| 视觉基线 | 1280/1024 地点树、1280 地点详情 | 与基线一致 | 符合 | PASS | `locations-1280.png`、`locations-1024.png`、`location-detail-1280.png` |
| a11y | `/locations` axe 扫描 | 无 serious/critical | 符合 | PASS | `a11y.spec.ts` 地点用例 |
| 键盘 | 导航 Tab 顺序、树展开按钮 Enter 切换 | 焦点可见、层级显隐正确 | 符合 | PASS | `keyboard.spec.ts` 地点树用例 |

全量回归：`npx playwright test` → **85 passed**（1.2m），含 CHG-001~006 全部用例。

## 4. F6 全量回归（最新提交 `7f0b4c1`）

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `go vet ./...` | 0 | PASS | 无输出 |
| `go build ./...` | 0 | PASS | 全部包编译通过 |
| `go test -count=1 ./...` | 0 | PASS | 含 `model/location_test.go` |
| `go test -count=1 -tags=integration ./internal/integration/...` | 0 | PASS | 10.3s |
| `npm run typecheck` | 0 | PASS | 无错误 |
| `npm run lint` | 0 | PASS | 无告警 |
| `npm run test` | 0 | PASS | 10 文件 / 70 用例 |
| `next build`（Node 22） | 0 | PASS | 10 页面静态导出，`/locations` 9.47 kB |
| `docker compose build app` | 0 | PASS | 镜像 `novel-settings:dev` 重建成功 |
| `docker compose up -d app` | 0 | PASS | 容器重建并 healthy |
| `npx playwright test --reporter=list` | 0 | PASS | **86 passed**（1.2m），含地点 4 例、视觉 3 例、a11y/键盘各 1 例 |

结论：F6 全量门禁 PASS，无回归。`gofmt -l .` 在 Windows 下因 CRLF 对既有文件有历史漂移，仅对本次改动文件执行 `gofmt -w`，无内容差异。

## 5. 审查发现

F4 独立审查结果见 `review.md`；F5 修复项与复审结论在 `review.md` 与 `tasks.md` 回填。

## 6. 已知限制

- `POST /locations/{id}/review` 按 D1 延后至 CHG-011：本轮只读 `review_status`、支持筛选与 Badge，无确认按钮。
- 地点列表一次取全量（≤1,000 节点），不做虚拟滚动；与 CHG-006 势力树一致。
- 删除阻断清单单次最多 50 条（`ReferenceListLimit`），超出部分不展示但阻断仍生效。
- 地点详情页编辑人物关系时，逐个调用人物 `PATCH`；人物数量很多时保存为串行请求。

## 7. 回滚验证

- `0007` Down 可删除 `location_factions`、`character_locations`、`locations`，不影响人物/势力本体；`TestMigrationRoundTrip` 已覆盖。
- 代码回滚到 `d8cce2a` 可移除地点入口；已有数据从删除前快照恢复。
- 回收站永久删除前生成 `trash_purge` 快照，失败即阻止。

## 8. F9 合并与 Smoke

| 步骤 | 命令/操作 | 结果 |
|---|---|---|
| 分支状态 | `git status -sb` | `main` 领先 `origin/main` 14 个提交；无分支合并冲突（本次直接在 `main` 分阶段提交） |
| 迁移版本 | `docker compose exec -T db psql -U novel -d novel -c "select max(version_id) from goose_db_version;"` | `7` |
| Smoke | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\smoke.ps1` | `SMOKE OK: version=0.1.0 db=ok cache=ok` |
| 推送 | `git push origin main` | `f977bbd..ef26638  main -> main` |

## 9. 结论

PASS。F3 自测、F4 独立审查（S1 全部修复）、F5 复审、F6 全量回归（86 passed）、F7 人工验收（ACCEPT）、F9 Smoke 与推送均通过。
