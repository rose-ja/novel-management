# ADR-002: Go + Next.js + PostgreSQL + Redis 技术栈

- Status: Accepted（2026-11-07）
- Date: 2026-11-07
- Decision owner: 项目发起人（已确认方向：方案一基础上加入 Redis 做缓存）；技术方案：架构师

## Context

需求要求：本地 Docker、单用户、中文界面、桌面浏览器、零成本、无云端大模型；功能覆盖复杂表单、树、席位表、多步骤导入、Toast 撤销；性能硬指标为检索 P95 ≤150ms、保存 ≤300ms、首屏 ≤2s。`PROJECT-SETUP.md` §10 技术栈原为 `TBD`。

项目发起人提供了个人学习路线（飞书 Wiki），其中业务后端主线为 Go + gin + gorm，前端 Next.js + Tailwind，数据库 PostgreSQL，并使用 Redis；Python/FastAPI 用于 RAG/Agent 项目。本项目无 AI 需求，因此不引入 Python 服务。

## Decision Drivers

- 与学习路线一致，便于独立复现与讲解。
- 中文检索可行性（PostgreSQL `pg_trgm` / 短词回退）。
- zip 解压、Markdown 模板解析、图片校验与缩略图处理的成熟度。
- 前端交互复杂度与 P3 设计系统落地成本。
- 长期零成本、无付费依赖、许可证合规。
- 测试生态与可维护性。

## Options Considered

| 维度 | A. Go + Next.js + PostgreSQL + Redis（选用） | B. FastAPI + React + SQLite | C. Node/NestJS + React + PostgreSQL + Redis | D. Go + Python 双服务 + Next.js + PG + Redis |
|---|---|---|---|---|
| 与学习路线一致 | 高 | 中 | 中 | 高（但 Python 无业务用途） |
| 中文检索 | `pg_trgm` + 短词 `ILIKE`，V-01 | FTS5 + 短词回退，V-01 | 同 A | 同 A |
| 部署复杂度 | 中（3 容器） | 低（单容器） | 中（3 容器） | 高（4 容器 + 双服务契约） |
| 事务/一致性 | 强（PG 单库事务） | 强（SQLite 单写者） | 强 | 跨服务一致性成本高 |
| 图片/zip 处理 | Go 标准库 + `imaging`，成熟 | Python Pillow，成熟 | Node 原生模块风险 | 同 A |
| 后台任务 | 进程内 goroutine + 状态表 | 进程内线程 + 状态表 | 进程内 + 可选 BullMQ | 跨服务任务，复杂 |
| 学习收益 | 高（分层、索引、缓存、迁移） | 中 | 中 | 高但偏离产品需求 |
| 主要代价 | 需维护 PG/Redis 容器与迁移 | 技术栈与路线不一致 | 偏离 Go 主线 | 引入无需求的 Python 服务 |

## Decision

采用 **A**，具体版本与依赖见 [tech-stack.md](file:///d:/Project/novel-management/project/tech-stack.md)。

- 后端：Go 1.24、gin、gorm（postgres driver）、pgx v5、goose、viper、cobra、validator、slog、ULID、goldmark、`disintegration/imaging`、go-redis/v9。
- 数据库：PostgreSQL 16 + `pg_trgm`。
- 缓存：Redis 7，仅 cache-aside，非关键路径（ADR-007）。
- 前端：Next.js 15 App Router（`output: 'export'`）、React 19、TypeScript 5、Tailwind CSS 4、TanStack Query 5、React Hook Form + Zod、Radix UI、lucide-react。
- 契约：OpenAPI 3.1 契约优先，`oapi-codegen` + `openapi-typescript`。
- 测试：Go testing/testify/testcontainers-go/miniredis、Vitest/RTL、Playwright、k6。

不引入：Python/FastAPI、消息队列、Elasticsearch/Meilisearch、MUI/AntD/Chakra/Bootstrap、Redux/Zustand/MobX、Prisma/Ent、云服务、大模型 SDK。

## Consequences

- 正面：技术栈与学习路线一致；PG 事务/索引强；Redis 故障可旁路；Next 静态导出由 Go 托管，无额外 Node 运行时；API 契约可生成双端类型。
- 负面：三容器运维与迁移复杂度高于单容器；Go 图片处理与前端构建需要多阶段镜像；`pg_trgm` 中文短词能力未验证（V-01）。
- 兼容策略：API 前缀 `/api/v1`；导出格式 `export_version=1`；模板 `template_version=1.0`。

## Risks and Mitigations

| 风险 | 缓解 |
|---|---|
| PG 中文短词检索不达标 | ADR-003 回退路径；`SearchService` 接口隔离；V-01 阻断 |
| Redis 成为隐性依赖 | `Cache` 接口 + Noop 实现；V-07；代码评审禁止关键路径直接读缓存 |
| 前端静态导出限制（无 SSR） | 所有数据经 API；路由全部客户端渲染；Playwright 验证 |
| 多容器启动顺序/健康检查 | compose healthcheck + app 启动重试；Smoke Test |
| 依赖升级破坏设计系统 | 锁定次版本；Playwright 视觉回归 |

## Follow-up Actions

- [x] 项目发起人确认技术栈（2026-11-07）。
- [ ] V-01 检索 PoC 通过后冻结搜索实现。
- [ ] 生成并提交 SBOM 与许可证清单。
