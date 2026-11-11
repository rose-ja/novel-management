# CHG-005 技术设计

- Change ID：`CHG-005-characters-media-chronicles`
- 关联：`proposal.md`；SPEC-CHAR-006/007；PG-03；`api-contract.md` §3.3/§3.8；`data-model.md` §3.13/§3.14
- 原则：最小实现、复用现有软删除根/撤销令牌/快照/错误码体系；不引入队列与云存储。

## 1. 总体方案

```
浏览器
  ├─ 详情：GET /characters/{id} → image{asset_id,url,width,height}
  ├─ <img src="/api/v1/media/{asset_id}?variant=thumb"> （loading=lazy）
  ├─ 编辑保存：PATCH /characters/{id} → POST/DELETE /characters/{id}/image
  └─ 大事记：GET/POST /chronicles、PATCH/DELETE /chronicles/{id} → Toast 撤销 /undo/{token}
后端
  httpapi ── Characters / Media / Chronicles
  service  ── Media（校验/重编码/缩略图/文件） / Chronicles（校验/排序）
           ── Characters（DTO 装配） / Trash（根联动/永久删除/撤销）
  repository ── media_assets / chronicles / characters / works
  DATA_DIR/images/{work_id}/{asset_id}.{ext}
```

- 图片处理同步完成：读取 ≤10MB → 魔数/解码 → 重编码（必要时缩放）→ 写原图与缩略图 → 事务写 DB。
- 列表缩略图批量查询：`media_assets WHERE entity_type='character' AND entity_id IN (...) AND deleted_at IS NULL`，避免 N+1。
- 大事记是子内容：单条删除只发放 8 秒撤销令牌，不进入回收站列表；人物/作品删除时统一 `deleted_root_id`。

## 2. 数据模型与迁移

### 2.1 `0005_characters_media_chronicles.sql`（Up）

```sql
-- 1) 移除 CHG-004 占位列
ALTER TABLE characters DROP COLUMN IF EXISTS image_path;

-- 2) 图片资产
CREATE TABLE media_assets (
  id               TEXT PRIMARY KEY,
  work_id          TEXT NOT NULL REFERENCES works(id) ON DELETE CASCADE,
  entity_type      TEXT NOT NULL CHECK (entity_type IN ('character')),
  entity_id        TEXT NOT NULL,
  original_path    TEXT NOT NULL,
  thumb_path       TEXT,
  mime             TEXT NOT NULL CHECK (mime IN ('image/jpeg','image/png')),
  byte_size        BIGINT NOT NULL CHECK (byte_size > 0 AND byte_size <= 10485760),
  width            INTEGER,
  height           INTEGER,
  sha256           CHAR(64) NOT NULL,
  version          INTEGER NOT NULL DEFAULT 1,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at       TIMESTAMPTZ,
  deleted_root_id  TEXT,
  deleted_by_action TEXT,
  CHECK ((deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
      OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL AND deleted_by_action IS NOT NULL))
);
CREATE UNIQUE INDEX media_one_per_entity_uq
  ON media_assets(entity_type, entity_id) WHERE deleted_at IS NULL;
CREATE INDEX media_work_deleted_idx ON media_assets(work_id, deleted_at);

-- 3) 大事记
CREATE TABLE chronicles (
  id               TEXT PRIMARY KEY,
  work_id          TEXT NOT NULL REFERENCES works(id) ON DELETE CASCADE,
  owner_type       TEXT NOT NULL CHECK (owner_type IN ('character')),
  owner_id         TEXT NOT NULL,
  text             TEXT NOT NULL CHECK (char_length(text) BETWEEN 1 AND 2000),
  time_label       TEXT CHECK (time_label IS NULL OR char_length(time_label) <= 200),
  sort_order       INTEGER NOT NULL DEFAULT 0,
  version          INTEGER NOT NULL DEFAULT 1,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at       TIMESTAMPTZ,
  deleted_root_id  TEXT,
  deleted_by_action TEXT,
  CHECK ((deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
      OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL AND deleted_by_action IS NOT NULL))
);
CREATE INDEX chronicles_owner_idx ON chronicles(owner_type, owner_id, sort_order);
CREATE INDEX chronicles_work_idx ON chronicles(work_id);
```

