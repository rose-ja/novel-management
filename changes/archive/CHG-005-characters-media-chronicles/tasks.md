# CHG-005 任务拆解

- Change ID：`CHG-005-characters-media-chronicles`
- 分支：`feat/chg-005-characters-media-chronicles`
- 前置：`proposal.md`、`design.md` 获人工批准后进入 F2
- 规则：每个任务可独立验证；T1～T14 顺序提交；不跨任务混改

## 里程碑

| 里程碑 | 任务 | 可观察结果 |
|---|---|---|
| M1 图片链路 | T1～T4、T9～T10、T12 | 上传/替换/移除/读取/缩略图可用 |
| M2 大事记 | T5～T6、T9、T11 | 增删改查、2,000 字、撤销可用 |
| M3 生命周期 | T7～T8 | 删除/恢复/永久删除联动与文件清理 |
| M4 质量与证据 | T13～T14 | 门禁全绿、证据归档 |

## T1 迁移与模型

- 依赖：无
- 文件：`backend/internal/migrations/0005_characters_media_chronicles.sql`、`backend/internal/model/{media,chronicle,errors}.go`
- 内容：
  1. 建 `media_assets`、`chronicles`（含 `version`）、索引与 CHECK；Down 逆序删除并恢复 `characters.image_path`。
  2. 新增实体/DTO/常量；新增 `ErrUnsupportedFileType`、`ErrFileTooLarge`、`ErrImageTooLarge`、`ErrChronicleTextRequired` 等哨兵。
  3. `migrations_test.go` 增加 0005 往返与列存在性断言。
- 验证：`go test ./...`；集成 `TestMigrationRoundTrip` 通过；`goose up/down` 无残留。
- DoD：迁移可逆；约束与 `design.md` §2 一致。

## T2 媒体仓储与核心服务

- 依赖：T1
- 文件：`repository/media.go`、`service/media.go`、`service/media_test.go`
- 内容：魔数/大小/像素校验、解码与重编码、缩略图（最长边 320）、原子写文件、替换/删除、批量查询、路径越界校验。
- 验证：单元覆盖 jpeg/png/webp/伪造/11MB/超大像素/缩略图尺寸；临时目录断言旧文件删除。
- DoD：非法输入不落盘；合法输入生成原图+缩略图；`MediaView` 完整。

## T3 媒体 HTTP 与路由

- 依赖：T2
- 文件：`httpapi/media.go`、`httpapi/characters.go`、`httpapi/router.go`、`httpapi/request.go`、`cmd/server/main.go`
- 内容：
  1. `POST/DELETE /characters/:id/image`（multipart、`MaxBytesReader`、缺文件 422）。
  2. `GET /media/:asset_id`（variant、ETag/304、Cache-Control、nosniff、inline）。
  3. 错误映射与路由注入。
- 验证：httptest 集成上传/读取/304；非法文件返回稳定错误码。
- DoD：契约端点可用；响应头符合 §3.3；不泄露绝对路径。

## T4 人物 DTO 装配

- 依赖：T2
- 文件：`service/characters.go`、`repository/media.go`、`service/characters_test.go`
- 内容：详情 `image`、列表 `image_thumb_url` 批量装配；无图/读取失败降级为 `null`，不影响人物请求。
- 验证：集成创建人物+上传图后列表/详情返回真实 URL；1000 人列表不新增 N+1（SQL 计数）。
- DoD：字段从占位变实值；旧前端兼容。

## T5 大事记仓储与服务

- 依赖：T1
- 文件：`repository/chronicles.go`、`service/chronicles.go`、`service/chronicles_test.go`
- 内容：列表/创建/编辑（`If-Match`）/删除（软删 + 撤销令牌）、文本与时间标签校验、`sort_order` 追加、owner 行锁校验。
- 验证：单元覆盖空/2,000/2,001、时间标签 200/201、排序、版本冲突、owner 不存在。
- DoD：校验与排序符合 SPEC-CHAR-007；软删除字段一致。

## T6 大事记 HTTP 与撤销扩展

- 依赖：T5
- 文件：`httpapi/chronicles.go`、`httpapi/router.go`、`service/trash.go`、`repository/chronicles.go`
- 内容：4 个端点；`Trash.Undo` 增加 `RootTypeChronicle` 分支（owner 活跃校验、恢复行）；不改变回收站列表根类型。
- 验证：集成 CRUD + 撤销 + 过期 + 重复撤销 + owner 已删返回 `PARENT_DELETED`。
- DoD：8 秒撤销可用；大事记不出现在回收站列表。

## T7 生命周期联动与文件清理

