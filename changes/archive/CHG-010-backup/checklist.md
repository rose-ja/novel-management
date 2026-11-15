# CHG-010 验收清单

## 导出

- [ ] 设置页「导出全量数据」可用；无作品时禁用并显示原因。
- [ ] Dialog 含「包含回收站数据」Checkbox，默认勾选。
- [ ] 生成中禁止重复提交，显示进度与阶段。
- [ ] 成功后显示文件名、大小、时间；可下载、复制路径；Toast 提示。
- [ ] 失败显示中文原因并可重试；不产生残缺 zip。
- [ ] zip 含 `manifest.json`、`checksums.sha256`、`README.txt`、`data/app.dump`、`images/`。
- [ ] `includes_trash` 与包内容一致；不含回收站的包无软删除根。
- [ ] manifest 记录 export/schema/app 版本、works、counts、images。

## CLI 恢复

- [ ] `restore <zip>` 在空环境恢复成功；摘要与 manifest 一致。
- [ ] `restore-snapshot <dir>` 恢复成功；旧 manifest 缺版本时兼容并告警。
- [ ] 版本不兼容、checksum 失败、zip 损坏、磁盘不足时拒绝且不覆盖数据。
- [ ] 非空实例无 `--yes` 时拒绝并提示。
- [ ] 恢复前自动生成 `cli_restore` 快照；失败保留原数据与图片并输出回滚命令。
- [ ] 恢复后写 `last_restore_at`，检索可用。

## 设置页

- [ ] 快照列表显示时间、来源、大小、状态、路径；路径可复制。
- [ ] 提供 `docker compose run --rm cli restore <zip>` 与 `restore-snapshot <dir>` 命令及注意事项。
- [ ] 系统信息显示版本、数据目录、访问地址、最近导出/恢复。
- [ ] 加载/空/错误状态完整；无英文堆栈与占位文本。
- [ ] 1024/1280 无横向溢出；axe 无 serious/critical。

## 门禁

- [ ] gofmt/vet/build/test PASS。
- [ ] 集成测试 PASS（或记录 BLOCKED 原因）。
- [ ] 前端 typecheck/lint/test PASS。
- [ ] Docker 构建 + 全量 Playwright PASS。
- [ ] 空环境恢复演练记录耗时与结果。
- [ ] SPEC/PG/契约/数据模型/roadmap 回写并归档。
