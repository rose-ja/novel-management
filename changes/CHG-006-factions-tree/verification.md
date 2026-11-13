# CHG-006 验证记录（F3）

- Change ID：`CHG-006-factions-tree`
- 分支：`feat/chg-006-factions-tree`
- 验证日期：2026-11-13
- 环境：Windows PowerShell 5.1；Docker 容器 `novel-management-app-1` @ `127.0.0.1:17860`；PostgreSQL 迁移版本 **v6**（`goose_db_version` 最新 version_id = 6）

## 1. 门禁命令与结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| 后端格式/静态检查 | `gofmt -w`（本次新增/修改文件）、`go vet ./...` | 通过（存量 CRLF 文件会被 `gofmt -l` 列出，不作为本变更门禁） |
| 后端构建 | `go build ./...` | 通过 |
| 后端单测 | `go test -count=1 ./...` | 全部 `ok`（config/httpapi/model/service） |
| 后端集成 | `$env:TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./...` | 全部 `ok`，含 `internal/integration` |
| 前端类型 | `npm run typecheck` | 通过 |
| 前端 Lint | `npm run lint` | 通过（0 warning） |
| 前端单测 | `npm run test` | 通过，含 `lib/factions-tree.test.ts` 7 项 |
| 前端构建 | `npx -y node@22 node_modules/next/dist/bin/next build` | 通过（Node 24 下 Next 构建失败，沿用项目既定 Node 22 方式） |
| E2E | `npx playwright test` | **75 passed**（含新增 `e2e/factions.spec.ts` 5 项） |
| 视觉基线 | `npx playwright test e2e/visual.spec.ts --update-snapshots` | 13 项更新并通过；新增势力树 1280/1024、势力详情 1280 |
| 可访问性 | `e2e/a11y.spec.ts`（axe） | 全部页面无 serious/critical，含 `/factions` |

## 2. 新增集成覆盖

- `TestMigrationRoundTrip`：迁移 v6；0006 Down 删除 `factions`、恢复 `chronicles.owner_type` 仅 character、清理势力大事记，人物 `faction_id` 列值保留。
- `TestFactionTreeCRUD`：万族 → 妖族 → 青丘三级创建、默认排序与计数、`q` 命中 + 祖先、`parent_id=root`。
- `TestFactionNameAndReferenceValidation`：`Sword`/`sword` 判重、跨作品父级/首领 `CROSS_WORK_REFERENCE`、名称 trim。
- `TestFactionMoveCycleAndVersion`：缺 `If-Match` → 428；移动到自身/子/深层后代 → `CYCLE_DETECTED`；旧版本 → `VERSION_CONFLICT`。
- `TestFactionDeleteAndDetail`：详情下级/地点空数组；`HAS_CHILDREN`；无引用删除返回 `undo_token`。
- `TestFactionMembersScopeAndCounts`：直属/子树成员与分页、摘要计数、换势力后两侧即时变化、非法 scope 422。
- `TestCharacterFactionIntegration`：人物设置/清空势力、跨作品与已删势力拒绝、列表 `faction` 与 `faction_id` 筛选。
- `TestCharacterDeleteBlockedByFactionLeader`：首领删除阻断（含回收站中的势力）；清空首领后可删。
- `TestFactionChronicleCRUDAndUndo`：`owner_type=faction` CRUD、版本冲突、2,000 字校验、删除撤销、已删势力写入被阻。
- `TestFactionDeleteMatrix` / `TestFactionTrashUndoRestorePurge` / `TestFactionRestoreBlockedByDeletedParent` / `TestWorkLifecycleCoversFactions`：删除矩阵、回收站势力根、父级缺失恢复阻断、作品整组生命周期与永久删除顺序。

## 3. E2E 新增

`e2e/factions.spec.ts`：

1. 三级势力默认展开两级；搜索命中高亮并保留祖先；无匹配空态与清除筛选。
2. 内联新建（顶层/指定下级）；详情编辑首领与简介；自身在上级下拉中禁用。
3. 成员「仅直属/含下级」切换；人物详情势力链接跳转势力详情。
4. 删除被下级阻止并列出引用链接。
5. 回收站「势力」筛选、父级作品名、恢复、再次删除后永久删除。

## 4. 手工验证

- 容器启动自动迁移到 v6，健康检查 `{"status":"ok","db":"ok","cache":"ok"}`。
- 既有 CHG-004/005 数据（人物、图片、大事记）在迁移后可用；人物列表与回收站全量 E2E 无回归。
- 总部/成员计数在 1000 节点规模下无专项压测；当前实现为一次取全量、前端 `useMemo` 组树。

## 5. 已知偏差与延后

| 项 | 说明 |
|---|---|
| UI-11 慢网先渲染文字、计数后到 | 未做专项自动化；本轮一次请求返回计数，无二次请求抖动。延后到性能专项。 |
| `GET /factions` 响应形态 | 返回 `{items,total}` 包装（与作品/人物列表一致），契约 §3.4 写的数组形态在 F10 回写更正。 |
| 校对端点 | `POST /factions/{id}/review` 按 D4 延后 CHG-011；势力默认 `reviewed`。 |
| 地点关系 | `relations.locations` 恒为空数组，CHG-007 接入。 |
| 契约错误码 | 未知 `owner_type` 沿用既有 `VALIDATION_ERROR` 422（字段 `owner_type`），与 CHG-005 一致。 |

## 6. 证据文件

- 迁移：`backend/internal/migrations/0006_factions_chronicles.sql`
- 集成：`backend/internal/integration/{factions,faction_members,faction_character,faction_chronicle,faction_lifecycle,migrations}_test.go`
- 前端单测：`frontend/lib/factions-tree.test.ts`
- E2E：`frontend/e2e/factions.spec.ts`
- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/`（新增 `factions-1280/1024`、`faction-detail-1280`）
