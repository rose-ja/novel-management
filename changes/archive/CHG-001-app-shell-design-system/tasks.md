# Change Tasks

- Change ID: CHG-001
- Goal: 交付应用壳层、设计令牌与基线组件、全局状态基座、统一错误契约与 E2E/axe 基线
- Related Spec: SPEC-PLAT-001、SPEC-PLAT-002、SPEC-PLAT-004、SPEC-PLAT-005（部署/安全基座）
- Dependencies: P7/P8 已完成
- Complexity: High（L）

## Scope

### In Scope

- 后端：错误体 `details`/`request_id`、`/meta` limits 补全、静态 `/settings` 回退用例。
- 前端：AppShell（顶栏/导航/内容区）、落地页、只读设置页、404、Token 收敛、基线组件、全局状态组件。
- 测试：后端单测、前端组件单测、Playwright E2E、axe、视觉基线；CI 接入。

### Out of Scope

- 作品 CRUD、检索、导入导出、回收站、人物/势力/地点/榜单/体系页面。
- 登录/权限、暗色主题、多语言、移动端适配。

### Allowed Files

`backend/internal/httpapi/**`、`frontend/app/**`、`frontend/components/**`、`frontend/lib/api/**`、`frontend/styles/**`、`frontend/e2e/**`、`frontend/playwright.config.ts`、`frontend/package.json`、`frontend/package-lock.json`、`frontend/vitest.config.ts`、`scripts/ci.ps1`、`changes/CHG-001-app-shell-design-system/**`、`project/page-specs/PG-00-global-shell.md`、`project/roadmap.md`、`project/specs/platform/specs.md`（仅裁决后的文档同步）。

### Forbidden / Approval Required

- 新增运行时依赖、改动数据库 schema、改动 `/api/v1` 成功响应结构、修改 `project/constitution.md`。
- 一次性实现未列入本变更的 8 个业务页面。

## Execution Tasks

- [x] T1 后端错误契约补全
  - Files: `backend/internal/httpapi/errors.go`、`errors_test.go`
  - Steps: `errorItem` 增加 `Details any`（`omitempty`）与 `RequestID string`（`json:"request_id"`）；新增 `abortErrorWithDetails`；从 `c.GetString(requestIDKey)` 填充。
  - Expected result: 404/400 响应体含 `code/message/request_id`，提供 details 时出现数组。
  - Tests: `go test ./internal/httpapi/...`；新增 2 条用例。

- [x] T2 `/meta` limits 与 `/settings` 契约收敛
  - Files: `backend/internal/httpapi/meta.go`、`settings.go`（如需）、`health_test.go`
  - Steps: 确认 `limits` 含 name=200、description=20000、relation_note=500、era_text=100、chronicle=2000、seat_max=200、category_max=10、upload_image_mb=10；`/settings` 未知态字段为 `null`。
  - Expected result: 与 `project/api-contract.md` 常量一致。
  - Tests: 现有 meta 用例扩展断言。

- [x] T3 静态托管回归
  - Files: `backend/internal/httpapi/static_test.go`
  - Steps: 构造临时 web 目录含 `index.html`、`settings.html`；断言 `/` 200、`/settings` 200、未知页 404、路径穿越 404。
  - Expected result: 无重定向循环。
  - Tests: `go test ./internal/httpapi/...`。

- [x] T4 设计令牌与全局样式收敛
  - Files: `frontend/styles/tokens.css`、`frontend/app/globals.css`
  - Steps: 补齐组件所需语义令牌（focus ring、overlay、motion、z-index）；确认 `prefers-reduced-motion` 下禁用动画。
  - Expected result: 组件不硬编码品牌色；对比度达标。
  - Tests: 视觉人工检查 + axe。

- [x] T5 壳层组件
  - Files: `frontend/components/shell/app-shell.tsx`、`top-bar.tsx`、`global-nav.tsx`、`app/layout.tsx`
  - Steps: 顶栏（产品名/当前作品位/本机状态胶囊）、导航（只显示已实现路由，`aria-current="page"`）、内容区；`<1024px` 显示桌面端提示，布局最小 1024px 并横向滚动（按 PG-00 §7，不做移动导航）。
  - Expected result: 壳层在 `/`、`/settings`、404 均稳定渲染，无布局跳动。
  - Tests: E2E（壳层）。

