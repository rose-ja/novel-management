# P6 Spec 与 MVP 路线图

你是规格与交付规划负责人。读取所有已批准的产品、需求、UX/UI、架构、ADR、P5 审查结论，以及 `project/specs/README.md` 的 Spec 状态与结构规则。

请先检查文档一致性；发现冲突暂停并按 `workflow/00-operating-manual.md` 的冲突规则处理。

完成：
1. 将系统应实现的目标行为整理到 `project/specs/`，按 `project/specs/README.md` 组织；每条 Spec 有稳定 ID、来源 Requirement/User Story/Page Spec ID、行为描述、边界和验收场景，新建或更新的 Spec 状态一律为 `planned`，不得写成当前已具备能力。
2. 将 MVP 拆成可验证的垂直切片变更，优先覆盖端到端用户价值。
3. 每个变更明确目标、关联 Spec ID、范围、非目标、模块/文件预期、DoD、测试、依赖、复杂度、风险和回滚。
4. 依赖图不得有环；说明每个增量的验证方式。
5. 建立需求 → User Story → Page Spec/API/Data → Spec → Change → Acceptance 的追踪矩阵。
6. 检查 UI 页面、交互状态和设计系统任务是否被纳入对应变更，不能只拆后端任务。

输出 `project/specs/` 和 `project/roadmap.md`。不为凑数量强行拆分；建议先规划可交付的首批变更。等待人工确认。
