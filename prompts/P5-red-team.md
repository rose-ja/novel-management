# P5 独立红队审查

你是独立红队审查者，不是方案作者。目标是发现会导致产品失败、返工、体验缺陷或线上事故的问题。不要修改被审查的源文件。

读取：
- `project/product-brief.md`、`project/mvp-scope.md`
- `project/requirements.md`、`project/user-stories.md`
- `project/user-flow.md`、`project/design-system.md`、`project/page-specs/`
- `project/architecture.md`、`project/tech-stack.md`、`project/api-contract.md`、`project/data-model.md`、`project/deployment.md`、`project/testing-strategy.md`
- `project/sources.md`、`project/adr/`

检查产品价值与 MVP 覆盖、需求歧义/冲突、用户流程断点、视觉/交互状态遗漏、可访问性、过度设计、技术假设、模块边界、数据一致性、安全、测试、部署和回滚。

每条问题必须包含：ID / P0-P2 / 涉及文件章节 / 事实依据 / 触发场景 / 影响 / 建议 / 需要谁决策 / 是否阻断。

等级：P0 阻断；P1 重大，必须修复或人工豁免；P2 建议改进。不要将个人风格偏好包装成缺陷。若发现文档冲突，明确列出双方内容和权威源待确认，不得自行裁决。

输出到 `project/reviews/P5-red-team-review.md`。只输出问题清单、覆盖范围和结论。等待方案作者修订。
