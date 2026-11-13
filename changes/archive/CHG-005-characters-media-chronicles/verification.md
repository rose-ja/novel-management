# Verification Report

- Change ID：CHG-005-characters-media-chronicles
- Commit：功能分支 `feat/chg-005-characters-media-chronicles`（F2 实施 + F3 自测）
- Environment：Windows + PowerShell 5.1；Go 1.27.3（交叉编译 linux/amd64）；Node.js 22；PostgreSQL 16（容器 `novel-management-db-1`）；应用容器 `novel-management-app-1` @ `http://127.0.0.1:17860`
- Tester：Agent（F2 实施；F3 自测）
- Date：2026-11-11

## 1. Static Checks

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd backend; gofmt -l .` | 0 | 无输出（仅 LF 副本检查） | — |
| `cd backend; go vet ./...` | 0 | 无输出 | — |
| `cd backend; go build ./...` | 0 | 无输出 | — |
| `cd frontend; npm run typecheck` | 0 | 无输出 | `tsc --noEmit` |
| `cd frontend; npm run lint` | 0 | 无告警 | ESLint |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 0 | 6 个静态路由导出成功（`/characters` 17.1 kB） | 构建输出 |

## 2. Unit / Integration Tests

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd backend; go test ./...` | 0 | config/httpapi/model/service 全部 ok | 含 `service/media_test.go` |
| `cd backend; go test ./internal/service/...` | 0 | 通过 | 魔数、重编码、缩略图 320、4096 上限、路径越界 |
| `cd backend; go test -count=1 -v -tags=integration ./internal/integration/...` | 0 | **70 个顶层用例 PASS / 0 FAIL（7.223s）** | 见 §3 |
| 其中 `TestMigrationRoundTrip` | 0 | 版本 5；0005 Down 删两表并恢复 `image_path`；0004 Down 清引用；重新 Up 成功 | 覆盖 DB-04/DB-05 |
| 其中 `TestCharacterImageUploadAndServe` | 0 | 上传/缩略图/详情/列表/替换/移除/ETag/304 | 覆盖 IMG-01/05/06/07/08/09/10/11 |
| 其中 `TestCharacterImageValidation` | 0 | 文本/损坏 PNG/GIF → `UNSUPPORTED_FILE_TYPE`；11MB → `FILE_TOO_LARGE`；无残留行 | 覆盖 IMG-02/03 |
| 其中 `TestChronicleCRUDAndUndo` | 0 | 排序 0/1、版本冲突、2,001 字、撤销、owner 404、`root_type=chronicle` 422 | 覆盖 CHR-01～06 |
| 其中 `TestCharacterMediaAndChroniclesLifecycle` | 0 | 人物删除隐藏、撤销恢复、永久删除清行与文件 | 覆盖 LIFE-01/02 |
| 其中 `TestWorkLifecycleCoversMediaAndChronicles` | 0 | 作品删除隐藏、撤销恢复、永久删除清行与文件 | 覆盖 LIFE-03/04 |

## 3. E2E Scenarios

环境：`http://127.0.0.1:17860`，容器内迁移日志 `successfully migrated database to version: 5`。

| Scenario | Steps | Expected | Actual | Pass/Fail | Evidence |
|---|---|---|---|---|---|
| 主图上传 | 编辑 → 选择 PNG → 保存修改 | 详情 120px 显示、侧栏「已上传」 | 通过 | PASS | `characters-media.spec.ts`「上传、展示与移除主图」 |
| 列表缩略图 | 返回列表 | 行内 40px 图片可见 | 通过 | PASS | 同上 |
| 移除主图 | 编辑 → 移除主图 → 确认 → 保存 | 占位「主图：姓名」、未上传 | 通过 | PASS | 同上 |
| 非法图片 | 选择文本伪装 PNG → 保存 | 字段级「仅支持 JPG、PNG 或 WebP 图片」；停留编辑态 | 通过 | PASS | 「非法图片在字段级被拒绝」 |
| 大事记 | 添加 2 条 → 编辑 → 删除 → 撤销 | 01/02 顺序、修订可见、删除后 Toast 撤销恢复 | 通过 | PASS | 「大事记新增、编辑、删除与撤销」 |
| 全量回归 | `npx playwright test` | 全部通过 | **66 passed（53.3s）** | PASS | 含 a11y/键盘/体系/回收站/作品/视觉 |
| F6 回归（F5 修复后） | 全量集成 + 全量 E2E | 全部通过 | **70 集成 PASS；66 E2E passed（53.7s）** | PASS | F5 后重跑 |
| 视觉基线 · 人物详情 | 更新 `character-detail-1280` | 含主图区与大事记区 | 通过 | PASS | `visual.spec.ts-snapshots/character-detail-1280-chromium-win32.png` |

## 4. Review Findings

独立审查（`review.md`）结论 **PASS**：无 S1，8 项 S2。F5 处理如下。

| ID | Severity | Finding | Fix / Waiver | Retest |
|---|---|---|---|---|
| R-01 | S2 | 上传主图与人物/作品删除并发可能产生已删人物下的活跃主图 | 已修复：上传事务内 `GetForUpdate` 锁定人物并复核 `work_id` | 4 个图片/大事记/生命周期集成用例 PASS |
| R-08 | S2 | 304 响应缺少缓存与安全头 | 已修复：304 前设置 `Cache-Control`/`nosniff`/`ETag` | 集成 `If-None-Match` 用例 PASS |
| R-02 | S2 | 列表图片查询失败被静默忽略 | 延后：可选附件降级为无图，登记可观测性改进 | — |
| R-03 | S2 | 40MP 解码峰值内存较高 | 延后：保留 40MP 上限，已在 §6 记录内存预算 | — |
| R-04 | S2 | 图片失败重试会再次 PATCH，版本递增 | 延后：当前可重试；登记“仅重试图片”改进 | — |
| R-05 | S2 | 上传错误未与文件按钮 `aria-describedby` 关联 | 延后：复用 Alert 提示，登记 a11y 改进 | — |
| R-06 | S2 | 列表视觉基线未包含真实图片 | 延后：详情基线已含主图；登记列表带图快照 | — |
| R-07 | S2 | 大事记创建与人物删除缺少真实交错测试 | 延后：锁已实现，登记并发测试 | — |