- [x] T6 基线 UI 组件（18 个，按裁决清单）
  - Files: `frontend/components/ui/{button,icon-button,input,form-field,select,dialog,drawer,toast,badge,skeleton,empty-state,table,pagination,filter-bar,card,progress,alert,page-header}.tsx`
  - Steps: 每个组件实现变体/尺寸/状态；Dialog/Drawer 使用 Radix 并处理 Esc/焦点返回；Table 支持 loading/empty；Progress 支持 determinate/indeterminate。
  - Expected result: 全部状态可演示且键盘可达。
  - Tests: 组件状态由 Playwright E2E 覆盖；本变更不引入 React Testing Library（未经批准的新依赖）。

- [x] T7 全局状态与错误映射
  - Files: `frontend/lib/api/client.ts`、`frontend/lib/api/types.ts`、`frontend/components/ui/alert.tsx`、`frontend/lib/messages.ts`、`frontend/components/providers/settings-provider.tsx`
  - Steps: `fetchJson` 解析 `{error:{code,message,request_id}}` 抛 `ApiError`；统一文案常量；页面用 Alert 展示并可重试。
  - Expected result: 失败时不显示堆栈；网络异常与接口异常文案区分。
  - Tests: `client.test.ts` + E2E。

- [x] T8 页面：落地页、设置页、404
  - Files: `frontend/app/page.tsx`、`frontend/app/settings/page.tsx`、`frontend/app/not-found.tsx`
  - Steps: 落地页读取 `/settings`，`current_work_id` 为 `null` 时显示「还没有作品」说明（不提供不可用按钮）；设置页只读展示版本/数据目录/快照保留；404 页含返回首页链接。
  - Expected result: 与 PG-00 空状态文案一致；无占位文本。
  - Tests: E2E。

- [x] T9 测试基建
  - Files: `frontend/playwright.config.ts`、`frontend/e2e/{shell,a11y,visual,keyboard}.spec.ts`、`frontend/package.json`
  - Steps: 新增 `@playwright/test`、`@axe-core/playwright`；`baseURL` 默认 `http://127.0.0.1:17860`；脚本 `test:e2e`、`test:e2e:update`；浏览器默认用本机已安装 Chrome（`channel: "chrome"`），本机无法下载 Playwright Chromium，可用 `E2E_CHANNEL=chromium` 切换（需先 `npx playwright install chromium`）。
  - Expected result: 4 个 spec 可运行（15 条用例）；视觉基线保存 1280/1024 截图。
  - Tests: 本任务自身即测试。

- [x] T10 CI 接入
  - Files: `scripts/ci.ps1`、`project/PROJECT-SETUP.md` §11
  - Steps: 前端 build 后启动/复用 compose 栈并等待 `/health` ok，执行 `npm run test:e2e`；缺失浏览器时明确报错（不静默跳过）。
  - Expected result: `CI PASSED` 覆盖 E2E。
  - Tests: 实际执行一次。

- [x] T11 文档同步（按裁决）
  - Files: `project/page-specs/PG-00-global-shell.md`、`project/roadmap.md`、`project/specs/platform/specs.md`
  - Steps: 记录导航阶段豁免、Tabs→IconButton 调整、默认入口时序。
  - Expected result: 文档与实现一致，无静默偏离。
  - Tests: 人工核对。

- [x] T12 验收与归档（F5/F6/F7/F8/F9/F10）
  - Files: `changes/CHG-001-app-shell-design-system/checklist.md`、`changes/CHG-001-app-shell-design-system/verification.md`、`changes/archive/**`
  - Steps: 逐条勾选 checklist、记录真实命令与证据、代码审查、合并提交、归档。
  - Expected result: 所有验收项有证据；无未执行却标记 PASS 的项。
  - Tests: 按 checklist 执行。

## Definition of Done

- [x] Required checks pass（`scripts/ci.ps1` 含 E2E）
- [x] Acceptance scenarios pass（proposal §Acceptance Criteria 1~8）
- [x] No unresolved P0
- [x] P1 handled or approved waiver
- [x] Documentation synchronized（PG-00/roadmap/SPEC/SETUP §11）
- [x] Rollback understood（revert 提交即可）

## Parallelization

- T1~T3（后端）与 T4~T8（前端）可并行。
- T9 依赖 T5~T8 完成后才能稳定断言；T10 依赖 T9。

## Risks

- Playwright 浏览器下载与静态导出路径是主要不确定性，已列入 design.md §9。
- 视觉基线跨机器差异，已限定本机生成。

## Human Approval Points

1. F1 结束：批准 proposal 的 4 项裁决（Tabs、导航阶段化、E2E 依赖、默认入口）。
2. F7 代码审阅：无需人工，按 workflow 自审 + 独立验收人。
3. F9 合并与 F10 归档：完成后向发起人汇报，不阻塞后续变更（用户已批准一次性推进）。