### 2.2 Down

```sql
DROP TABLE chronicles;
DROP TABLE media_assets;
ALTER TABLE characters ADD COLUMN image_path TEXT;
```

> `mime` 只允许重编码后的 `image/jpeg` / `image/png`（WebP 输入转 PNG）。契约允许 `image/webp`，实现不落库该 MIME；F10 在契约/数据模型注明。

### 2.3 多态约束

- `owner_type/entity_type` 由 CHECK 限定为 `character`（`faction` 在 CHECK 中不开放，CHG-006 再扩）；应用层校验 `owner/entity` 存在且同作品。
- 无多态 FK；生命周期由 service 事务维护。

## 3. 后端模块

### 3.1 `model/media.go`

```go
const (
  MaxImageBytes      = 10 << 20
  MaxImagePixels     = 40_000_000
  MaxImageDimension  = 4096
  ThumbnailLongEdge  = 320
  MediaEntityCharacter = "character"
)

type MediaAsset struct { /* 与表列一致；含 Deleted* */ }
type MediaView struct {
  AssetID string `json:"asset_id"`
  URL     string `json:"url"`
  ThumbURL string `json:"thumb_url"`
  Width   int    `json:"width"`
  Height  int    `json:"height"`
  ByteSize int64 `json:"byte_size"`
}
```

### 3.2 `service/media.go`

`Media` 依赖 `db *gorm.DB`、`dataDir string`、`now func() time.Time`。

**UploadCharacterImage(ctx, characterID, file multipart.File, header *multipart.FileHeader) (*model.MediaView, error)**

1. 校验 `header.Size` 1..10MB；用 `io.LimitReader` 再读 10MB+1 防伪造。
2. 魔数：JPEG `FF D8 FF`、PNG `89 50 4E 47 0D 0A 1A 0A`、WebP `RIFF....WEBP`；不匹配 → `UNSUPPORTED_FILE_TYPE`。
3. `image.DecodeConfig`（WebP 用 `x/image/webp` 注册）校验宽高 >0、像素 ≤40MP；随后完整 `Decode`。解码失败 → `UNSUPPORTED_FILE_TYPE`。
4. 规范化方向（EXIF 可选，本轮按 Go 解码结果）；若最长边 >4096 等比缩小；输出：
   - JPEG 输入 → JPEG（quality 85）
   - PNG 输入 → PNG（BestSpeed/Default）
   - WebP 输入 → PNG（保留 alpha）
5. 生成缩略图：最长边 320，保持比例（`x/image/draw.CatmullRom`）。
6. 查询人物（`GetActiveByWork` 不需要 work；`Get`）拿 `work_id`；不存在 → `NOT_FOUND`。
7. 路径：`images/{work_id}/{asset_id}.{jpg|png}`、`images/{work_id}/{asset_id}_thumb.{jpg|png}`；`assetID=model.NewID()`。
8. 写临时文件 `os.CreateTemp(dir, ".upload-*")` → 写入 → `Sync` → `Rename` 到最终路径；两个文件都成功后才写 DB。
9. 事务：旧活跃资产 `deleted_at=now, deleted_root_id=characterID, deleted_by_action='image_replace'`；插入新资产。失败 → 删除刚写文件并返回错误。
10. 提交后删除旧资产文件（`os.Remove` 失败仅 `slog.Warn`）。
11. 返回 `MediaView{URL: /api/v1/media/{id}, ThumbURL: ...?variant=thumb}`。

**DeleteCharacterImage(ctx, characterID) error**：事务内锁定活跃资产 → 软删除（`deleted_root_id=characterID`）；无资产 → `NOT_FOUND`；提交后删文件。

**ServeAsset(ctx, assetID, variant) (*model.MediaAsset, string, error)**：查活跃资产；`variant=thumb` 且 `thumb_path` 空 → 回退原图；校验路径在 `DATA_DIR` 下（`filepath.Clean` + `Rel` 不以 `..` 开头）；返回绝对路径。

**ListCharacterImages(ctx, ids []string) (map[string]*MediaView, error)**、**GetCharacterImage(ctx, id)**：批量/单条装配。

### 3.3 `model/chronicle.go` + `service/chronicles.go`

