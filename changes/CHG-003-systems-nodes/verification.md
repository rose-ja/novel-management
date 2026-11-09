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

- 独立 Reviewer 只读检查 F5 Diff 与受影响区域，结论 **PASS**：CR-01～CR-06、CR-08、CR-09 均已修复，S1 清零，无新增回归；CR-07 接受为已知性能项。
- 复审独立执行的门禁：`go vet ./...` 0；集成测试 `ok 4.842s`；`npm run lint`/`typecheck` 0；容器健康；`npx playwright test trash-system.spec.ts` 2 passed。
- 复审发现 `project/api-contract.md:102-103` 仍写 `NODE_LEVEL_LIMIT=409`、`NODE_PARENT_MISMATCH=400`，已同步为 422（与实现 `httpapi/request.go:93,97` 一致）。
- 非阻塞遗留：待校对筛选下 `HAS_CHILDREN` 的 UI E2E 未补；工作级 `PurgeGroup` 未纳入父子顺序专项测试。

## 9. F6 集成 / E2E 验证

- 执行人：Agent（F6 QA，独立于 F5 修复）
- 环境：Windows + PowerShell 5.1；应用 `http://127.0.0.1:17860`（容器 `novel-management-app-1`）；PostgreSQL 容器 `novel-management-db-1`；Playwright channel chrome、workers 1
- 环境确认：`GET /api/v1/health` → `{"status":"ok","db":"ok","cache":"ok","data_dir_writable":true,"version":"0.1.0"}`
- 验证后状态：`git status` 无改动；E2E/QA 前缀数据已清理，works total=0、trash total=0、`current_work_id=null`

### 9.1 执行命令与结果

| # | 命令 | Exit | 结果 |
|---|---|---:|---|
| 1 | `cd frontend; npx playwright test --reporter=list` | 0 | **52 passed / 0 failed / 0 skipped（34.0s）** |
| 2 | 临时 Playwright 用例（加载骨架、空态、控制台/网络错误；用后删除） | 0 | 3 passed（4.3s） |
| 3 | 临时 PowerShell API 边界脚本（用后删除） | 0 | 24 项断言，0 失败；`finally` 清理数据 |
| 4 | `GET works / trash / settings` 复核 | 0 | works=0、trash=0、`current_work_id=null` |

### 9.2 场景矩阵

| 类别 | 场景 | 预期 | 实际 | 判定 | 证据来源 |
|---|---|---|---|---|---|
| 核心成功 | 内置境界保护、类别 CRUD、两级节点、排序持久、停用只读、纪元文本 | 全部可用 | 全量 E2E 通过 | PASS | 实跑 `system.spec.ts`（11） |
| 核心成功 | 类别整组删除 + `new_name` 恢复、节点删除/撤销、回收站系统根 | 整组语义正确 | 全量 E2E 通过 | PASS | 实跑 `system.spec.ts`、`trash-system.spec.ts`（2） |
| 异常路径 | 第三级节点 | 422 `NODE_LEVEL_LIMIT` | 422，`details.field=parent_id` | PASS | 实跑 API |
| 异常路径 | 删除有子节点的父节点 | 409 `HAS_CHILDREN` + 子节点清单 | 409，`details.items` 含 `{id,name,entity_type:"system_node"}` | PASS | 实跑 API |
| 异常路径 | 删除类别→删除作品→恢复类别 | 409 `PARENT_DELETED` | 409 `PARENT_DELETED` | PASS | 实跑 API |
| 异常路径 | 恢复类别 `renames` 撞同组未改名兄弟 | 422 `RESTORE_CONFLICT`，不部分恢复 | 422，items 含兄弟节点；类别仍在回收站 | PASS | 实跑 API |
| 异常路径 | 恢复类别改用不冲突名称 | 200 且整组恢复 | 200，`restored_count=3` | PASS | 实跑 API |
| 边界 | 第 10 类（含内置）成功、第 11 类拒绝 | 201 / 409 `CATEGORY_LIMIT` | 201 / 409 `CATEGORY_LIMIT` | PASS | 实跑 API |
| 边界 | 内置「境界」改名/停用/删除 | 409 `CATEGORY_BUILTIN_IMMUTABLE` | 409 ×3 | PASS | 实跑 API |
| 边界 | 空名称、`PATCH parent_id:null` | 422 `VALIDATION_ERROR` | 422，字段 `name`/`parent_id` | PASS | 实跑 API |
| 数据隔离 | 跨作品类别/节点列表与跨作品父级 | 不串扰 / 422 `NODE_PARENT_MISMATCH` | 不串扰 / 422 | PASS | 实跑 API；集成 `TestSystemCrossWorkIsolation` 引用 F3 |
| UI 状态 | `/system`、`/trash` 加载/空/成功态 | 骨架、空态文案、成功态可见 | 全部可见（空节点「还没有节点」等） | PASS | 实跑临时用例 |
| 视口 | system 1280 / 1024 | 视觉基线差异 ≤ 0.02 | 通过 | PASS | 实跑 `visual.spec.ts` |
| 控制台/网络 | `/system`、`/trash` | 无未预期 4xx-5xx/pageerror | 0 个 API 4xx-5xx、0 个 pageerror；1 条既有 `favicon.ico` 404 | PASS（附既有低危项） | 实跑临时用例 |
| 旧功能回归 | 作品、回收站、设置、壳层、错误态、键盘、a11y | 不回归 | 全量 52 passed | PASS | 实跑全量 E2E |

