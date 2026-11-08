# Bootstrap Acceptance

> P8 独立验收记录。必须由非 P7 作者执行；不得修改 `project/bootstrap-verification.md`。

- Commit: `a9f7941`（分支 `chore/p7-bootstrap`，工作区干净）
- Environment: Windows 11；Docker Desktop 29.9.2 / Compose v5.6.1；Go 1.27.3（模块声明 `go 1.24.0`）；Node v24.22.1 / npm 11.19.1；PostgreSQL 16-alpine；Redis 7-alpine
- Independent Reviewer: P8 QA（非 P7 作者，AI 辅助）
- Date: 2026-11-07
- P7 verification record: `project/bootstrap-verification.md`
- Conclusion: FAIL（返回 P7）。运行时启动、API、数据库、页面与构建链路正常；`golangci-lint` 实际发现 6 个 error，且本地 CI 脚本未覆盖项目规定的质量门禁，故不能判定可进入 F1。

| Check | Command / Steps | Expected | Actual | Status | Evidence |
|---|---|---|---|---|---|
| Install / lockfile | `cd backend && go mod download`；`go mod verify`；`cd frontend && npm ci` | 依赖完整；锁文件不被改写 | `go mod download` 成功；`go mod verify` 输出 `all modules verified`；`npm ci` 成功（401 包），`package-lock.json` SHA256 `917B77...0CF21F` 前后一致，`git status` 无漂移 | PASS | 命令输出；锁文件哈希；`git status` 空输出 |
| Frontend / Backend startup | `docker compose build`；`docker compose up -d`；`docker compose ps` | app/db/cache 均启动，app healthy，仅宿主回环发布 | classic builder 与 buildx 均构建成功（`novel-settings:dev`）；app/db/cache 均 `Up (healthy)`；端口 `127.0.0.1:17860->17860` | PASS | `docker compose ps` 输出 |
| Database / Migration | server 自动 `goose.Up`；`psql` 查询 `goose_db_version`、`pg_extension`；`docker compose stop app` 后 `docker compose run --rm cli migrate` | 迁移到 version 1；`pg_trgm` 存在；CLI 可迁移且并发时被锁拒绝 | `goose_db_version` 含 0 与 1 两行（1 已应用）；`pg_trgm` 存在；CLI 迁移输出 `迁移完成`、`current version: 1`；app 持锁时 CLI 退出 1，报「另一个实例正在运行或正在恢复」 | PASS | psql 输出；CLI 输出 |
| Health check / API | `GET /api/v1/health`、`/meta`、`/settings`、`/api/v1/unknown`；`Host: evil.com`；检查响应头 | 200 + 契约字段；未知 API 404 JSON；非白名单 Host 403；CSP/安全头与 X-Request-ID | health `{"status":"ok","db":"ok","cache":"ok","data_dir_writable":true,"version":"0.1.0"}`；meta 含 `limits`/`export_versions`/`template_version`；settings 含 `data_dir`/`snapshot_keep`；未知 API 404 `{"error":{"code":"NOT_FOUND","message":"接口不存在"}}`；evil.com 403 `HOST_NOT_ALLOWED`；CSP、nosniff、DENY、no-referrer、Permissions-Policy 均存在；`X-Request-ID: qa-test-123` 被透传、缺省时自动生成 | PASS | curl / Invoke-WebRequest 输出 |
| Minimal UI / E2E | `GET /`；解析 `/_next` 资源并逐一请求；`GET /no-such-page` | 200 且含标题；静态资源 200；未知页面 404；E2E 尚未引入 | `/` 200，5206 字节，含「小说设定管理系统」；7 个 `/_next` CSS/JS 全部 200；未知页面 404 JSON；Playwright 未建立（N/A，见 CI 一致性） | PASS（最小链路）；E2E N/A | HTTP 响应；`docker compose logs app` |
| Lint / Format | `gofmt -l .`；`npm run lint`；`golangci-lint run ./...`（另试 `GOTOOLCHAIN=go1.24.13`） | 无未格式化文件；ESLint 无 error；golangci-lint 无 error | `gofmt -l .` 无输出；ESLint 通过；默认 Go 1.27.3 下 golangci-lint 因 `export data version 5 ... maximum supported version 2` 退出 7；改用已缓存的 `GOTOOLCHAIN=go1.24.13` 后正常运行并报告 6 个 `errcheck` error（exit 1） | FAIL（golangci-lint）；gofmt / ESLint PASS | 命令输出 |
| Typecheck | `go vet ./...`；`npm run typecheck` | 无类型错误 | 均退出 0 | PASS | 命令输出 |
| Unit / Integration | `go test ./... -count=1`；`npm test` | 单元测试通过 | 后端 `internal/config`、`internal/httpapi` ok；前端 1 文件 4 测试通过；集成测试未建立（N/A） | PASS（单元）；集成 N/A | 命令输出 |
| Build | `go build ./...`；`npm run build`；`docker compose build` | 二进制、静态导出与镜像构建成功 | 均成功；Next 静态导出 4 页至 `frontend/out`；镜像 `novel-settings:dev` 构建成功 | PASS | 命令输出 |
| CI config consistency | `powershell -File scripts/ci.ps1`；对照 `PROJECT-SETUP.md` §11 与 `testing-strategy.md` §9 | 脚本退出 0 且覆盖文档列出的门禁 | 脚本退出 0 并输出「CI 通过」，但仅执行 `go vet`、`go test`、`go build`、`npm ci/lint/typecheck/test/build`；缺 `gofmt -l`、`golangci-lint`、`govulncheck`、`-race`、集成、E2E、`docker compose build`、Smoke，与 §11/§9 不一致 | FAIL | 脚本内容与执行输出 |
| Secrets / logging | 检查 `.gitignore`、`.dockerignore`、`.env.example`；`docker compose logs app`；`docker compose exec app /app/cli doctor`；错误体 | 密钥不入库、日志不泄密、DSN 脱敏、错误体统一 | `.env` 已忽略且未跟踪；`.dockerignore` 排除 `.env`；`.env.example` 仅 3 个占位变量（`APP_VERSION`/`APP_PORT`/`POSTGRES_PASSWORD`），其余变量依赖代码默认值；doctor 中 DSN 为 `%2A%2A%2A`；日志为 slog 结构化 `key=value`，无口令/堆栈；错误体为 `{"error":{"code","message"}}` | PASS（存在文档偏差） | 配置文件；日志；doctor JSON |
| Directory vs architecture | 对照 `architecture.md` §5.3、`tech-stack.md`、实际目录 | 启动骨架与文档一致 | `backend/internal/{config,httpapi,observability,database,migrations}`、`cmd/server`、`cmd/cli`、`frontend/app`、`scripts/` 均存在；`service/domain/repository/model/search/cache/imports/media/snapshots/jobs`、`templates/`、`components/ui`、`features`、`hooks` 尚未建立（属 F1 计划）；`tech-stack.md` §6 所述 `.nvmrc` 与 `package.json engines` 缺失 | PASS（启动骨架），F1 模块按计划待建 | 目录清单；文档对照 |