```go
const MaxChronicleRunes = 2000
const MaxChronicleTimeLabelRunes = 200

type Chronicle struct { ID, WorkID, OwnerType, OwnerID, Text, *TimeLabel, SortOrder, Version, ... }
type ChronicleView struct { ID, OwnerType, OwnerID, Text string; TimeLabel *string; SortOrder, Version int; CreatedAt, UpdatedAt time.Time }
```

- `List(ctx, ownerType, ownerID)`：只允许 `character`；校验 owner 活跃；`ORDER BY sort_order, id`；返回空数组。
- `Create(ctx, in)`：`owner_type=character`，事务内 `characters.GetForUpdate` 确认活跃 → work_id；`text` trim，1..2000；`time_label` trim，空→nil，≤200；`sort_order = COALESCE(MAX(sort_order),0)+1`（按 owner 过滤）；插入。
- `Update(ctx, id, in)`：`If-Match` 必需；行锁；`version` 比对；指针字段：`text` 省略=不变、必填非空；`time_label` 显式 null=清空；`sort_order` ≥0；更新 `version+1`。
- `Delete(ctx, id)`：行锁；已删 → `ALREADY_DELETED`；软删除 `deleted_root_id = owner_id`、`deleted_by_action='chronicle_delete'`；插入 8 秒撤销令牌 `RootTypeChronicle`。
- 撤销在 `Trash.Undo` 新增分支：查已删 chronicle；校验 owner 仍活跃（否则 `PARENT_DELETED`）；恢复行（`deleted_at=NULL`，保留 `version`）。

### 3.4 生命周期联动（`repository` + `service`）

| 场景 | 行为 |
|---|---|
| 人物软删除 | `characters.SoftDelete` 同事务更新 `media_assets`、`chronicles`：`deleted_at=now, deleted_root_id=characterID, action=character_delete` |
| 人物恢复 | `characters.Restore` 同事务恢复 `deleted_root_id=characterID` 的媒体/大事记 |
| 作品整组软删除 | `works.SoftDeleteGroup` 追加按 `work_id` 更新两张表，`deleted_root_id=workID` |
| 作品整组恢复 | `works.RestoreGroup` 追加按 `work_id` 恢复 |
| 单人物永久删除 | `Characters.Purge` 先收集媒体路径 → 删除 `media_assets`/`chronicles` → 删除人物；返回路径 |
| 作品永久删除 | `works.PurgeGroup`/`PurgeAllDeleted` 收集 `work_id` 路径 → 删除两表 → 删除作品 |
| 清空回收站 | 复用上述方法；提交后统一删除文件 |
| 单条大事记删除 | 只软删除，不进回收站列表；8 秒内可撤销 |

**签名调整**：`Characters.Purge(ctx,id) (int64, []string, error)`、`Works.PurgeGroup(ctx,id) (int64, []string, error)`、`Works.PurgeAllDeleted(ctx) (int64, []string, error)`；调用方 `service/trash.go` 在事务提交后 `media.RemoveFiles(paths)`。锁序保持：人物行 → 引用 → 别名 → 媒体/大事记 → 人物。

### 3.5 `httpapi`

- `registerCharacters` 增加 `POST /:id/image`、`DELETE /:id/image`。
  - 上传前 `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxImageBytes+1MB)`；`ParseMultipartForm(1<<20)`；字段名 `file`；缺文件 → 422 `VALIDATION_ERROR field=file`。
  - 成功 201 + `MediaView`；`UNSUPPORTED_FILE_TYPE`/`FILE_TOO_LARGE`/`IMAGE_TOO_LARGE` 映射 422。
- `registerMedia`：`GET /media/:asset_id`，`variant` 默认 `thumb`；设置头并 `http.ServeContent`；支持 `If-None-Match` → 304。
- `registerChronicles`：GET/POST `/chronicles`、PATCH/DELETE `/chronicles/:id`。
- `abortServiceError` 新增错误码映射；`MediaView` 写入时使用相对 URL。

### 3.6 人物 DTO 装配

- `Characters.Get`：`Media.GetCharacterImage` 失败视为无图（`image=null`），不让详情失败。
- `Characters.List`：收集当前页 ID，`ListCharacterImages` 一次查询；`image_thumb_url` 缺失为 `null`。
- 列表/详情返回的 URL 不含主机，前端直接使用。

