# CHG-005 人物图片与大事记 — 独立代码审查报告

| 项目 | 内容 |
|---|---|
| 变更 | CHG-005「人物图片与大事记」 |
| 审查分支 | `feat/chg-005-characters-media-chronicles` |
| 审查基线 | `9dc06c0`（CHG-004 归档）→ HEAD `799bc9e` |
| 审查日期 | 2026-11-11 |
| 审查方式 | 独立静态审查 + 关键路径手工推演；未重复执行全量门禁，引用 `verification.md` |
| 结论 | **PASS**（无 S1；8 项 S2，其中 R-01/R-08 在 F5 修复，其余登记） |

## 审查范围

- 后端：`0005_characters_media_chronicles.sql`、`model/{media,chronicle,errors,character}.go`、`repository/{media,chronicles,characters,works}.go`、`service/{media,chronicles,characters,trash}.go`、`httpapi/{media,chronicles,characters,request,router}.go`、`cmd/server/main.go`、`go.mod/go.sum`。
- 前端：`lib/api/{media,chronicles,characters}.ts`、`components/characters/{character-image,image-uploader,chronicle-section,character-detail-page,character-table}.tsx`、`styles/components.css`、`lib/messages.ts`。
- 测试：`service/media_test.go`、`integration/{character_media,migrations,suite}_test.go`、`e2e/characters-media.spec.ts`、视觉基线。
- 规范：`changes/CHG-005-.../{proposal,design,tasks,checklist,verification}.md`、SPEC-CHAR-006/007、PG-03、`api-contract.md` §3.3/§3.8、`data-model.md` §3.13/§3.14。

## 问题清单

| ID | 严重度 | 文件与行号 | 关联需求或 Spec | 触发条件 | 实际影响 | 修复建议 | 验证建议 |
|---|---|---|---|---|---|---|---|
| R-01 | S2 | `backend/internal/service/media.go:91-121` | design §3.2；FR 生命周期一致性 | 上传主图与人物删除/作品整组删除并发：上传先 `Get` 人物，删除事务提交后再写媒体 | 可能在已删人物下留下活跃主图，永久删除已执行后文件/行残留 | 上传事务内 `GetForUpdate` 锁定人物并复核 `work_id` | 并发集成用例；F5 已修复 |
| R-02 | S2 | `backend/internal/service/characters.go`（列表图片装配） | design §3.6；可观测性 | `ListCharacterImages` 查询失败 | 错误被 `_` 丢弃，列表静默无图，排障困难 | 记录 `slog.Warn` 或注入 logger；不阻断列表 | 注入失败 DB 的测试或日志断言 |
| R-03 | S2 | `backend/internal/model/media.go`（`MaxImagePixels=40MP`） | SPEC-CHAR-006 §边界；安全 | 上传 10MB 内但 40MP 的 PNG（压缩比高） | 解码峰值约 160MB/请求，并发可耗尽内存 | 降至 20MP 或先按 `DecodeConfig` 拒绝超大边长；文档标注内存预算 | 构造 40MP 小文件测试 |
| R-04 | S2 | `frontend/components/characters/character-detail-page.tsx`（保存流程） | PG-03 §6.2 | 图片上传失败后反复点保存 | 每次重试都再次 PATCH，人物版本持续递增；其他标签页更易冲突 | 图片失败后提供“仅重试图片”路径，或记录 `profileSaved` 跳过 PATCH | 组件/E2E：失败后连续重试只发一次 PATCH |
| R-05 | S2 | `frontend/components/characters/image-uploader.tsx` | UI-04、A11Y-02 | 上传错误/提示 | 错误 `Alert` 未与文件按钮 `aria-describedby` 关联，读屏不连贯 | 为按钮/区域加 `aria-describedby` 指向提示与错误 | axe + 键盘用例 |
| R-06 | S2 | `frontend/e2e/visual.spec.ts` | TEST-05 | 列表视觉基线 | 仅详情基线覆盖真实图片；列表基线为无图占位 | 增加带图列表快照或断言 40px 图片存在 | 更新视觉用例 |
| R-07 | S2 | `backend/internal/integration/character_media_test.go` | design §6.2 并发 | 大事记创建与人物删除交错 | 代码已用行锁，但无真实交错测试；回归无法证明 | 增加 `-race` 交错用例（可控钩子/事务） | 并发集成 |
| R-08 | S2 | `backend/internal/httpapi/media.go:82-91` | api-contract §3.3 缓存 | `If-None-Match` 命中返回 304 | 304 未带 `Cache-Control`/`X-Content-Type-Options`，缓存语义不完整 | 在 304 前设置缓存与安全头 | 集成断言 304 头；F5 已修复 |

## 已检查且判定无问题的高风险点

- **文件路径安全**：路径由 DB 相对路径 + `safePath` 组装，拒绝绝对路径、`..`、卷名与越界；用户输入不参与文件名。
- **上传原子性**：先写临时文件并 `Sync`/`Rename`，DB 失败清理新文件；替换事务提交后才删旧文件。
- **并发唯一性**：活跃资产部分唯一索引 `(entity_type, entity_id)`；并发上传由 DB 兜底，失败方清理文件。
- **生命周期**：人物/作品删除、恢复、永久删除均在同一事务更新 `media_assets`/`chronicles`；永久删除收集路径后提交再删文件；人物、媒体、大事记锁序一致。
- **撤销**：大事记令牌仅恢复单条且校验 owner 活跃；`root_type=chronicle` 列表仍 422，不会误入回收站。
- **图片校验**：MIME + 魔数 + `DecodeConfig` + 像素上限；WebP 经 `x/image/webp` 解码后转 PNG；缩略图最长边 320。
- **HTTP 响应**：ETag 基于 sha256+variant；`nosniff`、`inline`、私有长缓存；不返回绝对路径。
- **前端保存顺序**：PATCH 成功后提交图片；失败保留编辑态并同步版本基线；取消编辑不调用上传。
- **可访问性**：主图使用姓名 alt/占位 `role="img"`；大事记使用 `<ol>`；Toast 复用 `aria-live`。

## F5 复审结论

**PASS**。R-01（上传与删除并发的孤儿风险）与 R-08（304 缓存/安全头）已修复并重跑相关集成用例；R-02～R-07 作为 S2 改进登记，不阻断验收。前端 `alt` 与撤销文案同步修正。

## 结论

- 无 S1；R-01（并发孤儿）与 R-08（304 缓存头）在 F5 修复并复审。
- 其余 S2 登记为后续改进：R-02 日志、R-03 内存上限、R-04 重试语义、R-05 a11y 关联、R-06 列表视觉、R-07 并发测试。
- F5 后需重跑：后端单测/集成、前端 typecheck/lint/单测、全量 E2E。
