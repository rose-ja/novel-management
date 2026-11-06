# Git Workflow

- 主分支保持可构建、可测试。
- 一个变更使用独立分支。
- 一个 PR 尽量对应一个可审查交付单元。
- 提交信息遵循团队约定；无约定时采用 Conventional Commits。
- 不提交 `.env`、密钥、构建产物和本地缓存。
- 合并前必须通过仓库要求的 CI。
- 不允许 Agent 擅自 force push、删除分支或覆盖他人工作。
- Worktree 并行时，每个 Agent 有明确文件所有权和分支。
