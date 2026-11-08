# Change Design

## 1. Summary

在 P7 空壳上增加三件事：(1) Go 侧统一错误契约补全 `details`/`request_id`，静态托管支持 Next 导出的多级页面；(2) 前端建立 `AppShell`（顶栏 + 导航 + 内容区）与全局状态组件，落地设计令牌；(3) 建立 Playwright + axe 测试基线。业务数据不进入本变更。

## 2. Current State

- 后端：`/api/v1/health`、`/meta`、`/settings` 已存在；错误体为 `{error:{code,message}}`，缺 `details`/`request_id`；静态托管支持 `x.html` 与 `x/index.html` 回退。
- 前端：仅 `app/page.tsx` 健康检查卡片；`styles/tokens.css` 已定义颜色/字体/间距令牌；无组件库、无路由页、无测试框架（除 vitest 单测）。
- 测试：`go test ./...`、`npm test`（vitest）可用；无 Playwright、无 axe。

## 3. Proposed Design

### 3.1 后端

- `internal/httpapi/errors.go`：`errorItem` 增加 `details,omitempty` 与 `request_id`；新增 `abortErrorWithDetails`；`request_id` 从 gin context 读取（无则空串）。
- `internal/httpapi/router.go`：`/meta`、`/settings` 保持；NoRoute 中 `/api/` 返回 JSON 404（带 request_id），其余走静态。
- `internal/httpapi/meta.go`：`limits` 补全 `name/description/relation_note/era_text/chronicle/seat_max/category_max`（与 `/meta` 契约常量一致）。
- 新增 `internal/httpapi/errors_test.go`：404 体含 `code/message/request_id`；`details` 仅在提供时出现。

### 3.2 前端结构

```text
frontend/
├─ app/
│  ├─ layout.tsx          # 引入 AppShell + ToastProvider
│  ├─ page.tsx            # 落地页：无作品空状态 / 有作品时的引导位
│  ├─ settings/page.tsx   # 只读设置（版本/数据目录/快照保留）
│  └─ not-found.tsx       # 404
├─ components/
│  ├─ shell/AppShell.tsx  # 顶栏 + 导航 + 内容区
│  ├─ shell/TopBar.tsx
│  ├─ shell/GlobalNav.tsx
│  └─ ui/…                # 基线组件（见 §3.3）
├─ lib/api/client.ts      # fetch 包装 + ApiError 映射
└─ lib/api/settings.ts
```

- 静态导出下无动态段路由；本变更路由仅 `/`、`/settings`，导航按裁决只呈现已实现项。
- 当前作品位：读取 `/api/v1/settings.current_work_id`；为 `null` 时显示「未选择作品」静态文本（CHG-002 接入切换器）。

### 3.3 基线组件

按裁决后的清单实现；每个组件包含：状态样式、`aria-*` 语义、键盘交互、loading/disabled 支持，并在 `frontend/components/ui/*.test.tsx` 覆盖关键状态。

### 3.4 全局状态

- `Skeleton`：列表/详情/文本三型。
- `EmptyState`：`page`/`section`/`inline` 三型，必须含说明文案；「下一步操作」由使用方传入（无可用操作时不渲染按钮）。
- `Alert`：`info/warning/danger/success`。
- `Toast`：`success/error/info/undo`，非阻塞，4 秒自动消失，`role="status"`。
- 统一文案常量：`加载失败，请重试`、`保存失败，请稍后重试`、`网络异常，请检查连接`。

### 3.5 测试

- 后端：`go test ./internal/httpapi/...` 覆盖错误体、meta 常量、静态页 `/settings`。
- 前端：vitest 组件测试（状态渲染、键盘、空状态无按钮）。
- E2E：Playwright `e2e/shell.spec.ts`（壳层渲染、导航、404、1024 无横向溢出、无 console error）、`e2e/a11y.spec.ts`（axe serious/critical 为 0）、`e2e/visual.spec.ts`（1280/1024 截图基线）。
- CI：`scripts/ci.ps1` 增加 `npx playwright test`；运行前确保 `docker compose up -d` 且前端已构建。

## 4. Alternatives Considered

| Option | Advantages | Disadvantages | Decision |
|---|---|---|---|
| 一次性铺满 9 条导航与页面骨架 | 完全贴合 PG-00 导航基线 | 未实现页面只能放占位内容或死链，违反「无占位/无死链」 | 待裁决（建议阶段化） |
| 只做壳层、不做 E2E | 交付快 | 违反 roadmap 测试项与 §9 门禁；axe/键盘回归无保障 | 否 |
| 引入 shadcn/ui 或组件库 | 组件更快 | 新增第三方依赖与样式体系，违背「不加未批准依赖」与设计令牌统一 | 否 |
| 手写轻量组件 + 设计令牌 | 完全可控、无额外运行时依赖 | 开发量较大 | 是 |

## 5. File / Module Change List

