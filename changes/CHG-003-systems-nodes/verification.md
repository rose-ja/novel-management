# Verification Report

- Change ID: CHG-003-systems-nodes
- Commit: 分支 `feat/chg-003-systems-nodes`（T1`feat：CHG-003 体系迁移与领域模型` → T11`test：CHG-003 体系 E2E、视觉基线与无障碍用例`，共 11 次提交）
- Environment: Windows + PowerShell 5.1；Go 1.26；Node.js 22；PostgreSQL 16（容器 `novel-management-db-1`）；应用容器 `novel-settings:dev` @ `http://127.0.0.1:17860`；未在本机执行 `-race`（无 cgo）
- Tester: Agent（F3 自动验证）
- Date: 2026-11-09

## 1. Static Checks

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd backend; gofmt -l .` | 0 | 32 个 `.go` 文件被标记 | Windows 工作区 `core.autocrlf=true`，全部文件为 CRLF，属环境性误报；同一命令在 LF 归一化副本上复核无输出 |
| LF 副本 `gofmt -l <tmp>` | 0 | 无输出（PASS） | 修复 `httpapi/system.go`、`repository/works.go`、`service/system.go` 三处对齐后复核 |
| `cd backend; go vet ./...` | 0 | 无输出 | — |
| `cd backend; golangci-lint run ./...` | 0 | `0 issues.` | golangci-lint v2.15.0 |
| `cd backend; govulncheck ./...` | 0 | `No vulnerabilities found`（0 处受影响调用） | 依赖包内 2 个漏洞与模块 1 个漏洞未被调用 |
| `cd frontend; npm run lint` | 0 | 无告警 | ESLint |
| `cd frontend; npm run typecheck` | 0 | 无输出 | `tsc --noEmit` |

## 2. Unit / Integration Tests

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd backend; go test ./...` | 0 | 全部 ok | `config`、`httpapi`、`model`、`service` 通过 |
| `TEST_DATABASE_URL=postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable; cd backend; go test -tags=integration ./internal/integration/...` | 0 | `ok ... 9.398s` | 新增体系用例 28 个：`system_test.go`（7）、`system_nodes_test.go`（6）、`system_repo_test.go`（7）、`system_trash_test.go`（7）、`system_refs_test.go`（2）；含 `TestMigrationRoundTrip`（0003 Down/Up 往返） |
| `cd frontend; npm test` | 0 | 5 个文件 / 30 个用例通过 | 含 `lib/api/system.test.ts` 11 个用例 |

## 3. E2E Scenarios

| Scenario | Steps | Expected | Actual | Pass/Fail | Evidence |
|---|---|---|---|---|---|
| 镜像重建与新代码生效 | `docker compose build app`；`docker compose up -d app`；`GET /api/v1/health` | 健康且体系接口可用 | `{"status":"ok","db":"ok","cache":"ok"}` | PASS | 容器 `novel-management-app-1` healthy |
| 体系功能 E2E | `cd frontend; npx playwright test system.spec.ts` | 11 个用例通过 | 11 passed | PASS | 内置保护、类别 CRUD、10 上限、两级、同级重名、排序持久、停用只读、引用阻断、整组删除+`new_name` 恢复、节点删除/撤销、`PARENT_DELETED`、1024 视口 |
| 全量 E2E | `cd frontend; npx playwright test` | 全部通过 | 50 passed（36.6s） | PASS | 含既有壳层/错误态/键盘/作品/回收站/设置页用例与视觉基线 |
| 无障碍 | `a11y.spec.ts` 含 `/system` | axe 无 serious/critical | 5 个页面全部通过 | PASS | — |
| 视觉基线 | `visual.spec.ts` 新增 system 1280/1024 | 差异 ≤ 0.02 | 通过（`maxDiffPixelRatio 0.02`） | PASS | 新增 `system-1280-chromium-win32.png`、`system-1024-chromium-win32.png`；既有 5 张基线逐字节一致 |
| 容器内迁移与表结构 | `docker exec novel-management-db-1 psql -U novel -d novel ...` | 0003 已应用、三表存在、内置境界唯一 | `goose_db_version` 版本 3 且 `is_applied=t`；`system_categories`/`system_nodes`/`character_system_refs` 存在 | PASS | 唯一性查询：多内置作品 0 行、缺内置作品 0 行 |

## 4. Review Findings

| ID | Severity | Finding | Fix / Waiver | Retest |
|---|---|---|---|---|
| R-01 | High | 空节点集时 `AssembleNodeTree` 返回 nil slice，序列化为 `{"items":null}`，前端 `node-tree` 在 `items.some` 处崩溃，类别下无节点时整页白屏 | `model/system.go` 改为 `roots := make([]SystemNode, 0, len(nodes))` | `system.spec.ts` 11 passed；`go test ./internal/model/...` 通过 |
| R-02 | High | 随类别整组删除的节点 `deleted_root_id` 为类别 ID，`RenameDeletedNode` 按节点自身 ID 匹配导致 `renames[]` 改名恒失败并误报 `ALREADY_DELETED` | 新增 `repository.RenameDeletedNodeInGroup`，整组恢复路径改用它 | `TestSystemTrashRestoreCategory` 通过 |
| R-03 | Low | `httpapi/system.go`、`repository/works.go`、`service/system.go` 存在 gofmt 对齐偏差 | `gofmt -w` 修正 | LF 副本 `gofmt -l` 无输出 |
| R-04 | Low | 回收站永久删除类别/节点的集成测试在 T3 引用检查之后构造顺序错误，被 `REFERENCE_CONFLICT` 提前阻断 | 调整测试为「删除进回收站 → 补引用验证阻断 → 清理引用 → 永久删除」 | `TestSystemTrashPurgeCategoryAndNode` 通过 |