## 4. 前端设计

### 4.1 API 层

- `lib/api/media.ts`
  - `uploadCharacterImage(id, file, onProgress?)`：使用 `XMLHttpRequest`，`POST /api/v1/characters/{id}/image`，`FormData("file")`，设置 `X-Requested-With`；解析统一错误体为 `ApiError`。
  - `deleteCharacterImage(id)`：`requestVoid(..., {method:"DELETE"})`。
- `lib/api/chronicles.ts`：`listChronicles({ownerType,ownerID})`、`createChronicle`、`updateChronicle(id,{text,timeLabel,sortOrder,version})`、`deleteChronicle(id)`。
- `lib/api/characters.ts`：`CharacterImage` 增加 `thumb_url`、`byte_size`（可选），不改现有字段。

### 4.2 主图

- `CharacterImage` 组件（只读）：120px（详情）/40px（列表 compact）；`<img loading="lazy" decoding="async">`，`alt="人物主图：{name}"`；`onError` 切占位；占位 = 雾绿渐变 + 姓名首字（`role="img"` 同 alt）。
- `ImageUploader`（编辑态）：
  - props：`name`、`current: CharacterImage | null`、`file: File | null`、`removed: boolean`、`uploading/percent/error`、`onSelect/onRemove/onRetry`。
  - 选择：`accept="image/jpeg,image/png,image/webp"`；客户端校验类型与 10MB，错误贴近控件；`URL.createObjectURL` 预览，替换/卸载时释放。
  - 移除：`Dialog` 确认「移除主图？」；确认后本地标记 removed，保存时调用 DELETE。
  - 上传中显示 `Progress` 线性百分比；失败显示原因 + 重试；不写入残缺图。
- 详情查看态：主图位于身份块左侧；侧栏档案显示「主图 已上传/未上传」。

### 4.3 编辑保存顺序

```
点击保存
 1. 本地校验（姓名/别名/大事记不影响人物保存）→ 失败则停留
 2. PATCH /characters/{id}（If-Match=version）
    ├ 409 → 冲突提示，保留输入
    └ 成功 → setDetail(updated)
 3. 图片：
    a. 有新文件 → POST image（进度）；失败 → 停留编辑态，提示「资料已保存，主图上传失败」
    b. 标记 removed 且无新文件 → DELETE image；404 视为已移除
    c. 无变化 → 跳过
 4. 重新 GET 详情（拿 image 与最新 version）→ 退出编辑 → Toast「已保存」
```

- `imageDirty = file !== null || removed`；保存按钮在 `!formDirty && !imageDirty` 时禁用。
- 取消编辑：若有新文件/removed 标记，进入未保存确认；确认后释放预览，不调用上传。
- 冲突：PATCH 409 时不提交图片；刷新后丢失本地选择（符合“不自动合并”）。

### 4.4 大事记

- `ChronicleSection`（详情查看态，位于主栏底部）：
  - 标题 + 「添加」按钮；空态「还没有大事记」+「添加第一条」。
  - 列表 `<ol>`：序号 `01/02`（等宽）、时间标签（灰）、正文 `white-space: pre-wrap`、右侧「编辑/删除」。
  - 新增/编辑：行内表单（时间标签输入 + textarea + 字数 `n/2000` + 保存/取消）；内容空/超限字段级错误并聚焦。
  - 删除：行内确认条「删除这条大事记？[取消][删除]」；确认后调用 DELETE，Toast 8 秒撤销；撤销调用 `/undo/{token}` 并刷新。
  - 失败：行内 Alert + 重试，保留输入；不整页报错。
- 详情页 `reloadTick` 复用：大事记变更后刷新详情与列表无关（列表不展示大事记）。

### 4.5 列表缩略图

- `character-table` 新增首列「主图」40px：`CharacterImage compact`；`image_thumb_url` 为空显示首字占位；`loading="lazy"`。
- 列宽固定，懒加载不改变行高；1024 宽度保持 40px，不因窄屏放大。
- 视觉基线更新：`characters-1280/1024`、`character-detail-1280`（含真实图片与占位两种）。

### 4.6 文案