### 9.3 F6 发现与处理

| ID | 级别 | 发现 | 处理 |
|---|---|---|---|
| F6-01 | 文档 | `proposal.md` AC-2 写「已有 9 个自定义类别，第 10 个成功」，与实现/checklist「上限 10 含内置」矛盾 | 改为「8 个自定义（含内置 9）→ 第 9 个成功；第 10 个自定义（共 11）→ 409」 |
| F6-02 | 文档 | `checklist.md` UI-01 引用的 `system.spec.ts` 无作品空态用例不存在 | 证据改为 F6 临时用例实跑（已清理），如实记录 |
| F6-03 | 既有问题 | `GET /favicon.ico` 404（全局壳层，非本变更引入） | 记录为已知低危项，不在 CHG-003 处理 |
| F6-04 | 契约观察 | `POST /api/v1/system-nodes` 显式 `parent_id:null` 返回 201（按根节点处理），`PATCH` 显式 null 返回 422（parent_id 不可改） | 符合 design §5「POST parent_id? 可选 / PATCH parent_id 不可改」，不视为缺陷 |

### 9.4 截图与日志

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/system-1280-chromium-win32.png`、`system-1024-chromium-win32.png`（本次全量通过）。
- Playwright 失败截图/追踪：无失败项，`test-results/` 无失败产物。
- API 边界脚本与临时 UI 用例执行后已删除，未留脚本文件；结果见 9.1/9.2。

### 9.5 F6 结论

**PASS**。核心成功路径、异常路径、边界条件、数据隔离、UI 状态、视口与旧功能回归均通过；F6-01/F6-02 文档矛盾已修正，F6-03 为既有全局低危项。可进入 F7 人工验收。

## 10. F9 合并与发布记录

| 项目 | 内容 |
|---|---|
| 版本 / Commit | `main` @ `cfb7d17`（功能分支 `feat/chg-003-systems-nodes`，17 个提交，`976ce53..cfb7d17` 快进合并） |
| 环境 | 本机（Windows + Docker Desktop 单用户本地）；`http://127.0.0.1:17860` |
| 时间 | 2026-11-09（Asia/Shanghai） |
| 操作者 | Agent（发布协调） |
| 审批 | 用户：F7 结论 **ACCEPT** + 明确授权「本地合并 main，不推送 origin」 |
| 部署步骤 | 未执行生产部署变更：容器镜像已在 F5 由同一功能代码构建（`58dc52f`）；合并后新增提交仅为文档，镜像无需重建。`docker compose ps` 中 `app`/`db` healthy。 |
| Smoke Test | `powershell -File scripts/smoke.ps1` → `SMOKE OK: version=0.1.0 db=ok cache=ok`；`GET /api/v1/health`、`GET /api/v1/meta`（`api_version=v1`）通过 |
| 异常 | PowerShell 用户 profile 的 `oh-my-posh` 未安装告警（环境噪音，不影响 Smoke 结果，exit 0） |
| 远程状态 | **未推送** `origin/main`（本地 main 领先 origin/main 30 个提交；按授权不推送、不创建远程 PR） |
| 回滚结果 | 未触发回滚。如需回退：`git reset --hard 976ce53`（本地）或 revert 合并提交；若已迁移，执行 0003 Down。 |

**发布结论**：本地集成通过，未发布到远程/公网；符合 `project/deployment.md`「未获批准不得发布到公网或修改部署拓扑」。
