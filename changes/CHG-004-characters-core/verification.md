# Verification Report

- Change ID: CHG-004-characters-core
- Commit: 功能分支 `feat/chg-004-characters-core`（T1～T14）
- Environment: Windows + PowerShell 5.1；Go 1.27.3（交叉编译 linux/amd64）；Node.js 22；PostgreSQL 16（容器 `novel-management-db-1`）；应用容器 `novel-management-app-1` @ `http://127.0.0.1:17860`
- Tester: Agent（F2 实施；F3/F5/F6 待补）
- Date: 2026-11-09

## 1. Static Checks

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd frontend; npm run typecheck` | 0 | 无输出 | `tsc --noEmit` |
| `cd frontend; npm run lint` | 0 | 无告警 | ESLint |
| `cd backend; go vet ./...` | NOT RUN | 待 F3 | — |

## 2. Unit / Integration Tests

| Command | Exit code | Result | Evidence |
|---|---:|---|---|
| `cd frontend; npm run test` | 0 | 6 个文件 / 43 个用例通过 | 含 `lib/api/characters.test.ts` 13 个用例 |
| `cd backend; go test ./...` | NOT RUN | 待 F3 | — |
| `TEST_DATABASE_URL=... go test -tags=integration ./internal/integration/...` | NOT RUN | 待 F3 | T5～T9 已提交集成用例 |

## 3. E2E Scenarios

| Scenario | Steps | Expected | Actual | Pass/Fail | Evidence |
|---|---|---|---|---|---|
| 容器内迁移 | `docker compose up -d app`；查看日志 | `goose` 迁移到 0004 | `successfully migrated database to version: 4` | PASS | `novel-management-app-1` 启动日志 |
| 人物列表 | `characters.spec.ts` 列表 2 例 | 新建/搜索别名/境界筛选/待校对/清除筛选/分页 | 全部通过 | PASS | 7 passed |
| 人物详情 | `characters.spec.ts` 详情 5 例 | 内联保存、409 冲突、未保存守卫、删除撤销、回收站恢复、a11y、1024 溢出 | 全部通过 | PASS | 7 passed |
| 视觉基线 · 人物 | `npx playwright test e2e/visual.spec.ts -g "人物"` | 列表 1280/1024、详情 1280 基线生成 | 通过 | PASS | `characters-1280/1024`、`character-detail-1280` PNG |
| 无障碍 | `a11y.spec.ts` 含 `/characters` | axe 无 serious/critical | 6 个页面全部通过 | PASS | — |
| 键盘可达 | `keyboard.spec.ts` 侧栏遍历 | 概览 → 体系 → 人物 → 回收站 → 设置 | 通过 | PASS | 导航新增「人物」项 |
| 体系引用阻断 | `system.spec.ts`「引用阻断」 | 人物引用类别 → 停用/删除 409，details 含人物名与 `in_trash` | 通过 | PASS | 改为经 API 创建真实人物引用 |
| 全量 E2E | `cd frontend; npx playwright test` | 全部通过 | 63 passed（49.8s） | PASS | 含既有壳层/作品/体系/回收站/设置/视觉基线 |

## 4. Review Findings

| ID | Severity | Finding | Fix / Waiver | Retest |
|---|---|---|---|---|
| — | — | 待 F5 复审 | — | — |

## 5. Implementation Differences（对照 `design.md` §3.1 / D1～D9）

| 项 | 设计 | 实施 |
|---|---|---|
| `faction_id` | 势力引用 | 无外键的 `TEXT`，本轮仅接受 `null`（D1，CHG-006 补 FK） |
| `location_links` | 地点关联 | 仅接受空数组（D1，CHG-007 开放） |
| `review_status` | 校对闭环 | DTO 只读返回；确认端点延后 CHG-011（D2） |
| 列表列 | 势力/地点/榜单 | 隐藏势力/地点筛选；榜单列固定「—」（D4） |
| `CharacterSummary` | 摘要 | 含 `names[]` 与 `version` |
| `DELETE /characters/{id}` | 乐观锁 | 要求 `If-Match`，返回 `undo_token` |

## 6. Known Limitations

- 本机 Docker BuildKit 构建会话反复 `EOF`，本轮镜像由交叉编译 Linux 二进制 + `docker cp` + `docker commit` 生成（布局与 Dockerfile 一致）；`Dockerfile` 未改动，Linux CI 应可正常构建。
- 尚未执行 `go vet`、后端单测与集成测试（F3 补）。
- 正文图片、势力/地点/榜单关系、校对确认分别由 CHG-005～008、011 承接。

## 7. Rollback Verification

- `TestMigrationRoundTrip`（0004 Down/Up）：NOT RUN，待 F3。
- 数据回滚依赖既有快照机制（删除/清空前落盘，保留 3 份），未新增回滚路径。

## 8. Conclusion

PARTIAL（F2 实施完成；F3/F5/F6 门禁待执行并回填）