### F5 附注

- 同时修正主图 `alt` 为「人物主图：{名}」，与 PG-03/清单一致；大事记撤销成功提示改为「已撤销删除」。
- F5 重跑：后端 4 个相关集成用例 PASS；前端 typecheck/lint/46 单测 PASS；新增 E2E 3 例 PASS。

## 5. Implementation Differences（对照 `design.md`）

| 项 | 设计 | 实施 |
|---|---|---|
| WebP 落库 MIME | 输入支持 WebP | 重编码为 PNG，落库 `image/png`；HTML 显示不受影响 |
| 缩略图 | 最长边 320 | 保持比例，最长边 320（800×600 → 320×240） |
| 大事记回收站 | `root_type=chronicle` 保留 | 列表返回 422；仅撤销令牌使用该类型 |
| 图片上传时机 | 保存时统一提交 | 已实现：先 PATCH，再上传/删除；失败保留编辑态 |
| 409 冲突 | 人物文字 PATCH | 图片端点无 `If-Match`，最后写入生效（契约一致） |
| 组件测试 | `image-uploader.test.tsx`、`chronicle-section.test.tsx` | 未落地：前端 vitest 仅配置 node 环境，无 jsdom/testing-library；以 E2E 覆盖交互（见 §6） |

## 6. Known Limitations

- 前端组件测试未接入（无 jsdom/testing-library 依赖），上传/大事记交互由 3 个 E2E 用例覆盖；单元层面仅覆盖 `validateImageFile`。
- 慢网（图片延迟 2s）与 40MP 像素上限未单独自动化；上限逻辑由代码与 4096 尺寸用例间接覆盖。
- 上传耗时未做基准；10MB 图片为重编码 + 双文件写入同步执行，低配机器可能超过 AC-26 的 1s 图片预算（AC-26 不含缩略图耗时）。
- 图片 EXIF 方向未处理（PG03-Q5 非 Must）。
- 人物新建 Drawer 内不提供上传，需创建后进入详情维护（F1 已确认的范围）。
- 本机 Docker BuildKit 与本轮部署方式：交叉编译 + `docker cp`，未重建镜像；Dockerfile 未改动。
- 全站 `/favicon.ico` 404 为 CHG-004 之前既有问题，不在本变更范围。

## 7. Rollback Verification

- `TestMigrationRoundTrip` 覆盖 0005 Down：删除 `media_assets`/`chronicles`，恢复 `characters.image_path`，人物与引用保留；再次 Up 成功。
- 文件回滚：永久删除事务提交后才删文件；若需恢复，使用操作前快照（`images/` 目录随快照保留）。
- 应用回滚：切回 CHG-004 提交并重新部署；0005 数据表保留但不被旧代码读取。

## 8. Conclusion

ACCEPTED（F7 人工验收 2026-11-13；F8 待授权推送/合并）

- 后端门禁全绿：gofmt / vet / build / 单测 / 70 集成（F5 后重跑）。
- 前端门禁全绿：typecheck / lint / 46 单测 / Node 22 构建 / 66 E2E（F5 后重跑）。
- F4 审查 PASS（无 S1，8 项 S2）；R-01/R-08 已修复，其余登记。
- 容器已迁移至版本 5；主图与大事记端到端可用；生命周期与文件清理有集成证据。
- 已知限制见 §6，未发现 S1 级阻塞问题。人工验收见 `acceptance.md`，PR 描述见 `pr.md`。

## 9. F9 合并与发布记录

| 项目 | 内容 |
|---|---|
| 版本 / Commit | `main` @ `2dade22`（功能分支 `feat/chg-005-characters-media-chronicles`，9 个提交，`bff1947..2dade22` 快进合并） |
| 环境 | 本机（Windows + Docker Desktop 单用户本地）；`http://127.0.0.1:17860` |
| 时间 | 2026-11-13（Asia/Shanghai） |
| 操作者 | Agent（发布协调） |
| 审批 | 用户：F7 结论 **ACCEPT** + 明确授权「本地合并，不推送」 |
| 部署步骤 | 未变更镜像：容器 `novel-management-app-1` 已运行 F5 修复后的同一功能代码；合并后新增提交仅为文档与测试基线。Docker Desktop 重新启动后 `docker compose ps` 三容器 healthy。数据库已迁移至版本 5。 |
| Smoke Test | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke.ps1` → `SMOKE OK: version=0.1.0 db=ok cache=ok`；`GET /api/v1/works` 200；`GET /characters` 200；日志 `successfully migrated database to version: 5` |
| 异常 | 首次 Smoke 失败：Docker Desktop 未运行（`npipe ... The system cannot find the file specified`）。启动 Docker Desktop 并等待引擎就绪后重试通过，无代码/数据问题。 |
| 远程状态 | **未推送** `origin/main`（本地 main 领先 64 个提交；按授权不推送、不创建远程 PR） |
| 回滚结果 | 未触发回滚。如需回退：`git reset --hard bff1947`（本地）或 revert；若已迁移，执行 0005 Down（图片文件保留但不可见）。 |

**发布结论**：本地集成通过，未发布到远程/公网；符合 `project/deployment.md`「未获批准不得发布到公网或修改部署拓扑」。
