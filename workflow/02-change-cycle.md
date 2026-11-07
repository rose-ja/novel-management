# 单次功能变更执行手册（F1～F10）

每个功能、Bug 修复或技术改造建立独立目录：`changes/CHG-001-short-name/`。使用 `python scripts/new_change.py CHG-001 short-name` 从模板创建。

## F1：任务拆解与变更设计

**提示词：** `prompts/F1-task-breakdown.md`

1. 确认用户价值、目标、范围和非目标。
2. 阅读相关 Spec、架构、ADR、AGENTS 和实际代码。
3. 检查需求与文档/代码是否冲突；冲突按总手册处理。
4. 定义允许文件、接口/数据影响、依赖、任务顺序、并行边界、测试、DoD、风险和回滚。
5. 输出 proposal/design/tasks/checklist。

通过：每项任务可执行，范围可控，验收可判定。

## F2：实现

**提示词：** `prompts/F2-implementation.md`

1. 检查 Git 分支、工作区和未提交修改。
2. 按任务顺序实施，每个任务后汇报文件和结果。
3. 不得顺手重构无关代码；超范围改动先暂停。
4. 完成后输出 Diff 摘要和已知问题。

通过：仅完成批准任务，代码变更可追踪。

## F3：开发者自测

**提示词：** `prompts/F3-self-test.md`

执行适用的 lint、format、typecheck、unit、integration、build 和回归测试。记录真实命令、退出码和证据。

通过：相关检查通过；未执行项标记 NOT RUN，阻断项标记 BLOCKED。

## F4：独立代码审查

**提示词：** `prompts/F4-code-review.md`

Reviewer 必须独立于作者，审查实际 Diff、Spec、任务卡和自测报告。检查正确性、需求一致性、异常边界、权限、安全、并发幂等、数据兼容、性能、副作用和测试。

通过：无未处理 S0；S1 有修复计划或人工豁免。

## F5：修复与复审

**提示词：** `prompts/F5-fix-and-rereview.md`

1. 作者逐条分析审查发现，修复 S0/S1。
2. 重新执行受影响测试。
3. 独立 Reviewer 对修复 Diff 复审。
4. 不成立的问题必须给出依据，并由人确认是否豁免。

通过：S0 清零，S1 已修复或批准豁免，复测通过。

## F6：集成与 E2E

**提示词：** `prompts/F6-e2e.md`

至少覆盖核心成功路径、相关异常路径、权限/数据隔离（适用时）、边界条件和受影响旧功能回归。保存日志、截图和报告。

通过：关键验收场景有真实证据；失败项已修复或明确阻断。

## F7：人工验收

**提示词：** `prompts/F7-human-acceptance.md`

人工检查业务行为、UI/UX、测试证据、范围、兼容性、安全风险和回滚。结论为 ACCEPT / REWORK / REPLAN。

通过：验收人明确 ACCEPT。

## F8：PR/MR

**提示词：** `prompts/F8-pr.md`

整理目标、Spec、改动范围、测试命令和结果、E2E 证据、风险、回滚、截图及评审关注点。未经授权不推送、不创建远程 PR。

通过：PR/MR 描述完整，CI 状态明确。

## F9：合并与发布

**提示词：** `prompts/F9-release.md`

1. 合并前确认审批和 CI。
2. 按发布方案执行部署；生产操作必须人工授权。
3. 执行部署后 Smoke Test，检查日志和关键业务行为。
4. 失败时按预案回滚或停止发布。

通过：发布状态、Smoke Test 和回滚状态有记录。

## F10：Spec 同步与归档

**提示词：** `prompts/F10-archive.md`

1. 将已实现且验收的行为同步到 `project/specs/`，并按 `project/specs/README.md` 更新状态。
2. 更新受影响架构、API、数据模型、ADR 和 roadmap。
3. 补全 verification/acceptance。
4. 将变更目录移动到 `changes/archive/`。
5. 检查文档与代码一致，列出遗留技术债。

通过：长期文档反映真实系统状态，变更已归档。
