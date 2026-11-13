# CHG-005 人物图片与大事记

- Change ID：`CHG-005-characters-media-chronicles`
- 分支：`feat/chg-005-characters-media-chronicles`
- 目标：人物可维护 1 张主图与按序大事记，缩略图懒加载不阻塞文字，形成完整人物档案。
- 关联 Spec：SPEC-CHAR-006、SPEC-CHAR-007
- 关联需求：FR-04、FR-06、FR-67；EX-22、EX-26；AC-02、AC-17、AC-26、AC-33
- 关联页面：PG-03 人物详情（§4.1 主图、§4.3 大事记、§6.1/§6.2、§8、§10）；PG-02 列表缩略图（§4 列、§8 图片）
- 关联契约：`api-contract.md` §3.3 图片端点、§3.8 大事记、§3.3 GET 媒体；`data-model.md` §3.13/§3.14
- 依赖：CHG-004（已完成并归档）
- 复杂度：M
- 状态：`archived`（2026-11-13 本地合并 `main`，未推送；F7 ACCEPT）

## 1. 用户价值

CHG-004 完成后人物档案只有文字，主图区与大事记区仍是占位。本变更补齐人物档案的最后两块内容：

1. 一眼认出角色：列表 40px、详情 120px 主图，缺失时用姓名首字占位；缩略图懒加载不阻塞文字。
2. 记录角色经历：大事记按手工顺序展示、可增删改，2,000 字上限，删除可 8 秒撤销。
3. 数据不悬空：删除/恢复/永久删除人物或作品时，图片与大事记随根联动；永久删除前有快照，文件与行一致清理。

## 2. 范围

### 2.1 In Scope

- 迁移 `0005_characters_media_chronicles`：`media_assets`、`chronicles`（含 `version`）、索引/CHECK、Down 可回滚；移除 CHG-004 的 `characters.image_path` 占位列。
- 图片上传/替换/移除：jpg/png/webp、≤10MB、MIME + 魔数 + 像素上限校验；按内容重编码；最长边 320px 缩略图。
- 图片读取：`GET /api/v1/media/{asset_id}?variant=thumb|original`，`ETag`/304、`Cache-Control: private, max-age=31536000, immutable`、`nosniff`、`Content-Disposition: inline`。
- 人物 DTO 接入：`CharacterDetail.image`、`CharacterSummary.image_thumb_url` 返回真实数据（列表批量装配，无 N+1）。
- 大事记 API：列表、创建、编辑（`If-Match` + `version`）、删除 + 8 秒撤销；2,000 字/时间标签 200 字校验。
- 前端：详情页主图展示/占位、编辑态 `ImageUploader`（本地预览、替换、移除确认、上传进度/错误）；列表 40px 懒加载缩略图；`ChronicleSection`（时间线、行内新增/编辑/删除轻确认、Toast 撤销、2,000 字字段级错误）。
- 生命周期联动：人物软删除/恢复/永久删除、作品整组软删除/恢复/永久删除、清空回收站，覆盖 `media_assets` 与 `chronicles`；永久删除后清理图片文件。
- 测试：后端单元 + 集成、前端单测、E2E-02/AC-33/AC-17、迁移往返、慢网人工检查。

### 2.2 Out of Scope（明确不做）

| 项目 | 承接/说明 |
|---|---|
| 势力/地点图片、多图相册、裁剪、拖拽排序 | 需求边界（SPEC-CHAR-006 §边界） |
| 新建人物 Drawer 内上传主图 | 本轮仅在详情编辑态维护；创建后进入详情再上传。PG-02 新建表单不新增上传区 |
| 图片异步任务队列、上传进度百分比之外的断点续传 | 同步生成缩略图；AC-26 不含缩略图耗时 |
| 大事记时间线可视化、跨人物聚合、日期解析、拖拽排序 | 需求边界；仅手工 `sort_order` 追加与显式编辑 |
| `root_type=chronicle` 回收站根 | 已裁决：大事记是子内容 + 轻撤销，不进回收站；契约字段标为保留未启用（F10 回写） |
| 图片 lightbox、EXIF 旋转编辑、裁剪 UI | PG03-Q5 非 Must；仅按 EXIF 方向自动纠正 |
| 导入/导出/快照格式变更 | CHG-010/011；快照已包含 `images/` 目录 |
| 校对确认、检索大事记正文 | CHG-011、CHG-009 |

## 3. 已确认决策

| ID | 决策 | 结论 | 理由 |
|---|---|---|---|
| D1 | 图片编解码依赖 | 新增 `golang.org/x/image`（纯 Go） | 需解码 WebP + 高质量缩放；标准库无 WebP；无 CGO。已在 F1 获用户批准 |
| D2 | 大事记删除归属 | 子内容 + 8 秒轻撤销，不进回收站 | 与 PG-03 §6.1「行内轻确认 + Toast」一致；避免新增回收站类型/UI。已获批准 |
| D3 | 编辑态上传时机 | 点「保存」时统一提交 | 取消编辑不产生上传；一个人物 1 张图，避免“先传后撤”脏数据。已获批准 |
| D4 | 缩略图生成 | 同步生成，最长边 320px，保持比例 | 实现简单、失败可回滚；AC-26 明确不含缩略图耗时 |
| D5 | 存储路径 | `{DATA_DIR}/images/{work_id}/{asset_id}.{ext}` 与 `{asset_id}_thumb.{ext}` | 与架构 §media 一致；路径来自 DB，禁止用户拼接 |
| D6 | 替换旧图 | 旧资产软删除、行保留、文件在事务提交后删除 | 契约「旧资产软删除」；文件删除失败只记日志，快照硬链接可恢复 |
| D7 | 大事记乐观锁 | `chronicles.version` + PATCH `If-Match` | 与契约 §3.8 一致；DB 模型补 `version`（F10 回写 data-model） |
| D8 | 图片与人物版本 | 图片上传/删除不递增人物 `version` | 契约无 `If-Match`；图片是附属资产，人物文字并发仍由 PATCH 保护 |
| D9 | 文件删除时机 | 永久删除事务提交后删除文件；失败只记日志并保留孤儿文件 | 避免 DB 回滚后文件已丢；快照兜底 |

