# CHG-011 设计

- Change ID：`CHG-011-import`
- 依据：`proposal.md`、`project/specs/migration/specs.md`、`project/page-specs/PG-11-import-wizard.md`、`project/adr/ADR-005-atomic-import.md`、`project/api-contract.md` §3.11/§3.14、`project/data-model.md` §3.16
- 目标：不新增第三方依赖，交付模板导入、原子提交、报告与校对。

## 1. 设计决策

| ID | 决策 | 理由 |
|---|---|---|
| D1 | Markdown 与图片 zip 作为两个 multipart 部分 | api-contract/PG-11；避免单包内嵌 Markdown 歧义 |
| D2 | 上传后自动后台校验，同时保留 `POST /imports/{id}/validate` 重校验 | 兼容 api-contract 与 ADR-005/SPEC-MIG-003 |
| D3 | 校验错误不使任务 failed：`validated + error_count>0`；系统级解析/IO 错误才 `failed` | 支持错误清单预览与修正后重传 |
| D4 | 多余图片阻断导入 | 用户 2026-11-15 确认（PG11-Q6） |
| D5 | 提交前快照 `reason=import`，失败中止 | SPEC-BACKUP-003 / SPEC-MIG-004 |
| D6 | 预生成 work_id，图片先落盘 `images/{work_id}/`，DB 单事务写入 | ADR-005 Rev.2；失败删除本次目录 |
| D7 | 提交同步返回 `completed + report` | api-contract §3.11；1000 人物目标 ≤90s |
| D8 | 成功后自动切换当前作品 | 用户 2026-11-15 确认（PG11-Q2） |
| D9 | 导入实体 `review_status=pending`、`reviewed_by='import'` | SPEC-MIG-005 |
| D10 | review 单条/批量统一返回状态子集 | 五类 DTO 结构差异大；F10 回写契约 |
| D11 | 类别全部按 normal 导入；未声明境界时建空内置 | 模板未定义纪元类别；满足 RT-08 |

## 2. 数据设计（迁移 0011）

```sql
CREATE TABLE import_runs (
  id               TEXT PRIMARY KEY,
  status           TEXT NOT NULL DEFAULT 'uploaded'
                   CHECK (status IN ('uploaded','validating','validated','running','completed','failed')),
  stage            TEXT,
  markdown_name    TEXT NOT NULL,
  archive_name     TEXT,
  markdown_path    TEXT NOT NULL,
  archive_path     TEXT,
  markdown_bytes   BIGINT NOT NULL CHECK (markdown_bytes >= 0),
  archive_bytes    BIGINT CHECK (archive_bytes IS NULL OR archive_bytes >= 0),
  template_version TEXT,
  work_name        TEXT,
  work_id          TEXT REFERENCES works (id) ON DELETE SET NULL,
  snapshot_id      TEXT REFERENCES snapshots (id) ON DELETE SET NULL,
  counts_json      JSONB,
  errors_json      JSONB,
  warnings_json    JSONB,
  report_json      JSONB,
  error_count      INTEGER NOT NULL DEFAULT 0 CHECK (error_count >= 0),
  error_code       TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at      TIMESTAMPTZ
);
CREATE INDEX import_runs_status_created_idx ON import_runs (status, created_at DESC);
CREATE INDEX import_runs_created_idx ON import_runs (created_at DESC);
```

- staging 固定在 `DATA_DIR/tmp/imports/{id}/`，存 `source.md`、`images.zip`；成功后清理。
- 图片落盘 `images/{work_id}/{assetID}.{ext}` 与 `_thumb`，与 `service/media.go` 一致。

## 3. 模板 v1.0

固定区块与字段见实施计划；解析结果 `Spec` 保存行号与原始值：

```go
type Spec struct {
    TemplateVersion string
    Work            WorkSpec
    Categories      []CategorySpec
    Nodes           []NodeSpec
    Factions        []FactionSpec
    Locations       []LocationSpec
    Characters      []CharacterSpec
    Rankings        []RankingSpec
    Chronicles      []ChronicleSpec
}
type Issue struct { Line int; Field, Code, Message string }
```

- `## 体系类别` 每行一个类别，`（内置）` 仅允许「境界」。
- `## 体系节点` 行格式 `[类别] 父 > 子`；父必须先出现；同级唯一。
- 人物动态键 = 已声明类别名；保留键：名称/别名/称号/简介/势力/地点/图片。
- 图片引用只允许相对路径；`..`、绝对路径、反斜杠规范化后越界均报错。
- 未知区块、未知字段、版本缺失/不符、重复实体名、坏引用均产生 error。
- 自由文本关系类型（`弟子`/`属下`）记 warning，不建引用。

## 4. 校验

