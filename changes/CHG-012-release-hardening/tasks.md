# CHG-012 任务

- 允许文件：`frontend/app/trash/page.tsx`、`frontend/components/{trash,characters,factions,locations,system,rankings,ui}/*`、`frontend/e2e/*`、`frontend/lib/{api,messages,hooks}/*`、`backend/cmd/cli/main.go`、`backend/internal/{service,httpapi}/*_test.go`、`scripts/perf.ps1`、`README.md`、`changes/CHG-012-*`、`project/*`。
- 禁止：新增业务表/字段；改动既有迁移；引入新依赖。

## T1 跨回收站引用定位（E2E-15）

- [x] 回收站页读取 `type`/`highlight`，跨页查找、高亮、滚动与焦点；未命中提示。
- [x] 人物/势力/地点删除对话框：回收站引用链接到对应 `/trash?type=…&highlight=…`。
- [x] E2E-15：榜单席位引用回收站人物 → 删除被阻止 → 定位高亮 → 恢复榜单 → 撤席 → 删除成功。

## T2 未保存守卫统一

- [x] 势力内联编辑改用统一确认对话框。
- [x] 体系类别/节点对话框关闭与切换选中加守卫。
- [x] 榜单新建/改名与席位备注加守卫。
- [x] E2E 覆盖势力与体系脏状态关闭。

## T3 状态与错误统一

- [x] 审计主页面 loading/empty/error 与筛选保留；补齐缺口。
- [x] `PRECONDITION_REQUIRED`(428) 文案与集成断言。
- [x] E2E-12 扩展：列表接口 500 + 重试、筛选无结果、无作品空态。

## T4 可访问性全页

- [x] axe 覆盖人物/势力/榜单详情（有数据）与新建/编辑/删除对话框、回收站有数据。
- [x] 修复扫描发现的 serious/critical。

## T5 键盘主流程

- [x] 键盘新建人物、编辑保存、删除确认、回收站恢复；Esc 回焦断言。

## T6 E2E-08 一致性

- [ ] 人物改名/换势力/改境界后，人物列表、势力详情、榜单席位、检索结果同步。

## T7 性能基线

- [ ] `cli seed` 批量造数（默认 1000 人物）。
- [ ] `e2e/perf.spec.ts` 首屏测量；`scripts/perf.ps1` 采样 P95 并输出报告。
- [ ] 实测达标：首屏 ≤2s、列表 P95 ≤500ms、检索 P95 ≤150ms。

## T8 安全与发布复测

- [ ] E2E-13：CSP 下检索/详情可用、无控制台 violation。
- [ ] 复跑恶意 zip/路径穿越/伪造图片用例。
- [ ] 导出 → CLI 恢复 → 计数一致；快照创建/恢复；`cli migrate` 可回滚。
- [ ] README 步骤复核并补充性能/发布检查。

## T9 F3~F10 收口

- [ ] 全量门禁、集成、E2E、Docker、性能报告。
- [ ] 独立审查与修复；verification/review/acceptance/pr。
- [ ] 归档并回写 Spec/PG/契约/roadmap/README。
