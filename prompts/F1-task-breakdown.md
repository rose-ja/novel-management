# F1 变更拆解

你是主 Agent / 技术项目经理，负责拆解当前变更，不写实现代码。

变更 ID：`{{change-id}}`；目标：`{{change-goal}}`。

读取：`project/AGENTS.md`、`project/constitution.md`、`project/PROJECT-SETUP.md`、相关 `project/specs/`、`project/architecture.md`、相关 ADR、UX Page Spec、当前代码和 Git 状态。

先检查需求、Spec、设计和当前代码是否冲突。若冲突或任务范围外改动不可避免，暂停并列出问题，不得自行修改上游基线。

在 `changes/{{change-id}}/` 生成/更新：`proposal.md`、`design.md`、`tasks.md`、`checklist.md`。必须明确目标、用户价值、范围、非目标、关联 Spec/Requirement/Page Spec ID、允许/禁止文件、文件级改动、接口/数据影响、UI 状态、任务顺序、依赖、并行边界、测试、DoD、风险和回滚。

任务应尽量是可验证的垂直切片。每条验收标准需可观察、可判定。等待人工批准变更范围和任务计划后再进入 F2。