在 `lib/messages.ts` 增加：主图标题/上传/替换/移除/移除确认/上传中/上传失败/已上传/未上传/资料已保存但主图失败；大事记标题/空态/添加/编辑/删除确认/内容必填/超限/保存/取消/撤销提示/已移入回收站。禁止英文堆栈。

## 5. 错误与边界

| 场景 | 状态码/错误码 | 前端表现 |
|---|---|---|
| 缺文件/字段名错误 | 422 `VALIDATION_ERROR` field=file | 字段级提示 |
| >10MB | 422 `FILE_TOO_LARGE` | 「图片不能超过 10MB」 |
| MIME/魔数不符/解码失败 | 422 `UNSUPPORTED_FILE_TYPE` | 「仅支持 JPG/PNG/WebP」 |
| 像素 >40MP / 边长 >4096 缩放后仍异常 | 422 `IMAGE_TOO_LARGE` | 「图片尺寸过大」 |
| 人物不存在/已删 | 404 `NOT_FOUND` | 详情 404 态 |
| 无图删除 | 404 `NOT_FOUND` | 视为已移除 |
| 大事记空/超 2000 | 422 `VALIDATION_ERROR` field=text | 字段级错误并聚焦 |
| 大事记版本冲突 | 409 `VERSION_CONFLICT` | 固定文案 + 刷新 |
| 撤销过期 | 409 `UNDO_EXPIRED` | Toast「撤销窗口已关闭」 |
| 人物/作品已删后操作大事记 | 404/409 `PARENT_DELETED` | 提示并刷新 |

- 路径安全：文件路径只来自 DB；`ServeAsset` 校验 `filepath.Rel(dataDir, path)` 不含 `..`；不返回绝对路径。
- 响应头：`Content-Type` 按 MIME；`X-Content-Type-Options: nosniff`；`Content-Disposition: inline; filename="image"`。
- 缓存：`ETag: "{sha256}-{variant}"`；`If-None-Match` 命中返回 304；不变更原图则缓存长期有效。

## 6. 测试设计

### 6.1 后端单元

- `service/media_test.go`：魔数矩阵（jpeg/png/webp/伪 PNG/文本）、10MB 边界、像素上限、重编码 MIME、缩略图最长边 320、路径越界拒绝、替换后旧文件删除（临时目录）。
- `service/chronicles_test.go`：空/空白/2,000/2,001 字、时间标签 200/201、排序追加、`If-Match` 缺失/过期、owner 不存在。

### 6.2 集成（真实 PostgreSQL + `t.TempDir()` 作为 DATA_DIR）

- 迁移往返：0005 Up → 播种媒体/大事记 → Down → 断言表不存在、`characters.image_path` 恢复 → Up。
- 上传：multipart jpg/png/webp → 201；GET thumb/original 200 + ETag + 304；替换旧图（旧行软删、旧文件删除）；DELETE 204/404。
- 非法文件：11MB、GIF 伪装 jpg、文本、超大像素 → 422 且无文件/行写入。
- 人物详情/列表：`image`/`image_thumb_url` 正确，批量查询 SQL 次数不随行数增长。
- 生命周期：人物删除→媒体/大事记隐藏；恢复→回来；永久删除→行与文件为空；作品删除/恢复/永久删除；PurgeAll。
- 大事记：CRUD、排序、2,000 字、撤销、重复撤销、过期、owner 删除时拒绝、跨作品隔离。
- 引用/锁：并发上传替换同一人物（唯一索引 + 事务）；Chronicle POST 与人物删除交错不产生孤儿活跃大事记。

### 6.3 前端单测

- `lib/api/media.test.ts`（XHR mock：进度、错误体映射）、`chronicles.test.ts`（查询/If-Match/错误）。
- `image-uploader.test.tsx`：类型/大小本地校验、预览 URL 释放、移除确认、上传中禁用。
- `chronicle-section.test.tsx`：空态、添加、2,001 字错误、编辑、删除确认与撤销回调、失败保留输入。

### 6.4 E2E（`e2e/characters.spec.ts` 扩展）

