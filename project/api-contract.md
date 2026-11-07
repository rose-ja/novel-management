# API 契约（API Contract）

| 项目 | 内容 |
|---|---|
| 文档版本 | v1.0 |
| 状态 | Approved（2026-11-07，项目发起人） |
| 日期 | 2026-11-07 |
| 实现技术栈 | Go 1.24 + gin + gorm；PostgreSQL 16；Redis 7（仅缓存） |
| 基础路径 | `/api/v1` |
| 协议 | HTTP/1.1，JSON（`application/json`）；上传 `multipart/form-data` |
| 认证 | **无登录、无 Token、无 Cookie**；仅本地回环访问 |
| 上游 | [requirements.md](file:///d:/Project/novel-management/project/requirements.md)、[architecture.md](file:///d:/Project/novel-management/project/architecture.md)、[data-model.md](file:///d:/Project/novel-management/project/data-model.md)、PG-00~PG-13 |

> 契约中的字段名以本文件为准；数据库列见 data-model.md。错误码为稳定标识，中文 `message` 可调整文案但不得改变语义。

---

## 1. 通用约定

### 1.1 请求头

| Header | 必填 | 说明 |
|---|---|---|
| `Host` | 是 | 必须为 `127.0.0.1:17860` / `localhost:17860`；否则 `403 HOST_NOT_ALLOWED` |
| `Origin` / `Referer` | 写请求 | 存在时必须同源；缺失（curl/CLI）允许；跨源 `403 FORBIDDEN_ORIGIN` |
| `X-Requested-With` | 写请求 | 前端固定发送 `XMLHttpRequest`，作为本地 CSRF 的第二道检查 |
| `If-Match` | 更新/删除 | 乐观锁版本，如 `If-Match: "3"`；缺失或过期返回 `409 VERSION_CONFLICT` |
| `Content-Type` | 是 | JSON 或 multipart；不支持的类型 `415 UNSUPPORTED_MEDIA_TYPE` |

- 无 CORS 头；服务端不返回 `Access-Control-Allow-*`。
- 不使用 Cookie/Session；无 `Authorization` 头语义。

### 1.2 响应约定

- 成功：`2xx` + JSON；创建返回 `201` 与 `Location` 头；删除返回 `200`（软删除，带 `undo_token`）或 `204`（无 body 的永久操作）。
- 所有实体响应包含：`id`、`created_at`、`updated_at`、`version`、`deleted_at`（未删为 `null`）。
- 时间格式：ISO-8601 UTC，如 `2026-11-07T02:15:30Z`。
- ID：26 位 ULID 字符串。
- 列表响应：

```json
{
  "items": [ ... ],
  "page": 1,
  "page_size": 20,
  "total": 125,
  "total_pages": 7
}
```

- 分页：`page` 默认 1；`page_size` 默认 20、最大 100（检索固定 20，FR-30）。
- 排序：`sort` 取值白名单（`updated_at:desc` 默认、`name:asc`）；非法值 `422`。
- 空值：可选字段用 `null`，不省略键（便于前端类型稳定）。
- 缓存提示：`GET /search` 与作品摘要响应带 `X-Cache: HIT|MISS|BYPASS`；`BYPASS` 表示 Redis 不可用或未启用，不影响响应正确性（QA-07）。
- 请求追踪：所有响应带 `X-Request-Id`，与错误体 `request_id` 一致。

### 1.3 错误响应

```json
{
  "error": {
    "code": "REFERENCE_CONFLICT",
    "message": "该人物仍被 2 处引用，无法删除",
    "details": [
      { "type": "faction_leader", "entity_id": "...", "name": "山海妖族", "field": "leader" },
      { "type": "ranking_seat", "entity_id": "...", "name": "封神榜", "field": "seat_no", "value": 7 }
    ],
    "request_id": "req_01H..."
  }
}
```

- `details` 可选；引用类错误必须给出可定位信息（FR-38）。
- 校验错误：

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "请求字段校验失败",
    "details": [{ "field": "name", "rule": "required", "message": "主名称不能为空" }],
    "request_id": "req_..."
  }
}
```

### 1.4 错误码表

| Code | HTTP | 含义 | 典型触发 |
|---|---|---|---|
| `VALIDATION_ERROR` | 422 | 字段格式/长度/必填失败 | 空名称、超长、非法枚举 |
| `NOT_FOUND` | 404 | 实体不存在或已永久删除 | 非法 ID |
| `NAME_CONFLICT` | 409 | 同作品同类型名称重复（`name_key` 冲突） | 新建/改名/恢复/导入 |
| `REFERENCE_CONFLICT` | 409 | 仍被其他实体引用，不能删除/停用/缩容 | 删除人物/势力/地点/类别/节点 |
| `HAS_CHILDREN` | 409 | 存在下级节点（结构约束） | 删除势力/地点；删除体系主节点（C-01 默认阻止） |
| `CYCLE_DETECTED` | 400 | 父级形成环 | 移动势力/地点/节点 |
| `CROSS_WORK_REFERENCE` | 400 | 引用其他作品实体 | 人物→势力/地点/节点等 |
| `CATEGORY_LIMIT` | 409 | 类别已达 10 类 | 新建类别 |
| `CATEGORY_BUILTIN_IMMUTABLE` | 409 | 内置「境界」不可改名/停用/删除 | 对应操作 |
| `CATEGORY_DISABLED` | 409 | 引用已停用类别 | 人物保存体系引用 |
| `NODE_LEVEL_LIMIT` | 409 | 体系节点超过两级 | 新建子节点于子节点下 |
| `NODE_PARENT_MISMATCH` | 400 | 节点与类别/父级不匹配 | 体系引用写入 |
| `SEAT_CONFLICT` | 409 | 同一人物同榜已占其他席位 | 指派/交换 |
| `SEAT_OUT_OF_RANGE` | 400 | 席位号超出 1..N | 指派/扩容/缩容 |
| `SEAT_SHRINK_OCCUPIED` | 409 | 缩容目标尾部席位有占用 | 减少席位数 |
| `DUPLICATE_RELATION` | 409 | 同一对关系重复创建 | 地点-势力、人物-地点关联 |
| `VERSION_CONFLICT` | 409 | `If-Match` 版本不一致 | 并发编辑 |
| `PARENT_DELETED` | 409 | 恢复时父级仍在回收站 | 回收站恢复 |
| `REFERENCE_MISSING` | 409 | 恢复时引用对象缺失 | 回收站恢复 |
| `ALREADY_DELETED` | 409 | 重复删除/撤销已过期 | 二次删除、undo |
| `UNDO_EXPIRED` | 409 | 撤销窗口已过 | 超过 8 秒 |
| `SNAPSHOT_FAILED` | 503 | 快照生成失败，操作未执行 | 删除作品/清空回收站/导入/恢复 |
| `IMPORT_INVALID` | 400 | 导入模板/字段/引用/图片错误 | 预览阶段 |
| `IMPORT_STATE_CONFLICT` | 409 | 状态不允许该操作 | 重复 commit、未校验即提交 |
| `IMPORT_TEMPLATE_VERSION` | 400 | 模板版本缺失或不支持 | 上传解析 |
| `UNSUPPORTED_FILE_TYPE` | 415 | 图片/上传文件类型不支持 | 图片上传、zip |
| `FILE_TOO_LARGE` | 413 | 超过单文件/总量上限 | 图片 >10MB、zip 超限 |
| `PATH_TRAVERSAL` | 400 | 压缩包含越界路径 | 导入 zip |
| `EXPORT_IN_PROGRESS` | 409 | 已有导出任务运行 | 重复创建导出 |
| `IMPORT_IN_PROGRESS` | 409 | 已有导入任务运行 | 重复上传 |
| `HOST_NOT_ALLOWED` | 403 | Host 不在白名单 | 非本机访问 |
| `FORBIDDEN_ORIGIN` | 403 | 写请求跨源 | 恶意网页 CSRF |
| `OFFLINE` | 503 | 服务不可用/正在恢复 | 启动锁、容器停止 |
| `INTERNAL_ERROR` | 500 | 未预期错误 | 兜底；不含堆栈 |

### 1.5 认证与授权

- 本版无认证、无角色、无权限判断；所有端点对本地调用者开放（requirements §2.3）。
- 授权边界由网络层承担：仅 `127.0.0.1` 监听 + Host/Origin 校验 + 无 CORS + CSP。
- “权限不足”记为 `N/A`（EX-05）；未来公网部署需新增反代认证与访问控制，并新增 ADR。

### 1.6 幂等与并发

- **创建**：同一名称重复提交由部分唯一索引兜底，第二次返回 `409 NAME_CONFLICT`（EX-04）；不引入通用 `Idempotency-Key`。
- **更新/删除**：`If-Match` 版本号实现乐观锁；版本不匹配返回 `409 VERSION_CONFLICT`，不自动合并（EX-07）。
- **撤销**：`undo_token` 一次性；重复使用返回 `409 ALREADY_DELETED`/`UNDO_EXPIRED`。
- **导入提交**：`import_id` 状态机保证一个导入只成功提交一次；重复 commit 返回同一结果（200）。
- **导出**：同一时间仅允许一个 `running` 导出；重复创建返回 `409 EXPORT_IN_PROGRESS`。
- **删除**：重复 DELETE 返回 `409 ALREADY_DELETED`（根已被删除且仍在回收站）。

### 1.7 兼容策略

- 路径版本 `/api/v1`；新增字段只能追加且可选；删除/改名字段必须升 `/api/v2`。
- 枚举新增值视为兼容；前端必须容忍未知枚举并显示原文。
- 导出包 `export_version=1`；恢复至少兼容上一版本（data-model §7）。
- OpenAPI：仓库内 `openapi.yaml` 为唯一契约源；后端用 `oapi-codegen` 生成类型与路由绑定，前端用 `openapi-typescript` 生成类型。CI 校验生成物与 YAML 一致。
- OpenAPI 文档：`/api/openapi.json`（只读静态文件）；生产镜像默认不暴露 `/docs`。

---

## 2. 实体 DTO

### 2.1 Work

```json
{
  "id": "01H...", "name": "山海", "description": "…",
  "counts": { "characters": 1000, "factions": 200, "locations": 200, "rankings": 5 },
  "created_at": "…", "updated_at": "…", "version": 3, "deleted_at": null
}
```

### 2.2 Character

```json
{
  "id": "01H...", "work_id": "01H...",
  "name": "司空摘天",
  "names": [
    { "id": "01H...", "kind": "alias", "value": "神逆", "sort_order": 0 },
    { "id": "01H...", "kind": "title", "value": "山海帝君", "sort_order": 0 }
  ],
  "description": "…",
  "faction": { "id": "01H...", "name": "山海妖族" },
  "system_refs": [
    { "category_id": "01H...", "category_name": "境界", "node": { "id": "01H...", "name": "帝境" }, "child_node": { "id": "01H...", "name": "五重天" } }
  ],
  "location_links": [
    { "location_id": "01H...", "name": "东海", "note": "常驻" }
  ],
  "image": { "asset_id": "01H...", "url": "/api/v1/media/01H...?variant=thumb", "width": 800, "height": 1200 },
  "review_status": "reviewed",
  "created_at": "…", "updated_at": "…", "version": 2, "deleted_at": null
}
```

### 2.3 Faction / Location

```json
{
  "id": "01H...", "work_id": "01H...", "name": "山海妖族",
  "parent": { "id": "01H...", "name": "万族" },
  "leader": { "id": "01H...", "name": "司空摘天" },
  "description": "…", "review_status": "reviewed",
  "children_count": 3, "member_count": 12,
  "created_at": "…", "updated_at": "…", "version": 1, "deleted_at": null
}
```

地点额外字段：`type_text`；`characters_count`、`factions_count`。

### 2.4 Ranking / Seat

```json
{
  "id": "01H...", "work_id": "01H...", "name": "封神榜",
  "seat_count": 100, "occupied_count": 23, "empty_count": 77,
  "review_status": "reviewed",
  "created_at": "…", "updated_at": "…", "version": 1, "deleted_at": null
}
```

```json
{ "seat_no": 7, "character": { "id": "01H...", "name": "司空摘天" }, "note": "并列", "version": 2 }
```

### 2.5 SystemCategory / SystemNode

```json
{ "id": "01H...", "work_id": "01H...", "name": "境界", "category_type": "normal", "is_enabled": true, "is_builtin": true, "sort_order": 0, "node_count": 15, "reference_count": 980 }
```

```json
{ "id": "01H...", "category_id": "01H...", "name": "帝境", "parent": null, "sort_order": 0, "era_start_text": null, "era_end_text": null, "reference_count": 120, "children": [ { "id": "01H...", "name": "五重天", "sort_order": 0 } ] }
```

### 2.6 Chronicle

```json
{ "id": "01H...", "owner_type": "character", "owner_id": "01H...", "text": "…", "time_label": "帝元 300 年", "sort_order": 0, "created_at": "…", "updated_at": "…", "deleted_at": null }
```

### 2.7 检索结果 SearchResult

```json
{
  "groups": [
    {
      "entity_type": "character",
      "total": 12,
      "items": [
        {
          "entity_id": "01H...", "entity_type": "character", "title": "司空摘天",
          "subtitle": "人物 · 山海妖族 · 帝境五重天",
          "matched_fields": ["name"],
          "snippet": "司空摘天（神逆）（山海帝君）",
          "score": 100,
          "updated_at": "…",
          "url": "/characters/01H..."
        }
      ]
    }
  ],
  "total": 12,
  "page": 1,
  "page_size": 20,
  "query": { "q": "山海 帝君", "tokens": ["山海", "帝君"], "filters": { "entity_types": ["character"] } }
}
```

---

## 3. 端点清单

### 3.1 系统与设置

#### GET /api/v1/health
- Purpose：容器、数据库与缓存健康检查。
- Request：无。
- Response `200`：`{ "status": "ok|degraded", "version": "1.0.0", "db": "ok", "cache": "ok|degraded|disabled", "data_dir_writable": true }`。
- 语义：`db` 不可用时 `status=degraded`（HTTP `503`）；`cache` 不可用时 `status=ok`、`cache=degraded`，检索自动旁路（QA-07）。
- Errors：`503 OFFLINE`。
- Tests：启动 smoke；DB 只读、Redis 停止两种降级路径。

#### GET /api/v1/meta
- Response：`{ "app_version", "api_version": "v1", "export_versions": [1], "template_version": "1.0", "cache_enabled": true, "cache_ttl_sec": 60, "limits": { "name":200, "description":20000, "chronicle":2000, "relation_note":500, "era_text":100, "seat_max":200, "category_max":10, "upload_image_mb":10 } }`。
- Purpose：前端显示与校验常量；不替代后端校验。`cache_enabled` 只用于状态展示，前端不依赖缓存。

#### GET /api/v1/settings
- Response：`{ "current_work_id": "01H...|null", "data_dir": "/data", "snapshot_keep": 3, "last_export_at": "…|null", "last_restore_at": "…|null", "app_version": "…" }`。
- Note：`data_dir` 只返回逻辑路径，不回显宿主机绝对路径。

#### PATCH /api/v1/settings
- Request：`{"current_work_id": "01H..."}`。
- Response：同 GET。
- Errors：`404 NOT_FOUND`（作品不存在）。
- Idempotency：重复设置同一作品返回 200。

### 3.2 作品 Works（FR-01~03）

#### GET /api/v1/works
- Query：`q`（可选，名称模糊）、`page`、`page_size`。
- Response：`Work[]` 分页。
- 仅未删除作品。

#### POST /api/v1/works
- Request：`{ "name": "山海", "description": "…|null" }`。
- Response `201`：`Work`；自动创建内置「境界」类别（`is_builtin=true`、`is_enabled=true`、`category_type=normal`）。
- Errors：`VALIDATION_ERROR`、`NAME_CONFLICT`。
- 软上限：超过 10 本时响应含 `warnings: [{code:"WORK_SOFT_LIMIT", message:"…"}]`，不阻断（C-04）。

#### GET /api/v1/works/{work_id}
- Response：`Work`。
- Errors：`NOT_FOUND`。

#### PATCH /api/v1/works/{work_id}
- Request：`{ "name": "…", "description": "…|null" }`；`If-Match` 必填。
- Response：`Work`。
- Errors：`NAME_CONFLICT`、`VERSION_CONFLICT`。

#### DELETE /api/v1/works/{work_id}
- Request：`{ "confirm_name": "山海" }`（必须与作品名完全一致）。
- Behavior：生成快照 → 整本软删除（全部实体同根）→ 返回 `undo_token`（8 秒）。
- Response：`{ "deleted": true, "undo_token": "…", "snapshot_id": "…" }`。
- Errors：`VALIDATION_ERROR`（名称不符）、`SNAPSHOT_FAILED`、`ALREADY_DELETED`。

### 3.3 人物 Characters（FR-04~08）

#### GET /api/v1/characters
- Query：`work_id`（必填）、`q`、`faction_id`、`system_node_id`、`location_id`、`review_status`、`sort`、`page`、`page_size`。
- Response：`CharacterSummary[]`：`id, name, faction:{id,name}, system_summary:string, image_thumb_url, review_status, updated_at`。
- 仅未删除；`work_id` 缺失 `422`。

#### POST /api/v1/characters
- Request：
  ```json
  {
    "work_id": "01H...",
    "name": "司空摘天",
    "names": [{ "kind": "alias", "value": "神逆" }],
    "description": null,
    "faction_id": "01H...|null",
    "system_refs": [{ "category_id": "01H...", "node_id": "01H..." }],
    "location_links": [{ "location_id": "01H...", "note": "常驻" }]
  }
  ```
- Response `201`：`Character`。
- Validation：名称必填/≤200；别名/称号 ≤20 软上限；`system_refs` 每类别至多 1，节点必须属于类别且类别启用；全部引用同作品。
- Errors：`NAME_CONFLICT`、`CROSS_WORK_REFERENCE`、`CATEGORY_DISABLED`、`NODE_PARENT_MISMATCH`。

#### GET /api/v1/characters/{id}
- Response：`Character`（含 `relations` 摘要：势力、地点、榜单、体系、大事记计数）。
- Errors：`NOT_FOUND`。

#### PATCH /api/v1/characters/{id}
- Request：任意字段子集（`name`、`names`、`description`、`faction_id`、`system_refs`、`location_links`）；`If-Match` 必填。
- Response：`Character`。
- Behavior：`names` 为全量替换；`system_refs` 为全量替换；`location_links` 为全量替换。
- Errors：`NAME_CONFLICT`、`VERSION_CONFLICT`、`CROSS_WORK_REFERENCE`。

#### DELETE /api/v1/characters/{id}
- Behavior：引用检查：被势力设为首领、被榜单占用 → 阻止并列出引用；人物自身的别名/称号、体系引用、地点关系、大事记、图片随人物软删除（不阻止）。
- Response：`{ "deleted": true, "undo_token": "…" }`。
- Errors：`REFERENCE_CONFLICT`（`details` 列出首领/席位）、`ALREADY_DELETED`。

#### POST /api/v1/characters/{id}/image
- Request：multipart `file`；jpg/png/webp，≤10MB。
- Response `201`：`{ "asset_id", "url", "thumb_url", "width", "height", "byte_size" }`。
- Behavior：校验 MIME + 魔数；按内容重编码；缩略图（最长边 320px）可异步生成，原图先返回。
- Errors：`UNSUPPORTED_FILE_TYPE`、`FILE_TOO_LARGE`、`VALIDATION_ERROR`。
- Idempotency：重复上传替换旧图（旧资产软删除）。

#### DELETE /api/v1/characters/{id}/image
- Response `204`；无图返回 `404 NOT_FOUND`。

#### GET /api/v1/media/{asset_id}
- Query：`variant=thumb|original`（默认 `thumb`）。
- Response：二进制图片；`Cache-Control: private, max-age=31536000, immutable`；`ETag`。
- 安全：仅未删除资产；`Content-Disposition: inline`；`nosniff`。

### 3.4 势力 Factions（FR-09~12）

#### GET /api/v1/factions
- Query：`work_id`、`q`、`parent_id`（`root` 表示顶层）、`flat=true|false`、`page`、`page_size`。
- Response：`FactionSummary[]`（含 `children_count`、`leader`、`member_count`）。

#### POST /api/v1/factions
- Request：`{ "work_id", "name", "parent_id": null, "leader_character_id": null, "description": null }`。
- Response `201`：`Faction`。
- Errors：`NAME_CONFLICT`、`CYCLE_DETECTED`、`CROSS_WORK_REFERENCE`。

#### GET /api/v1/factions/{id}
- Response：`Faction` + `relations`：`parent`、`children`、`leader`、`locations`、`chronicle_count`。
- Errors：`NOT_FOUND`。

#### PATCH /api/v1/factions/{id}
- Request：`name`、`parent_id`、`leader_character_id`、`description`；`If-Match`。
- 移动校验：目标父级不能是自身或后代（`CYCLE_DETECTED`）。
- Errors：`NAME_CONFLICT`、`CYCLE_DETECTED`、`CROSS_WORK_REFERENCE`、`VERSION_CONFLICT`。

#### DELETE /api/v1/factions/{id}
- 校验：先检查下级（`HAS_CHILDREN`），再检查人物归属引用（`REFERENCE_CONFLICT`）；首领、地点关联不阻止，随势力删除隐藏。
- Response：`{ "deleted": true, "undo_token": "…" }`。
- Errors：`HAS_CHILDREN`、`REFERENCE_CONFLICT`。

#### GET /api/v1/factions/{id}/members
- Query：`scope=direct|subtree`（默认 `direct`）、`page`、`page_size`。
- Response：`CharacterSummary[]`；`subtree` 递归包含下级节点成员（FR-10）。

### 3.5 地点 Locations（FR-13~16）

#### GET /api/v1/locations
- Query：`work_id`、`q`、`parent_id`（`root`）、`type_text`、`flat`、`page`、`page_size`。
- Response：`LocationSummary[]`。

#### POST /api/v1/locations
- Request：`{ "work_id", "name", "parent_id": null, "type_text": null, "description": null }`。
- Response `201`：`Location`。
- Errors：`NAME_CONFLICT`、`CYCLE_DETECTED`、`CROSS_WORK_REFERENCE`。

#### GET /api/v1/locations/{id}
- Response：`Location` + `relations`：`parent`、`children`、`characters`、`factions`。
- Errors：`NOT_FOUND`。

#### PATCH /api/v1/locations/{id}
- 字段：`name`、`parent_id`、`type_text`、`description`；`If-Match`。
- Errors：`NAME_CONFLICT`、`CYCLE_DETECTED`、`VERSION_CONFLICT`。

#### DELETE /api/v1/locations/{id}
- 校验：`HAS_CHILDREN` → `REFERENCE_CONFLICT`（人物/势力关联不阻止？按 FR-15 有下级或被引用时阻止；地点被引用需阻止）→ 软删除。
- Response：`{ "deleted": true, "undo_token": "…" }`。

#### GET /api/v1/locations/{id}/characters
- Response：`{ "items": [{ "character_id", "name", "note" }] }`。

#### GET /api/v1/locations/{id}/factions
- Response：`{ "items": [{ "faction_id", "name", "note" }] }`。

#### POST /api/v1/locations/{id}/factions
- Request：`{ "faction_id": "01H...", "note": null }`。
- Response `201`；重复关联 `409 DUPLICATE_RELATION`。
- DELETE /api/v1/locations/{id}/factions/{faction_id}：`204`。

> 人物↔地点关系通过 `PATCH /characters/{id}` 的 `location_links` 全量替换维护，避免双入口（FR-14）。

### 3.6 榜单 Rankings（FR-17~20）

#### GET /api/v1/rankings
- Query：`work_id`、`page`、`page_size`。
- Response：`RankingSummary[]`（含 `occupied_count`、`empty_count`）。

#### POST /api/v1/rankings
- Request：`{ "work_id", "name", "seat_count": 100 }`。
- Response `201`：`Ranking`；同时生成 1..N 空席位。
- Errors：`NAME_CONFLICT`、`VALIDATION_ERROR`（1–200）。

#### GET /api/v1/rankings/{id}
- Query：`page`、`page_size`（席位分页，默认 20，可 50/100）。
- Response：`Ranking` + `seats` 分页 + `summary`。
- Errors：`NOT_FOUND`。

#### PATCH /api/v1/rankings/{id}
- Request：`{ "name": "…", "seat_count": 150 }`；`If-Match`。
- 扩容：追加空席位；缩容：仅允许尾部连续空席，否则 `SEAT_SHRINK_OCCUPIED`（`details` 列占用人）。
- Errors：`NAME_CONFLICT`、`SEAT_SHRINK_OCCUPIED`、`SEAT_OUT_OF_RANGE`、`VERSION_CONFLICT`。

#### DELETE /api/v1/rankings/{id}
- 行为：整榜 + 席位软删除；不检查席位占用（FR-20 允许整榜进回收站）。
- Response：`{ "deleted": true, "undo_token": "…" }`。

#### GET /api/v1/rankings/{id}/seats
- Query：`page`、`page_size`、`occupied=true|false`。
- Response：`{ "items": [{ "seat_no", "character": {...}|null, "note" }], "page", ... }`。

#### PUT /api/v1/rankings/{id}/seats/{seat_no}
- Request：`{ "character_id": "01H...|null", "note": "并列|null" }`；`If-Match` 为席位 `version`。
- Behavior：`character_id=null` 清空；同一人物同榜已有席位 → `SEAT_CONFLICT`。
- Response：席位（含新 `version`）。
- Errors：`SEAT_CONFLICT`、`SEAT_OUT_OF_RANGE`、`CROSS_WORK_REFERENCE`、`VERSION_CONFLICT`。

#### POST /api/v1/rankings/{id}/seats/{seat_no}/move
- Request：`{ "target_seat_no": 12, "mode": "empty|swap" }`；`If-Match` 为源席位 `version`。
- Behavior：`empty` 要求目标为空；`swap` 交换两人；目标越界 `SEAT_OUT_OF_RANGE`。
- Response：`{ "from": {...}, "to": {...} }`。
- Errors：`SEAT_CONFLICT`（目标非空且 mode=empty）、`SEAT_OUT_OF_RANGE`、`VERSION_CONFLICT`。

### 3.7 体系 System（FR-21~26）

#### GET /api/v1/system-categories
- Query：`work_id`、`include_disabled=true|false`。
- Response：`SystemCategory[]`（含 `node_count`、`reference_count`）。

#### POST /api/v1/system-categories
- Request：`{ "work_id", "name": "圣体", "category_type": "normal|era", "is_enabled": true }`。
- Response `201`；`is_builtin` 恒为 false（境界由作品创建时自动生成）。
- Errors：`CATEGORY_LIMIT`、`NAME_CONFLICT`、`VALIDATION_ERROR`。

#### PATCH /api/v1/system-categories/{id}
- Request：`{ "name": "…", "is_enabled": true|false }`（`category_type` 不可改）；`If-Match`。
- 停用校验：必须无引用，否则 `REFERENCE_CONFLICT`（`details` 含引用数量与位置）。
- 内置类别：改名/停用 → `CATEGORY_BUILTIN_IMMUTABLE`。
- Errors：`NAME_CONFLICT`、`REFERENCE_CONFLICT`、`CATEGORY_BUILTIN_IMMUTABLE`、`VERSION_CONFLICT`。

#### DELETE /api/v1/system-categories/{id}
- Behavior：引用检查 → 无引用则类别 + 全部节点整组软删除（`deleted_root_id=category_id`）。
- 内置类别 → `CATEGORY_BUILTIN_IMMUTABLE`。
- Response：`{ "deleted": true, "undo_token": "…", "affected_nodes": 8 }`。
- Errors：`REFERENCE_CONFLICT`、`CATEGORY_BUILTIN_IMMUTABLE`。

#### GET /api/v1/system-categories/{id}/nodes
- Query：`include_disabled`、`flat`。
- Response：`SystemNode[]`（树形，含 `children`、`reference_count`）。

#### POST /api/v1/system-nodes
- Request：`{ "category_id", "parent_id": null, "name", "era_start_text": null, "era_end_text": null }`。
- 校验：类别启用；父级同类别且为根节点（两级上限）；era 字段仅 `era` 类别。
- Response `201`：`SystemNode`。
- Errors：`NODE_LEVEL_LIMIT`、`NODE_PARENT_MISMATCH`、`NAME_CONFLICT`、`CATEGORY_DISABLED`。

#### PATCH /api/v1/system-nodes/{id}
- Request：`name`、`sort_order`、`era_start_text`、`era_end_text`；`If-Match`。
- 改名同步由引用 join 自动生效（FR-25）；`parent_id` 不在 PATCH 中修改。
- Errors：`NAME_CONFLICT`、`VERSION_CONFLICT`。

#### DELETE /api/v1/system-nodes/{id}
- 校验：引用（人物引用）→ `REFERENCE_CONFLICT`；含子节点 → `HAS_CHILDREN`（C-01 默认）。
- Response：`{ "deleted": true, "undo_token": "…" }`。

#### POST /api/v1/system-nodes/reorder
- Request：`{ "category_id": "01H...", "parent_id": "01H...|null", "ordered_ids": ["…"] }`。
- 校验：集合必须完整且同父级；返回 `204`。
- Errors：`VALIDATION_ERROR`。

### 3.8 大事记 Chronicles（FR-04/11）

#### GET /api/v1/chronicles
- Query：`owner_type=character|faction`、`owner_id`。
- Response：`Chronicle[]`，按 `sort_order`。

#### POST /api/v1/chronicles
- Request：`{ "owner_type", "owner_id", "text", "time_label": null }`；`sort_order` 追加末尾。
- Response `201`；文本来空 `VALIDATION_ERROR`。
- Errors：`NOT_FOUND`（owner 不存在/已删）。

#### PATCH /api/v1/chronicles/{id}
- Request：`text`、`time_label`、`sort_order`；`If-Match`。
- Errors：`VALIDATION_ERROR`、`VERSION_CONFLICT`。

#### DELETE /api/v1/chronicles/{id}
- 软删除单条，返回 `{ "deleted": true, "undo_token": "…" }`。

### 3.9 检索 Search（FR-27~33）

#### GET /api/v1/search
- Query：
  - `work_id`（必填）
  - `q`（可选；为空且无筛选 → `400 VALIDATION_ERROR`，前端显示 FR-31 提示）
  - `entity_types`（逗号分隔：`character,faction,location,ranking,system_node`；默认全部）
  - `faction_id`、`system_node_id`、`location_id`、`ranking_id`（条件检索，AND）
  - `page`（固定 20/页）
- 语义：空格分词 AND；每个词需命中名称/别名/称号/简介/关联实体名；不搜大事记（FR-28）。
- 实现：PostgreSQL `search_documents` + `pg_trgm` GIN 索引；3 字符以上走 `ILIKE '%token%'`，1–2 字走同一匹配（预计顺序扫描，数据量 1500 行内可接受）。V-01 未通过前不冻结该实现（ADR-003）。
- Response：`SearchResult`（§2.7），按实体类型分组；`total` 为全部匹配数；响应头 `X-Cache`。
- 排序：名称 > 别名/称号 > 简介 > 关联实体名；同权重 `updated_at DESC`（FR-30）。
- 缓存：key 由 `work_id` + 检索参数哈希构成，TTL 60s；作品任何写操作递增 epoch 使旧键失效；Redis 故障时旁路。
- Errors：`VALIDATION_ERROR`（空查询/非法类型）。
- 性能：P95 ≤150ms（V-01）。

### 3.10 回收站 Trash（FR-57~63）

#### GET /api/v1/trash
- Query：`work_id`、`root_type=work|character|faction|location|ranking|system_category|system_node|chronicle`、`page`、`page_size`。
- Response：
  ```json
  { "items": [{ "root_type": "character", "root_id": "01H...", "name": "司空摘天", "deleted_at": "…", "parent_name": "山海妖族", "affected_count": 3 }], "page": 1, ... }
  ```
- 只返回根记录；不含已永久删除。

#### POST /api/v1/trash/{root_type}/{root_id}/restore
- Request：`{ "new_name": "…|null" }`（名称冲突时可改名）。
- Behavior：整组恢复；校验父级、引用、名称。
- Response：恢复后的实体摘要 + `restored_count`。
- Errors：`NAME_CONFLICT`、`PARENT_DELETED`、`REFERENCE_MISSING`、`NOT_FOUND`。

#### DELETE /api/v1/trash/{root_type}/{root_id}
- Request：`{ "confirm": true }`；生成快照（`trash_purge`）→ 物理删除整组。
- Response `204`。
- Errors：`VALIDATION_ERROR`、`SNAPSHOT_FAILED`。

#### DELETE /api/v1/trash
- Request：`{ "confirm": "PURGE_ALL", "work_id": "01H...|null" }`；清空当前作品或全部作品回收站；先快照。
- Response：`{ "purged": 12, "snapshot_id": "…" }`。
- Errors：`VALIDATION_ERROR`、`SNAPSHOT_FAILED`。

#### POST /api/v1/trash/undo
- Request：`{ "undo_token": "…" }`。
- Response：`{ "restored": true, "root_type": "…", "root_id": "…" }`。
- Errors：`UNDO_EXPIRED`、`ALREADY_DELETED`。

### 3.11 导入 Imports（FR-39~46）

#### GET /api/v1/imports/template
- Response：`text/markdown` 附件 `template-v1.0.md`；`Content-Disposition: attachment`。

#### GET /api/v1/imports/template/example
- Response：示例 zip（含 Markdown + images），用于用户对照整理。

#### POST /api/v1/imports
- Request：multipart：`markdown`（文件，≤20MB）、`archive`（zip，≤6GB 可配）。
- Response `202`：`{ "import_id": "01H...", "status": "uploaded" }`。
- 后台：`validating → validated|failed`。
- Errors：`IMPORT_IN_PROGRESS`、`UNSUPPORTED_FILE_TYPE`、`FILE_TOO_LARGE`、`PATH_TRAVERSAL`、`IMPORT_TEMPLATE_VERSION`。

#### GET /api/v1/imports/{import_id}
- Response：
  ```json
  {
    "import_id": "01H...", "status": "validated", "file_name": "…", "template_version": "1.0",
    "counts": { "characters": 1000, "factions": 200, "locations": 200, "rankings": 5, "system_nodes": 150, "images": 480 },
    "errors": [{ "line": 42, "field": "人物/司空摘天/体系引用", "code": "IMPORT_INVALID", "message": "类别「圣体」未定义" }],
    "warnings": [], "error_count": 1, "work_id": null, "snapshot_id": null, "created_at": "…", "finished_at": "…"
  }
  ```

#### POST /api/v1/imports/{import_id}/commit
- Request：`{ "work_name": "山海（导入）" }`（可选；默认取模板作品名）。
- Behavior：仅在 `status=validated` 且 `error_count=0` 时允许；快照 → 单事务写入 → `completed`。
- Response：`{ "status": "completed", "work_id": "01H...", "report": {...}, "review_url": "/characters?review_status=pending" }`。
- Errors：`IMPORT_STATE_CONFLICT`、`IMPORT_INVALID`、`SNAPSHOT_FAILED`、`NAME_CONFLICT`（作品名）。
- 幂等：重复调用返回同一 `completed` 结果。

#### GET /api/v1/imports/{import_id}/report
- Response：导入报告（只读）：文件名、作品、计数、错误数、耗时、校对清单、快照 ID。

### 3.12 导出 Exports（FR-52~53）

#### POST /api/v1/exports
- Request：`{ "include_trash": true }`。
- Response `202`：`{ "export_id": "01H...", "status": "queued" }`。
- Errors：`EXPORT_IN_PROGRESS`。
- 后台任务：`running → completed|failed`。
- 两种模式：
  - `include_trash=true`：直接 `pg_dump -Fc` 当前数据库，回收站记录随库导出。
  - `include_trash=false`：复制到临时数据库，删除软删除根与隐藏关系后 `pg_dump -Fc`，再删除临时库；导出包不含回收站数据。
  - 两种导出包均可用同一 CLI 恢复（AC-12）；`manifest.json` 记录 `includes_trash`。

#### GET /api/v1/exports
- Response：最近任务列表（`id, status, include_trash, byte_size, created_at, finished_at`）。

#### GET /api/v1/exports/{export_id}
- Response：任务详情 + `file_name`（如 `novel-settings-20261107-101530.zip`）+ `file_path`（逻辑路径）。

#### GET /api/v1/exports/{export_id}/download
- Response：`application/zip` 流式下载；`Content-Disposition: attachment; filename="…"`。
- Errors：`NOT_FOUND`（未完成/失败）。

### 3.13 快照 Snapshots（FR-55）

#### GET /api/v1/snapshots
- Response：`[{ "id", "reason", "created_at", "db_bytes", "image_count", "path" }]`，按时间倒序；最多展示 10 条，保留 3 份。
- 只读；不提供页内恢复（D-05）。

---

## 4. 契约测试与追溯

| 端点组 | Requirement | Page |
|---|---|---|
| `/works` | FR-01~03 | PG-00、PG-13 |
| `/characters`、`/media` | FR-04~08 | PG-02、PG-03 |
| `/factions`、`/chronicles` | FR-09~12 | PG-04、PG-05 |
| `/locations` | FR-13~16 | PG-06、PG-07 |
| `/rankings` | FR-17~20 | PG-08、PG-09 |
| `/system-categories`、`/system-nodes` | FR-21~26 | PG-10 |
| `/search` | FR-27~33 | PG-01 |
| `/imports` | FR-39~46 | PG-11 |
| `/trash` | FR-57~63 | PG-12 |
| `/exports`、`/snapshots`、`/settings` | FR-52~56、FR-65 | PG-13 |

- 每个端点至少一条契约测试（OpenAPI schema 校验）与一条行为测试；错误码必须有对应测试用例。
- 变更 API 必须同步本文件、data-model.md 与 OpenAPI，并更新 ADR（如涉及重大决策）。

---

## 5. 契约决策记录

| 编号 | 问题 | 当前契约 | 备选 |
|---|---|---|---|
| C-01 | 删除体系主节点含子节点 | `409 HAS_CHILDREN`（已确认 2026-11-07） | 整组软删除 + 返回 `affected_nodes` |
| C-02 | 回收站单项恢复是否快照 | 不快照（已确认 2026-11-07） | 恢复前快照（响应增加 `snapshot_id`） |
| C-03 | 字段长度上限 | `/meta` 常量（已确认 2026-11-07） | 产品确认后调整 |
| C-04 | 作品软上限 | `warnings[]` 不阻断（已确认 2026-11-07） | 硬限制 `409 WORK_LIMIT` |
| D-02 | 中文检索实现 | `pg_trgm` + 短词 `ILIKE`（已确认 2026-11-07；V-01 待执行） | `pg_bigm` / `zhparser` / 外部搜索 |
| D-08 | Redis 使用范围 | 仅缓存检索与摘要，故障旁路（已确认 2026-11-07） | 引入队列/会话（不采用） |
| D-09 | 前端交付方式 | Next.js 静态导出，由 Go 托管（已确认 2026-11-07） | Node 运行时 + Nginx |
| C-07 | 引用检查是否包含回收站记录 | 包含（已确认 2026-11-07） | 仅未删除记录 |
| — | 地点被人物/势力引用时删除是否阻止 | 阻止（FR-15 原文，已确认） | 允许级联解除关系 |