| Path | Change | Reason | New/Modify |
|---|---|---|---|
| `backend/internal/httpapi/errors.go` | 补 `details`/`request_id` | 统一错误契约 | Modify |
| `backend/internal/httpapi/router.go` | API 404 带 request_id | 契约一致 | Modify |
| `backend/internal/httpapi/meta.go` | 补全 limits 常量 | CHG-001 DoD | Modify |
| `backend/internal/httpapi/errors_test.go` | 新增错误体测试 | 覆盖契约 | New |
| `backend/internal/httpapi/static_test.go` | 补 `/settings` 静态页用例 | 防止导出路径回退失效 | Modify |
| `frontend/app/layout.tsx` | 挂载 AppShell 与 ToastProvider | 壳层 | Modify |
| `frontend/app/page.tsx` | 改为落地页（无作品空状态） | 替换健康检查页 | Modify |
| `frontend/app/settings/page.tsx` | 只读设置页 | 导航落地 | New |
| `frontend/app/not-found.tsx` | 404 页 | 路由契约 | New |
| `frontend/components/shell/*` | 顶栏、导航、壳层 | PG-00 | New |
| `frontend/components/ui/*` | 基线组件 | SPEC-PLAT-004 | New |
| `frontend/lib/api/client.ts`、`settings.ts` | 请求包装、错误映射 | PLAT-002 | New |
| `frontend/e2e/*`、`playwright.config.ts` | E2E/axe/视觉基线 | roadmap 测试项 | New |
| `frontend/package.json` / `package-lock.json` | 新增 Playwright、axe devDependencies 与脚本 | 测试基线 | Modify |
| `scripts/ci.ps1` | 增加 E2E 门禁 | §9 | Modify |
| `project/page-specs/PG-00-global-shell.md` | 记录阶段豁免（若批准） | 文档一致 | Modify |

## 6. API / Data Contract

本变更不改既有成功响应；仅调整错误体：

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "接口不存在",
    "request_id": "0f1c..."
  }
}
```

- `details` 为可选数组，元素含 `field`/`rule`/`message` 或引用定位信息。
- `/meta` 的 `limits` 键与 `project/api-contract.md` 一致；`/settings` 只读字段保持 `null` 语义。

## 7. Data Flow / Call Sequence

1. 浏览器请求 `/` → Go 静态托管返回 `index.html` → React 水合 → 壳层渲染 → 请求 `/api/v1/settings`。
2. `settings.current_work_id === null` → 壳层显示「未选择作品」+ 落地页空状态文案。
3. 请求失败 → `client.ts` 映射为 `ApiError` → 页面显示 Alert（统一文案）+ 重试。
4. 未知路径 → Go 返回 404 JSON（`/api/*`）或静态 404 页（页面路由）。

## 8. Error / Boundary / Permission Handling

- 前端不展示堆栈、SQL、文件路径；只展示 `code` 对应的中文文案。
- 网络错误与 5xx 均可重试一次（手动点击），不自动循环。
- 权限错误本版 N/A（单用户），不实现提示。
- 静态路径穿越继续由 `path.Clean` + `..` 检查拦截。

## 9. Risks and Mitigations

- 视觉基线跨机器抖动：阈值 2%，仅在 Windows 本机生成基线并在文档标注。
- Playwright 浏览器下载失败：本机无法从 Playwright CDN / 镜像完成下载，浏览器改为复用本机已安装 Chrome（`channel: "chrome"`），`E2E_CHANNEL=chromium` 可切回自带 Chromium；已在 PROJECT-SETUP §11 记录。
- 导航阶段化可能与 PG-00 视觉基线不一致：在 PG-00 记录豁免，CHG-012 收口时复核。

## 10. Test Strategy

| 层 | 覆盖 |
|---|---|
| 单元 | 错误体结构、limits 常量、API 客户端错误映射与文案常量、空状态无按钮 |
| 集成 | `/health`、`/meta`、`/settings`、CSP、静态页 `/settings`、API 404 |
| E2E | E2E-12 壳层：加载壳层、404、1024 无溢出、`<1024` 提示、无 console error；组件状态通过页面级 E2E 覆盖 |
| A11y | axe `/`、`/settings`、404，无 serious/critical |
| 视觉 | 1280/1024 截图基线 |

## 11. Rollback

- 无数据库变更；仅前端与后端错误体字段新增（向后兼容）。
- 回退方式：`git revert` 本变更提交，重建镜像；静态导出与容器卷无影响。

## 12. Open Questions

| ID | 问题 | 建议 | 需批准 |
|---|---|---|---|
| Q-01 | Tabs 是否纳入基线（design-system 明确不用，roadmap/SPEC 列出） | 以 design-system 为准，用 IconButton 替换并同步上游文档 | 是 |
| Q-02 | 导航是否一次铺满 9 项 | 阶段化，只呈现已实现路由，PG-00 记录豁免 | 是 |
| Q-03 | 是否新增 Playwright/axe devDependencies | 批准，roadmap CHG-001 测试项要求 | 是 |
| Q-04 | `/search` 何时成为默认入口 | CHG-009 完成后再重定向；本变更 `/` 为落地页 | 是 |
