# Acceptance Checklist — CHG-002

格式：`- [ ] AC-编号 | 关联 Requirement/Spec/Page ID | 验证方式 | 预期结果 | 证据`

## Functional

- [ ] AC-001 | SPEC-WORK-001 / 3.2 Works | `POST /works` 创建「山海」 | 201，含 `id/version=1/deleted_at=null`，同事务创建内置「境界」类别 | 集成测试 + `curl`
- [ ] AC-002 | SPEC-WORK-001 | 重复创建同名作品 | `409 NAME_CONFLICT`，`works` 表仍 1 条 | 集成测试
- [ ] AC-003 | SPEC-WORK-001 | `GET /works?q=山&page=1&page_size=20` | 只返回未删除且匹配的作品；分页字段完整 | 集成测试
- [ ] AC-004 | SPEC-PLAT-003 | `PATCH` 不带 / 带错 / 带对 `If-Match` | 前两者 `409 VERSION_CONFLICT`；正确时 `version` +1 | 集成测试
- [ ] AC-005 | SPEC-WORK-002 | `DELETE /works/{id}` 且 `confirm_name` 不符 | `422 VALIDATION_ERROR`，作品未删除 | 集成测试
- [ ] AC-006 | SPEC-BACKUP-003 / EX-23 | 注入快照失败后删除作品 | `503 SNAPSHOT_FAILED`，作品仍在 | 集成测试（假快照服务）
- [ ] AC-007 | SPEC-WORK-002 | 删除后 8 秒内 `POST /trash/undo` | `200 restored=true`，作品恢复可见 | 集成 + E2E
- [ ] AC-008 | SPEC-WORK-002 | 超过 8 秒撤销；重复撤销 | `409 UNDO_EXPIRED` / `409 ALREADY_DELETED` | 集成测试
- [ ] AC-009 | SPEC-TRASH-002 | 恢复同名冲突且未提供 `new_name` | `422 RESTORE_CONFLICT` + `details.items`；不部分恢复 | 集成测试
- [ ] AC-010 | SPEC-TRASH-003 | 恢复时提供 `new_name` | 成功且作品名更新 | 集成测试
- [ ] AC-011 | SPEC-BACKUP-003 / FR-55 | 连续 4 次快照 | 目录与记录只剩最近 3 份 | 单测 + 容器内实测
- [ ] AC-012 | SPEC-BACKUP-003 | 快照五件套 | `app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256` 均存在且校验通过 | 容器内实测
- [ ] AC-013 | SPEC-WORK-003 / PG-00 | 顶栏切换作品后刷新页面 | 当前作品保持；`/settings.current_work_id` 一致 | E2E
- [ ] AC-014 | SPEC-WORK-001 / PG-00 | 顶栏新建作品 | 新建后自动成为当前作品，列表出现 | E2E
- [ ] AC-015 | SPEC-TRASH-001 / PG-12 | 打开 `/trash`（有被删作品） | 显示名称/删除时间/影响条数，可恢复/永久删除 | E2E
- [ ] AC-016 | PG-12 | 无被删作品 | 空状态，不渲染不可用按钮 | E2E
- [ ] AC-017 | SPEC-BACKUP-004 / PG-13 | 设置页快照卡片 | 显示时间/原因/大小/状态；无被删作品时为「暂无快照」 | E2E

## UI / UX

- [ ] AC-101 | PG-00 | 作品切换器键盘可达 | 方向键/Tab 可遍历，Esc 关闭，焦点可见 | E2E
- [ ] AC-102 | PG-12 | 1024 宽度 | 无横向溢出 | E2E
- [ ] AC-103 | SPEC-PLAT-004 | axe 扫描 `/`、`/trash`、`/settings` | 无 serious/critical | E2E
- [ ] AC-104 | SPEC-PLAT-003 | 重命名表单有未保存修改时切换作品 | 弹出确认，取消则不切换 | E2E

## Error / Boundary

- [ ] AC-201 | SPEC-PLAT-002 | 接口 500 / 网络异常 | 显示对应文案并可重试，不显示堆栈 | E2E（路由拦截）
- [ ] AC-202 | SPEC-PLAT-002 | 名称超长（>200）或为空 | `422 VALIDATION_ERROR`，字段级错误提示 | 集成 + E2E
- [ ] AC-203 | SPEC-WORK-002 | 删除对话框作品名不匹配 | 确认按钮禁用 | E2E

## Permission / Security

- [ ] AC-301 | SPEC-PLAT-005 | 写请求跨源 | `403 FORBIDDEN_ORIGIN`（沿用 P7 中间件） | 集成测试
- [ ] AC-302 | SPEC-PLAT-005 | 写请求缺 `X-Requested-With` | 前端始终携带；服务端按既有规则处理 | 代码审查 + E2E 抓包

## Regression

- [ ] AC-401 | CHG-001 基线 | `/health`、`/meta`、静态页、404 | 行为不变 | 集成 + E2E
- [ ] AC-402 | 工程门禁 | `scripts/ci.ps1` | 退出码 0 | 实测输出
- [ ] AC-403 | 迁移 | `goose up` → `down` → `up` | 可往返，数据不残留 | 集成测试
- [ ] AC-404 | 视觉基线 | 概览/设置/回收站 1280 与 1024 | 差异 ≤2% | Playwright 视觉基线