## Differences from P7 Record

1. **npm 缓存限制未复现**：P7 称默认缓存 `D:\SoftWare\Node24\node_cache` 不可写、需加 `--cache`。本次直接写入该目录成功，`scripts/ci.ps1` 中未加参数的 `npm ci` 也成功，故该限制在当前环境不成立；`npm ci` 包数为 401（P7 为 402）。
2. **Docker buildx 403 未复现**：P7 称 buildx 对镜像代理 HEAD 返回 403、只能用 `DOCKER_BUILDKIT=0`。本次 classic builder 与 buildx 均构建成功（基础镜像/层命中缓存），未观察到 403；该限制可能在冷缓存/联网拉取时仍存在，但当前不阻断。
3. **golangci-lint 状态与 P7 不同**：默认 Go 1.27.3 的 `export data version 5` 报错复现；但 `GOTOOLCHAIN=go1.24.13` 工具链已缓存在本地模块目录，lint 正常完成并报告 6 个 `errcheck` error。P7「工具链下载过慢、未能完成」不再成立，因此不能以「环境限制」豁免 lint。
4. **CI「通过」的覆盖范围不足**：P7 以 `scripts/ci.ps1` 退出 0 作为 CI 通过证据；脚本未执行 §11/§9 中的多项门禁，无法证明 golangci-lint、gofmt、govulncheck、集成/E2E、镜像构建与 Smoke 均通过。
5. **数据库版本行**：P7 记录「version 1」；实测 `goose_db_version` 同时存在 version 0（goose 初始化行）与 version 1，有效迁移版本仍为 1，结论不受影响。
6. **日志格式文档偏差**：`deployment.md` §7 要求 JSON 行；实现为 `slog.NewTextHandler` 的 `key=value` 文本行，字段为 `time/level/msg/method/path/status/latency_ms/request_id`，与文档列出的 `ts/action/entity_type/entity_id/code/duration_ms/cache_status` 不完全对应。
7. **环境变量示例偏差**：`deployment.md` §2.1 列出 14 个变量，`.env.example` 仅 3 个；其余变量由 `internal/config` 提供默认值，功能不受影响，但示例未完整对齐。

