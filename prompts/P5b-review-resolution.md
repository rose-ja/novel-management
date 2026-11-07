# P5b 红队问题处理与复审

你是原方案作者/架构负责人。请读取 `project/reviews/P5-red-team-review.md` 及 P2～P4 所有相关源文件。

逐条处理每个 Finding：
- ACCEPT：接受并修改。
- PARTIAL：部分接受，说明边界。
- REJECT：不接受，提供可验证依据；涉及产品/架构风险时必须由人批准。
- DEFER：延期，说明风险、责任人和触发条件。

强制要求：
1. S0 必须修复；S1 必须修复或取得人工书面豁免。
2. 修改应落在权威源文件，而非只改审查报告或下游摘要。
3. 若改动产品目标/MVP，更新 `product-brief.md` / `mvp-scope.md`，并重新检查 requirements。
4. 若改动需求规则，更新 `requirements.md` / `user-stories.md`，并检查 page specs、architecture 和 specs。
5. 若改动 UX/UI，更新 `user-flow.md`、`design-system.md`、相关 `page-specs/`，并检查架构影响。
6. 若改动架构，更新对应 architecture/API/data/tech-stack/ADR，并评估 roadmap/PoC。
7. 在每个被修改文档的变更记录中写明 Finding ID、决策、原因和日期。
8. 不要自动覆盖人工确认的决策；需要新决策时暂停提问。

完成后更新 `project/reviews/P5-red-team-review.md` 的处理状态，并请求独立 Reviewer 复审 S0/S1。输出修改文件、未解决风险、人工豁免和复审结论。S0 未清零或 S1 未有批准豁免，不得宣布 P5 通过。
