# PR/MR 描述 — CHG-003 体系类别与节点

> 状态：本地分支已就绪，**未推送、未创建远程 PR**（按 F8 规则等待用户明确授权）。
> 分支：`feat/chg-003-systems-nodes` → 目标 `main`；提交范围 `3c2c10a^..HEAD`（16 个提交）。

## 1. 标题与目标

**feat: 体系类别与两级节点（CHG-003）**

让用户在 `/system` 为每本作品维护「体系」：新增/改名/停用/启用/删除类别（最多 10 类含内置「境界」），维护两级节点（同级唯一、可排序、纪元起止文本），并通过回收站整组恢复/撤销/永久删除；同时为 CHG-004 的人物动态表单提供「启用的类别 + 可选节点」与引用阻断骨架。

## 2. 关联

- Requirement：AC-07、AC-13、AC-14；EX-16、EX-21
- Spec：SPEC-SYS-001～005（`project/specs/systems/specs.md`）
- Page Spec：PG-10 体系配置（`project/page-specs/PG-10-system-config.md`）
- 契约：`project/api-contract.md`、`project/architecture/data-model.md`
- 变更文档：`changes/CHG-003-systems-nodes/`（proposal、design、tasks、checklist、review、verification、acceptance）

## 3. 范围与非目标

**In scope**

- 类别生命周期：新增/改名/停用/启用/删除；上限 10（含内置）；内置「境界」不可改名/停用/删除；类型 `normal`/`era` 创建后不可变；名称唯一（NFKC + lower + 空白折叠）。
- 两级节点：主层级/子层级、同级唯一、同级排序、纪元起止文本；父节点有子节点时删除阻断。
- 引用阻断骨架：`character_system_refs` 表与类别/节点引用计数；停用/删除/永久删除被引用时阻断（人物写入延后 CHG-004）。
- 回收站：`work` / `system_category` / `system_node` 三类根；类别整组软删除/恢复（`new_name` + `renames[]` 原子恢复）、节点单项删除/撤销/恢复；8 秒撤销窗口。
- 前端 `/system` 页面与回收站三类根适配（类型 Badge、父级、筛选、对话框文案）。

**非目标（按 checklist §7 延后）**

- 人物表单动态字段、人物侧改名同步（CHG-004）
- 导入模板、待校对确认 `/reviews`（CHG-011）
- 检索包含类别/节点（CHG-009）
- 类别自定义排序/复制/跨作品共享、三级及以上节点、跨类移动、纪元日期计算/格式校验、类别 `description`。

## 4. 主要文件/模块变更

**后端**

- 迁移：`backend/internal/migrations/0003_system_nodes.sql`（`system_categories`、`system_nodes`、`character_system_refs`、部分唯一索引、软删除一致性 CHECK）。
- 模型：`model/system.go`、`model/errors.go`（错误码与新 `HasChildrenError`）。
- 仓储：`repository/system.go`、`repository/works.go`（含 `ActiveName`、`ListNodeChildren`、子先父后永久删除）。
- 服务：`service/system.go`（类别/节点生命周期、事务内引用检查）、`service/trash.go`（三类根列表、整组恢复、永久删除、撤销）。
- HTTP：`httpapi/system.go`、`httpapi/trash.go`、`httpapi/request.go`（错误码与 `details.items` 映射）。
- 集成测试：`internal/integration/system_*.go`（体系 33 个用例，含跨作品隔离、回收站恢复/永久删除、迁移往返）。

**前端**

- API：`frontend/lib/api/system.ts`（9 个函数 + 11 个单测）、`lib/api/trash.ts`。
- 页面/组件：`app/system/page.tsx`、`components/system/{system-page,category-list,category-detail,node-tree,system-dialogs,reference-dialog}.tsx`、`app/trash/page.tsx`、`components/trash/trash-dialogs.tsx`、`lib/messages.ts`。
- E2E：`frontend/e2e/system.spec.ts`（11）、`frontend/e2e/trash-system.spec.ts`（2）；视觉基线 `system-1280/1024`。

## 5. UI 截图

- 视觉基线：`frontend/e2e/visual.spec.ts-snapshots/system-1280-chromium-win32.png`、`system-1024-chromium-win32.png`（全量视觉用例通过，差异 ≤ 0.02）。
- 本次未录制视频。