- 名称与长度复用 `model.Normalize*`。
- 同作品唯一：作品名全局、人物/势力/地点/榜单/体系节点同级。
- 引用：人物→势力/地点/体系节点；势力→上级/首领；地点→上级/势力；榜单席位→人物；大事记→人物/势力。
- 体系：类别 ≤10；节点最多两级；父先于子；节点类别存在。
- 图片：Markdown 引用集合 vs zip 条目集合；缺失、多余、格式不支持、>10MB 均为 error。
- 作品名与现有作品冲突由 service 在创建前检查（校验时也提示）。

## 5. 提交顺序

1. `import_runs` 行锁；completed → 返回既有报告；非 validated/error_count>0 → `IMPORT_STATE_CONFLICT`/`IMPORT_INVALID`。
2. 预生成 `workID`；`Snapshot.Create(ctx, "import")`；失败 → `SNAPSHOT_FAILED`。
3. 读取并重编码引用图片 → 写 `images/{workID}/`；构建 `media_assets` 行。
4. 单事务：
   work（不建默认境界）→ categories（补空境界）→ nodes（父先子后）→ factions（先无首领）→ locations → characters + names + system refs + locations → factions.leader 回填 → location_factions → rankings + seats → chronicles → media_assets → `search.Refresher.RefreshWork` → `SetCurrentWork` → import 置 completed。
5. 事务失败：回滚 DB，删除 `images/{workID}` 与 staging，置 `failed`。
6. 成功：清 staging，写 `report_json`、`finished_at`。

- 幂等：同一 import_id 重复 commit 返回首次报告；并发 commit 由行锁与状态机保护。
- 超时：handler/服务使用 `context.WithTimeout(cfg.ImportTimeoutMin)`。

## 6. API

| 方法 | 路径 | 行为 |
|---|---|---|
| GET | `/api/v1/imports/template` | `text/markdown` 附件 |
| GET | `/api/v1/imports/template/example` | 示例 zip |
| POST | `/api/v1/imports` | multipart `markdown` + `archive` → 202 `{import_id,status}`，后台校验 |
| POST | `/api/v1/imports/{id}/validate` | 202 重跑校验 |
| GET | `/api/v1/imports/{id}` | 状态、计数、错误、警告、作品、快照 |
| POST | `/api/v1/imports/{id}/commit` | 同步提交 → 200 `{status,work_id,report,review_url}` |
| GET | `/api/v1/imports/{id}/report` | 报告 |
| POST | `/api/v1/{resource}/{id}/review` | 单条确认 |
| POST | `/api/v1/reviews/batch` | 批量确认 |

错误码：`IMPORT_IN_PROGRESS`(409)、`IMPORT_STATE_CONFLICT`(409)、`IMPORT_INVALID`(422)、`IMPORT_TEMPLATE_VERSION`(400)、`IMPORT_INSUFFICIENT_STORAGE`(507)、`IMPORT_TIMEOUT`(504)、`PATH_TRAVERSAL`(400)、`UNSUPPORTED_FILE_TYPE`(422)、`FILE_TOO_LARGE`(422)、`REVIEW_NOT_FOUND`(404)、`REVIEW_INVALID_STATE`(409)、`SNAPSHOT_FAILED`(503)。

## 7. 前端

- `/import`：五步向导；上传用 XHR 显示进度；校验每 1s 轮询；确认 Dialog 同步等待并显示阶段文案；报告与校对分组。
- 设置页新增「数据安全 → 导入飞书正文」卡片链接。
- 校对清单按 5 类拉取 pending；体系节点先拉类别再逐类拉 pending 节点。
- 复用 Dialog/Progress/Table/Badge/Alert/Toast/EmptyState/Button；新增 stepper 与错误清单样式。
- a11y：`aria-current="step"`、`role="progressbar"`、`aria-live`、`aria-label`、键盘文件选择。

## 8. 门禁

- 后端：`gofmt -l <改动文件>`、`go vet ./...`、`go build ./...`、`go test -count=1 ./...`
- 集成：`TEST_DATABASE_URL=… go test -count=1 -tags=integration ./internal/integration/...`
- 前端：`npm run typecheck`、`npm run lint`、`npm run test`
- 构建：`docker compose build app`
- E2E：`npx playwright test --reporter=list`

## 9. 风险与回滚

- 代码回滚：回退 0011 迁移与本次代码；staging 清理；已导入作品可软删除或从快照恢复。
- 提交失败不触碰既有作品；快照可回退。
- 图片目录按 work_id 隔离，失败只删除本次目录。

## 10. 变更历史

| 日期 | 版本 | 说明 |
|---|---|---|
| 2026-11-15 | v0.1 | F1 初版：模板 v1.0、导入状态机、原子提交、review 接口、PG-11 |
