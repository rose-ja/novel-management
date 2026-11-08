# Bootstrap Acceptance（P8 复验）

> P8 独立复验记录，由非 P7 作者、亦非首轮 P8 验收人的第二位独立验收人执行。本文件覆盖首轮 FAIL 记录；`project/bootstrap-verification.md` 保持只读未改动。

- Commit: `5ed9216`（P8 首轮 FAIL 后的返工提交），基线 `a9f7941`（P7 初始化）；分支 `chore/p7-bootstrap`；复验开始时 `git status` 干净
- Environment: Windows 11；Docker 29.9.2 / Compose v5.6.1；Go 1.27.3（本地工具链）+ 模块 `go 1.26.0`（容器 `golang:1.26-alpine` → `go1.26.10`）；Node v24.22.1 / npm 11.19.1（`frontend/.nvmrc` = 22，`engines.node >=22`，Dockerfile `node:22-alpine`）；PostgreSQL 16-alpine；Redis 7-alpine；golangci-lint v2.15.0（built with go1.27.3）；govulncheck v1.9.0
- Independent Reviewer: P8 QA（第二位独立验收人，非 P7 作者）
- Date: 2026-11-08（命令执行时区 Asia/Shanghai）
- P7 verification record: `project/bootstrap-verification.md`（仅读，未修改）
- Conclusion: **PASS（可进入 F1）**。首轮两项阻断（lint 未真实运行、CI 门禁覆盖不足）均已关闭并有实跑证据；运行时链路、迁移、API 契约、页面、全部门禁、CI 脚本与文档一致性均复核通过。剩余事项为非阻断的流程确认与后续阶段工作（见 §5）。

本文件所有结论均来自本次实际执行的命令与响应；未运行项标 N/A，环境阻塞项标 BLOCKED，未引用 P7 记录作为通过证据。

## 1. 验收结果