## 4. 允许与禁止的文件

### 4.1 允许修改

- 后端：`internal/migrations/0005_*.sql`、`internal/model/{media,chronicle,errors}.go`、`internal/repository/{media,chronicles,characters,works}.go`、`internal/service/{media,chronicles,characters,trash}.go`、`internal/httpapi/{media,chronicles,characters,router,request}.go`、`cmd/server/main.go`、`internal/config/config.go`（如新增常量）、`internal/integration/*_test.go`、`go.mod`/`go.sum`（仅 `golang.org/x/image`）。
- 前端：`lib/api/{media,chronicles,characters,client}.ts`、`lib/messages.ts`、`components/characters/{character-detail-page,character-image,image-uploader,chronicle-section,character-table}.tsx`、`styles/components.css`、`e2e/characters.spec.ts`、`e2e/fixtures/*`、相关单测。
- 文档：`changes/CHG-005-characters-media-chronicles/*`（F1～F10）。

### 4.2 禁止修改

- 不改势力/地点/榜单/检索/导入/校对/导出/快照格式与页面。
- 不改 CHG-004 已验收的文字字段语义、删除矩阵与回收站根类型（`character` 等）。
- 不新增远程服务、云存储、图片 CDN、遥测。
- 不修改 `project/` 上游基线；F10 才允许回写 Spec/契约/数据模型。
- 不重构无关组件；`character-detail-page.tsx` 只做本变更所需的图片/大事记接入。

## 5. 接口与数据影响

- 新增 `POST/DELETE /api/v1/characters/{id}/image`、`GET /api/v1/media/{asset_id}`、`GET/POST /api/v1/chronicles`、`PATCH/DELETE /api/v1/chronicles/{id}`。
- 现有响应字段从占位变实值：`CharacterSummary.image_thumb_url`、`CharacterDetail.image`（结构不变）；新增 `Chronicle` DTO。
- 数据表新增 `media_assets`、`chronicles(version)`；删除 `characters.image_path`。
- 错误码新增：`UNSUPPORTED_FILE_TYPE`（422）、`FILE_TOO_LARGE`（422）、`IMAGE_TOO_LARGE`（422，像素上限）。
- 生命周期：`deleted_root_id` 统一指向删除根（人物 ID 或作品 ID）；恢复按根恢复；永久删除清理行与文件。

## 6. UI 状态

- 主图：empty（姓名首字 + 雾绿渐变）/ loading（浅色块）/ ready（`loading="lazy"`）/ error（占位，不显示破图）；编辑态 selected/replaced/removed/uploading（进度）/ failed（字段级错误 + 重试）。
- 大事记：loading（Skeleton）/ empty（「还没有大事记」+「添加第一条」）/ ready / adding / editing / deleting（行内确认条）/ error（保留输入 + 重试）。
- 保存：资料保存与图片提交分两步；资料成功而图片失败时停留在编辑态并提示「资料已保存，主图上传失败，可重试」。

## 7. DoD

1. ≤10MB 的 jpg/png/webp 可上传并在详情 120px、列表 40px 显示；webp 可被解码并重编码。
2. >10MB、伪造 MIME、非图片、超大像素被拒绝，返回稳定错误码，不写入残缺文件。
3. 列表文字先渲染，缩略图 `loading="lazy"` 不阻塞；慢网下无布局抖动。
4. 大事记可新增/编辑/删除，顺序稳定，2,000 字超限字段级错误；删除 8 秒内可撤销。
5. 人物/作品删除、恢复、永久删除后，图片与大事记一致隐藏/恢复/清理；永久删除后无文件残留。
6. `go test ./...`、集成、前端 typecheck/lint/单测、全量 E2E 通过；迁移 0005 Up/Down 往返通过。
7. 文档与证据（verification/review/acceptance）齐全，S1/S2 处理或登记。

## 8. 风险与回滚

| 风险 | 影响 | 缓解 |
|---|---|---|
| 图片解码炸弹 / 超大像素 | 内存耗尽 | 10MB 流式限制 + 像素上限 40MP + 尺寸上限 4096 |
| WebP 重编码兼容性 | 图片显示异常 | 统一转 PNG；E2E/单测覆盖真实 webp 样本 |
| 文件与 DB 不一致 | 孤儿文件/破图 | 先写文件后提交 DB；失败清理新文件；孤儿文件仅记日志并由后续清理任务处理（不在本轮） |
| 永久删除漏文件 | 磁盘泄漏 | 事务内收集路径、提交后删除；集成测试断言目录为空 |
| 大事记并发/根联动 | 脏数据 | 行锁 + 版本 + 统一 `deleted_root_id`；集成覆盖 |
| 上传耗时 | AC-26 1s | 10MB 本机目标；重编码/缩略图同步执行，超时仅影响图片步骤，不阻塞文字保存 |

**回滚**：回退本分支；执行 `0005` Down（先删 `media_assets`/`chronicles`，再恢复 `characters.image_path`）；关闭详情图片/大事记 UI。已上传文件保留但不可见；快照可恢复。
