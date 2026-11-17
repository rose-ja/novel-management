# CHG-012 验证记录（F3）

- 日期：2026-11-17
- 分支：`main`
- 代码基线：`4c54507`（F5 审查修复后）
- 环境：Windows + Docker Compose（PostgreSQL 16、Redis 7、`novel-settings:dev`）

## 1. 门禁结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| 格式 | `gofmt -l cmd/cli` | PASS（无输出） |
| 静态检查 | `go vet ./...` | PASS |
| 后端单测 | `go test -count=1 ./...` | PASS（backup/config/httpapi/imports/migrations/model/service） |
| 集成测试 | `$env:TEST_DATABASE_URL=…; go test -count=1 -tags=integration ./internal/integration/...` | PASS（14.6s，scratch 库 `novel_it`） |
| 前端类型 | `npm run typecheck` | PASS |
| 前端 lint | `npm run lint` | PASS |
| 前端单测 | `npm test` | PASS（15 文件 / 91 用例） |
| 镜像构建 | `docker compose build app` | PASS |
| 烟测 | `scripts/smoke.ps1` | PASS（db=ok cache=ok） |
| E2E 全量 | `npx playwright test` | PASS（128 passed / 1 skipped，跳过项为未设置 `PERF_WORK_ID` 的性能用例） |
| E2E 回归（审查修复后） | `npx playwright test a11y locations rankings system keyboard trash error-states` | PASS（59/59） |

## 2. 性能基线（1000 人物）

`scripts/perf.ps1` 输出：`artifacts/perf/20261117-105258/report.json`（gitignored）。

| 指标 | 实测 | 阈值 | 结果 |
|---|---|---|---|
| 首屏可交互 | 221ms | ≤2000ms | PASS |
| 人物列表 P95 | 5ms | ≤500ms | PASS |
| 检索 P95 | 10ms | ≤150ms | PASS |

- 造数：`cli seed --work perf-baseline-* --characters 1000 --factions 50 --locations 50`，单事务批量写入并重建检索。
- 采样：20 次请求去掉首个预热值计算 P95；首屏由 Playwright 测量人物列表首个数据行可见。
- 演练后清理：`works=0 trash=0`。

## 3. 安全与发布演练

### 3.1 CSP 与安全用例

- E2E-13（`security.spec.ts`）：响应头含 `default-src 'self'` 与 `script-src`；详情/检索可用；无 CSP 控制台违规。
- 路径穿越与恶意输入：`backup`（`TestExtractZipRejectsTraversal`）、`imports`（`TestInventoryRejectsTraversal`、`TestValidateRejectsAbsoluteImagePath`）、`httpapi`（`TestStaticHandlerRejectsTraversal`）、`service`（`TestEncodeImageRejectsInvalid`）均 PASS。
- 集成：`TestImportRejectsMissingImage`、`TestImportSnapshotFailureRollsBack`、`TestRestoreArchiveRestoresImagesAndWritesSnapshot`、`TestCharacterImageValidation` 均 PASS。

### 3.2 导出 → CLI 恢复

```text
恢复完成：/data/exports/novel-settings-20261117-102722.zip
数据版本：11
作品：2，人物：1，图片：0，耗时 508ms
恢复后 health=ok db=ok；works=2，演练作品人物=1，与恢复前一致
```

### 3.3 快照创建与恢复

```text
删除演练作品生成快照：20261117T022748Z-work_delete（ok）
恢复完成：/data/snapshots/20261117T022748Z-work_delete
作品：2，人物：1，耗时 504ms；恢复后演练作品与人物均回归
```

### 3.4 迁移回滚

- `TestMigrationRoundTrip` PASS：0011→0002 逐级 Down、表/列/数据断言，再重复 Up；覆盖 `import_runs`、`export_runs`、检索、榜单、地点、势力、图片、人物与引用。
- 测试运行在独立 scratch 库，不影响应用库。

## 4. 已知限制

1. 回收站深链最多查找前 1000 条；实体随祖先整组删除时记录为祖先 `root_id`，按实体 id 深链会进入未命中提示（design §1.1 已记录）。
2. `cli seed` 有意保留造数作品供复测；性能脚本结束不自动删除，需按 verification §2 的清理步骤回收。`perf.spec.ts` 会在结束后恢复原 `current_work_id`。
3. 性能数据为本地单次采样，受机器负载影响；脚本内已设置阈值，超标退出非 0。
4. 席位备注守卫覆盖「取消」确认；页面级离开由榜单改名守卫拦截，未单独对备注做全局导航拦截。
5. 势力删除对话框的 `in_trash` 分支依赖后端引用字段，目前后端不赋值，该分支不可达（无行为影响）。
