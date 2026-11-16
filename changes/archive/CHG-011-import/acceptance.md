# CHG-011 验收记录（F7）

- 日期：2026-11-16
- 基线：`a79d1c2` + 全量回归
- 验收依据：`proposal.md` Acceptance Criteria、`project/specs/migration/specs.md`、`project/page-specs/PG-11-import-wizard.md`

| # | 验收项 | 结果 | 证据 |
|---|---|---|---|
| 1 | 合法 Markdown + zip 上传后创建记录并进入校验；预览显示八类计数与图片匹配 | PASS | 真实演练：validated/0 错误；计数 1/1/1/1/2/3/1/1；E2E-09 |
| 2 | 非 `.md`/非 zip、超大、单图超限、磁盘不足、超时均有中文错误且不写业务数据 | PASS | 前端类型/大小校验；服务端扩展名与大小限制；超时 120 分钟；磁盘预检（可用量 >0 时） |
| 3 | 版本不符、未知区块/字段、必填、重名、坏引用、类别/节点非法、图片问题进入错误清单并禁止确认 | PASS | 解析/校验单测；E2E 错误态与缺图阻断 |
| 4 | 确认前自动快照；快照失败阻止导入 | PASS | `TestImportSnapshotFailureRollsBack`；报告 `snapshot_id` |
| 5 | 原子写入八类实体、关系、图片与检索；成功后自动切换当前作品；报告与预览一致 | PASS | 真实演练；`TestImportHappyPathAndReview`；E2E 顶栏断言 |
| 6 | 提交失败无半成品；重复 commit 返回同一结果 | PASS | 快照失败回滚；失败清理图片与 staging；重复提交幂等 |
| 7 | 导入实体 `pending`，单条/批量确认仅作用 pending，写 reviewed 信息并刷新检索 | PASS | 集成单条 + 批量；`POST /reviews/batch`、`/:resource/:id/review` |
| 8 | 批量确认已 confirmed 计 skipped，不存在 ID 整批失败 | PASS | `TestImportHappyPathAndReview` 批量与 404 用例 |
| 9 | 五步步骤条、错误清单、右侧摘要、确认 Dialog、报告、校对分组与空状态完整；1024 不溢出 | PASS | E2E `import.spec.ts`；视觉基线 1024/1280 |
| 10 | axe 无 serious/critical；`aria-current`、`progressbar`、`aria-live`、键盘选择文件 | PASS | `a11y.spec.ts` 导入页；`keyboard.spec.ts` 步骤条与文件选择 |
| 11 | 全量门禁、集成、前端单测、Docker、Playwright 通过并记录 | PASS | verification.md §1（115/115） |

## 结论

CHG-011 达到 proposal 的全部验收标准，roadmap DoD 的大样本（200/100/100/20/500）导入实测通过（提交 361ms、计数一致），可归档并进入 CHG-012。已知限制（上传不可取消、200 项校对上限、硬崩溃图片残留）已在 verification §6 记录。