| # | Check | Command / Steps | Expected | Actual | Status | Evidence |
|---|---|---|---|---|---|---|
| 1 | 依赖与锁文件可复现 | `cd backend && go mod download`；`go mod verify`；`cd frontend && npm ci`；前后 `Get-FileHash` + `git status --porcelain` | 依赖完整、校验通过、锁文件哈希不变、工作区无漂移 | `go mod download` exit 0；`go mod verify` → `all modules verified`；`npm ci` added 401 packages exit 0；`go.sum` SHA256 `D1D99FF0E2251289D20CA0D6FCEEFC5CDA60766D737301787D51EF4B65483ABB`、`package-lock.json` SHA256 `3795EB1797A3FB762FB16DD96B20CCAB6170C1C16756CDD09AFD1D43B547BADA` 前后一致；`git status --porcelain` 空 | PASS | 命令输出；文件哈希；`git status` |
| 2 | 容器栈启动与健康（含重启 app） | `docker compose build`；`docker compose up -d`；`docker compose up -d --force-recreate app`；`docker compose ps` | 构建成功；app 以新镜像重建并 healthy；db/cache healthy；仅宿主回环发布 | `docker compose build` exit 0（镜像 `novel-settings:dev`，ID `b52017431bba`，665MB）；强制重建后 34s `Up (healthy)`；三容器均 healthy；`127.0.0.1:17860->17860/tcp`，db/cache 仅 `5432/tcp`、`6379/tcp` 未发布 | PASS | `docker compose ps`；build 日志 |
| 3 | 迁移与 `pg_trgm` | app 启动自动 `goose.Up`；`docker compose logs app`；`psql -c "select version_id, is_applied, tstamp from goose_db_version order by id"`；`pg_extension` 查询；`docker compose run --rm cli migrate` | 迁移应用到 version 1；`pg_trgm` 存在；app 运行时 CLI 迁移被锁拒绝 | 日志 `数据库迁移完成`、`goose: no migrations to run. current version: 1`；`goose_db_version` 实为 **2 行**：`0 | t | 2026-11-07 14:56:12.354818` 与 `1 | t | 2026-11-07 14:56:12.367605`（0 号为 goose 初始化行，有效版本为 1）；`pg_trgm 1.6`；CLI `migrate` 退出 1 并输出 `Error: 另一个实例正在运行或正在恢复`；当前仅有 `goose_db_version` 表（业务表属 F1） | PASS | psql 输出；`docker compose logs app`；CLI 输出 |
| 4 | API 契约与安全 | `GET /api/v1/health`、`/meta`、`/settings`、`/api/v1/unknown`；`curl -H "Host: evil.com"`；响应头检查；`X-Request-Id` 透传与自动生成 | 200 + 契约字段；未知 API 404 JSON；非白名单 Host 403；CSP/安全头齐全；request-id 透传 | health `{"cache":"ok","data_dir_writable":true,"db":"ok","status":"ok","version":"0.1.0"}`；meta 含 `api_version/app_version/cache_enabled/cache_ttl_sec/export_versions/limits/template_version`；settings 含 `app_version/current_work_id/data_dir/last_export_at/last_restore_at/snapshot_keep`；未知 API 404 `{"error":{"code":"NOT_FOUND","message":"接口不存在"}}`；`evil.com` 403 `{"error":{"code":"HOST_NOT_ALLOWED","message":"请求来源不在 Host 白名单内"}}`；CSP、`nosniff`、`DENY`、`no-referrer`、`Permissions-Policy` 均在；`X-Request-Id: qa-recheck-456` 原样回传，缺省时自动生成 16 位 hex | PASS | curl / Invoke-WebRequest 输出 |
| 5 | 首页与静态资源 | `GET /`；`curl -I`；`curl -w num_redirects`；解析并请求全部 `/_next` 资源；`GET /no-such-page` | 200、含标题、无重定向循环；静态资源 200；未知页面 404 | `/` 200，`num_redirects=0`，`Content-Length: 5206` 字节（字符数 5028），`<title>小说设定管理系统</title>` 与 `<h1>` 均存在；7 个 `/_next` JS/CSS 全部 200（共约 469KB）；`/no-such-page` 404 `application/json` | PASS | HTTP 响应头与正文 |
| 6 | 质量门禁 | `gofmt -l .`；`golangci-lint run ./...`；`go vet ./...`；`govulncheck ./...`；`go test ./... -count=1`；`npm run lint/typecheck/test/build`；`go build ./...`；`docker compose build`；镜像内 `grep -aoE 'go1\.[0-9]+(\.[0-9]+)?' /app/server \| sort -u` | 全部 exit 0；lint 0 issues；无已知可利用漏洞；镜像 Go ≥1.26 | `gofmt -l .` 无输出；`golangci-lint` → `0 issues.`；`go vet` exit 0；`govulncheck` → `Your code is affected by 0 vulnerabilities`（另有 2 个导入包/1 个依赖模块的漏洞未被调用，不计影响）；`go test` → `internal/config` ok 0.128s、`internal/httpapi` ok 0.322s；前端 lint/typecheck 无输出、vitest 4/4 通过、`next build` 静态导出 4 页；`go build ./...` exit 0；`docker compose build` exit 0；镜像内匹配 `go1.26`、`go1.26.10` | PASS | 命令输出；镜像内二进制 |
| 7 | `scripts/ci.ps1` 完整执行 | `powershell -File scripts/ci.ps1`；对照 `PROJECT-SETUP.md` §11 与 `testing-strategy.md` §9 | 退出 0 且输出 `CI PASSED`；覆盖 §11 列出的门禁；未建立项显式 SKIP 且说明如实 | 退出 0，输出 `CI PASSED`；实际执行 gofmt → go vet → golangci-lint（0 issues）→ govulncheck（0 影响）→ `go test ./...` → `go build`（server/cli 到 TEMP 后清理）→ `npm ci` → `npm run lint/typecheck/test/build` → `docker compose build`；三处 SKIP 均打印原因：`go test -race`（Windows 无 cgo 工具链）、集成测试（未建立，F1）、Playwright E2E（未建立，F6）。独立验证：本机 `go test -race` 确报 `-race requires cgo`，`CGO_ENABLED=1` 时因无 gcc 失败，SKIP 理由属实；另在 `golang:1.26-alpine` 容器内 `CGO_ENABLED=1 go test -race ./...` 全部通过（config 1.010s、httpapi 1.029s），说明竞态检查在当前代码上可用 | PASS | 脚本输出；`go env`；容器 -race 输出 |
| 8 | 文档一致性 | 全仓搜索 `1.24` 与 `go 1\.`；对照 `deployment.md` §2.1、`.env.example`、`.nvmrc`、`package.json engines`、`bootstrap-verification.md` | 无 Go 1.24 构建基线残留；示例变量与文档对齐；Node 版本策略一致；P7 记录与实测相符 | 无实际基线残留（`1.24` 仅出现在 `bootstrap-verification.md`、`tech-stack.md` TS-Q1、ADR-002 的历史说明「P7 由 1.24 上调」及本文件旧版本中）；`go.mod` `go 1.26.0`、Dockerfile 与 `deployment.md` §3.1 片段均为 `golang:1.26-alpine`；`.env.example` 16 个变量覆盖 `deployment.md` §2.1 全部 14 项（另含 compose 用的 `APP_VERSION`、`POSTGRES_PASSWORD`）；`frontend/.nvmrc`=22 已被 git 跟踪，`engines.node >=22`，与 `node:22-alpine` 一致；P7 记录主要结论与实测一致，差异见 §4 | PASS（记录有 3 处不完整，非失实） | 搜索输出；配置文件；`git ls-files` |
| 9 | Smoke 脚本 | `powershell -File scripts/smoke.ps1` | 输出 `SMOKE OK` 且退出 0 | `SMOKE OK: version=0.1.0 db=ok cache=ok`，exit 0 | PASS | 脚本输出 |