## 5. Known Limitations

- 导入模板与「确认校对」接口属 CHG-011：本轮只实现 `review_status=pending` 筛选与 DTO 字段，不提供确认操作。
- 人物表单联动与引用写入属 CHG-004：`character_system_refs` 本轮仅建骨架，`character_id` 无外键、不返回人物名称（`name` 恒为 `null`）。
- 名称唯一键：体系采用 NFKC + lower + trim + 连续空白折叠；`works` 仍为 `lower(trim)`，差异留 CHG-012。
- 类别列表不提供拖拽排序（符合 PG10-Q4）。
- 本地 `gofmt -l .` 受 Windows CRLF 影响不可作为门禁，以 LF 副本或 Linux CI 为准。
- 未执行 `go test -race`（本机无 cgo 工具链）。
- `scripts/ci.ps1` 在本机于 gofmt 步骤中止（同一 CRLF 原因），其余步骤已逐条单独执行并全部通过。

## 6. Rollback Verification

- `TestMigrationRoundTrip`：`migrations.Down` 回滚 0003 后 `system_nodes`、`character_system_refs` 消失且 `works`/`system_categories` 保留；再回滚 0002 后全部新表消失；两次 `Up` 后表结构恢复且 `app_settings` 仅 1 行。PASS。
- 数据回滚依赖既有快照机制（删除/清空前落盘，保留 3 份），未新增回滚路径。

## 7. Conclusion

PASS

（`gofmt -l .` 的 32 个文件标记为 Windows CRLF 环境性误报，已在 LF 归一化副本上验证格式正确；其余门禁与容器核对全部通过。）

## 8. F5 Review Findings Fix & Retest

- 输入：`review.md` 的 F4 结论（FAIL：S0=0、S1=3、S2=6）
- 修复范围：CR-01～CR-06、CR-08、CR-09；CR-07 记录为已接受的性能类 S2
- 执行人：Agent（F5 修复）

### 8.1 修复项与重测

| Finding | Severity | Fix summary | Retest command | Result |
|---|---|---|---|---|
| CR-01 | S1 | 回收站按 `root_type` 渲染类型/名称/父级与筛选；恢复/永久删除文案按根类型 | `npx playwright test`（含 `e2e/trash-system.spec.ts` 2 用例） | PASS（52 passed） |
| CR-02 | S1 | `Works.ActiveName` 仅返回未删除作品；恢复/撤销类别时作品在回收站返回 `PARENT_DELETED` | `go test -tags=integration ./internal/integration/...`（`TestSystemTrashCategoryRequiresActiveWork`） | PASS |
| CR-03 | S1 | `PurgeNode` 检查含软删子节点并返回 `HAS_CHILDREN`；`PurgeAll` 子先父后；`PurgeCategoryGroup` 先子行后根行 | 集成（`TestSystemTrashPurgeNodeWithDeletedChild`、`TestSystemTrashPurgeAllParentChild`、`TestSystemTrashPurgeCategoryWithTwoLevels`） | PASS |
| CR-04 | S2 | 恢复类别按最终名称做同级去重，并与活跃节点比对，冲突返回 `RESTORE_CONFLICT` | 集成（`TestSystemTrashRestoreCategoryRenamedCollidesSibling`） | PASS（422，未部分恢复） |
| CR-05 | S2 | `HasChildrenError` 映射 409 `HAS_CHILDREN`，`details.items` 含子节点 id/name | 集成（`TestSystemNodesDelete` details 断言） | PASS |
| CR-06 | S2 | 类别更新/删除、节点删除、回收站永久删除的引用检查移入事务 | `go vet` + 集成回归 | PASS |
| CR-07 | S2 | 不修复，接受为已知性能项（N+1/全表聚合不影响正确性） | — | ACCEPTED |
| CR-08 | S2 | 新增 5 个后端集成用例 + 2 个回收站 E2E | 集成/Playwright | PASS |
| CR-09 | S2 | `checklist.md` ND-02/ND-03、`proposal.md` AC-6 统一为 422 `NODE_LEVEL_LIMIT` | 文档一致性检查 | PASS |

### 8.2 F5 门禁记录（2026-11-09）

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd backend; go vet ./...` | 0 | 无输出 | — |
| `cd backend; go test -count=1 ./...` | 0 | 全部 ok | `config`、`httpapi`、`model`、`service` |
| `TEST_DATABASE_URL=...; cd backend; go test -count=1 -tags=integration ./internal/integration/...` | 0 | `ok ... 10.123s` | 新增 5 个回收站用例，体系用例总数 33 |
| `cd frontend; npm run lint` | 0 | 无告警 | ESLint |
| `cd frontend; npm run typecheck` | 0 | 无输出 | `tsc --noEmit` |
| `cd frontend; npm run test` | 0 | 5 个文件 / 30 个用例通过 | vitest |
| `cd frontend; npm run build` | 0 | 静态导出成功 | Next.js `output: export` |
| `docker compose build app; docker compose up -d app; GET /api/v1/health` | 0 | `{"cache":"ok","data_dir_writable":true,"db":"ok","status":"ok","version":"0.1.0"}` | 容器 `novel-management-app-1` |
| `cd frontend; npx playwright test --reporter=list` | 0 | **52 passed (36.7s)** | 含 `trash-system.spec.ts` 2 条、`system.spec.ts` 11 条 |

### 8.3 F5 复审结论

- 待独立 Reviewer 依据 F5 Diff 给出 PASS/FAIL；S1 清零且无新增回归后方可进入 F6。
