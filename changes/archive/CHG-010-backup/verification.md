# CHG-010 验证记录（F3）

- 日期：2026-11-15
- 分支：`main`
- 代码基线：`f9429ae`（F5 修复后）
- 环境：Windows + Docker Compose（PostgreSQL 16、Redis 7、`novel-settings:dev`）

## 1. 门禁结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| 格式 | `gofmt -l <改动文件>` | PASS（无输出） |
| 静态检查 | `go vet ./...` | PASS |
| 构建 | `go build ./...` | PASS |
| 后端单测 | `go test -count=1 ./...` | PASS（backup/httpapi/migrations/model/service） |
| 集成测试 | `TEST_DATABASE_URL=… go test -count=1 -tags=integration ./internal/integration/...` | PASS（21.2s） |
| 前端类型 | `npm run typecheck` | PASS |
| 前端 lint | `npm run lint` | PASS |
| 前端单测 | `npm run test -- --run` | PASS（15 文件 / 91 用例） |
| 镜像构建 | `docker compose build app` | PASS |
| E2E | `npx playwright test --reporter=list` | PASS（109/109，1.7m） |

## 2. 导出实测

- 真实导出日志：4 次，耗时 97.17ms / 97.67ms / 114.53ms / 144.66ms；包大小 122–141 KB（小数据集）。
- 解包结构（`novel-settings-20261115-203854.zip`）：

```text
manifest.json
checksums.sha256
README.txt
data/app.dump
```

- manifest 实测：

```json
{
  "export_version": 1,
  "schema_version": 10,
  "app_version": "0.1.0",
  "includes_trash": true,
  "works": [{"id": "01M7FKFF5WN3TS3N0PJ60NYBR0", "name": "E2E-导出页-mwhtmmka"}],
  "counts": {"works": 1, "system_categories": 1, "characters": 0, "media_assets": 0},
  "images": {"count": 0, "files": []}
}
```

- 不含回收站模式由集成测试覆盖（fake dumper + manifest `includes_trash=false`）；真实临时库过滤 SQL 在 Docker 恢复演练中执行。
- checksum 校验：集成测试 `backup.VerifyChecksums` 与恢复流程均通过。

## 3. 恢复演练

```text
docker compose stop app
docker compose run --rm cli restore /data/exports/novel-settings-20261115-205821.zip --yes
→ 恢复完成，数据版本 10，恢复前快照 20261115T125916Z-cli_restore，515ms

docker compose run --rm cli restore-snapshot /data/snapshots/20261115T125903Z-work_delete --yes
→ 恢复完成，数据版本 10，恢复前快照 20261115T125918Z-cli_restore，505ms

docker compose start app
→ /api/v1/health: {"status":"ok","db":"ok","cache":"ok","data_dir_writable":true}
→ /api/v1/settings: last_restore_at=2026-11-15T20:59:18.96572+08:00
```

- 恢复前自动快照记录在设置页可见；`goose: no migrations to run. current version: 10`。
- 恢复失败路径（版本不兼容、checksum 篡改、非空无 `--yes`、runner 失败）由单测/集成覆盖，均不调用 `pg_restore` 或保留原数据。

## 4. 与计划的偏差

- 导出包采用 ADR-006 结构（用户 2026-11-15 裁决），未实现 SPEC-BACKUP-001 的 `data.json/media/meta.json`；F10 已回写 Spec。
- 真实 `pg_dump/pg_restore` 仅在 Docker 演练中执行；Windows 单测/集成使用注入的 fake dumper/runner，避免依赖本机 PostgreSQL 客户端。

## 5. 已知限制

- 数据库恢复成功后的图片替换、迁移、检索重建、计数核对属于非事务步骤；失败时返回自动快照路径与回滚命令，需手工回滚（快照保留）。
- 图片缺失时导出跳过该文件并记录警告，manifest 以实际复制结果计数。
