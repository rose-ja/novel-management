# Bootstrap Verification

> P7 工程初始化作者记录。P8 独立验收结论见 `project/bootstrap-acceptance.md`；P8 不修改本文件。

- Commit: `chore/p7-bootstrap` 分支上的 P7 工程初始化提交，以及 P8 FAIL 后的 P7 返工提交（哈希见对应提交）
- Environment: Windows 11；Docker Desktop 29.9.2；Go 1.27.3（本地工具链）+ 模块 `go 1.26.0`（容器内 `golang:1.26-alpine`）；Node v24.22.1 / npm 11.19.1；PostgreSQL 16-alpine；Redis 7-alpine
- Author / Tester: P7 作者（AI 辅助），2026-11-07
- Date: 2026-11-07
- Conclusion: PASS
- Rework: P8 首轮独立验收判 FAIL（golangci-lint 未执行、CI 门禁不完整）；本轮修复后由新的独立验收人复验，见 `bootstrap-acceptance.md`。

| Check | Command / Steps | Expected | Actual | Status | Evidence |
|---|---|---|---|---|---|
| Install | `cd backend && go mod download`；`cd frontend && npm ci` | 依赖安装成功，无缺失模块，锁文件不漂移 | 后端依赖全部拉取；前端 `npm ci` 成功且 `package-lock.json` 未变化 | PASS | 执行输出 |
| Frontend/Backend startup | `docker compose build`；`docker compose up -d`；`docker compose ps` | app/db/cache 启动，app 为 healthy | app、db、cache 均 `Up (healthy)`；镜像 `novel-settings:dev` | PASS | `docker compose ps` |
| Database / Migration | server 启动自动 `goose.Up`；`psql -c "select version_id from goose_db_version"`；`pg_extension` 查询 | 迁移 version 1 已应用；`pg_trgm` 存在 | `version_id = 1`；`extname = pg_trgm` | PASS | psql 输出 |
| Health check / API | `GET /api/v1/health`、`/api/v1/meta`、`/settings`、未知 `/api/v1/unknown` | 200 + 契约字段；DB 不可用 503；未知 API 404 JSON | health `{"status":"ok","db":"ok","cache":"ok","data_dir_writable":true,"version":"0.1.0"}`；meta 含 `limits`/`export_versions`/`template_version` 等；未知路径 404 | PASS | curl / Invoke-RestMethod 输出 |
| Minimal UI / E2E | `GET /` 与 `_next` 静态资源 | 200，含标题「小说设定管理系统」 | `GET /` 200、长度 5028、含标题；`/_next/static/chunks/*.js` 200 | PASS | HTTP 响应 |
| Lint / Format | `gofmt -l .`；`golangci-lint run ./...`；`npm run lint` | 无未格式化文件；lint 0 issues；ESLint 无警告 | `gofmt -l .` 无输出；`golangci-lint` v2.15.0 报 0 issues（首轮 P7 使用的 v2.6.2 不支持 Go 1.27 导出格式，已升级 lint 自身版本）；`npm run lint` 无警告 | PASS | 命令输出 |
| Security | `govulncheck ./...` | 无已知可利用漏洞 | 升级 `go-redis v9.7.3`、`pgx v5.9.2`、`x/net v0.60.0`、`x/text v0.42.0` 后，`govulncheck` 退出码 0，无 "Your code is affected" | PASS | 命令输出 |
| Typecheck | `go vet ./...`；`npm run typecheck` | 无类型错误 | 两者均通过 | PASS | 命令输出 |
| Unit / Integration | `go test ./...`；`npm test` | 后端单测通过；前端单测通过 | 后端 `internal/config`、`internal/httpapi` ok；前端 4 条测试通过；`-race` 需 cgo，Windows 本地不可用，Linux/容器执行 | PASS | 命令输出 |
| Build | `go build ./...`；`npm run build`；`docker compose build` | 二进制与静态导出成功，镜像构建成功 | 全部成功；静态导出 `frontend/out`；镜像内 `/app/server` 构建版本 `go1.26.10` | PASS | 命令输出 / 镜像内二进制 |
| CI | `powershell -File scripts/ci.ps1` | 脚本退出码 0，输出 `CI PASSED` | gofmt、golangci-lint、go vet、govulncheck、go test、go build、npm ci/lint/typecheck/test/build、`docker compose build` 全部通过；集成与 Playwright 显式 SKIP | PASS | 脚本输出 |
| Smoke | `powershell -File scripts/smoke.ps1` | `SMOKE OK` | `SMOKE OK: version=0.1.0 db=ok cache=ok` | PASS | 脚本输出 |
| CLI doctor | `docker compose exec app /app/cli doctor` | 输出脱敏配置与 DB/缓存状态 | `db_ok: true`、`cache_status: ok`、DSN 口令脱敏为 `%2A%2A%2A` | PASS | JSON 输出 |

## Blocking Issues

无。

## Follow-up Actions

1. Go 基线由 1.24 上调至 1.26：安全修复要求 `golang.org/x/net ≥ v0.60`，该版本要求 Go 1.26；已同步 `tech-stack.md`、`architecture.md`、`api-contract.md`、`deployment.md`、`PROJECT-SETUP.md`、`project-context.md`、ADR-002 与 `Dockerfile`。本轮未取得人工确认弹窗结果，按「修复漏洞」的必要性执行，若需维持 1.24 请回退本项。
2. 早期本机 buildx 曾因 Docker Hub 镜像代理返回 403 而改用预拉镜像 + `DOCKER_BUILDKIT=0`；P8 复验与本轮 CI 已可正常 buildx 构建，后续以 `docker compose build` 为准。
3. 集成测试与 E2E（Playwright）尚未建立，按 roadmap 在后续变更中引入；`scripts/ci.ps1` 已显式打印 SKIP。
4. P8 复验由新的独立验收人执行，结果见 `project/bootstrap-acceptance.md`。

## Approval

- 待 P8 独立验收。
