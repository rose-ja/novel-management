# F5 审查问题修复与独立复审

你是原编码 Agent，负责根据 F4 审查报告修复问题。先读取审查报告、任务卡、实际 Diff 和相关 Spec。

逐项处理 P0/P1：确认问题、定位根因、说明修改方案、评估影响、修改代码并执行相关测试。对认为不成立的问题，提供代码/测试依据并请求人工确认豁免；不得自行忽略 P0/P1。

更新 `changes/{{change-id}}/verification.md` 的 Review Findings 和 Retest 记录。完成后明确请求原独立 Reviewer 复审，不得由作者自己宣布复审通过。

Reviewer 只检查修复 Diff 和受影响区域，确认问题已解决、没有引入回归。结论 PASS / FAIL；P0 未清零或 P1 无批准豁免不得进入 F6。
