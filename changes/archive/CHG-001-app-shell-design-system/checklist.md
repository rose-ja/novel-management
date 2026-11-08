# Acceptance Checklist

格式：`- [ ] AC-编号 | 关联 Requirement/Spec/Page ID | 验证方式 | 预期结果 | 证据`

## Functional

- [x] AC-001 | SPEC-PLAT-001 / PG-00 | `curl -s http://127.0.0.1:17860/` | 200，HTML 含产品名与壳层结构 | `status=200`；E2E `shell.spec.ts`「渲染顶栏、导航与内容区」
- [x] AC-002 | SPEC-PLAT-001 / PG-00 | 浏览器打开 `/`（无作品） | 显示「还没有作品」说明，无空白页、无不可用按钮 | E2E「无作品时展示空状态说明」（断言主区无「新建作品」按钮）
- [x] AC-003 | SPEC-PLAT-001 / PG-00 | 访问不存在页面 | 显示 404 页与返回首页入口 | E2E「未知路由返回可返回的 404 页」（状态码 404 + 「页面不存在」+「返回首页」）
- [x] AC-004 | SPEC-PLAT-001 / PG-00 | `curl /api/v1/unknown` | JSON 404，含 `code/message/request_id` | `{"error":{"code":"NOT_FOUND","message":"接口不存在","request_id":"chg001-probe"}}`
- [x] AC-005 | SPEC-PLAT-001 / PG-13 | 打开 `/settings` | 只读展示版本、数据目录、快照保留；无编辑入口 | E2E「设置页只读展示系统信息」（断言无「保存」按钮）；`/api/v1/settings` 实测
- [x] AC-006 | SPEC-PLAT-005 | `GET /api/v1/meta` | `limits` 含 name=200、description=20000、relation_note=500 | 实测 8 项 limits 全量返回；`health_test.go` 断言

## UI / UX（适用时）

- [x] AC-101 | Page Spec PG-00 | 对照 Design System 令牌与布局 | 顶栏/导航/内容区层级正确，无硬编码品牌色 | 视觉基线 `overview-1280`/`overview-1024`；`components.css` 只引用 `tokens.css` 变量
- [x] AC-102 | Page Spec PG-00 | Loading/Empty/Error/404 状态 | 状态完整、文案统一、可重试 | E2E `shell.spec.ts` + `error-states.spec.ts`（网络异常、500、重试）
- [x] AC-103 | Page Spec PG-00 §7 | 1280 与 1024 宽度；900 宽度 | 1280/1024 无横向溢出；<1024 显示桌面端提示、侧栏保留、无移动导航 | E2E「1280/1024 宽度无横向溢出」+「<1024 显示桌面端提示且不提供移动导航」；两档视觉基线
- [x] AC-104 | SPEC-PLAT-004 | 键盘 Tab/Shift+Tab/Esc | 跳过链接首个可聚焦；侧栏导航可键盘遍历且焦点可见 | E2E `keyboard.spec.ts`（2 条）。Esc 关闭 Dialog 无本变更使用点，转 CHG-002（已记 verification §5）
- [x] AC-105 | SPEC-PLAT-004 | axe 扫描 `/`、`/settings`、404 | 无 serious/critical | E2E `a11y.spec.ts` 3 条通过

## Error / Boundary

- [x] AC-201 | SPEC-PLAT-002 | 接口不可用时的页面表现 | 显示「网络异常，请检查连接」，不显示堆栈 | E2E `error-states.spec.ts`「接口网络异常时…」（路由 abort 模拟；静态页与 API 同为 Go 托管，整机不可用时页面无法加载，故用接口级模拟）
- [x] AC-202 | SPEC-PLAT-002 | 后端返回 500 | 显示「加载失败，请重试」+ 手动重试 | E2E `error-states.spec.ts`「接口 500 时…」，点击重试后恢复正常
- [x] AC-203 | SPEC-PLAT-005 | 请求 `/../../etc/passwd` | 404，不读取外部文件 | `curl --path-as-is` → `status=404`；`static_test.go` 含穿越用例

## Permission / Security

- [x] AC-301 | SPEC-PLAT-005 | 非白名单 Host 请求 | 403 `HOST_NOT_ALLOWED` | 实测 `403` + `{"code":"HOST_NOT_ALLOWED","request_id":"750bb36890398017"}`
- [x] AC-302 | SPEC-PLAT-005 | 响应头检查 | 含 CSP、X-Content-Type-Options、X-Frame-Options、Referrer-Policy | 实测四项均存在（CSP 含 `frame-ancestors 'none'`）
- [x] AC-303 | SPEC-PLAT-005 | 错误响应与日志 | 不含堆栈、SQL、文件路径、DSN 口令 | 错误体仅 `code/message/request_id`；日志为 JSON（`ts/level/msg`），启动日志 DSN 已脱敏（P8 验收项）

## Regression

- [x] AC-401 | P7 基线 | `/api/v1/health` | 200，db/cache/data_dir_writable 正常 | `{"cache":"ok","data_dir_writable":true,"db":"ok","status":"ok","version":"0.1.0"}`
- [x] AC-402 | P7 基线 | `scripts/smoke.ps1` | `SMOKE OK` | `SMOKE OK: version=0.1.0 db=ok cache=ok`
- [x] AC-403 | 工程门禁 | `scripts/ci.ps1` | 退出码 0，含 E2E | 实测 `CI PASSED`，E2E 15 passed
- [x] AC-404 | 视觉基线 | 1280/1024 截图对比 | 差异 ≤2% | `playwright test` 视觉基线两条通过（`maxDiffPixelRatio: 0.02`）