## Blocking Issues

1. **后端 Lint 未通过**：`GOTOOLCHAIN=go1.24.13 golangci-lint run ./...` 报告 6 个 `errcheck` error：
   - `cmd/cli/main.go:53`、`cmd/cli/main.go:110`：`db.SQL.Close` 返回值未检查；
   - `cmd/cli/main.go:115`：`client.Close` 返回值未检查；
   - `cmd/server/main.go:50`：`db.SQL.Close` 返回值未检查；
   - `internal/config/config_test.go:14`：`os.Unsetenv` 返回值未检查；
   - `internal/config/config_test.go:15`：`os.Setenv` 返回值未检查。
   `testing-strategy.md` §9 明确规定后端 Lint「任何 error」即阻塞，故不能进入 F1。
2. **CI 门禁与文档不一致**：`scripts/ci.ps1` 未执行 `gofmt -l .`、`golangci-lint`、`govulncheck`、`-race` 单测、集成测试、E2E、`docker compose build`、Smoke；其「CI 通过」不能作为 §11/§9 的合规证据。

## Follow-up Actions

1. 修复上述 6 处 `errcheck`（如 `defer func() { _ = db.SQL.Close() }()`、`client.Close()` 显式忽略/处理、测试中 `_ = os.Setenv(...)`），使 `golangci-lint run` 归零。
2. 将 `scripts/ci.ps1` 补齐到 `PROJECT-SETUP.md` §11 与 `testing-strategy.md` §9 的要求：`gofmt -l`、`golangci-lint`（固定 `GOTOOLCHAIN=go1.24.13` 或统一工具链）、`govulncheck`、`-race`、集成/E2E（未建立时显式标注并跟踪）、`docker compose build`、Smoke；或在文档中明确 P7 启动阶段的最小门禁子集。
3. 统一日志实现与 `deployment.md` §7：改用 JSON handler/补齐字段，或修订文档为结构化文本格式。
4. 补齐 `.env.example` 与 `deployment.md` §2.1 的变量清单（可标注「默认值由代码提供」）。
5. 补 `.nvmrc` / `package.json engines`，并与 Dockerfile 的 Node 22 统一版本策略。
6. 修复后重新执行 P8 独立验收；运行时链路（启动、迁移、API、静态页面、构建、Smoke）已通过，可作为复验基线。

## Approval

- P8 独立验收结论：FAIL，返回 P7 修复 Blocking Issues 1–2；修复后需重新独立验收。
- 验收时容器栈已恢复为 app/db/cache 全部 `Up (healthy)`，可通过 `http://127.0.0.1:17860` 访问。
- 验收人：P8 QA（独立于 P7 作者）；日期：2026-11-07。
