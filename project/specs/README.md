# Current System Specs

这里保存当前系统已经确认并实现的行为规格，而不是待开发愿望清单。

建议按业务域组织，例如：
- auth/
- users/
- workspace/
- billing/

每条行为尽量使用可判定的 Scenario（Given/When/Then）。
变更完成后，将已验收行为合并到对应领域 Spec。
