# Proposal — CHG-002 作品生命周期与回收站/快照基座

- Change ID: CHG-002
- Status: Approved
- Owner: 作者（单用户）
- Created at: 2026-11-08
- Approved at: 2026-11-08（F1 裁决：真实快照五件套；顶栏切换器 + 独立回收站页）

## 1. Why

CHG-001 交付了壳层与只读状态，但系统还没有「作品」这一层：没有作品就无法隔离人物/势力/地点数据，后续 CHG-003~009 全部无落点；`/settings` 的 `current_work_id` 永远是 `null`。

同时，规格要求所有破坏性操作（删除作品、永久删除、导入、恢复）前先落快照，失败则阻止操作（FR-55、EX-23）。这条安全底线需要在有第一个破坏性入口之前建立，否则后续变更会先长出手，再补安全网。

本变更因此同时交付：作品生命周期（创建/切换/重命名/删除/撤销/恢复）、回收站基座、快照基座（真实五件套 + 保留 3 份）、乐观锁与未保存保护的前端基座。

## 2. What Changes

### 2.1 用户可感知

- 顶栏「作品位」从只读占位变为切换器：列出全部作品、可新建、可切换、可重命名、可删除。
- 删除作品需要二次确认（输入作品名一致），成功后 8 秒内可「撤销」。
- 新增 `/trash` 回收站页：按作品筛选、查看被删作品、恢复（同名冲突时要求改名）、永久删除、清空。
- `/settings` 增加快照列表卡片（时间、触发原因、大小、状态）；`current_work_id` 反映真实当前作品。
- 未保存保护：切换/删除作品前若表单有未保存修改，先确认。

### 2.2 系统内部

- 迁移 `0002_works_trash_snapshot`：`works`、`app_settings`、`snapshots`、`undo_tokens` 四表及其索引/约束。
- 新增 `internal/{model,repository,service}/works.go`、`trash.go`、`snapshot` 相关包；`internal/httpapi/works.go`、`trash.go`、`snapshots.go`。
- 快照服务：真实产出 `app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256` 到 `/data/snapshots/{ts}-{reason}`，保留最近 3 份。
- 运行时镜像加入 PostgreSQL 客户端（`pg_dump`），用于快照导出。
- 前端：`components/shell/work-switcher.tsx`、`components/works/*`、`app/trash/page.tsx`、`components/settings/snapshot-list.tsx`、`lib/api/works.ts`、`lib/api/trash.ts`、`lib/api/snapshots.ts`。

## 3. Why Not Alternatives

- **只做元数据快照**：无法真正验收 EX-23（快照失败阻止删除），且后续 CHG-010 会推翻重做存储格式。已否决（用户裁决：真实五件套）。
- **独立「作品管理」页**：与顶栏切换器功能重叠，CHG-001 已把作品上下文放在顶栏，用户裁决保持原地扩展。
- **`updated_at` 做乐观锁**：规格明确只用 `version`（P5b RT-16）。
- **撤销窗口放内存**：容器重启即丢失，且 undo 需要幂等（重复/过期要返回明确 code）；落库更简单可控。

## 4. Acceptance Criteria

1. Given 空库，When `POST /api/v1/works` 创建「山海」，Then `201` 返回作品且同事务创建内置「境界」类别。
2. Given 已有作品「山海」，When 再次创建同名，Then `409 NAME_CONFLICT`，不产生第二条记录。
3. Given 作品列表非空，When `GET /api/v1/works?q=山`，Then 只返回未删除且名称匹配的作品，分页字段完整。
4. Given 作品 `version=1`，When `PATCH` 不带 `If-Match`，Then `409 VERSION_CONFLICT`；带 `"2"` 时 `409`；带 `"1"` 时成功且 `version=2`。
5. Given `DELETE /works/{id}` 且 `confirm_name` 与作品名不符，Then `422 VALIDATION_ERROR` 且作品未被删除。
6. Given 快照生成失败，When 删除作品，Then `503 SNAPSHOT_FAILED`，作品仍在且 `GET /works` 可见（EX-23）。
7. Given 删除成功，When 8 秒内 `POST /trash/undo`，Then `200 restored=true`；超过 8 秒或重复撤销返回 `409 UNDO_EXPIRED` / `409 ALREADY_DELETED`。
8. Given 回收站存在被删作品，When `POST /trash/work/{id}/restore` 且同名作品已存在、未提供 `new_name`，Then `422 RESTORE_CONFLICT` 且不部分恢复。
9. Given 第 4 次快照成功，When 查看 `/data/snapshots` 与 `GET /snapshots`，Then 只剩最近 3 份且记录与目录一致（FR-55）。
10. Given `/settings`，When 读取，Then `current_work_id` 为真实当前作品；无作品时为 `null`。
11. Given 顶栏切换器，When 切换作品，Then 当前作品持久化（重启后仍是该作品），并在页面刷新后保持一致。
12. Given `/trash`，When 无被删作品，Then 显示空状态且不渲染不可用按钮。

## 5. Out of Scope

- 人物/势力/地点等实体的回收站条目（本变更只交付作品根；其余 root_type 端点保留并对空数据返回空列表）。
- `/api/v1/works` 的 10 本软上限提示（`WORK_SOFT_LIMIT` 记入后续作品管理增强）。
- 快照恢复（`restore-snapshot` CLI）、导出（`/exports`）→ CHG-010。
- 快照差异备份、定时快照、自定义保留数 → 明确不做（SPEC-BACKUP-003 边界）。
- 回收站批量恢复/分组恢复外的复杂改名编排。

## 6. Risks

| 风险 | 影响 | 处置 |
|---|---|---|
| 镜像加 pg_dump 体积/版本不匹配 | 快照失败 | 用 `postgresql16-client` 与 DB 主版本对齐；启动时校验 `pg_dump --version` 并写健康信息 |
| 软删除整组事务较长 | 锁竞争 | 单事务 + 批量 `UPDATE ... WHERE work_id = ?`；作品量级本地单用户可接受 |
| 快照与删除共用事务边界 | 快照成功但删除回滚 → 残留快照 | 允许残留（快照是审计资源，不做补偿删除），保留策略自然回收 |
| undo 依赖时钟 | 跨时区误判 | 全部用 UTC 时间戳，服务端判定 |

## 7. Decisions（F1 裁决记录）

- D-01 快照实现深度：真实五件套（用户裁决 2026-11-08）。
- D-02 页面落点：顶栏作品切换器 + `/trash` 独立页 + 设置页快照卡片（用户裁决 2026-11-08）。
- D-03 快照文件命名以 `api-contract.md` §3.14（P5b RT-06）为准：`app.dump`、`images/`、`snapshot.json`、`manifest.json`、`checksums.sha256`；`backup/specs.md` 的旧命名同步修订。
- D-04 撤销 token 落 `undo_tokens` 表，一次性、8 秒有效。
- D-05 当前作品存 `app_settings.current_work_id`，切换即写入，不做会话态。
