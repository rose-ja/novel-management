# 技术栈（Tech Stack）

| 项目 | 内容 |
|---|---|
| 文档版本 | v1.0 |
| 状态 | Approved（2026-11-07，项目发起人） |
| 日期 | 2026-11-07 |
| 选择依据 | 项目发起人的学习路线（Go/gin/gorm、Next.js、PostgreSQL、Redis）+ 产品约束（本地、零成本、无 AI） |
| 配套 | [architecture.md](file:///d:/Project/novel-management/project/architecture.md)、[deployment.md](file:///d:/Project/novel-management/project/deployment.md) |

> 版本号在脚手架阶段写入 `go.mod` / `package.json` / `docker-compose.yml` 并锁定；本文给出版本线，不保证具体补丁号。

---

## 1. 总览

| 层 | 选型 | 版本线 |
|---|---|---|
| 语言（后端） | Go | 1.24.x |
| Web 框架 | gin | v1.10.x |
| ORM | gorm + gorm postgres driver（pgx v5） | v1.25+ / v1.5+ |
| 数据库 | PostgreSQL | 16.x（`postgres:16-alpine`） |
| 检索扩展 | `pg_trgm`（contrib） | 随 PG 16 |
| 缓存 | Redis（仅缓存） | 7.2.x（`redis:7.2-alpine`，BSD-3-Clause） |
| 迁移 | goose（SQL migrations，embed） | v3.x |
| 配置 | viper + 环境变量 | v1.20.x |
| CLI | cobra | v1.10.x |
| 校验 | go-playground/validator | v10.x |
| 日志 | Go 标准库 `log/slog`（JSON） | 随 Go |
| ID | `oklog/ulid/v2` | v2.1.x |
| Markdown 解析 | `yuin/goldmark` | v1.7.x |
| 图片处理 | `disintegration/imaging` + `golang.org/x/image/webp` | 最新稳定 |
| Redis 客户端 | `redis/go-redis/v9` | v9.x |
| 前端框架 | Next.js（App Router，静态导出） | 15.x |
| UI 语言 | React + TypeScript | 19.x / 5.x |
| 样式 | Tailwind CSS + P3 CSS 变量 | 4.x |
| 数据请求 | TanStack Query | 5.x |
| 表单 | React Hook Form + Zod | 7.x / 3.x |
| 无样式组件 | Radix UI 原语 | 最新稳定 |
| 图标 | lucide-react | 最新稳定 |
| 契约生成 | OpenAPI 3.1 + oapi-codegen（Go）+ openapi-typescript（TS） | 最新稳定 |
| 后端测试 | `testing` + testify + testcontainers-go + httptest | 最新稳定 |
| 前端测试 | Vitest + Testing Library + Playwright | 3.x / 最新稳定 |
| 压测 | k6 | 最新稳定 |
| 静态检查 | golangci-lint、go vet、govulncheck、ESLint、tsc | 最新稳定 |
| 容器 | Docker + Docker Compose | Docker Desktop 最新稳定 |

---

## 2. 后端依赖

### 2.1 runtime

| 模块 | 用途 |
|---|---|
| `github.com/gin-gonic/gin` | HTTP 路由、中间件、绑定 |
| `gorm.io/gorm`、`gorm.io/driver/postgres` | ORM、事务、预加载 |
| `github.com/jackc/pgx/v5` | PostgreSQL 驱动（由 gorm driver 使用） |
| `github.com/pressly/goose/v3` | 迁移（embed SQL） |
| `github.com/spf13/viper` | 配置加载（env + 默认值） |
| `github.com/spf13/cobra` | CLI（restore/reindex/doctor） |
| `github.com/go-playground/validator/v10` | 请求字段校验 |
| `github.com/oklog/ulid/v2` | 26 位可排序 ID |
| `github.com/redis/go-redis/v9` | Redis 缓存 |
| `github.com/yuin/goldmark` | 固定模板 Markdown 解析 |
| `github.com/disintegration/imaging` | 缩略图、重编码 |
| `golang.org/x/image/webp` | WebP 解码 |
| `log/slog`（标准库） | 结构化日志 |
| `archive/zip`、`crypto/sha256`、`mime`（标准库） | 导入、校验、上传 |

### 2.2 dev/test

| 模块 | 用途 |
|---|---|
| `github.com/stretchr/testify` | 断言与 suite |
| `github.com/testcontainers/testcontainers-go` | 集成测试启动 PG/Redis |
| `github.com/alicebob/miniredis/v2` | 缓存单元测试 |
| `net/http/httptest`（标准库） | API handler 测试 |
| `github.com/oapi-codegen/oapi-codegen/v2` | 由 OpenAPI 生成 Go 类型/服务器接口 |
| `golangci-lint` | 静态检查 |
| `govulncheck` | 依赖漏洞扫描 |

- 不使用 `sqlmock` 作为主要数据库测试手段（不 mock DB，见 testing-strategy §6）。
- 不使用 GORM `AutoMigrate` 作为生产迁移；迁移只走 goose。

---

## 3. 前端依赖

### 3.1 runtime

| 模块 | 用途 |
|---|---|
| `next` | App Router、静态导出、路由 |
| `react`、`react-dom` | UI |
| `typescript` | 类型 |
| `tailwindcss` | Token 映射与工具类 |
| `@tanstack/react-query` | 服务端状态、缓存、轮询 |
| `react-hook-form` | 表单状态 |
| `zod` | 表单校验 |
| `@hookform/resolvers` | RHF + Zod 适配 |
| `@radix-ui/react-dialog`、`react-select`、`react-dropdown-menu`、`react-tooltip`、`react-alert-dialog` | 可访问性原语 |
| `lucide-react` | 图标 |
| `openapi-fetch` | 类型化 API 客户端 |

### 3.2 dev

| 模块 | 用途 |
|---|---|
| `vite`? | 不使用；Next.js 自带构建 |
| `openapi-typescript` | 由 OpenAPI 生成 TS 类型 |
| `vitest`、`@testing-library/react`、`@testing-library/user-event` | 组件测试 |
| `playwright` | E2E 与视觉回归 |
| `eslint`、`eslint-config-next`、`typescript-eslint` | 静态检查 |
| `prettier` | 格式化（可选） |

---

## 4. 明确不引入

| 类别 | 不引入 | 理由 |
|---|---|---|
| Python/FastAPI | 不作为业务服务 | 无 AI/RAG/Agent 需求；双服务增加契约与运维成本 |
| 队列/消息中间件 | Redis Stream、Kafka、RabbitMQ、Celery、BullMQ | 导入/导出用进程内单 worker + 状态表即可 |
| 搜索服务 | Elasticsearch、Meilisearch、OpenSearch | PG `pg_trgm` 足够；避免额外服务 |
| 缓存框架 | 不引入 Redis 之外的缓存 | Redis 仅 cache-aside |
| 前端组件库 | MUI、AntD、Chakra、Bootstrap | 与 P3 设计系统冲突；Radix + Tailwind 足够 |
| 状态管理大库 | Redux、Zustand、MobX | TanStack Query + Context 足够 |
| 其他 ORM | Prisma、Ent | gorm 已覆盖 |
| 云服务 | 对象存储、云数据库、付费 API、大模型 | 零成本、本地私密 |
| 站点框架 | Nginx/反向代理（本版） | Go 直接托管静态资源；公网阶段再加 |

---

## 5. 方案理由

### 5.1 为什么 Go + gin + gorm

- 与学习路线的业务后端主线一致（W4–W8、W15–W17）。
- 单二进制部署、启动快、内存占用低；标准库覆盖 zip/图片/HTTP。
- gin 中间件模型适合本地 CSRF/Host/Origin/request-id；gorm 事务与预加载满足引用校验。
- 代价：需要手写更多分层与错误映射；通过 `oapi-codegen` + 统一 `apperr` 包降低重复。

### 5.2 为什么 PostgreSQL

- 事务、部分唯一索引、`CHECK`、外键、`JSONB` 成熟；比 SQLite 更适合多容器与并发读写。
- `pg_trgm` GIN 索引支持中文子串检索；短词用 `ILIKE` 回退（V-01）。
- `pg_dump`/`pg_restore` 提供一致性备份与恢复；比 SQLite 文件复制更可靠。
- 代价：多一个容器与连接配置；本地 Docker Desktop 资源占用增加。

### 5.3 为什么 Redis 只做缓存

- 检索结果与作品摘要重复读取多；cache-aside 可降低 PG 压力并稳定 P95。
- Redis 故障时旁路，不影响正确性；不持久化、不发布端口、不存正文。
- 不做队列：导入/导出任务用进程内 worker + DB 状态表，避免 Redis 成为关键路径。
- 代价：缓存一致性需要 epoch 失效；多一个容器。

### 5.4 为什么 Next.js + Tailwind

- 与学习路线 W2/W17 一致；App Router + 静态导出可托管为纯前端 SPA。
- Tailwind 可直接消费 P3 CSS 变量，Radix 提供 Dialog/Drawer/Select 的可访问性。
- 代价：静态导出不支持 SSR/Server Actions；本项目不需要。

### 5.5 为什么不用 Python/FastAPI

- 学习路线中的 FastAPI 用于 RAG/Agent；本项目明确不调用大模型、不做语义检索。
- 引入 Python 服务会造成“双服务但无业务理由”，违反简单成熟原则。

---

## 6. 版本与兼容策略

- API 前缀 `/api/v1`；破坏性变更升 `/api/v2`。
- 数据库迁移版本由 goose 管理；启动时 `goose.Up`，失败不启动。
- 导出包 `export_version=1`；至少兼容上一版本（见 deployment.md）。
- 导入模板 `template_version=1.0`；版本不匹配在预览阶段报错。
- 浏览器：Chrome/Edge 最新两个大版本；最低窗口 1024px。
- Go/Node 版本写入 `go.mod` / `.nvmrc` / `package.json engines`；CI 与 Docker 使用同一版本。

---

## 7. 开发命令（计划）

| 用途 | 命令 |
|---|---|
| 后端依赖 | `cd backend && go mod download` |
| 启动后端 | `go run ./cmd/server` |
| 迁移 | `goose -dir internal/migrations postgres "$DATABASE_URL" up`（或启动自动） |
| 后端单测 | `go test ./internal/... -run Unit` |
| 后端集成 | `go test ./... -tags=integration` |
| Lint | `golangci-lint run` |
| Vet | `go vet ./...` |
| 漏洞 | `govulncheck ./...` |
| 前端依赖 | `cd frontend && npm ci` |
| 前端开发 | `npm run dev` |
| 前端构建 | `npm run build`（`output: 'export'`） |
| 前端检查 | `npm run lint && npm run typecheck && npm run test` |
| E2E | `npx playwright test` |
| 全栈 | `docker compose up -d --build` |
| 压测 | `k6 run tests/k6/search.js` |
| CLI 恢复 | `go run ./cmd/cli restore ./data/exports/<zip>` |
| CLI 重建检索 | `go run ./cmd/cli reindex --work <id>` |
| CLI 自检 | `go run ./cmd/cli doctor` |

---

## 8. 许可证

| 组件 | 许可证 |
|---|---|
| Go 标准库 | BSD-3-Clause |
| gin、gorm、pgx、goose、viper、validator、goldmark、imaging、go-redis、ulid、testify、testcontainers-go、miniredis | MIT / BSD 类（以仓库 LICENSE 为准） |
| cobra | Apache-2.0 |
| PostgreSQL | PostgreSQL License |
| Redis 7.2 | BSD-3-Clause（固定 7.2，避免 7.4 之后的 RSALv2/SSPL） |
| Next.js、React、Tailwind、Radix、TanStack、RHF、Zod、Vitest | MIT |
| lucide-react | ISC |
| Playwright | Apache-2.0 |

- 生成 SBOM（`go list -m all` + `npm sbom`）；禁止 GPL/AGPL 依赖进入发布镜像。
- 若后续替换 Redis 为 Valkey（BSD-3-Clause），客户端仍使用 go-redis，协议兼容。

---

## 9. 版本与工具决策

以下决策随 P4 整体批准（2026-11-07）；后续升级需记录 ADR 或变更说明。

| 编号 | 决策 | 采用 | 备选（未采用） |
|---|---|---|---|
| TS-Q1 | Go 版本 | 1.24.x | 1.25.x |
| TS-Q2 | Next.js 交付 | 静态导出，Go 托管（D-09 已确认） | Node 运行时 + Nginx 反代 |
| TS-Q3 | 中文检索扩展 | `pg_trgm` + 短词 `ILIKE`（D-02 已确认，V-01 待执行） | `pg_bigm` / `zhparser` |
| TS-Q4 | 集成测试数据库 | testcontainers-go | 固定 `TEST_DATABASE_URL` |
| TS-Q5 | Redis 版本/替代 | Redis 7.2（BSD） | Valkey 8（BSD） |
| TS-Q6 | OpenAPI 来源 | 契约优先：YAML → oapi-codegen/openapi-typescript | swag 注解生成 |
