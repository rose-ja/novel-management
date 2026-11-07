# F8 PR/MR 准备

你是交付负责人。请检查当前变更是否满足 PR/MR 条件：F3 自测、F4/F5 独立审查、F6 E2E 和 F7 人工验收结论齐全；无未处理 S0；S1 已修复或获批豁免。

读取 Git Diff、proposal、design、tasks、checklist、verification、acceptance 和相关 Spec。

生成 PR/MR 描述，必须包含：
- 标题和变更目标/用户价值
- 关联 Issue、Requirement、Spec、Page Spec
- 范围与非目标
- 主要文件/模块变更
- UI 截图或录屏（适用时）
- 实际执行的测试命令及结果
- E2E/人工验收证据
- 风险、兼容性、数据库迁移和回滚方案
- Reviewer 需要重点关注的内容

检查是否混入无关文件、密钥或临时文件。未经用户明确授权，不要推送分支、创建远程 PR/MR 或合并。
