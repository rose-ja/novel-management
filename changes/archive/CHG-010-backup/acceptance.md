# CHG-010 验收记录（F7）

- 日期：2026-11-15
- 基线：`f9429ae` + 回归
- 验收依据：`proposal.md` Acceptance Criteria、`project/specs/backup/specs.md`、`project/page-specs/PG-13-settings-backup.md`

| # | 验收项 | 结果 | 证据 |
|---|---|---|---|
| 1 | 设置页可导出；Dialog 询问是否包含回收站，默认勾选；无作品禁用并说明 | PASS | E2E `settings-backup.spec.ts` 3 用例；键盘用例 |
| 2 | 后台导出；成功展示文件名/大小/时间，可下载/复制路径；失败中文原因与重试，不产生残缺 zip | PASS | E2E 导出成功卡片+下载 200；`TestExportFailureRecordsErrorCode` |
| 3 | 含/不含回收站包结构与 `includes_trash` 一致；不含包无软删除根 | PASS | `TestExportArchiveStructureAndChecksums`；临时库清理 SQL |
| 4 | 包含 manifest/checksums/data/app.dump/images/README；manifest 记录版本/works/counts | PASS | 真实 zip 解包记录（verification §2） |
| 5 | 空环境 `cli restore` 恢复数据、图片、关系与检索；摘要与 manifest 一致 | PASS | F9 Docker 演练：zip 恢复 515ms，健康检查通过 |
| 6 | 版本/checksum/zip/磁盘异常拒绝且不覆盖 | PASS | `TestRestoreArchiveRejectsNewerSchema`、`...ChecksumMismatch`、`ReadZipFile` 限额、磁盘预检 |
| 7 | 非空需 `--yes`；恢复前快照；失败保留原数据并输出回滚命令 | PASS | `TestRestoreRequiresForceOnNonEmptyInstance`；`TestRestoreArchiveRestoresImagesAndWritesSnapshot`；演练日志 |
| 8 | `restore-snapshot` 可恢复；旧 manifest 兼容告警；写 `last_restore_at` | PASS | 集成恢复用例；F9 快照恢复 505ms；settings API |
| 9 | 快照列表显示时间/来源/大小/状态/路径；路径可复制；提供两条命令与注意事项 | PASS | E2E `settings-snapshots.spec.ts`；PG-13 页面 |
| 10 | 系统信息显示版本、数据目录、访问地址、最近导出/恢复 | PASS | `shell.spec.ts` 设置页；`/api/v1/settings` 实测 |
| 11 | 失败提示中文、无堆栈；重试边界明确 | PASS | 错误码映射、E2E 无 console 错误 |
| 12 | 全量门禁、集成、前端单测、Docker、Playwright、空环境演练通过 | PASS | verification.md §1/§3 |

## 结论

CHG-010 达到 proposal 的全部验收标准，可归档并进入 CHG-011。已知限制（恢复后非事务步骤失败需按命令回滚）已在 review/verification 中记录。
