# CHG-011 验证记录（F3）

- 日期：2026-11-16
- 分支：`main`
- 代码基线：`a79d1c2`（F5 修复后）
- 环境：Windows + Docker Compose（PostgreSQL 16、Redis 7、`novel-settings:dev`）

## 1. 门禁结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| 格式 | `gofmt -l <改动文件>` | PASS（无输出） |
| 静态检查 | `go vet ./...` | PASS |
| 构建 | `go build ./...` | PASS |
| 后端单测 | `go test -count=1 ./...` | PASS（backup/httpapi/imports/migrations/model/service） |
| 集成测试 | `TEST_DATABASE_URL=… go test -count=1 -tags=integration ./internal/integration/...` | PASS（22.2s） |
| 前端类型 | `npm run typecheck` | PASS |
| 前端 lint | `npm run lint` | PASS |
| 前端单测 | `npm run test -- --run` | PASS（15 文件 / 91 用例） |
| 镜像构建 | `docker compose build app` | PASS |
| E2E | `npx playwright test --reporter=list` | PASS（115/115，1.8m） |

## 2. 真实导入演练

使用服务端 `GET /api/v1/imports/template/example` 下载示例包，按 API 走完整链路（上传 → 校验 → 提交 → 报告），完成后删除并永久清理演练作品。

```text
import_id=01M7HD1FYTHG3C5MB1K1YT67VX
validate_status=validated errors=0
commit_ms=124
report.elapsed_ms=232
report.work_name=验收-导入-132453
report.snapshot_id=01M7HD1G2Z6P6PN0RYMYH98GY8
counts={characters:1, factions:1, locations:1, rankings:1,
        system_categories:2, system_nodes:3, chronicles:1,
        images:1, reviewable:7}
images={matched:1, missing:[], extra:[], unsupported:[], too_large:[]}
errors=[] warnings=[]
```

- 提交后作品成为当前作品（E2E `import.spec.ts` 断言顶栏作品名切换）。
- 清理结果：`works=0 trash=0`。

## 3. 大样本验收（200/100/100/20/500）

生成 5 个体系类别、500 个体系节点、100 势力、100 地点、200 人物、20 榜单、500 大事记的模板（95.6 KB，无图片），按 API 上传并提交：

```text
validate_status=validated errors=0
counts={characters:200, factions:100, locations:100, rankings:20,
        system_categories:5, system_nodes:500, chronicles:500,
        images:0, reviewable:920}
commit_wall_ms=361  report_elapsed_ms=298
提交后计数与预览一致，作品自动成为当前作品
```

- 层级抽查：人物体系引用命中 100 个境界节点；榜单 20 个各 5 席；大事记 500 条按人物归属写入。
- 演练后删除并永久清理，`works=0 trash=0`。
- 用户签收（AC-31）由本次变更交付说明与验收记录承担，未单独取得外部签字。

## 4. 错误与回滚实测

- 模板版本不符 → `发现 1 个错误` 且「确认导入」禁用（E2E）。
- 缺图 → 错误清单显示 `缺失 1`，提交被拒绝且不创建作品（E2E + `TestImportRejectsMissingImage`）。
- 快照失败 → 状态 `failed`、`error_code=SNAPSHOT_FAILED`、无作品（`TestImportSnapshotFailureRollsBack`）。
- 导出排队时提交导入 → `ErrExportInProgress`；导入校验中创建导出 → `ErrImportInProgress`（`TestImportAndExportAreMutuallyExclusive`）。
- 失败分支清理 `tmp/imports/<id>`，启动时清理中断遗留目录（`MarkInterrupted`）。

## 5. 与计划的偏差

- 模板 v1.0 采用「区块 + 行字段」Markdown（用户已确认），未实现 SPEC-MIG-002 示例中的逗号类别写法；同时兼容 `体系类别：A,B` 行。
- 导入报告不提供文件下载（proposal 中列为 Should，未纳入基线）。
- 校对清单前端按实体各取前 200 项（对 200/100/100/20/500 验收样本足够），未做分页。

## 6. 已知限制

- 上传/提交期间无法主动取消（`uploadImport` 支持 signal，但向导未接入）。
- 校验轮询遇到网络错误会停止并给出重试入口（重新上传），不自动重试。
- 进程在图片落盘后、事务提交前硬崩溃时，`images/<work_id>` 可能残留；常规失败路径已清理。
