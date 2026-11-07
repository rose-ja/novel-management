# Bootstrap Verification

> P7 工程初始化作者记录。P8 独立验收结论见 `project/bootstrap-acceptance.md`；P8 不修改本文件。

- Commit: `chore/p7-bootstrap` 分支上的 P7 工程初始化提交（本文件随该提交一起入库，哈希见该提交本身）
- Environment: Windows 11；Docker Desktop 29.9.2；Go 1.27.3（本地工具链）+ 模块 `go 1.24.0`（容器内 `golang:1.24-alpine`）；Node v24.22.1 / npm 11.19.1；PostgreSQL 16-alpine；Redis 7-alpine
- Author / Tester: P7 作者（AI 辅助），2026-11-07
- Date: 2026-11-07
- Conclusion: PASS（`golangci-lint` 在本地工具链下 BLOCKED，见 Follow-up 1）

| Check | Command / Steps | Expected | Actual | Status | Evidence |
|---|---|---|---|---|---|
| Install | `cd backend && go mod download`；`cd frontend && npm ci --cache "$env:USERPROFILE\.npm"` | 依赖安装成功，无缺失模块 | 后端依赖全部拉取；前端 402 包安装成功（默认 npm 缓存目录不可写，改用用户缓存） | PASS | 执行输出 |
| Frontend/Backend startup | `docker compose build`；`docker compose up -d`；`docker compose ps` | app/db/cache 启动，app 为 healthy | app、db、cache 均 `Up (healthy)`；镜像 `novel-settings:dev` | PASS | `docker compose ps` |
| Database / Migration | server 启动自动 `goose.Up`；`psql -c "select version_id from goose_db_version"`；`pg_extension` 查询 | 迁移 version 1 已应用；`pg_trgm` 存在 | `version_id = 1`；`extname = pg_trgm` | PASS | psql 输出 |
| Health check / API | `GET /api/v1/health`、`/api/v1/meta`、`/settings`、未知 `/api/v1/unknown` | 200 + 契约字段；DB 不可用 503；未知 API 404 JSON | health `{"status":"ok","db":"ok","cache":"ok","data_dir_writable":true,"version":"0.1.0"}`；meta 含 `limits`/`export_versions`/`template_version` 等；未知路径 404 | PASS | curl / Invoke-RestMethod 输出 |
| Minimal UI / E2E | `GET /` 与 `_next` 静态资源 | 200，含标题「小说设定管理系统」 | `GET /` 200、长度 5028、含标题；`/_next/static/chunks/*.js` 200 | PASS | HTTP 响应 |
| Lint / Format | `gofmt -l .`；`golangci-lint run ./...`；`npm run lint` | 无未格式化文件；lint 无问题；ESLint 无警告 | `gofmt -l .` 无输出；`npm run lint` 无警告；`golangci-lint` v2.6.2 在本地 Go 1.27.3 下无法完成类型检查（`export data version 5 is greater than maximum supported version 2`），改用 `GOTOOLCHAIN=go1.24.13` 又因工具链下载过慢未能完成 | PASS（gofmt / ESLint）/ BLOCKED（golangci-lint） | 命令输出 |
| Typecheck | `go vet ./...`；`npm run typecheck` | 无类型错误 | 两者均通过 | PASS | 命令输出 |
| Unit / Integration | `go test ./...`；`npm test` | 后端单测通过；前端单测通过 | 后端 `internal/config`、`internal/httpapi` ok；前端 4 条测试通过 | PASS | 命令输出 |
| Build | `go build ./...`；`npm run build`；`docker compose build` | 二进制与静态导出成功，镜像构建成功 | 全部成功；静态导出 `frontend/out`；镜像 665MB | PASS | 命令输出 |
| CI | `powershell -File scripts/ci.ps1` | 脚本退出码 0，输出「CI 通过」 | 后端 vet/test/build 与前端 ci/lint/typecheck/test/build 全部通过 | PASS | 脚本输出 |
| Smoke | `powershell -File scripts/smoke.ps1` | `SMOKE OK` | `SMOKE OK: version=0.1.0 db=ok cache=ok` | PASS | 脚本输出 |
| CLI doctor | `docker compose exec app /app/cli doctor` | 输出脱敏配置与 DB/缓存状态 | `db_ok: true`、`cache_status: ok`、DSN 口令脱敏为 `%2A%2A%2A` | PASS | JSON 输出 |

## Blocking Issues

无。

## Follow-up Actions

1. `golangci-lint`（v2.6.2）在本地 Go 1.27.3 工具链下报 `export data version 5 is greater than maximum supported version 2`，无法完成类型检查；需以 `GOTOOLCHAIN=go1.24.13 golangci-lint run` 运行（或统一本地 Go 版本至 1.24/1.25）。该问题属本地工具链版本错配，不影响构建与测试结论。
2. Docker Hub 镜像源（`mirror.aliyuncs.com`）对 buildx 的 manifest HEAD 请求返回 403：本机改为先 `docker pull` 预拉基础镜像、再以 `DOCKER_BUILDKIT=0 docker compose build` 构建。此为环境限制，Dockerfile 本身无需改动；换用可用的 registry 镜像源即可恢复 buildx 构建。
3. 集成测试与 E2E（Playwright）尚未建立，按 roadmap 在后续变更中引入。
4. P8 独立验收需由非 P7 作者执行，并填写 `project/bootstrap-acceptance.md`。

## Approval

- 待 P8 独立验收。