### 补充证据（不单独计项）

| 项 | 实际 | Status |
|---|---|---|
| CLI doctor | `docker compose run --rm cli doctor` → `db_ok: true`、`cache_status: "ok"`、`data_dir_writable: true`、`database: postgres://novel:%2A%2A%2A@db:5432/novel?sslmode=disable`（口令脱敏） | PASS |
| 日志格式与脱敏 | 应用日志为 JSON 行：`{"ts":...,"level":"INFO","msg":"http","method":"GET","path":"...","status":200,"duration_ms":0,"request_id":"..."}`；启动日志数据库串为 `%2A%2A%2A`；无堆栈、无正文、无宿主机绝对路径 | PASS |
| 密钥保护 | `.env` 未跟踪（`.gitignore` 含 `.env`、`.env.local`）；`.dockerignore` 排除 `.env`、`.env.local`、`data/`、`node_modules/`、`*.dump`、`exports/`、`snapshots/` | PASS |
| 目录与架构 | `backend/cmd/{server,cli}`、`internal/{config,httpapi,observability,database,migrations}`、`frontend/{app,lib,styles}`、`scripts/` 与 `architecture.md` §5.3 的启动骨架一致；`service/domain/repository/model/search/cache/imports/media/snapshots/jobs/templates`、`frontend/{components/ui,features,hooks}`、`scripts/fixtures/perf_seed.go` 尚未建立，属 F1/后续计划 | PASS（骨架），F1 待建 |

## 2. 首轮 FAIL 阻断项复核

| 首轮阻断 | 本次实测 | 结论 |
|---|---|---|
| 1. 后端 Lint 未通过（6 个 `errcheck`，需 `GOTOOLCHAIN=go1.24.13` 才能复现） | 默认 Go 1.27.3 下 `golangci-lint` v2.15.0 直接运行成功，输出 `0 issues.`，exit 0；`cmd/cli`、`cmd/server`、`config_test.go` 原 6 处 `Close`/`Setenv` 返回值均已处理 | **已关闭** |
| 2. CI 门禁与文档不一致（缺 gofmt/golangci-lint/govulncheck/-race/集成/E2E/镜像构建/Smoke） | `scripts/ci.ps1` 现执行 gofmt、golangci-lint、go vet、govulncheck、go test、go build、npm ci/lint/typecheck/test/build、`docker compose build` 并输出 `CI PASSED`；-race、集成、Playwright 三处显式 SKIP 并打印原因；与 `PROJECT-SETUP.md` §11「本地 CI」行完全对应 | **已关闭** |

首轮 Follow-up 3–6 的处置：日志已改 JSON 行；`.env.example` 已补齐；`.nvmrc`/`engines` 已补；运行时链路复验仍通过。

## 3. 与首轮 FAIL 记录的差异

1. 首轮 lint 只能在 `GOTOOLCHAIN=go1.24.13` 下运行并报 6 个 error；本次默认工具链即可运行且 0 issues，说明 lint 自身版本与 Go 基线均已统一。
2. 首轮 `scripts/ci.ps1` 仅 7 条命令且无 SKIP 说明；本次覆盖 12 类命令并显式打印 3 类 SKIP。
3. 首轮 `.env.example` 仅 3 个变量，本次 16 个；首轮缺 `.nvmrc`/`engines`，本次已补。
4. 首轮日志为 `key=value` 文本，本次为 JSON 行（`ts/level/msg/method/path/status/duration_ms/request_id`）。
5. 首轮首页长度记 5206 字节 / 5028 字符；本次测得一致（HTTP 字节 5206，字符 5028）。
6. 首轮未实测 `-race`；本次在容器内实测通过，同时确认 Windows 本地 SKIP 理由（无 cgo/gcc）属实。

## 4. 与 P7 记录（`bootstrap-verification.md`）的差异

