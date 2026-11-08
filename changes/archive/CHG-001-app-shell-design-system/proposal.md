# Change Proposal

- ID: CHG-001
- Title: 工程基线、应用壳层与设计系统基座
- Status: Approved（F1，2026-11-07）
- Owner: 项目发起人
- Related Spec: SPEC-PLAT-001、SPEC-PLAT-002、SPEC-PLAT-004、SPEC-PLAT-005（部署/安全基座部分）
- Related Issue: 无（本地单用户项目，无外部 issue 系统）

## Problem / Opportunity

P7 已交付可运行的 Go + Next.js 空壳与三容器部署，但当前前端只有一个健康检查页：没有统一顶栏/导航、没有设计令牌落地、没有全局 loading/empty/error/404 语义，也没有可复用的 UI 组件库。若直接进入业务切片（作品、体系、人物），每个页面都会各自实现布局与状态，导致视觉不一致、状态语义分裂、后续返工。

本变更建立后续 11 个变更共同承载的「壳层 + 设计系统 + 全局状态」基座，使每个业务切片只需填充内容区。

## User Value

- 用户打开系统即看到一致的浅绿色渐变壳层、清晰的导航与当前作品位置，不会遇到白屏、裸报错或死链。
- 无作品时明确告知下一步（创建作品在 CHG-002 提供），而不是空白页。
- 后续所有页面共享同一套按钮/表单/对话框/提示语义，操作预期稳定。

## Goal

`docker compose up -d` 后浏览器访问 `http://127.0.0.1:17860` 可见完整应用壳层；设计令牌与基线组件可被后续变更直接复用；全局加载/空/错误/404 状态与统一错误契约生效；壳层通过 axe 扫描与键盘可达检查。

## In Scope

- 前端壳层：顶栏（产品名、版本、当前作品位、设置入口）、全局导航、内容区、侧栏容器、详情页返回位；`/` 落地页与 404 页。
- 设计令牌落地：沿用 `frontend/styles/tokens.css` 并补齐组件所需令牌；页面禁止硬编码品牌色。
- 基线组件（按 F1 裁决的数量与清单）：具备 default/hover/focus/disabled/loading/error 状态与键盘可达。
- 全局状态基座：Skeleton 加载、EmptyState（区分无数据与筛选无结果，本变更先提供组件与壳层用例）、Alert 错误提示、Toast 非阻塞提示、统一文案。
- 后端：统一错误 DTO 补齐 `details` 与 `request_id`；`/meta`、`/settings` 只读字段按契约收敛；CSP/安全头回归测试。
- 测试：后端错误映射与常量单测；前端组件/状态单测；Playwright 壳层 E2E（E2E-12 壳层部分）、axe 壳层扫描、1280/1024 视觉基线截图。
- 文档：同步 `PROJECT-SETUP.md` §11 的 E2E 命令、`bootstrap` 相关记录不适用。

## Out of Scope

- 任何业务实体 CRUD：作品、人物、势力、地点、榜单、体系、导入、回收站。
- 检索功能本体（CHG-009）：本变更不实现检索输入与结果。
- 导入/导出/快照界面能力（CHG-010）。
- 登录、账号、权限、暗色主题、多语言、移动端专门适配。
- Redis 缓存、性能压测（CHG-009/CHG-012）。

## Acceptance Criteria

1. `docker compose up -d` 后 `http://127.0.0.1:17860/` 返回壳层页面，含顶栏、导航与内容区；刷新无 404，无重定向循环。
2. 无作品时壳层显示明确空状态文案（不出现空白页），且不提供不可用按钮。
3. 未知路由返回 404 页面（非裸 JSON），页面含返回首页入口；未知 `/api/*` 仍返回 JSON 404 且含 `request_id`。
4. 基线组件全部具备规定状态；Tab 键可遍历、焦点可见、Esc 可关闭 Dialog、焦点返回触发元素。
5. axe 扫描 `/`、`/settings`、404 页无 serious/critical 问题。
6. 1280 宽布局完整、1024 宽不横向溢出；两种宽度具视觉基线截图。
7. `/api/v1/meta` 返回 `limits`（name=200、description=20000、relation_note=500）与版本；`/api/v1/settings` 返回只读版本/数据目录，未知字段为 `null`。
8. `scripts/ci.ps1` 全绿（新增 E2E 门禁），后端 `golangci-lint` 0 issues、前端 lint/typecheck/单测零错误。

## Constraints and Assumptions

- Next.js 静态导出，无 SSR/Server Actions；所有数据经 `/api/v1`；详情页使用查询参数（本变更无详情页）。
- 单用户、本地回环；CSP 使用 `script-src 'self' 'unsafe-inline'`（Next hydration 需要）。
- Go 1.26 基线（P7 已调整，待发起人追认）。
- 导航与路由是否一次性铺满、Tabs 是否纳入组件基线存在上游文档冲突，见 design.md §12 与 proposal 的 Human Approval。

## Dependencies

- P7 工程初始化与 P8 验收通过（已完成）。
- 新增开发依赖：`@playwright/test`、`@axe-core/playwright`（E2E-12 与 axe 基线所需，roadmap CHG-001 测试项明确要求）。

## Risks

| 风险 | 影响 | 缓解 |
|---|---|---|
| Next 静态导出与 Go 托管路径不一致 | 页面 404 或白屏 | 复用 P7 的 `.html`/`index.html` 回退逻辑并补 `/settings` 用例 |
| CSP 阻断 Next 资源 | 壳层不可交互 | E2E 在真实容器内运行，断言无 console error |
| axe 基线与组件实现反复 | 组件返工 | 先实现语义结构（label/role/focus），再补视觉 |
| 视觉基线跨机器差异 | CI 截图抖动 | 基线限定本机 Windows 环境生成，差异阈值 2%，并在文档标注 |
| 导航提前铺满导致死链 | 违反「无死链」 | 按裁决只暴露已实现路由（推荐），未实现模块随对应变更加入 |

## Human Approval

F1 裁决（2026-11-07，项目发起人确认）：

1. **Tabs 冲突**：以 `design-system.md` §5.1 为准，组件基线用 IconButton 替换 Tabs；同步修订 `roadmap.md` CHG-001 与 SPEC-PLAT-004 清单。
2. **导航阶段化**：本变更只展示已实现路由（`/`、`/settings`）；其余导航随 CHG-002~CHG-010 增量加入，并在 `PG-00-global-shell.md` 记录阶段豁免。
3. **E2E 依赖**：批准新增 `@playwright/test`、`@axe-core/playwright` 两个 devDependency。
4. **默认入口**：本变更 `/` 作为壳层落地页（无作品空状态）；CHG-009 完成后改为重定向 `/search`，并在 SPEC-PLAT-001 记录时序。
