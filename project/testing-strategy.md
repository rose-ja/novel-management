# 测试策略（Testing Strategy）

| 项目 | 内容 |
|---|---|
| 文档版本 | v1.0 |
| 状态 | Approved（2026-11-07，项目发起人） |
| 日期 | 2026-11-07 |
| 上游 | [constitution.md](file:///d:/Project/novel-management/project/constitution.md) §5、[requirements.md](file:///d:/Project/novel-management/project/requirements.md) NFR/AC、[architecture.md](file:///d:/Project/novel-management/project/architecture.md)、[tech-stack.md](file:///d:/Project/novel-management/project/tech-stack.md) |
| 后端工具 | Go `testing`、testify、testcontainers-go（PostgreSQL 16、Redis 7）、miniredis、`net/http/httptest` |
| 前端工具 | Vitest、Testing Library、MSW（仅组件）、Playwright |
| 质量工具 | golangci-lint、go vet、govulncheck、ESLint、tsc、k6 |

> 未实际执行的测试不得标记 PASS；未运行标记 `NOT RUN`；环境阻塞标记 `BLOCKED`。

---

## 1. Test Pyramid

| 层级 | 范围 | 工具 | 目标数量 | 运行时机 |
|---|---|---|---|---|
| 单元 | 纯规则：名称规范化、树环、席位、引用、解析、长度、缓存键 | `go test` + testify | ≥120 用例 | 每次提交 |
| 组件 | React 组件状态、表单校验、空/错/加载 | Vitest + Testing Library | ≥40 用例 | 每次提交 |
| 集成 | service + 真实 PostgreSQL 16 + Redis（miniredis）+ HTTP 中间件 | `go test -tags=integration` + testcontainers-go + httptest | ≥60 用例 | 每次提交 |
| E2E | 12 条用户旅程 + 视觉基线 + 安全冒烟 | Playwright | 12–15 条 | PR 影响 UI/API；发布前全量 |
| 性能基准 | V-01~V-07、P-01~P-03 | k6 + Go benchmark | 10 组 | 里程碑与发布前 |

- 覆盖率目标：`internal/domain` 与 `internal/service` ≥80%；全局不设硬门槛（避免为覆盖率写无效测试）。
- 风险优先：搜索、删除/恢复、导入原子性、快照/恢复、引用完整性必须有测试（constitution §5）。
- 竞态检测：单元与集成测试在 CI 使用 `-race`；集成测试串行执行，避免共享数据库状态。

---

## 2. Unit Test Scope

| 包 / 文件 | 必测规则 | 关联 |
|---|---|---|
| `internal/domain/names.go` | NFKC、casefold、trim、连续空白折叠；中英文混排；全角/半角；空字符串 | FR-48 |
| `internal/domain/tree.go` | 自引用、后代引用、深度遍历；势力/地点无深度限制；体系节点最多两级 | FR-09/13/22 |
| `internal/domain/seats.go` | 1–200；只扩容；缩容尾部全空；同榜一人一席；move empty/swap | FR-17/18 |
| `internal/domain/references.go` | 人物删除引用矩阵（首领/席位/地点关系）；势力/地点下级与引用；类别/节点停用与引用；恢复校验 | FR-37/38/49/50 |
| `internal/domain/categories.go` | 最多 10 类；内置境界不可改/停/删；类型不可变；停用前无引用 | FR-21/25 |
| `internal/domain/chronicles.go` | 文本非空；排序号；随实体删除 | FR-04/11 |
| `internal/imports/parser.go` | 模板版本、区块顺序、未知区块、必填、`|` 分隔、引用解析、续行、错误行号 | FR-39~41、FR-51 |
| `internal/imports/zip.go` | `../`、绝对路径、符号链接、超数量、超大小、压缩炸弹上限 | NFR §8.7、R-05 |
| `internal/media/validate.go` | 扩展名/MIME/魔数一致；jpg/png/webp；≤10MB | FR-04/43 |
| `internal/search/query.go` | 分词、AND 组合、字段权重、1–2 字回退路径、分页上限 | FR-27~33 |
| `internal/cache/key.go` | 键格式、参数哈希稳定、epoch 递增、TTL 上限、Noop 降级 | QA-07 |
| `internal/config/config.go` | 环境变量默认值、非法端口/URL、`APP_HOST` 非回环警告 | deployment §2 |

- 单元测试不启动 gin、不连接 PostgreSQL/Redis；纯函数输入/输出。
- 每个规则至少：正常、边界、异常三例。
- 时间与 ULID：注入 `Clock` 与 `IDGenerator` 接口；单元测试固定时间与序列。

---

## 3. Integration Test Scope

环境：testcontainers-go 启动 `postgres:16-alpine` 与 `redis:7-alpine`；每个测试包共享容器、每个用例使用独立 schema 或 `TRUNCATE` 重置；执行 goose 迁移；`httptest.NewServer(router)` 走真实路由、中间件与错误映射。

| 场景 | 断言 | 关联 |
|---|---|---|
| 作品创建 | 自动创建内置「境界」；名称唯一；跨作品同名人物允许 | FR-01/03、AC-01 |
| 人物 CRUD | 名称唯一、别名去重、体系引用每类 0..1、跨作品引用拒绝 | FR-04/05/08/23 |
| 势力树 | 父子、环拒绝、删除有下级阻止、首领引用阻止 | FR-09~12 |
| 地点树 | 类型自由文本、人物/势力双向关系、删除引用阻止 | FR-13~16 |
| 榜单 | 创建 100 席、扩容、缩容占用阻止、换位、一人同榜一席 | FR-17~20 |
| 体系 | 类别上限、内置不可变、停用引用阻止、节点两级、排序、改名同步 | FR-21~26 |
| 检索 | 空格 AND、名称/别名/称号/简介/关联名命中、分组、排序、空白提示、作品隔离 | FR-27~33 |
| 缓存 | HIT/MISS 行为；写操作 epoch 失效；Redis 停止后 `BYPASS` 且结果正确 | QA-07、D-08 |
| 软删除 | 作品整本/类别整组/人物大事记同根；回收站只列根；撤销 8 秒；恢复冲突 | FR-57~63 |
| 导入 | 成功原子写入；失败 0 残留；图片匹配；错误行号；重复 commit 幂等；重启 running→failed | FR-39~46 |
| 导出/快照 | 任务状态、zip 结构、manifest、checksums、保留 3 份；含/不含回收站两种包 | FR-52~56、AC-12 |
| 迁移 | 空库 `goose up`；表/索引/`pg_trgm` 存在；`goose down-to 0`（可逆时） | data-model §7 |
| 并发 | 两个 `If-Match` 相同版本，后写返回 `VERSION_CONFLICT`；两个导入/导出并发限制 | EX-07 |
| 契约 | 每个端点响应通过 OpenAPI schema 校验；错误码与 §1.4 一致 | api-contract |

- 数据库测试不 mock gorm/PostgreSQL（Mocking Policy §6）。
- Redis：集成测试优先使用 miniredis（快、无容器）；缓存降级与真实 Redis 行为另用 testcontainers 覆盖。
- 每个写操作必须验证：事务回滚、软删除过滤、`search_documents` 同步、缓存 epoch 失效。

---

## 4. E2E Critical Journeys

Playwright 对 `docker compose up` 或本地 `go run ./cmd/server + npm run dev`；Chromium（Chrome/Edge 等价）。

| 用例 | 旅程 | 验收点 | 关联 |
|---|---|---|---|
| E2E-01 | 创建作品 → 切换 → 删除作品（输入名称确认）→ 回收站恢复 | 数据隔离、整本恢复、快照提示 | US-001、FR-01~03 |
| E2E-02 | 新建人物（Drawer）→ 别名/称号 → 图片上传 → 内联编辑 → 删除被阻止 | 表单校验、图片预览、引用提示 | US-002、FR-04~08 |
| E2E-03 | 建立势力树 → 设首领 → 成员范围切换 → 删除有下级阻止 | 树展开、成员推导、边界错误 | US-003、FR-09~12 |
| E2E-04 | 建立地点树 → 关联人物与势力 → 双向跳转 | 关系可达、无死链 | US-004、FR-13~16 |
| E2E-05 | 创建 50 席榜单 → 指派/换位 → 扩容 → 缩容占用阻止 | 席位交互、空席、错误清单 | US-005、FR-17~20 |
| E2E-06 | 新增「圣体」类别 → 添加两级节点 → 人物引用 → 改名同步 → 停用被阻止 | 动态字段、同步、引用提示 | US-006、FR-21~26 |
| E2E-07 | 快速检索 2 字/空格 AND → 条件检索 → 无结果提示 | 分组、摘要、可跳转、不编造 | US-007、FR-27~33 |
| E2E-08 | 改名/换势力/改境界 → 人物页/势力页/榜单页同步 | 三类一致性 | US-008、FR-34~36、AC-09 |
| E2E-09 | 导入向导：上传 → 校验错误 → 修正 → 预览 → 确认 → 报告 → 待校对 | 原子导入、错误定位 | US-009、FR-39~46 |
| E2E-10 | 导出 zip → 检查文件结构 → CLI 空环境恢复 → 计数一致 | 可恢复性、图片数量 | US-010、FR-52~56 |
| E2E-11 | 删除人物 → toast 撤销 → 回收站恢复 → 永久删除确认 | 8 秒撤销、二次确认、快照 | US-011、FR-57~63 |
| E2E-12 | 空数据/加载/断网/未保存离开/键盘焦点 | 状态与可访问性 | US-012、EX-01~08 |

视觉基线：7 个原型页（检索、人物详情、势力详情、榜单详情、体系、导入、回收站）截图对比 `design-system.md` §10 清单；允许字体渲染差异，不允许布局/间距/颜色偏差。

- E2E 不使用真实私密设定；使用固定合成数据。
- 失败用例保留 trace、截图与请求日志（不含正文），便于定位。
- E2E-10 的 CLI 恢复在 compose 的一次性 `cli` 容器执行，验证两种导出包。

---

## 5. Test Data and Fixtures

| Fixture | 内容 | 用途 |
|---|---|---|
| `works/basic` | 2 本作品，每本 20–30 人物、10 势力、10 地点、2 榜单、3 类别 | 功能与 E2E（A-02 真实样例的脱敏替代） |
| `perf/large` | 1 作品 1000 人物、200 势力、200 地点、5×100 席、10 类别、150 节点、500 图片（1KB 占位） | V-01/V-02/V-03 |
| `import/good` | 模板 v1.0 Markdown + 20 图片 zip | 导入成功 |
| `import/bad-version` | 缺版本/错版本 | 预览错误 |
| `import/bad-ref` | 引用不存在、重复名、缺图 | 错误清单 |
| `import/malicious` | `../`、绝对路径、符号链接、超大条目 | 安全测试 |
| `images/*` | jpg/png/webp、MIME 伪装、>10MB、0 字节 | 上传校验 |
| `text/xss` | `<script>`、`<img onerror>`、Markdown 链接 | XSS 转义 |

- 生成器 `scripts/fixtures/perf_seed.go`（`go run`）可复现；固定随机种子。
- 集成测试通过 SQL 或 service 创建数据；性能数据由 seeder 直接批量 `COPY` 写入，避免逐条 API 调用。
- 真实设定只在本机手工验收使用，不进入仓库、不写入日志、不上传。

---

## 6. Mocking Policy

- **不 mock 数据库**：集成测试使用真实 PostgreSQL 16（testcontainers-go）；不 mock gorm。
- **不 mock 文件系统**：使用 `t.TempDir()`；快照/导出/恢复测试真实读写。
- **Redis**：单元与集成默认 miniredis；缓存故障通过关闭 miniredis 或指向不可用端口模拟。
- **仅 mock 时间与 ID**：`Clock`、`IDGenerator` 接口；不 mock 内部 service。
- **不 mock 外部服务**：本版不调用云、大模型、飞书，不存在网络 mock。
- 前端：MSW 仅用于组件测试；E2E 使用真实后端，不拦截 API。
- `pg_dump`/`pg_restore`：集成测试使用真实二进制（CI 镜像与 app 镜像一致）；不做命令级 mock。

---

## 7. Security / Permission Tests

| 测试 | 断言 |
|---|---|
| Host 白名单 | `Host: evil.com` → `403 HOST_NOT_ALLOWED` |
| Origin 校验 | 写请求 `Origin: https://evil.com` → `403 FORBIDDEN_ORIGIN`；同源通过；无 Origin（curl）通过 |
| 无 CORS | 响应不含 `Access-Control-Allow-Origin` |
| 安全头 | CSP、`X-Content-Type-Options: nosniff`、`Referrer-Policy` 存在 |
| XSS | 名称/简介/大事记含脚本 → 页面以文本显示，无执行；API 不返回 HTML |
| SQL 注入 | 名称含 `' OR 1=1` → 参数化查询，正常保存/检索 |
| 路径穿越 | zip 含 `../../etc/passwd` → `400 PATH_TRAVERSAL`，无文件写出 |
| 图片伪装 | `.jpg` 实为 HTML/PHP → `415 UNSUPPORTED_FILE_TYPE` |
| 上传大小 | 11MB 图片 → `413`；zip 超限 → `413` |
| Redis 暴露 | compose 中 redis 无宿主端口映射；`docker compose config` 校验 |
| 数据库暴露 | postgres 无宿主端口映射；仅内部网络可达 |
| 日志隐私 | 触发错误后检查日志：无正文、无绝对路径、无堆栈、无 Redis 值 |
| 权限 | 本版无认证；确认无 `/login`、无用户表、无 401/403 业务分支（EX-05 = N/A） |

- 安全测试纳入集成套件；恶意 fixture 独立目录，禁止在正常数据中复用。

---

## 8. Performance Tests

环境记录必须包含：主机 CPU/内存、Docker Desktop 版本、镜像 digest、PostgreSQL/Redis 版本、数据集版本、缓存开关状态。

| 编号 | 数据集 | 测量 | 通过标准 |
|---|---|---|---|
| V-01 | `perf/large` 1000 人物 | 20 次查询（2 字/3 字/空格 AND/条件）API 端到端，缓存关闭 | P95 ≤150ms；30 条预设查询集正确 |
| V-02 | 导入 1000 人物 + 500 图片 + 200 势力 + 150 节点 | 总耗时、失败残留 | ≤90s；失败 0 写入 |
| V-03 | 5GB 导出 + 快照 | 耗时、文件校验、恢复计数 | 导出成功；checksum 通过；恢复一致 |
| V-04 | Docker volume 恢复中断模拟 | 重启后状态 | 恢复前或恢复后之一，无半状态 |
| V-05 | 10MB 图片上传 | 首响应时间 | ≤1s（不含异步缩略图），缩略图最终成功 |
| V-06 | 错误模板样例 | 错误定位 | 全部进入清单，0 写入 |
| V-07 | Redis 停止/重启 | 检索与保存可用性、延迟 | 自动旁路；无 5xx；延迟不超过基线 2 倍 |
| P-01 | 人物列表 1000 条 | 列表 API | P95 ≤500ms |
| P-02 | 保存人物（含关系） | 保存 API | P95 ≤300ms |
| P-03 | 首屏可交互 | Playwright 计时 | ≤2s（本地） |

- 脚本：`tests/k6/search.js`、`tests/k6/list.js`、`tests/k6/save.js`；导入/导出为 Go 基准或 PowerShell 脚本。
- 性能脚本输出 JSON 报告，写入 `artifacts/perf/{date}/`；不将完整数据提交仓库。
- 任一 Must 指标未达标：阻止发布，登记风险与优化计划（constitution §9）。
- V-01 未通过前不得宣称搜索方案达标（ADR-003）。

---

## 9. CI Quality Gates

| 检查 | 命令 | PR 必跑 | 阻塞条件 |
|---|---|---|---|
| 后端格式 | `gofmt -l .` | 是 | 有未格式化文件 |
| 后端 Lint | `golangci-lint run` | 是 | 任何 error |
| 后端静态检查 | `go vet ./...` | 是 | 任何报告 |
| 后端漏洞 | `govulncheck ./...` | 是 | 已知可利用漏洞 |
| 后端单测 | `go test -race ./internal/... -run Unit` | 是 | 失败 |
| 后端集成 | `go test -race -tags=integration ./...` | 是 | 失败 |
| 前端 Lint | `npm run lint` | 是 | error |
| 前端类型 | `npm run typecheck` | 是 | error |
| 前端组件 | `npm run test` | 是 | 失败 |
| 构建 | `npm run build` + `go build ./...` + `docker build .` | 是 | 失败 |
| 迁移 | 空库 `goose up` + `goose down-to 0`（可逆时） | 是 | 失败 |
| OpenAPI 契约 | `oapi-codegen`/`openapi-typescript` 生成并对比快照 | 是（API 变更） | 非预期差异 |
| E2E 冒烟 | `npx playwright test --grep @smoke` | 是 | 失败 |
| E2E 全量 | `npx playwright test` | 影响 UI/API | 失败 |
| 依赖/许可证 | SBOM + 禁止 GPL/AGPL | 是 | 违规依赖 |
| 密钥扫描 | 禁止 `.env`、私钥、Token、连接串 | 是 | 命中 |

- CI 只跑在本地/自托管环境；不依赖付费 SaaS（零成本约束）。
- 发布前人工执行性能基准与恢复演练，结果附在发布记录。

---

## 10. Commands

| Check | Command | Required on PR? |
|---|---|---|
| Install (backend) | `cd backend && go mod download` | — |
| Install (frontend) | `cd frontend && npm ci` | — |
| Format (backend) | `gofmt -l .` | Yes |
| Lint (backend) | `golangci-lint run` | Yes |
| Vet (backend) | `go vet ./...` | Yes |
| Vulnerability | `govulncheck ./...` | Yes |
| Unit (backend) | `go test -race ./internal/... -run Unit` | Yes |
| Integration (backend) | `go test -race -tags=integration ./...` | Yes |
| Lint (frontend) | `npm run lint` | Yes |
| Typecheck (frontend) | `npm run typecheck` | Yes |
| Component tests | `npm run test` | Yes |
| Build | `npm run build` | Yes |
| Migration check | `goose -dir internal/migrations postgres "$TEST_DATABASE_URL" up`（空库） | Yes |
| E2E smoke | `npx playwright test --grep @smoke` | Yes |
| E2E full | `npx playwright test` | By impact |
| Performance | `k6 run tests/k6/search.js`（需 `perf/large`） | Release |
| Restore drill | `docker compose run --rm cli restore /data/exports/<zip>` | Release / 迁移前 |

> 命令在 scaffolding 后必须回写 `PROJECT-SETUP.md` §11 与 `deployment.md`；当前为计划命令，未执行。

---

## 11. 追溯

| 需求/风险 | 测试 |
|---|---|
| FR-01~26 核心 CRUD 与规则 | 单元 + 集成 + E2E-01~06 |
| FR-27~33 检索 | 集成检索矩阵 + V-01 + E2E-07 |
| FR-34~38 一致性/引用 | 单元引用矩阵 + E2E-08 |
| FR-39~46 导入 | 单元解析 + 集成原子性 + V-02/V-06 + E2E-09 |
| FR-47~51 校验 | 单元 + 集成错误码矩阵 |
| FR-52~56 导出/恢复 | 集成 + V-03/V-04 + E2E-10 |
| FR-57~63 回收站 | 集成 + E2E-11 |
| FR-64~69 视觉/交互 | Playwright 截图 + E2E-12 |
| NFR 安全 | §7 安全测试 |
| NFR 性能 | §8 性能测试 |
| QA-07 Redis 降级 | 集成缓存用例 + V-07 |
