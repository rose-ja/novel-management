# PR/MR 描述 — CHG-005 人物图片与大事记

> 状态：本地分支已就绪，**未推送、未创建远程 PR、未合并**（按 F8 规则等待用户明确授权）。
> 分支：`feat/chg-005-characters-media-chronicles` → 目标 `main`；提交范围 `b85521f..HEAD`（8 个提交）。
> 门禁：F3 自测、F4 独立审查 PASS（无 S1 / 8 项 S2）、F5 修复+复审 PASS、F6 回归/E2E PASS、F7 人工验收 **ACCEPT**（2026-11-13）。

## 1. 标题与目标

**feat: 人物图片与大事记（CHG-005）**

为人物档案补齐视觉与时间线：主图上传/替换/移除、列表缩略图、详情 120px 展示，以及按顺序维护的大事迹时间线（含 8 秒轻撤销）。补齐后人物不再是纯文本档案，且删除/恢复/永久删除全链路覆盖图片文件与大事记，避免孤儿数据与残余文件。

## 2. 关联

- Requirement：FR-04（人物档案）、FR-47（媒体）、FR-48（大事记）、FR-66/FR-67（回收站与文件清理）
- 验收：AC-01、AC-26（图片响应）、AC-33（列表性能）；EX-04、EX-07、EX-21
- Spec：SPEC-CHAR-006（主图）、SPEC-CHAR-007（大事记）、SPEC-TRASH-002/003/004（生命周期联动）、SPEC-WORK-001
- Page Spec：PG-02 人物列表、PG-03 人物详情
- 契约：`project/api-contract.md`（F10 回写图片/大事记最终行为）、`project/data-model.md`（`media_assets`、`chronicles`）
- 变更文档：`changes/CHG-005-characters-media-chronicles/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 迁移 `0005_characters_media_chronicles`：新增 `media_assets`、`chronicles`；删除 `characters.image_path` 占位列；活跃主图唯一索引、软删除一致性 CHECK。
- 图片：`POST/DELETE /characters/{id}/image`；服务端 MIME + 魔数 + 像素/尺寸校验；JPEG→JPEG、PNG→PNG、WebP→PNG 重编码；最长边 320 缩略图；`GET /media/{id}?variant=thumb|original`（ETag/304、私有长缓存、`nosniff`）。
- 大事记：`GET/POST /characters/{id}/chronicles`、`PATCH/DELETE /chronicles/{id}`；`sort_order` 追加排序、2,000 字上限、`time_label` 可空/可清空、`version` + `If-Match`、删除返回 8 秒 `undo_token`。
- 生命周期联动：人物/作品软删除隐藏、恢复还原、永久删除清理 `media_assets`/`chronicles` 行与磁盘文件。
- 前端：详情主图区（上传/替换/移除、本地预览、进度、字段级错误）、列表 40px 缩略图、详情大事记时间线（空态/新增/编辑/行内删除确认/Toast 撤销）、保存时统一提交（先 PATCH 人物，再上传/删除图片）。

**非目标（已登记延后）**

| 项目 | 承接 |
|---|---|
| 图片裁剪、相册、多图、EXIF 方向 | 不做（需求边界 / PG03-Q5 非 Must） |
| 现有主图以外的人物图片管理、作品封面 | 后续变更 |
| `root_type=chronicle` 进入回收站列表 | 保留未启用（返回 422） |
| 大事记排序拖拽、富文本、批量操作 | 后续 UI 变更 |
| 新建人物 Drawer 内上传 | 不做（创建后进入详情维护） |

## 4. 主要文件/模块变更

**后端**

- 迁移：`backend/internal/migrations/0005_characters_media_chronicles.sql`（Up/Down 可逆，Down 恢复 `image_path`）。
- 模型：`model/media.go`（`MediaAsset`/`Chronicle` 与上限常量）、`model/character.go`（`Image *MediaView`、移除 `ImagePath`）、`model/errors.go`（`UNSUPPORTED_FILE_TYPE`、`FILE_TOO_LARGE`、`IMAGE_TOO_LARGE`、`CHRONICLE_OWNER_UNSUPPORTED`）。
- 仓储：`repository/media.go`、`repository/chronicles.go`（软删/恢复/按根或按作品清理）；`repository/characters.go`、`repository/works.go`（软删/恢复/永久删除联动，Purge 返回待删文件路径）。
- 服务：`service/media.go`（校验、重编码、缩略图、安全的路径构造与原子写入）、`service/chronicles.go`（排序、版本、撤销）、`service/trash.go`（`chronicle` 撤销分支、永久删除后 `removeMedia` 清理文件）。
- HTTP：`httpapi/media.go`、`httpapi/chronicles.go`、`httpapi/{characters,request,router}.go`、`cmd/server/main.go`。
- 依赖：新增 `golang.org/x/image`（WebP 解码），经用户批准。
- 测试：`service/media_test.go`、`integration/character_media_test.go`、`integration/migrations_test.go`（v5 往返）、`integration/suite_test.go`。

**前端**

- API：`frontend/lib/api/media.ts`（XHR 上传 + 进度 + 错误映射、客户端预校验）、`lib/api/chronicles.ts`、`lib/api/characters.ts`（`thumb_url`/`byte_size`）、`lib/messages.ts`。
- 组件：`components/characters/character-image.tsx`、`image-uploader.tsx`、`chronicle-section.tsx`、`character-detail-page.tsx`、`character-table.tsx`；样式 `styles/components.css`。
- 测试：`lib/api/media.test.ts`（3 组）、`e2e/characters-media.spec.ts`（3 例）、视觉基线 `character-detail-1280` 更新。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/character-detail-1280-chromium-win32.png`（含 120px 主图区与大事记区）。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; gofmt -l .` / `go vet ./...` / `go build ./...` | 0，无输出 |
| `cd backend; go test ./...` | 全部 ok（含 `service/media_test.go`） |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | **70 个顶层用例 PASS / 0 FAIL（7.223s）**，F5 后重跑 |
| 其中 `TestMigrationRoundTrip` | 版本 5；0005 Down/Up 往返通过 |
| 其中 `TestCharacterImageValidation` | 文本/损坏 PNG/GIF → 422 `UNSUPPORTED_FILE_TYPE`；11MB → 422 `FILE_TOO_LARGE`；无残留行 |
| `cd frontend; npm run typecheck` / `npm run lint` | 0 |
| `cd frontend; npm run test` | 46 tests 通过 |
| `cd frontend; npx -y node@22 node_modules/next/dist/bin/next build` | 静态导出成功（本机 Node 24 报 `spawn UNKNOWN`，Node 22 通过） |
| `cd frontend; npx playwright test` | **66 passed（53.7s）**，F5 后重跑 |
| 容器迁移 | `successfully migrated database to version: 5` |

## 7. E2E 与人工验收证据

- E2E 新增 3 例（`characters-media.spec.ts`）：上传/展示/列表缩略图/移除主图；非法图片在保存后字段级拒绝并停留编辑态；大事记新增/编辑/删除/Toast 撤销。
- 集成覆盖：上传替换旧图软删与文件清理、ETag/304、缩略图 320、大事记排序与版本冲突与 2,001 字、人物与作品生命周期（隐藏/恢复/永久删除清行与文件）。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-13，验收人：项目负责人），接受 §4 所列 S2 遗留。
- 审查：`review.md` F4 **PASS**（无 S1，8 项 S2）；F5 修复 R-01（上传与删除并发）、R-08（304 缓存/安全头），复审 PASS。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0005 新增 2 表并删除 `characters.image_path`；Down 删表并恢复该列，`TestMigrationRoundTrip` 覆盖带数据往返。旧前端不再读取 `image_path`，无跨版本数据依赖。
- **兼容性**：`CharacterDetail.image` 由字符串占位改为对象/`null`；新增 `image_thumb_url`。删除人物图片端点无 `If-Match`，并发以最后写入生效（契约一致）。`root_type=chronicle` 为保留值，回收站列表返回 422。
- **文件一致性**：图片写入采用临时文件 + rename；DB 事务提交后才删除旧文件；永久删除事务提交后再清理文件，避免回滚后文件丢失。
- **安全**：MIME + 魔数 + 像素 + 尺寸四重校验；路径仅由 DB 构造并拒绝绝对路径/`..`/卷名/越界；响应 `nosniff` + inline + 私有缓存；写请求沿用 `X-Requested-With` 与作品隔离。
- **性能**：缩略图最长边 320，列表用 thumb variant；40MP 解码峰值内存较高（R-03，已登记）。
- **回滚方案**：回退本分支提交并重新部署；若已迁移，执行 0005 Down（图片文件保留但不可见），或从删除前快照恢复（快照含 `images/`）。

## 9. Reviewer 重点关注

1. `service/media.go` 上传事务：`GetForUpdate` 锁定人物 + 复核 `work_id` + 活跃唯一索引的并发语义（R-01）。
2. `service/trash.go` 与 `repository/{characters,works}.go` 永久删除的锁序与「先提交、后删文件」顺序，确保无孤儿行/孤儿文件。
3. `httpapi/media.go` 的 304 分支头部设置顺序（`Cache-Control`/`nosniff`/`ETag`）与 `ServeContent` 用法（R-08）。
4. 图片校验链（Content-Length 预判、`ParseMultipartForm`、魔数、像素上限、尺寸缩放）与错误码映射。
5. `httpapi/chronicles.go` 用 `json.RawMessage` 识别 `time_label: null` 的显式清空语义。
6. 前端保存顺序（PATCH → 上传/删除）与失败后保留编辑态的重试语义（R-04）。
7. 文档一致性：`api-contract.md`、`data-model.md`、SPEC-CHAR-006/007 由 F10 回写。

## 10. 已知遗留（不阻断合并）

| ID | 内容 | 承接 |
|---|---|---|
| R-02 | 列表图片查询失败静默降级为无图 | 可观测性改进 |
| R-03 | 40MP 解码峰值内存较高 | 性能专项 |
| R-04 | 图片失败重试会再次 PATCH，人物版本递增 | 后续「仅重试图片」改进 |
| R-05 | 上传错误与文件按钮缺少 `aria-describedby` 关联 | 后续 a11y 改进 |
| R-06 | 列表视觉基线未含真实图片 | 视觉基线补充 |
| R-07 | 大事记创建与人物删除缺少真实交错测试 | 并发测试专项 |
| UI-04/07/09、A11Y-02、LIFE-05/06 | 上传百分比、前端焦点、慢网、键盘遍历、快照含图片、并发用例未自动化 | 后续测试专项 |
| — | 前端组件测试未接入 jsdom/testing-library，交互由 E2E 覆盖 | 测试基础设施 |
| — | 图片 EXIF 方向未处理；无裁剪/相册 | 需求边界 |