## 6. 实际执行的测试与结果

| 命令 | 结果 |
|---|---|
| `cd backend; go vet ./...` | 0，无输出 |
| `cd backend; go test -count=1 ./...` | 全部 ok |
| `TEST_DATABASE_URL=...; go test -count=1 -tags=integration ./internal/integration/...` | `ok ... 10.123s`（体系 33 个用例） |
| `cd backend; golangci-lint run ./...` | `0 issues.` |
| `cd backend; govulncheck ./...` | `No vulnerabilities found` |
| `cd frontend; npm run lint` | 0 |
| `cd frontend; npm run typecheck` | 0 |
| `cd frontend; npm run test` | 5 files / 30 tests 通过 |
| `cd frontend; npm run build` | Next.js 静态导出成功 |
| `docker compose build app; docker compose up -d app` | 容器 healthy，`/api/v1/health` → `{"status":"ok","db":"ok","cache":"ok"}` |
| `cd frontend; npx playwright test --reporter=list` | **52 passed / 0 failed（34.0s）** |
| F6 API 边界脚本（24 项断言） | 24/24 通过（含 `NODE_LEVEL_LIMIT`、`HAS_CHILDREN` details、`PARENT_DELETED`、`RESTORE_CONFLICT`、`CATEGORY_LIMIT`、`CATEGORY_BUILTIN_IMMUTABLE`） |

## 7. E2E 与人工验收证据

- E2E：`system.spec.ts` 11 条（内置保护、类别 CRUD、10 类上限、两级节点、同级重名、排序持久、停用只读、引用阻断、整组删除 + 恢复、节点删除/撤销、1024 视口）；`trash-system.spec.ts` 2 条（类别整组显示/筛选/恢复、节点显示/筛选/永久删除）；`a11y` 含 `/system`，axe 无 serious/critical。
- 集成：新增 5 个回归用例覆盖 F4 审查缺陷（作品在回收站时恢复类别、含软删子节点的永久删除、清空回收站父子顺序、类别两级永久删除、恢复改名撞未改名兄弟）。
- 人工验收：`acceptance.md` 结论 **ACCEPT**（2026-11-09，用户选择；F5 复审 PASS、F6 PASS）。
- 审查：`review.md` F4 初审 FAIL（S0=0/S1=3/S2=6）→ F5 修复 → 独立 Reviewer 复审 **PASS**（S1 清零，CR-07 性能类 S2 接受）。

## 8. 风险、兼容性、迁移与回滚

- **迁移**：0003 新增三表与索引；Down 可回滚，`TestMigrationRoundTrip` 覆盖 Up/Down 往返。0002 表与数据保留。
- **兼容性**：错误码沿用既有集合并新增体系语义；`NODE_LEVEL_LIMIT`/`NODE_PARENT_MISMATCH` 统一为 422（契约文档已同步）。
- **数据一致性**：类别/节点删除的引用检查与写入同事务；永久删除按「引用→子节点→根节点→类别→作品」顺序，避免 FK RESTRICT。
- **性能**：回收站列表仍有 N+1/全表聚合（CR-07，S2），数据量增长后需专项优化。
- **安全**：写请求保持 `X-Requested-With` 守卫；类别/节点按 `work_id` 隔离；无新增依赖与密钥。
- **回滚方案**：回退本 PR 提交；若已迁移，执行 0003 Down（`migrations.Down`）恢复至 0002 结构；快照机制未改动，不影响既有备份。

## 9. Reviewer 重点关注

1. `trash.go` 整组恢复的最终名称预检与 `renames[]` 原子性（CR-04）。
2. `PurgeNode`/`PurgeAll`/`PurgeCategoryGroup` 的删除顺序与含软删子节点检查（CR-03）。
3. `UpdateCategory`/`DeleteCategory`/`DeleteNode` 的事务边界与引用检查（CR-06）。
4. 回收站前端三类根的展示/筛选/文案是否完整（CR-01）。
5. 文档一致性：`proposal.md` AC-2、`checklist.md` ND-02/ND-03、`project/api-contract.md` 状态码。
6. 合并方式：建议 squash merge 或保留 16 个提交；需用户授权后再推送与创建 PR。
