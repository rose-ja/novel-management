# Vibe Coding Project Template

一套面向从零开发与持续迭代的、工具无关的 AI 辅助软件工程模板。

## 适用范围

- 从一个产品想法开始的新项目
- 已有项目中的中大型功能开发
- Web 全栈、AI 应用、Agent、内部工具和个人产品
- Codex、Claude Code、Cursor、Pi 等 AI 编程工具

## 核心原则

1. 先明确产品与验收标准，再设计和编码。
2. 项目级长期文档与单次变更文档分离。
3. 每个变更有明确范围、任务、DoD 和验证证据。
4. AI 不得伪称执行过命令、测试或部署。
5. 高风险操作、需求变更、架构变更和生产发布由人确认。
6. 流程按风险裁剪，不为小任务执行完整项目流程。

## 快速开始

### A. 创建新项目

1. 复制本模板到新仓库。
2. 阅读 `workflow/00-operating-manual.md`。
3. 填写 `project/PROJECT-SETUP.md`。
4. 执行 `python scripts/init_project.py`，完成项目名和基础占位符替换。
5. 依次执行 `workflow/01-project-bootstrap.md` 中的项目级阶段。
6. 在 `changes/` 中创建第一个变更，按 `workflow/02-change-cycle.md` 执行。

### B. 接入已有项目

1. 不要直接重建目录或覆盖现有规范。
2. 先填写 `project/PROJECT-SETUP.md` 的“已有项目接入”部分。
3. 让 Agent 只读分析仓库，生成现状基线和风险清单。
4. 人工确认后，再建立项目级文档和变更目录。
5. 从 Level 1 或 Level 2 流程开始，不必重走完整产品探索。

## 目录导航

- `workflow/`：实际执行手册
- `project/`：项目级长期文档模板
- `changes/_template/`：每次功能变更的模板
- `prompts/`：可直接复制给 Agent 的阶段提示词
- `standards/`：Agent、Git、安全、测试等规则
- `scripts/`：创建变更、初始化和文档检查脚本
- `examples/`：示例文件

## 推荐工作方式

- 一个项目只维护一份权威的项目级事实文档。
- 一个正在开发的需求对应一个 `changes/CHG-xxx-name/` 目录。
- 变更完成后，把确认后的行为合并回 `project/specs/`，然后归档变更。
- 不要让 `README`、架构文档、提示词和代码分别维护互相矛盾的事实。

## 必须人工确认的事项

- 产品目标、MVP 范围与验收口径
- 技术栈与重大架构决策
- 数据库破坏性迁移、权限模型和安全策略
- 超出任务范围的改动
- 生产环境部署、数据操作和回滚