- 依赖：T2、T5
- 文件：`repository/characters.go`、`repository/works.go`、`service/characters.go`、`service/trash.go`
- 内容：
  1. 人物软删除/恢复：同事务更新/恢复 `media_assets`、`chronicles`。
  2. 作品整组软删除/恢复：按 `work_id` 更新/恢复两表。
  3. 永久删除：`Purge`/`PurgeGroup`/`PurgeAllDeleted` 收集媒体路径、删除两表、提交后删文件；锁序保持。
- 验证：集成人物/作品删除→恢复→永久删除；断言行数与 `DATA_DIR/images` 文件；并发用例覆盖锁序。
- DoD：删除/恢复/清理三方一致；无孤儿活跃记录。

## T8 回收站回归

- 依赖：T7
- 文件：`service/trash_test.go`、`integration/character_trash_test.go`、`integration/regression_test.go`
- 内容：确认人物/作品永久删除、清空回收站、撤销仍通过；快照包含图片目录；不回退 CHG-004 行为。
- 验证：全量集成；`TestCharacterPurgeSkipsRestoredRoot` 继续通过。
- DoD：无回归；旧数据无 `image_path` 依赖。

## T9 前端 API 与文案

- 依赖：T3、T6
- 文件：`frontend/lib/api/{media,chronicles,characters,client}.ts`、`frontend/lib/messages.ts`、对应单测
- 内容：XHR 上传（进度、错误体映射）、删除图片、大事记 CRUD（`If-Match`）、消息文案。
- 验证：`npm run test`；mock 覆盖成功/错误/进度/网络异常。
- DoD：类型与契约一致；无英文堆栈文案。

## T10 主图查看与 ImageUploader

- 依赖：T9
- 文件：`components/characters/character-image.tsx`、`image-uploader.tsx`、`character-detail-page.tsx`、`styles/components.css`
- 内容：120px 查看/占位/懒加载；编辑态选择/预览/替换/移除确认/上传进度/错误；保存顺序按 `design.md` §4.3；侧栏主图状态。
- 验证：组件测试 + E2E 上传/移除；a11y alt/占位；取消编辑不上传。
- DoD：取消不产生上传；资料成功图片失败可重试；无破图。

## T11 大事记 UI

- 依赖：T9
- 文件：`components/characters/chronicle-section.tsx`、`character-detail-page.tsx`、`styles/components.css`
- 内容：时间线、空态、行内新增/编辑、2,000 字计数与字段级错误、删除轻确认 + Toast 撤销、失败保留输入。
- 验证：组件测试 + E2E；键盘可达、`aria-live`。
- DoD：SPEC-CHAR-007 场景 1～3 通过。

## T12 列表缩略图与视觉

- 依赖：T9、T10
- 文件：`components/characters/character-table.tsx`、`styles/components.css`、`frontend/e2e/visual.spec.ts`、快照
- 内容：40px 首列、`loading="lazy"`、占位、行高稳定；更新 1280/1024 基线。
- 验证：E2E 慢网路由延迟断言文字先渲染；视觉对比。
- DoD：AC-33 通过；无布局抖动。

## T13 测试补齐与门禁

- 依赖：T1～T12
- 文件：`backend/internal/{service,integration}/*_test.go`、`frontend/**/*.test.ts(x)`、`frontend/e2e/characters.spec.ts`、`e2e/fixtures/*`
- 内容：补齐 `design.md` §6 全部用例；E2E 上传 jpg/webp/非法、移除、大事记 CRUD/上限/撤销、慢网。
- 验证：后端 gofmt/vet/build/单测/集成；前端 typecheck/lint/单测/构建；全量 E2E。
- DoD：门禁全绿；失败用例有记录。

## T14 证据与文档

- 依赖：T13
- 文件：`changes/CHG-005-characters-media-chronicles/{verification,review,acceptance}.md`、`checklist.md`
- 内容：回填命令与结果、E2E 证据、风险/遗留、检查清单勾选；F10 再回写 `project/`。
- 验证：文档可复核；命令可重跑。
- DoD：F3 证据完整，等待 F4 独立审查。

## 任务依赖图

```
T1 → T2 → T3 → T9 → T10 → T12
 │     └→ T4
 └→ T5 → T6 → T9 → T11
        └→ T7 → T8
T1～T12 → T13 → T14
```

## 并行边界

- T2 与 T5 可并行（不同表/文件）。
- T10 与 T11 可在 T9 后并行。
- T3 与 T6 可并行（不同 handler），但都要改 `router.go`，合并时按顺序提交。
- T7 修改 `characters/works/trash`，避免与 T4/T6 同文件并行。

## 测试命令

```
cd backend
gofmt -l .
go vet ./...
go test ./...
$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
go test -count=1 -tags=integration ./internal/integration/...

cd frontend
npm run typecheck
npm run lint
npm run test
npx -y node@22 node_modules/next/dist/bin/next build
npx playwright test
```
