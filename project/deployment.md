# 部署与运维（Deployment & Operations）

| 项目 | 内容 |
|---|---|
| 文档版本 | v1.1 |
| 状态 | 修订（P5b，2026-11-07，待复审） |
| 日期 | 2026-11-07 |
| 目标环境 | Windows + Docker Desktop（单用户本地） |
| 访问方式 | `http://127.0.0.1:17860`（仅本机回环） |
| 拓扑 | 三容器：`app`（Go + Next 静态资源）、`db`（PostgreSQL 16）、`cache`（Redis 7）；`cli` 一次性运维容器 |
| 上游 | [architecture.md](file:///d:/Project/novel-management/project/architecture.md)、[data-model.md](file:///d:/Project/novel-management/project/data-model.md)、[ADR-006](file:///d:/Project/novel-management/project/adr/ADR-006-export-restore-cli.md)、[ADR-007](file:///d:/Project/novel-management/project/adr/ADR-007-redis-cache.md)、[PROJECT-SETUP.md](file:///d:/Project/novel-management/project/PROJECT-SETUP.md) |

---

## 1. Environments

| 环境 | 用途 | 部署方式 | 数据 |
|---|---|---|---|
| 开发（local-dev） | 编码、单测、组件测试 | `docker compose up -d db cache` + `go run ./cmd/server` + `npm run dev` | `backend/.data/` 临时目录（不入库）；本地 PG volume |
| 集成（local-ci） | 集成/E2E/性能 | Docker Compose（本地构建） | 一次性 volume，测试后删除 |
| 生产（local-prod） | 用户实际使用 | Docker Compose + 固定镜像 tag | `pgdata` + `appdata` named volume |

- 本版无云端/公网环境；未来公网部署必须新增访问控制、HTTPS、备份策略，并另行批准。
- 不引入 staging/prod 多环境差异；配置项只有连接串、端口、数据目录、日志级别与上传上限。

---

## 2. Configuration and Secrets

### 2.1 环境变量

| 变量 | 默认 | 说明 | 敏感 |
|---|---|---|---|
| `APP_HOST` | `0.0.0.0` | 容器内绑定地址；**容器内必须监听 `0.0.0.0`**，宿主仅通过 compose 的 `127.0.0.1:17860:17860` 回环可达，不监听局域网（P5b RT-02） | 否 |
| `APP_PORT` | `17860` | 服务端口；Host/Origin 白名单由此派生 | 否 |
| `ALLOWED_HOSTS` | `127.0.0.1:${APP_PORT},localhost:${APP_PORT}` | Host 白名单（逗号分隔）；改端口无需改白名单（P5b RT-23） | 否 |
| `DATA_DIR` | `/data` | 图片/导出/快照/临时目录（volume） | 否 |
| `DATABASE_URL` | `postgres://novel:${POSTGRES_PASSWORD}@db:5432/novel?sslmode=disable` | PostgreSQL 连接串（容器内） | 是（含口令） |
| `REDIS_ADDR` | `cache:6379` | Redis 地址；空值表示禁用缓存 | 否 |
| `REDIS_PASSWORD` | 空 | 可选；本版默认无口令且不暴露端口 | 是（若设置） |
| `CACHE_TTL_SEC` | `60` | 检索/摘要缓存 TTL（上限 300） | 否 |
| `MAX_UPLOAD_MB` | `6144` | 导入 zip 上限（6GB，可调） | 否 |
| `SNAPSHOT_KEEP` | `3` | 快照保留份数 | 否 |
| `IMPORT_TTL_MIN` | `120` | 导入临时目录过期 | 否 |
| `IMPORT_TIMEOUT_MIN` | `120` | 导入上传/解析 HTTP 超时（分钟，P5b RT-12） | 否 |
| `LOG_LEVEL` | `INFO` | `DEBUG`/`INFO`/`WARN`/`ERROR` | 否 |
| `TZ` | `Asia/Shanghai` | 展示时区；DB 存 UTC | 否 |

- 导入预检：上传前校验可用空间 ≥ `zip_size + 预计解压大小 + 1GB 快照余量`；不足返回 `IMPORT_INSUFFICIENT_STORAGE`，超时返回 `IMPORT_TIMEOUT`（P5b RT-12）。
- `ALLOWED_HOSTS` 为空时由 `APP_PORT` 派生默认白名单；容器内 `0.0.0.0` 仅影响监听地址，不改变主机白名单（P5b RT-02/23）。

### 2.2 密钥

- 本版**无第三方密钥**：无账号、无 Token、无云服务、无大模型 API、无对象存储。
- PostgreSQL 口令仅用于容器内连接，`.env` 不提交仓库；仓库提供 `.env.example`（本地占位值，明确标注仅本地使用）。
- Redis 默认不设口令、不映射宿主端口；若未来启用口令，使用本地未提交 `.env` 或 Docker secret。
- `.dockerignore` 排除 `.env`、`data/`、`node_modules/`、`*.dump`、`snapshots/`、`exports/`。
- 日志与错误信息不得输出 `DATABASE_URL` 中的口令。

### 2.3 数据目录布局

```text
/appdata/                    # appdata named volume，挂载到 /data
├─ images/{work_id}/         # 人物主图与缩略图
├─ snapshots/                # 自动快照（app.dump + images + snapshot.json + manifest.json + checksums.sha256，保留 3 份）
├─ exports/                  # 导出 zip
├─ tmp/imports/{id}/         # 导入临时目录（TTL 清理）
├─ tmp/restore-{ts}/         # 恢复临时目录
└─ logs/app.log              # 结构化日志（滚动，可关闭文件输出只用 stdout）

pgdata/                      # pgdata named volume，PostgreSQL 数据目录
```

- 所有路径为容器内逻辑路径；API/日志不暴露宿主机绝对路径。
- 使用 named volume 而非 Windows 绑定目录：避免 NTFS 权限/硬链接差异；用户通过导出包做外部备份。
- 数据目录权限仅容器运行用户可读写。

---

## 3. Build Artifact

### 3.1 多阶段 Dockerfile（计划）

```dockerfile
# stage 1: frontend static export
FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build          # next build，output: 'export' → /web/out

# stage 2: Go binaries
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cli ./cmd/cli

# stage 3: runtime（含 pg_dump 16）
FROM postgres:16-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/server /out/cli ./
COPY --from=web /web/out ./web
COPY backend/templates ./templates
ENV APP_HOST=0.0.0.0 APP_PORT=17860 DATA_DIR=/data
EXPOSE 17860
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl -fsS http://127.0.0.1:17860/api/v1/health || exit 1
ENTRYPOINT ["/app/server"]
```

- 镜像打标签：`novel-settings:<version>-<git-sha>`；禁止只用 `latest` 发布。
- `postgres:16-bookworm` 作为运行基座，确保 `pg_dump`/`pg_restore` 版本与服务端一致。
- 构建产物包含前端 `out`、Go 二进制、嵌入式迁移、导入模板与示例。
- 不把测试数据、设计稿、`.env` 打入镜像。
- 镜像必须支持 `linux/amd64`（Windows Docker Desktop 默认），可选 `arm64`。

### 3.2 docker-compose.yml（计划）

```yaml
services:
  app:
    image: novel-settings:${APP_VERSION:-dev}
    build: .
    ports:
      - "127.0.0.1:17860:17860"
    environment:
      APP_HOST: 0.0.0.0
      APP_PORT: 17860
      ALLOWED_HOSTS: 127.0.0.1:${APP_PORT:-17860},localhost:${APP_PORT:-17860}
      DATA_DIR: /data
      DATABASE_URL: postgres://novel:${POSTGRES_PASSWORD:-nm_local_only}@db:5432/novel?sslmode=disable
      REDIS_ADDR: cache:6379
      CACHE_TTL_SEC: 60
      TZ: Asia/Shanghai
    volumes:
      - appdata:/data
    depends_on:
      db:
        condition: service_healthy
      cache:
        condition: service_started
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: novel
      POSTGRES_USER: novel
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-nm_local_only}
      TZ: UTC
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U novel -d novel"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    # 不映射宿主端口

  cache:
    image: redis:7-alpine
    command: ["redis-server", "--save", "", "--appendonly", "no",
              "--maxmemory", "256mb", "--maxmemory-policy", "allkeys-lru"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
    restart: unless-stopped
    # 不持久化、不映射宿主端口

  cli:
    image: novel-settings:${APP_VERSION:-dev}
    profiles: ["tools"]
    entrypoint: ["/app/cli"]
    environment:
      DATA_DIR: /data
      DATABASE_URL: postgres://novel:${POSTGRES_PASSWORD:-nm_local_only}@db:5432/novel?sslmode=disable
    volumes:
      - appdata:/data
    depends_on:
      db:
        condition: service_healthy

volumes:
  pgdata:
  appdata:
```

- 端口绑定必须带 `127.0.0.1:` 前缀；`db`/`cache` 不发布端口。
- `cache` 使用 `allkeys-lru`，内容可随时丢弃；`depends_on` 不等待其健康，保证 Redis 故障时 app 仍能启动并旁路。
- `pgdata` 与 `appdata` 分离：数据库与图片/导出/快照独立备份。

---

## 4. Database Migration

- 迁移文件嵌入 Go 二进制（`internal/migrations`，goose v3）。
- 启动顺序：
  1. 校验 `DATA_DIR` 可写；
  2. 读取配置并连接 PostgreSQL（带重试，最多 60s）；
  3. 获取 PostgreSQL advisory lock（`pg_try_advisory_lock(17860)`），失败则退出并提示「另一个实例正在运行或正在恢复」；
  4. 执行 `goose.Up`（单事务/每个迁移）；
  5. 扫描 `import_runs`/`export_runs` 中 `running` → `failed`；
  6. 启动 HTTP 服务。
- 迁移失败：进程退出（非 0），日志输出失败版本与错误；不得以旧 schema 启动。
- 破坏性迁移：必须先导出或快照；在副本数据上演练并记录。
- 降级：不保证自动降级；回滚通过恢复升级前快照/导出包（§8）。
- CLI 手动迁移：先 `docker compose stop app`（释放 advisory lock），再 `docker compose run --rm cli migrate`；CLI 获取**同一** advisory lock，锁不可用（app 仍运行）时退出并输出 `APP_RUNNING`（P5b RT-22）。

---

## 5. Deployment Steps

### 5.1 首次部署

1. 安装 Docker Desktop（Windows），启用 WSL2 后端。
2. 获取仓库并构建：`docker compose build`。
3. 复制 `.env.example` 为 `.env`（可选，仅改 `POSTGRES_PASSWORD`/端口/上限）。
4. 启动：`docker compose up -d`。
5. 验证：`docker compose ps` 中 `app`/`db` healthy（`cache` up）；浏览器打开 `http://127.0.0.1:17860`。
6. 首次进入：创建作品 → 配置类别/榜单 → 录入或导入。

### 5.2 升级

1. 设置页执行「全量导出」，确认 zip 可下载。
2. 停止：`docker compose down`（保留 `pgdata` 与 `appdata`）。
3. 构建/拉取新镜像（记录版本与 digest）。
4. 启动：`docker compose up -d`；观察迁移日志。
5. 执行 Smoke Test（§6）。
6. 失败：按 §9 回滚镜像；若 schema 已变更，恢复升级前快照/导出包。

### 5.3 恢复（CLI）

```powershell
# 1) 停止应用，避免并发写入
docker compose stop app

# 2) 将导出 zip 放入 appdata 卷的 exports 目录（或通过 docker cp）
# 3) 执行恢复（一次性 cli 容器）
docker compose run --rm cli restore /data/exports/novel-settings-20261107-101530.zip

# 4) 启动并验证
docker compose up -d
```

CLI 行为：
1. 校验 `manifest.json`、`export_version`、`schema_version`、`checksums.sha256`；
2. 预检磁盘空间（解压后大小 + 当前快照）；
3. 生成恢复前快照（`restore`）；
4. 解压到 `/data/tmp/restore-{ts}/`，校验图片数量与引用；
5. `pg_restore --clean --if-exists --no-owner --single-transaction -d "$DATABASE_URL"`；
6. 事务成功后替换 `images/`（旧目录先移至临时备份，再重命名新目录）；
7. 执行 `goose.Up` 对齐当前版本；`reindex`（可选）；
8. 输出恢复计数与结果路径。

- 恢复期间 app 停止，health 返回 `OFFLINE` 属预期；失败时 `pg_restore` 事务回滚，原数据库不变；图片替换失败则用恢复前快照回退。
- 恢复完成后在浏览器抽查：作品数、人物数、图片、榜单、回收站。
- **恢复命令唯一入口**（P5b RT-18）：`docker compose run --rm cli restore <zip>`；`scripts/restore.sh` 不作为权威路径，删除或仅保留为薄封装（内部仍调用同一 CLI）。
- 恢复成功后写入 `app_settings.last_restore_at`，设置页展示（P5b RT-06）。

#### 快照恢复（P5b RT-06）

```powershell
docker compose stop app
docker compose run --rm cli restore-snapshot /data/snapshots/20261107-101530-import
docker compose up -d
```

- 快照目录固定含 `app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`。
- CLI 校验 `manifest.json`、`checksums.sha256`、`schema_version`/`app_version` 兼容性后，恢复 DB + images；版本不兼容时拒绝并提示使用导出包跨版本恢复。
- 快照恢复同样先做一次恢复前快照（若空间允许），失败可用上次快照回退。
- 快照触发以 requirements FR-55 为准：CLI 全量恢复、导入、删除作品、永久删除、清空回收站；回收站单项恢复不生成；手动快照仅 CLI 可选，无页面入口（P5b RT-17）。

### 5.4 无 Docker 的开发启动（可选）

```powershell
# 基础设施
docker compose up -d db cache

# 后端（仅本机开发时绑定回环；容器内默认 0.0.0.0）
cd backend
$env:APP_HOST="127.0.0.1"
$env:DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
$env:REDIS_ADDR="127.0.0.1:6379"
go run ./cmd/server

# 前端（另一个终端）
cd frontend
npm ci
npm run dev          # next dev，/api 代理到 127.0.0.1:17860
```

> 开发环境需临时映射 `db`/`cache` 端口（覆盖文件 `compose.dev.yml`），生产 compose 不映射。

---

## 6. Smoke Test

发布/升级后必须执行（可由 `scripts/smoke.ps1` 自动化）：

| 检查 | 期望 |
|---|---|
| `GET /api/v1/health` | `200`，`status=ok`，`db=ok`，`cache=ok|degraded` |
| `GET /api/v1/meta` | `200`，`cache_enabled` 与配置一致 |
| 首页加载 | `200`，无控制台错误，首屏 ≤2s |
| 创建作品 | `201`；自动出现「境界」类别 |
| 创建人物 + 别名 | `201`；检索可命中别名 |
| 检索缓存 | 第二次相同查询 `X-Cache: HIT`（Redis 正常时） |
| Redis 停止 | 检索返回 `200` 且 `X-Cache: BYPASS`；恢复 Redis 后恢复 `HIT` |
| 删除无引用人物 → 撤销 | toast 出现；撤销后恢复 |
| 检索 2 字查询 | 返回结果或明确空提示，无 500 |
| 设置页 | 显示版本、数据目录、快照列表 |
| 日志 | 无堆栈、无正文、无绝对路径、无连接串口令 |

- Smoke 失败视为发布失败；记录 `FAIL` 与日志位置。
- 每次发布保留 Smoke 输出（`artifacts/smoke/{date}.txt`）。

---

## 7. Observability / Logs

- 日志：stdout（`docker compose logs -f app`）+ 可选 `/data/logs/app.log`（10MB × 5 滚动）。
- 格式：JSON 行：`ts, level, request_id, action, entity_type, entity_id, code, duration_ms, cache_status`。
- 禁止记录：设定正文、完整文件路径、堆栈（生产）、请求 body、Redis 键值、`DATABASE_URL` 口令。
- 健康检查：`GET /api/v1/health`；Docker healthcheck 使用同一端点。
- PostgreSQL：错误日志走容器 stdout；慢查询日志默认关闭（数据量小）。
- Redis：不开启持久化；`INFO` 仅用于人工排查，不进入应用日志。
- 导入报告、快照列表、最近导出/恢复时间在设置页可见。
- 无 APM、无遥测、无外部日志服务。
- 故障排查顺序：`docker compose ps` → `docker compose logs --tail=200 app` → `db` 健康 → `/api/v1/health` → 磁盘空间 → 卷权限 → Redis `ping`。

---

## 8. Backup / Restore

| 类型 | 触发 | 位置 | 保留 |
|---|---|---|---|
| 自动快照 | CLI 全量恢复、导入、删除作品、永久删除、清空回收站（回收站单项恢复不生成；手动快照仅 CLI 可选） | `appdata/snapshots/{ts}-{reason}/` | 最近 3 份 |
| 全量导出 | 用户手动（设置页） | `appdata/exports/novel-settings-YYYYMMDD-HHmmss.zip` | 用户自管 |
| 用户外部备份 | 手动复制导出包 | 用户自选（移动硬盘/网盘） | 用户自管 |

- 快照内容（固定）：`app.dump`（`pg_dump -Fc`）+ `images/`（同 volume 硬链接，失败降级为复制）+ `snapshot.json` + `manifest.json` + `checksums.sha256`；不含 `snapshots/` 自身。
- 快照恢复：`docker compose run --rm cli restore-snapshot <dir>`；校验 manifest/checksums/版本后恢复 DB + images，并写 `last_restore_at`（P5b RT-06）。
- 导出包：`manifest.json`、`checksums.sha256`、`data/app.dump`、`images/`；`includes_trash` 控制是否含回收站记录（不含时由临时库过滤后转储）。
- 恢复前必须验证：`export_version`、`schema_version`、checksum、图片数量、引用完整、磁盘空间（导出包与快照同一套校验逻辑）。
- 迁移前、公网部署前必须执行一次空环境恢复演练并留存记录（FR-56）。
- 文档提示：导出包与 `appdata` 包含私密内容，备份到外部时自行加密。
- **禁止**直接复制 `pgdata` 目录作为备份（可能不一致）；必须使用 `pg_dump` 或导出包。

---

## 9. Rollback

| 场景 | 回滚方式 |
|---|---|
| 新镜像启动失败/迁移失败 | `docker compose down` → 切回上一镜像 tag → `up`；若 schema 已变更，恢复升级前导出包/快照 |
| 功能回归 | 回滚镜像 tag；保留数据；记录回归用例 |
| 数据误操作 | 回收站恢复 → 若已永久删除，使用最近快照或导出包（CLI） |
| 导入失败 | 不写入正式库；修正文件后重跑（原子性保证） |
| 恢复失败 | `pg_restore` 单事务回滚，原数据不变；图片未替换；修复 zip 后重试；或用恢复前快照 |
| 快照恢复失败 | 校验失败即中止、不触碰数据；DB 恢复失败回滚；可用更早快照重试（`restore-snapshot`，P5b RT-06/18） |
| Redis 故障 | 无需回滚；应用自动旁路，恢复 Redis 后缓存自然重建 |

- 禁止通过手工 `DELETE FROM` 或直接修改 `pgdata` 回滚；必须走快照/导出恢复。
- 回滚后必须再次执行 Smoke Test。

---

## 10. Release Approval

发布前检查（全部通过才可发布）：

- [ ] 所有 Must 需求验收通过；`NOT RUN`/`BLOCKED` 已记录并有人工豁免。
- [ ] CI 质量门禁通过（testing-strategy §9）。
- [ ] 性能基准 V-01~V-07 达标；结果附发布记录。
- [ ] 空环境恢复演练通过（FR-56），含/不含回收站两种导出包。
- [ ] 快照恢复演练通过（`restore-snapshot`，至少一次），记录耗时与结果。
- [ ] Smoke Test 通过。
- [ ] 迁移在副本数据上演练通过。
- [ ] 版本 tag、镜像 digest、变更说明与回滚步骤已记录。
- [ ] 项目发起人批准（架构、数据模型、部署方式变更均需批准）。

审批人：项目发起人本人。未获批准不得发布到公网或修改部署拓扑。

---

## 11. 决策与默认项

| 编号 | 问题 | 采用 | 备选 |
|---|---|---|---|
| D-05 | 恢复方式 | CLI 停止应用后全量恢复；快照恢复为 `restore-snapshot <dir>`，唯一入口 `docker compose run --rm cli ...`（已确认 2026-11-07；P5b RT-06/18） | 页内在线恢复 |
| D-08 | Redis 部署 | 独立容器、无持久化、不暴露端口（已确认 2026-11-07） | 与 app 同容器 / 不使用 Redis |
| D-09 | 前端交付 | Next 静态导出，由 Go 托管（已确认 2026-11-07） | Node 运行时 + Nginx |
| — | 数据卷类型 | named volume `pgdata`/`appdata` | Windows 绑定目录（不推荐） |
| — | 端口 | `17860` | 用户自定义 |
| — | 镜像分发 | 本地构建 | `docker save` 离线分发 |
| — | 数据库口令默认值 | `.env` 本地占位 `nm_local_only` | 首次启动生成随机口令 |

---

## 12. 变更记录

| 日期 | 版本 | Finding ID | 决策 | 原因/变更摘要 |
|---|---|---|---|---|
| 2026-11-07 | v1.1 | RT-02 | 容器内 `APP_HOST=0.0.0.0`、`APP_PORT=17860`；compose 发布 `127.0.0.1:17860:17860`；删除“禁止改为 0.0.0.0” | 容器内监听 loopback 会导致宿主端口转发不可达 |
| 2026-11-07 | v1.1 | RT-06 | 快照目录固定结构；新增 `restore-snapshot`；`last_restore_at` | 快照原先不可恢复，FR-55 无法验收 |
| 2026-11-07 | v1.1 | RT-09 | 恢复整组支持 `renames[]`（API 契约） | 子实体冲突需逐项改名 |
| 2026-11-07 | v1.1 | RT-12 | `IMPORT_TIMEOUT_MIN=120`；空间预检与错误码 | 大包上传/解析易超时或写满磁盘 |
| 2026-11-07 | v1.1 | RT-17 | 快照触发补“永久删除”，明确回收站单项恢复不生成、手动快照仅 CLI | 对齐 FR-55 |
| 2026-11-07 | v1.1 | RT-18 | 恢复唯一入口 `docker compose run --rm cli restore/restore-snapshot`；`scripts/restore.sh` 非权威 | 消除两条恢复路径的歧义 |
| 2026-11-07 | v1.1 | RT-22 | 手动迁移前 `stop app`；CLI 获取同一 advisory lock，锁占用退出 `APP_RUNNING` | 避免迁移与在线写并发 |
| 2026-11-07 | v1.1 | RT-23 | 新增 `ALLOWED_HOSTS`，默认由 `APP_PORT` 派生 | 改端口不再触发 403 |
