# CHG-011 任务拆解

- Change ID：`CHG-011-import`
- 依据：`proposal.md`、`design.md`
- 规则：每项完成后独立提交；门禁命令见 `design.md` §8。

## 1. 任务列表

### T1 迁移 0011 与导入模型

- **文件**：`backend/internal/migrations/0011_import_runs.sql`、`backend/internal/model/import.go`、`backend/internal/migrations/migrations.go`、`backend/internal/migrations/migrations_test.go`。
- **验收**：Up/Down 成功；状态 CHECK 六态；索引；`LatestVersion=11` 与嵌入 SQL 最大版本一致。
- **验证**：gofmt/vet/build/迁移用例。

### T2 模板 v1.0 与解析器

- **文件**：`backend/internal/imports/{template,types,parser}.go`、`template-v1.0.md`、`example-v1.0.md` 与单测。
- **验收**：合法模板全部区块解析；版本缺失/不符、未知区块/字段、重复记录产生定位错误。
- **验证**：`go test ./internal/imports/...`。

### T3 校验、图片匹配与报告

- **文件**：`backend/internal/imports/{validate,images,report}.go` 与单测。
- **验收**：类别 ≤10、节点两级/父先子后、引用、同名、缺图/多图/坏格式/超 10MB；八类计数。
- **验证**：单测。

### T4 上传、校验任务与仓储

- **文件**：`backend/internal/repository/imports.go`、`backend/internal/service/import.go`（CreateUpload/RunValidate/MarkInterrupted/清理）。
- **验收**：流式落盘；磁盘预检；状态机；staging 清理；中断标记。
- **验证**：`go test ./internal/service/...`。

### T5 原子提交

- **文件**：`backend/internal/service/import.go`（Commit 与落图）。
- **验收**：快照、图片、单事务八类写入、检索重建、当前作品切换、重复 commit 幂等、失败 0 残留。
- **验证**：单测 + 集成。

### T6 导入 HTTP 与装配

- **文件**：`backend/internal/httpapi/imports.go`、`router.go`、`request.go`、`backend/cmd/server/main.go`。
- **验收**：模板/示例下载、multipart 上传、validate、commit、report；错误码与中文文案。
- **验证**：`go build ./...` + 集成。

### T7 校对接口

- **文件**：`backend/internal/service/review.go`、`backend/internal/httpapi/reviews.go`、`model/errors.go`。
- **验收**：单条幂等；批量仅 pending；skipped；不存在整批失败；版本 +1；检索同步。
- **验证**：单测 + 集成。

### T8 前端 API 与文案

- **文件**：`frontend/lib/api/imports.ts`、`frontend/lib/api/reviews.ts`、`frontend/lib/messages.ts` 与单测。
- **验收**：类型完整；上传进度；错误码映射；URL 纯函数。
- **验证**：`npm run typecheck` + `npm run test`。

### T9 PG-11 页面与设置入口

- **文件**：`frontend/app/import/page.tsx`、`frontend/components/import/*`、`frontend/components/settings/import-section.tsx`、`frontend/app/settings/page.tsx`、`frontend/styles/components.css`。
- **验收**：五步、上传进度、错误清单、右侧摘要、确认 Dialog、报告、校对分组/批量、空状态、1024 布局、a11y。
- **验证**：typecheck/lint/test。

### T10 集成、E2E 与视觉

- **文件**：`backend/internal/integration/import_test.go`、`review_test.go`、`frontend/e2e/import.spec.ts`、`a11y.spec.ts`、`keyboard.spec.ts`、`visual.spec.ts`。
- **验收**：合法/非法夹具、回滚、幂等、review；E2E-09 全链路；视觉基线。
- **验证**：集成 + Docker 构建 + Playwright。

### T11 收口

- **文件**：`changes/CHG-011-import/{verification,review,acceptance,pr}.md`、归档目录、Spec/PG/契约/数据模型/roadmap。
- **验收**：F3~F10 完成；文档同步并推送。
- **验证**：全量门禁。

## 2. 依赖图

```text
T1 ─ T2 ─ T3 ─ T4 ─ T5 ─ T6 ─ T8 ─ T9 ─ T10 ─ T11
T1 ─ T7 ───────────────────────┘
```

## 3. 门禁

- 每个后端任务：gofmt（改动文件）、go vet、go build、相关 go test。
- T5/T6/T7 后：全量集成。
- 每个前端任务：typecheck、lint、test。
- T10 后：Docker 构建 + 全量 Playwright。
