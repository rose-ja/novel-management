# ADR-001: 模块化单体与三容器部署

- Status: Accepted（2026-11-07）
- Date: 2026-11-07
- Decision owner: 项目发起人（已确认 2026-11-07）；技术方案：架构师

## Context

本项目是单用户、本地 Docker Desktop 运行的小说设定管理工具，零成本、离线、无登录、仅监听 `127.0.0.1`。业务包含作品、人物、势力、地点、榜单、体系、检索、导入、导出、快照、回收站等能力模块，数据规模上限为单作品 1000 人物、500 图片、作品 ≤10（`requirements.md` §6.7）。

项目发起人要求按个人学习路线选择技术栈，路线业务后端主线为 Go + gin + gorm、PostgreSQL 和 Redis。需要决定：采用单体、模块化单体、微服务，以及容器数量。

## Decision Drivers

- 部署必须一条命令启动，用户能自行操作 Docker。
- 数据一致性优先：人物/势力/榜单/体系之间存在大量引用与删除前置校验，跨服务事务会增加复杂度。
- 零成本：只使用开源、本地可运行的基础设施。
- 可学习：分层、事务、索引、缓存边界清晰，便于独立复现与讲解。
- Redis 只能是非关键路径，不能因缓存故障导致数据不可用。
- 未来公网部署不得被堵死。

## Options Considered

| 方案 | 优点 | 代价 |
|---|---|---|
| A. 单体应用（按页面分层） | 最简单 | 模块边界模糊，检索/引用/导入规则易散落，测试困难 |
| B. 模块化单体 + 三容器（推荐） | 单事务、模块职责清晰；PG/Redis 独立升级与观测；Redis 可旁路 | Compose 与运维略复杂；需遵守依赖方向 |
| C. 模块化单体 + 单容器内嵌 PG/Redis | 部署单元最少 | 容器内多进程、数据卷混用、升级与故障隔离差 |
| D. 微服务（API/搜索/导入独立） | 独立扩展 | 分布式事务、网络故障、运维成本与零成本目标冲突 |

## Decision

采用 **B：模块化单体 + 三容器**。

- 一个应用镜像 `app`：Go + gin 提供 `/api/v1`，并托管 Next.js 静态导出产物；单进程、单写入口。
- `db`：PostgreSQL 16，唯一持久化业务数据库；不暴露宿主端口。
- `cache`：Redis 7，仅做 cache-aside 缓存；无持久化、不暴露端口、不可用时自动旁路（ADR-007）。
- `cli`：同一镜像的一次性运维容器，执行 migrate/restore/reindex/doctor。
- 后端按业务能力分模块：`httpapi → service → domain → repository/model`；基础设施 `cache`、`search`、`imports`、`media`、`snapshots`、`jobs` 由 service 调用，不反向依赖 `httpapi`。
- 模块之间只通过应用服务函数与 DTO 交互，不直接读写对方表。
- 不引入消息队列、服务发现、Celery/BullMQ；后台任务使用进程内单 worker + 状态表，重启后 `running → failed`。

## Consequences

- 正面：事务边界清晰；PG 提供可靠索引与约束；Redis 故障不影响正确性；测试可用 testcontainers 复现真实环境；搜索实现可在 `SearchService` 边界替换。
- 负面：三容器比单容器更重；长任务（导入/导出）与 API 争用同一进程；模块隔离靠代码评审与架构测试。
- 约束：禁止跨模块直接 SQL；跨模块写操作必须经 service；Redis 不得成为任何写路径的必要条件。

## Risks and Mitigations

| 风险 | 缓解 |
|---|---|
| 模块边界被侵蚀 | 架构测试禁止 `httpapi` import repository/gorm；评审清单 |
| 单进程后台任务重启丢失 | `import_runs`/`export_runs` 状态表；启动时 `running → failed` |
| Redis 故障影响可用性 | cache-aside + TTL + epoch；失败直接旁路（QA-07/V-07） |
| PG 运维复杂度高于 SQLite | 固定版本、迁移嵌入二进制、pg_dump/pg_restore 封装为 CLI |
| 长任务阻塞请求 | 单 worker 限流；导入/导出异步轮询；V-02 基准 |

## Follow-up Actions

- [x] 项目发起人确认本 ADR（2026-11-07）。
- [ ] 编码阶段建立模块依赖检查测试。
- [ ] V-02/V-07 验证导入性能与 Redis 降级。
