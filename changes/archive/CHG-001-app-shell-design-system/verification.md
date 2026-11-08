# Verification

- Change ID: CHG-001
- Verified at: 2026-11-08
- Verifier: 作者自验 + 真实命令复跑（F6）
- Environment: Windows / Docker Desktop；`http://127.0.0.1:17860`；镜像 `novel-settings:dev`

## 1. 命令与结果

| 命令 | 工作目录 | 结果 |
|---|---|---|
| `gofmt -l .` | `backend` | 无输出（无未格式化文件） |
| `go vet ./...` | `backend` | 通过 |
| `go test ./...` | `backend` | 通过（`internal/config`、`internal/httpapi` 全绿） |
| `golangci-lint run ./...` | `backend` | `0 issues` |
| `govulncheck ./...` | `backend` | `Your code is affected by 0 vulnerabilities` |
| `npm ci` | `frontend` | 成功 |
| `npm run lint` | `frontend` | 通过 |
| `npm run typecheck` | `frontend` | 通过 |
| `npm test` | `frontend` | 3 个文件 / 10 条测试通过（`client.test.ts` 4、`messages.test.ts` 2、其余 4） |
| `npm run build` | `frontend` | 静态导出成功，`out/` 含 `index.html`、`settings.html`、`404.html` |
| `powershell -File scripts/ci.ps1` | 仓库根 | `CI PASSED`（含 image build 与 15 条 E2E） |
| `powershell -File scripts/smoke.ps1` | 仓库根 | `SMOKE OK: version=0.1.0 db=ok cache=ok` |

`scripts/ci.ps1` 实际输出（末段）：

```
== e2e ==
Running 15 tests using 1 worker
  ✓  1 [chromium] › e2e\a11y.spec.ts:11:3 › 概览 无 serious/critical 无障碍问题
  ✓  2 [chromium] › e2e\a11y.spec.ts:11:3 › 设置 无 serious/critical 无障碍问题
  ✓  3 [chromium] › e2e\a11y.spec.ts:11:3 › 404 无 serious/critical 无障碍问题
  ✓  4 [chromium] › e2e\keyboard.spec.ts:5:3 › 键盘可达 › 首个 Tab 聚焦跳过链接并可跳到主要内容
  ✓  5 [chromium] › e2e\keyboard.spec.ts:16:3 › 键盘可达 › 侧栏导航可用键盘遍历
  ✓  6..13 [chromium] › e2e\shell.spec.ts › 应用壳层（8 条）
  ✓ 14–15 [chromium] › e2e\visual.spec.ts › 视觉基线 概览页 1280 / 1024
  15 passed (12.4s)
CI PASSED
```

错误态另跑一次（`npm run test:e2e -- e2e/error-states.spec.ts`）：`2 passed (2.0s)`。

## 2. 接口与安全实测

| 检查 | 命令 | 实际响应 |
|---|---|---|
| 落地页 | `curl -s -o NUL -w "%{http_code}" http://127.0.0.1:17860/` | `200` |
| API 404 契约 | `curl -H "X-Request-ID: chg001-probe" /api/v1/unknown` | `{"error":{"code":"NOT_FOUND","message":"接口不存在","request_id":"chg001-probe"}}` |
| `/meta` limits | `curl /api/v1/meta` | `limits":{"category_max":10,"chronicle":2000,"description":20000,"era_text":100,"name":200,"relation_note":500,"seat_max":200,"upload_image_mb":10}` |
| `/settings` | `curl /api/v1/settings` | `{"app_version":"0.1.0","current_work_id":null,"data_dir":"/data","last_export_at":null,"last_restore_at":null,"snapshot_keep":3}` |
| `/health` | `curl /api/v1/health` | `{"cache":"ok","data_dir_writable":true,"db":"ok","status":"ok","version":"0.1.0"}` |
| Host 白名单 | `curl -H "Host: evil.example.com" /api/v1/health` | `403` `{"error":{"code":"HOST_NOT_ALLOWED",...,"request_id":"750bb36890398017"}}` |
| 路径穿越 | `curl --path-as-is /../../etc/passwd` | `404` |
| 安全响应头 | `curl -D - http://127.0.0.1:17860/` | `Content-Security-Policy`、`Referrer-Policy: no-referrer`、`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY` 均存在 |

## 3. 实现说明与偏差

1. **导航阶段化**（F1 裁决 A2）：侧栏只渲染 `/`（概览）与 `/settings`（设置），保留「资料 / 工具」分组标题；未实现项不渲染，避免死链。
2. **Tabs → IconButton**（F1 裁决 A1）：`design-system.md` 本版不含 Tabs，已同步 `roadmap.md` 与 `specs/platform/specs.md`。
3. **`<1024px`**：按 PG-00 §7 显示「本系统按桌面端设计」提示，布局 `min-width: 1024px` 允许横向滚动，不提供移动端抽屉导航。原 checklist 中「导航可折叠」表述已更正。
4. **E2E 浏览器**：本机无法从 Playwright CDN / npmmirror 下载 Chromium（下载超时、`__dirlock` EPERM），改为复用本机已安装 Chrome（`channel: "chrome"`），可用 `E2E_CHANNEL=chromium` 切回自带 Chromium。已在 tasks.md T9 与 PROJECT-SETUP §11 记录。
5. **组件单测**：不引入 React Testing Library（未经批准的依赖），组件状态由页面级 E2E 覆盖，`tasks.md` T6 已改写。
6. **`scripts/ci.ps1` 必须保持 ASCII**：Windows PowerShell 5 以 ANSI 读取无 BOM 的 UTF-8，中文注释会吞掉后续语句（本变更期间实际踩到）。已在脚本头部写明。
7. **CI 缓存目录**：`npm` 与 `golangci-lint` 缓存统一指向仓库内 `.tmp/`（已 gitignore），避免本机全局缓存目录不可写导致门禁失败。

## 4. 视觉基线

- `frontend/e2e/visual.spec.ts-snapshots/overview-1280-chromium-win32.png`
- `frontend/e2e/visual.spec.ts-snapshots/overview-1024-chromium-win32.png`
- 比对阈值 `maxDiffPixelRatio: 0.02`；基线仅在本机生成，跨机器差异见 design.md §9。

## 5. 遗留与技术债

| 项 | 说明 | 归属 |
|---|---|---|
| Dialog/Drawer 的 Esc 与焦点返回 | 组件已实现（Radix），但 CHG-001 无页面使用点，E2E 未覆盖交互 | CHG-002 首次落地时补 E2E |
| 集成测试 | 仍为 SKIP（测试策略要求先建立集成层） | 后续变更 |
| 作品切换器 | 只读占位 | CHG-002 |
| 默认入口 | `/` 为落地页 | CHG-009 改为跳转 `/search` |
| Go 1.26 基线 | P7 为修复漏洞从 1.24 上调，已在 `project/bootstrap-verification.md` 标注待追认 | 追认已有记录 |

## 6. 结论

- Required checks：通过（含 E2E 15 条）。
- Acceptance：proposal §Acceptance Criteria 1~8 全部满足，证据见 checklist。
- P0：无。P1：无遗留未处理项（上述 5 条均为已批准的分阶段实施或已记录技术债）。
- 回滚：`git revert` 对应提交即可，不涉及数据迁移与 schema 变更。
