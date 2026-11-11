# CHG-005 验收清单

> 勾选规则：F3/F6 实测通过后勾选并写证据（命令、用例名、截图名）。未实现项显式标注「延后」及承接变更。

## 1. 数据与迁移

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| DB-01 | `media_assets` 字段、索引、MIME/大小 CHECK 与设计一致 | `0005_*.sql`；迁移测试 | ☐ |
| DB-02 | `chronicles` 字段、`version`、owner 索引、2,000 字 CHECK 与设计一致 | `0005_*.sql`；迁移测试 | ☐ |
| DB-03 | 活跃主图唯一 `(entity_type, entity_id)`；软删后可再传 | 集成用例 | ☐ |
| DB-04 | 0005 Down 删除两表并恢复 `characters.image_path`，数据不残留 | `TestMigrationRoundTrip` | ☐ |
| DB-05 | 软删除三件套 CHECK 对两表生效（禁止半删状态） | 集成约束用例 | ☐ |

## 2. 图片上传与读取

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| IMG-01 | ≤10MB jpg/png/webp 上传 201，返回 `asset_id/url/thumb_url/width/height/byte_size` | 集成 + E2E | ☐ |
| IMG-02 | 服务端校验 MIME + 魔数；伪造扩展名/文本/GIF 返回 422 `UNSUPPORTED_FILE_TYPE` | 单元 + 集成 | ☐ |
| IMG-03 | >10MB 返回 422 `FILE_TOO_LARGE`，无文件/行写入 | 单元 + 集成 | ☐ |
| IMG-04 | 像素 >40MP 返回 422 `IMAGE_TOO_LARGE`；最长边 >4096 等比缩小 | 单元 | ☐ |
| IMG-05 | JPEG/PNG 重编码；WebP 解码后转 PNG；落库 MIME 为 jpeg/png | 单元 + 集成 | ☐ |
| IMG-06 | 缩略图最长边 320、保持比例；详情/列表使用正确 variant | 单元 + 集成 | ☐ |
| IMG-07 | `GET /media/{id}?variant=thumb|original` 返回正确 Content-Type、`Cache-Control`、`nosniff`、inline | 集成 | ☐ |
| IMG-08 | `ETag` 稳定，`If-None-Match` 返回 304 | 集成 | ☐ |
| IMG-09 | 替换主图：旧资产软删除、旧文件删除；新图立即可读 | 集成 | ☐ |
| IMG-10 | 移除主图：204；无图再删 404；人物保留且 `image=null` | 集成 + E2E | ☐ |
| IMG-11 | 详情 `image`、列表 `image_thumb_url` 返回真实值，无图返回 `null` | 集成 | ☐ |
| IMG-12 | 路径只由 DB 构造，越界/`..` 被拒绝，不泄露绝对路径 | 单元 | ☐ |

## 3. 大事记

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| CHR-01 | 列表按 `sort_order,id` 返回；无条目返回空数组 | 集成 | ☐ |
| CHR-02 | 创建追加末尾；文本必填、1～2,000 字；时间标签可空、≤200 | 单元 + 集成 | ☐ |
| CHR-03 | 编辑携带 `If-Match`；版本不一致 409 `VERSION_CONFLICT`；`time_label=null` 可清空 | 集成 | ☐ |
| CHR-04 | 删除软删除并返回 `undo_token`；8 秒内撤销恢复；重复撤销 `ALREADY_DELETED`；过期 `UNDO_EXPIRED` | 集成 + E2E | ☐ |
| CHR-05 | owner 不存在/已删返回 404 或 `PARENT_DELETED`；跨作品不可读写 | 集成 | ☐ |
| CHR-06 | 大事记不出现在回收站列表；`root_type=chronicle` 仍为保留未启用 | 集成 | ☐ |

## 4. 生命周期联动

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| LIFE-01 | 人物软删除后图片/大事记隐藏；恢复后原样回来 | 集成 | ☐ |
| LIFE-02 | 人物永久删除后 `media_assets`/`chronicles` 行与文件均清理 | 集成 | ☐ |
| LIFE-03 | 作品整组软删除/恢复覆盖两表（`deleted_root_id=work_id`） | 集成 | ☐ |
| LIFE-04 | 作品永久删除/清空回收站后文件与行清理 | 集成 | ☐ |
| LIFE-05 | 永久删除前快照包含 `images/`，失败阻止删除 | 集成/回归 | ☐ |
| LIFE-06 | 并发：上传替换唯一索引、Chronicle 创建与人物删除不产生孤儿 | `-race` 集成 | ☐ |

## 5. UI 与交互

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| UI-01 | 详情 120px 主图；缺失显示姓名首字 + 雾绿渐变，无破图 | E2E + 视觉 | ☐ |
| UI-02 | 列表 40px 缩略图，`loading="lazy"`，行高稳定 | E2E + 视觉 | ☐ |
| UI-03 | 编辑态选择图片本地预览；替换/移除需确认；取消编辑不上传 | E2E | ☐ |
| UI-04 | 上传中显示百分比；失败字段级错误并保留其他输入；可重试 | 组件测试 | ☐ |
| UI-05 | 资料保存成功但图片失败时提示且停留编辑态 | E2E | ☐ |
| UI-06 | 大事记空态、添加/编辑/删除、序号与 `pre-wrap` 展示 | E2E | ☐ |
| UI-07 | 2,001 字阻止保存、字段级错误并聚焦 | 组件测试 + E2E | ☐ |
| UI-08 | 删除大事记行内确认 + Toast 撤销 | E2E | ☐ |
| UI-09 | 慢网下文字先渲染，图片后到，无布局抖动 | E2E（route 延迟） | ☐ |
| UI-10 | 1280/1024 布局无横向溢出；图片不变形 | E2E | ☐ |

## 6. 可访问性

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| A11Y-01 | 主图 `alt="人物主图：{名}"`；占位同 alt 且 `role="img"` | 组件测试 | ☐ |
| A11Y-02 | 上传/移除/大事记按钮键盘可达，焦点可见；Dialog 焦点陷阱与归还 | E2E + axe | ☐ |
| A11Y-03 | 大事记使用列表语义；Toast `aria-live`；无 serious/critical | axe | ☐ |

## 7. 测试与证据

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| TEST-01 | 后端 gofmt/vet/build/单测全绿 | `verification.md` §1 | ☐ |
| TEST-02 | 集成用例覆盖迁移/图片/大事记/生命周期/并发 | `verification.md` §2 | ☐ |
| TEST-03 | 前端 typecheck/lint/单测/Node 22 构建通过 | `verification.md` §1 | ☐ |
| TEST-04 | 全量 E2E 通过，含新增图片/大事记/慢网用例 | `verification.md` §3 | ☐ |
| TEST-05 | 视觉基线更新（列表/详情含图与占位） | `visual.spec.ts-snapshots/` | ☐ |

## 8. 文档

| ID | 验收点 | 证据 | 状态 |
|---|---|---|---|
| DOC-01 | SPEC-CHAR-006/007 状态与缺口在 F10 更新 | `project/specs/characters/specs.md` | ☐ |
| DOC-02 | `api-contract.md` 回写图片/大事记最终行为（MIME、`root_type=chronicle` 保留、version） | `project/api-contract.md` | ☐ |
| DOC-03 | `data-model.md` 回写 `chronicles.version`、`media_assets.mime` 实际值、移除 `image_path` | `project/data-model.md` | ☐ |
| DOC-04 | verification/review/acceptance 齐全并含证据 | `changes/CHG-005-.../` | ☐ |