1. **`goose_db_version` 行数**：P7 记「`version_id = 1`」，实测为 0、1 两行（0 为 goose 初始化行），有效迁移版本仍为 1，不影响结论。
2. **`-race` SKIP 未在 P7 记录中体现**：P7 只提「集成与 Playwright 显式 SKIP」，脚本实际还有 `go test -race` SKIP。P7 在 `PROJECT-SETUP.md` §11 与 Follow-up 中已说明 `-race` 需 cgo，故非失实，仅记录不完整。
3. **「govulncheck 无 'Your code is affected'」表述**：实际输出为 `Your code is affected by 0 vulnerabilities`，结论一致；另有 2 个导入包、1 个依赖模块存在未被调用的漏洞（`-show verbose` 可查），不构成可利用风险，但 P7 未披露该数字。
4. **首页长度统计口径**：P7 记 5028，为字符数；HTTP `Content-Length` 为 5206 字节，两者不矛盾。
5. **Go 基线 1.24→1.26 未经人工批准**：P7 已在 Follow-up 1 披露「本轮未取得人工确认弹窗结果」。技术实现与文档一致，且为消除 govulncheck 漏洞的必要条件，故不判阻断；但按 `AGENTS.md`「技术栈变更需人工批准」，需项目发起人追认（见 §5.1）。

除上述外，P7 记录的安装、启动、迁移、API、页面、门禁、构建、CI、Smoke、doctor 结论均与本次实测一致。

## 5. 非阻断观察与后续动作

### 5.1 需人工确认

1. **Go 1.26 基线追认**：`golang.org/x/net v0.60.0`（govulncheck 修复所需）要求 Go 1.26，P7 已将 `go.mod`、`Dockerfile`、`tech-stack.md`、`architecture.md`、`api-contract.md`、`deployment.md`、`PROJECT-SETUP.md`、`project-context.md`、ADR-002 全部同步为 1.26。工程侧一致，请项目发起人确认保留该基线（若要求回退 1.24，将重新引入已知漏洞）。

### 5.2 文档与脚本的轻微不一致（不阻断启动）

1. `testing-strategy.md` §9 列出的「迁移空库 up + down-to 0」「OpenAPI 契约」「SBOM/许可证」「密钥扫描」尚未进入 `scripts/ci.ps1`；`PROJECT-SETUP.md` §11 的「本地 CI」行也未声称覆盖它们。当前脚本与 §11 一致，建议在 F1 引入 OpenAPI 时同步补齐这些门禁。
2. `scripts/ci.ps1` 未调用 `scripts/smoke.ps1`（Smoke 在 §11 为独立命令，本次已单独执行通过）。建议后续将 Smoke 并入 CI 作为启动栈冒烟。
3. `deployment.md` §7 要求的日志字段为 `action/entity_type/entity_id/code/cache_status`，当前请求日志字段为 `method/path/status/duration_ms/request_id`；JSON 行格式已达标，业务字段待 F1 业务日志落地时补齐。
4. `.env.example` 注释「修改口令时同步修改此连接串」在 Compose 下不生效：`docker-compose.yml` 的 app 环境用 `POSTGRES_PASSWORD` 拼装 `DATABASE_URL`，改口令会自动同步；该注释仅适用于无 Docker 的本地运行，建议澄清。
5. `architecture.md` §5.3 的 `scripts/fixtures/perf_seed.go` 与 `backend/templates/` 尚未建立，属 F1/性能阶段计划，不属 P7 启动范围。

## 6. 环境恢复与清理

- 验收结束时已执行 `docker compose up -d` 并强制重建 app；`docker compose ps` 为 app/db/cache 全部 `Up (healthy)`，可通过 `http://127.0.0.1:17860` 访问。
- 镜像 `novel-settings:dev`（`b52017431bba`）为本次 CI 构建产物。
- 临时脚本 `%TEMP%\novel-qa-api.ps1` 已删除；`scripts/ci.ps1` 生成的 `novel-server-ci.exe`/`novel-cli-ci.exe` 已由脚本自身清理。
- 未修改业务代码、未改 `bootstrap-verification.md`、未执行 git commit；`git status` 中仅本文件为新增/修改。

## Approval

- P8 复验结论：**PASS**，可进入 F1。首轮两项阻断均已关闭，无遗留阻断项。
- 前提：§5.1 的 Go 1.26 基线需项目发起人追认；§5.2 各项作为 F1 及后续阶段的跟踪项，不阻断启动。
- 验收人：P8 QA（第二位独立验收人，独立于 P7 作者与首轮 P8 验收人）；日期：2026-11-08。
