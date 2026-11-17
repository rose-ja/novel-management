# CHG-012 验收记录（F7）

- 日期：2026-11-17
- 基线：`667d183` + 全量回归
- 验收依据：`proposal.md` 验收标准、`project/specs/`、`project/page-specs/`、`project/deployment.md` §10

| # | 验收项 | 结果 | 证据 |
|---|---|---|---|
| 1 | 回收站引用可阻断并深链定位：`/trash?type=…&highlight=…` 命中高亮、滚动与回焦，未命中提示 | PASS | E2E-15（`trash.spec.ts`）；`locations/characters/factions` 删除对话框链接；深链仅定位一次 |
| 2 | 势力/地点/体系/人物/榜单/作品编辑表单脏状态关闭或站内跳转弹出统一确认 | PASS | 势力/体系/榜单新建/容量/调席/改名/备注守卫；E2E 势力、体系、榜单改名；人物/地点/作品/导入既有用例 |
| 3 | 主列表/详情/检索/回收站/设置具备 loading/empty/error；失败中文提示可重试且不泄露堆栈 | PASS | 页面审计；E2E-12 扩展（500+重试保留筛选、筛选空态、无作品空态）；428 文案统一 |
| 4 | axe 列表与详情（有数据）及新建/编辑/删除对话框无 serious/critical | PASS | `a11y.spec.ts` 17 项（含人物/势力/榜单详情、对话框、回收站有数据） |
| 5 | 纯键盘完成人物新建、编辑、删除、回收站恢复，Esc 关闭回焦 | PASS | `keyboard.spec.ts`「人物键盘主流程」；Dialog 关闭显式回焦 |
| 6 | 1000 人物首屏 ≤2s、列表 P95 ≤500ms、检索 P95 ≤150ms | PASS | `scripts/perf.ps1`：221ms / 5ms / 10ms，报告 `artifacts/perf/20261117-105258/report.json` |
| 7 | CSP 下检索/详情可用且无控制台违规；恶意 zip/路径穿越/伪造图片被拒 | PASS | E2E-13（`security.spec.ts`）；backup/imports/httpapi/service 安全用例；相关集成用例 |
| 8 | 导出→CLI 恢复→计数一致；迁移 Up/Down 通过；快照可创建/恢复 | PASS | 导出恢复 508ms、works=2 chars=1；快照恢复 504ms；`TestMigrationRoundTrip` |
| 9 | E2E-08 一致性：改名/换势力/改境界后列表、势力详情、榜单席位、检索同步 | PASS | `consistency.spec.ts` |
| 10 | README 步骤可复现（启动/门禁/性能/发布与恢复） | PASS | README 新增「本项目」章节；本次门禁与演练按该步骤执行 |

## 结论

CHG-012 达到 proposal 的全部验收标准：跨回收站引用定位、未保存守卫、状态与错误、a11y、键盘、性能、安全与发布演练均有可复现证据；审查发现的阻断/高优先级问题已修复并回归。可归档并更新 Spec/PG/契约/roadmap。已知限制见 `verification.md` §4。
