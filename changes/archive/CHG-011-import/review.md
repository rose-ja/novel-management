# CHG-011 独立审查与修复记录（F4/F5）

- 审查范围：`83e5547..cd868a4`（CHG-011 全部改动）
- 审查方式：独立只读审查（非实现者视角），随后由实现者逐条核对、修复并回归
- 结论：发现 1 个 P0、4 个 P1；已全部修复并补充用例；P2 处理 3 项，其余记录为已知限制

## 1. P0 问题与处置

| # | 问题 | 处置 | 验证 |
|---|---|---|---|
| 1 | 两级节点校验用「父级已有子节点」标记判定层级，同一父级下第二个子节点被误报「最多两级」 | 改为 `declared`/`topLevel` 两张表：父级必须已声明且为一级节点；兄弟节点仅按 `(类别, 父级, 名称)` 去重 | 新增 `TestValidateAllowsSiblingNodesUnderSameParent` |

## 2. P1 问题与处置

| # | 问题 | 处置 | 验证 |
|---|---|---|---|
| 1 | 人物引用二级节点时只校验父节点与「有子节点」标记，不校验子节点本身，坏引用静默丢失 | `nodeExists` 直接校验 `cat\x00parent\x00name`；一级引用不再接受子节点名 | 新增 `TestValidateRejectsMissingChildNodeReference` |
| 2 | `GetForUpdate` 在事务外执行即释放，状态置 running 无 CAS，并发 commit 可双写、失败覆盖 completed | 新增 `startCommit`：单事务内锁 `app_settings` 与导入行，校验状态/错误数/其他导入/导出后置 running；完成后才进入快照与写入 | 集成 happy path + 重复提交幂等 PASS |
| 3 | 导入与导出互斥检查未共享锁，二者可同时通过 | 导入提交与导出创建均先 `SELECT … FOR UPDATE app_settings`，再检查对方任务 | 新增 `TestImportAndExportAreMutuallyExclusive` |
| 4 | 失败与中断分支不清理 staging，6GB 级上传残留 | `failRun` 统一清理 `tmp/imports/<id>`；`MarkInterrupted` 启动时清空 `tmp/imports/*` | 集成快照失败用例 PASS |
| 5 | zip 条目 `./images/x.png` 在清单中规范化为 `images/x.png`，提交读取时精确匹配失败 | `backup.ReadZipFile` 统一规范化条目名后再匹配 | 新增 `TestReadZipFileMatchesDotSlashEntry` |

## 3. P2 问题与处置

| # | 问题 | 处置 |
|---|---|---|
| 1 | 图片「超过 10MB」只在摘要计数，明细未展示 | 校验页明细列表补充 `too_large` 文件名 |
| 2 | 图片路径未拒绝 `C:/x.png` 盘符绝对路径 | `NormalizeImagePath` 增加盘符判断并报路径错误 |
| 3 | 校对清单固定 200 项、上传不可取消、轮询不自动重试 | 记录为已知限制（见 verification §5） |
| 4 | 未知字段错误行号取 `###` 行；体系类别行按自由文本接受 | 属模板语义选择，保留现状并在模板文档说明 |

## 4. 复审结果

- 修复后执行 F6 全量回归：后端单测、集成（22.2s）、前端类型/lint/单测（91）、Docker 构建、Playwright 115 用例全部 PASS。
- 未发现新增 P0/P1。