- 上传 jpg → 详情 120px 可见、列表 40px 可见；刷新后仍在。
- 上传 webp → 成功显示；上传 11MB/GIF 伪装 → 字段级错误且不产生图片。
- 移除主图 → 占位，刷新后仍为空。
- 大事记：添加两条 → 顺序 01/02；编辑；2,001 字阻止；删除后 8 秒内撤销恢复。
- 慢网：`page.route` 延迟 media 响应 2s，断言姓名/标题先出现、无布局抖动。
- 视觉基线：更新列表/详情快照；a11y 扫描详情含图/大事记。

### 6.5 门禁命令

```
cd backend; gofmt -l .; go vet ./...; go test ./...
$env:TEST_DATABASE_URL="postgres://novel:nm_local_only@127.0.0.1:5432/novel?sslmode=disable"
go test -count=1 -tags=integration ./internal/integration/...
cd frontend; npm run typecheck; npm run lint; npm run test
npx playwright test
```

## 7. 兼容性与迁移

- 现有 `CharacterDetail.image`/`image_thumb_url` 字段名与类型不变，只从 `null` 变实值；前端旧逻辑无需改动即可兼容。
- `characters.image_path` 无生产数据；0005 删除该列，Down 恢复空列。
- 快照/导出已包含 `images/`，无需改格式；图片文件路径相对 `DATA_DIR`，容器重建后仍可读。
- 图片 MIME 落库为 jpeg/png；契约中的 `image/webp` 仅作为输入格式，F10 同步说明。

## 8. 实施顺序与并行边界

1. T1 迁移与模型 → T2 媒体仓储/服务 → T3 媒体 HTTP → T4 人物 DTO 装配。
2. T5 大事记仓储/服务/撤销 → T6 大事记 HTTP。
3. T7 生命周期联动（删除/恢复/永久删除/文件清理）→ T8 人物删除/恢复/清除回归。
4. T9 前端 API/消息 → T10 主图查看/上传组件 → T11 大事记组件 → T12 列表缩略图与样式。
5. T13 测试补齐（单元/集成/前端/E2E）→ T14 证据与文档。

T2/T5 可并行（不同表）；T10/T11 可在 T9 后并行；T7 依赖 T2/T5；T12 依赖 T9/T10。

## 9. 文件级改动清单

| 层 | 文件 | 改动 |
|---|---|---|
| DB | `backend/internal/migrations/0005_characters_media_chronicles.sql` | 新建两表、索引、CHECK；删 `image_path` |
| Model | `model/media.go`、`model/chronicle.go`、`model/errors.go` | 实体/DTO/错误哨兵 |
| Repo | `repository/media.go`、`repository/chronicles.go`、`repository/characters.go`、`repository/works.go` | CRUD、批量查询、根联动、Purge 返回路径 |
| Service | `service/media.go`、`service/chronicles.go`、`service/characters.go`、`service/trash.go` | 编解码/缩略图/文件、校验/撤销、DTO、Purge 文件清理 |
| HTTP | `httpapi/media.go`、`httpapi/chronicles.go`、`httpapi/characters.go`、`httpapi/router.go`、`httpapi/request.go` | 路由、multipart、二进制响应、错误映射 |
| 启动 | `cmd/server/main.go` | 注入 Media/Chronicles 服务与 `DATA_DIR` |
| 前端 API | `lib/api/media.ts`、`lib/api/chronicles.ts`、`lib/api/characters.ts`、`lib/api/client.ts` | 上传/删除/CRUD、FormData |
| 前端 UI | `components/characters/character-image.tsx`、`image-uploader.tsx`、`chronicle-section.tsx`、`character-detail-page.tsx`、`character-table.tsx`、`styles/components.css`、`lib/messages.ts` | 主图/占位/上传/时间线/缩略图 |
| 测试 | `backend/internal/{service,integration}/*_test.go`、`frontend/lib/api/*.test.ts`、`frontend/components/characters/*.test.tsx`、`frontend/e2e/characters.spec.ts`、`frontend/e2e/fixtures/*` | 单元/集成/E2E/视觉 |

## 10. 待人工确认（F1）

1. 是否批准新增 `golang.org/x/image`（已口头/弹窗确认，需在 F7 记录）。
2. 是否接受大事记“子内容 + 轻撤销”、不进入回收站列表（已确认）。
3. 是否接受本轮仅详情编辑态维护主图，新建 Drawer 不加上传（待批准）。
