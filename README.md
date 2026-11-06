# Vibe Coding Project Template

一套面向从零开发与持续迭代的、工具无关的 AI 辅助软件工程模板。适用于 Codex、Claude Code、Cursor、Pi 等工具。

## 核心原则

1. 需求、UX/UI、架构和实现分阶段确认，不让 Agent 自行补全关键决策。
2. 项目级长期文档与单次变更文档分离。
3. 任何已确认信息发生冲突，都必须先解决冲突，再继续下游工作。
4. 需求变化必须同步更新其上游依据、下游规格和变更历史，不能只改一份文档。
5. 每个变更有明确范围、任务、DoD 和真实验证证据。
6. 作者与独立审查者分离；测试、审查、人工验收各自独立。
7. 流程按风险裁剪，小任务不必执行完整项目流程。

## 快速开始

### 新项目

1. 复制模板到新仓库。
2. 阅读 `workflow/00-operating-manual.md`。
3. 填写 `project/PROJECT-SETUP.md`，确认 `project/constitution.md`。
4. 执行 `python scripts/init_project.py`（先阅读脚本说明并检查替换结果）。
5. 按 `workflow/01-project-bootstrap.md` 执行 P0～P8，并调用对应 `prompts/P*.md`。
6. 通过 P8 后，使用 `python scripts/new_change.py CHG-001 feature-name` 创建首个变更。
7. 按 `workflow/02-change-cycle.md` 执行 F1～F10，并调用对应 `prompts/F*.md`。

### 已有项目接入

1. 不要覆盖现有规范或重建工程。
2. 填写 `project/PROJECT-SETUP.md` 的 Existing 部分。
3. 先让 Agent 只读检查代码库、Git 状态、运行命令和现有文档。
4. 将实际情况与项目文档对齐；冲突先记录、确认、修订。
5. 根据改动风险选择 Level 1/2/3，不必重走完整产品探索。

## 目录导航与职责

- `workflow/`：阶段顺序、输入输出、通过标准、回退规则，是总操作手册。
- `prompts/`：按阶段编号的可复制提示词；每份提示词必须与 workflow 和文件目录一一对应。
- `project/`：跨变更长期有效的项目事实与规范。
- `project/page-specs/`：页面级 UX/UI 规格。
- `project/specs/`：稳定的产品行为规格，作为当前行为的长期基线。
- `project/reviews/`：项目级独立审查报告。
- `changes/_template/`：每次功能变更的任务、设计、验收和验证模板。
- `changes/CHG-xxx/`：单次变更工作区；完成后归档到 `changes/archive/`。
- `standards/`：安全、测试、Git 等工程标准。
- `scripts/`：初始化、创建变更和模板检查脚本。

## 文档权威关系

- 产品方向与 MVP：`product-brief.md`、`mvp-scope.md`（人工确认后生效）。
- 需求基线：`requirements.md`、`user-stories.md`。
- UX/UI 基线：`user-flow.md`、`design-system.md`、`page-specs/`。
- 架构与技术决策：`architecture.md`、`tech-stack.md`、`adr/`、API/数据模型文档。
- 当前系统行为：`specs/`。
- 单次变更范围：`changes/CHG-xxx/`。

若文档相互矛盾，不允许 Agent 自行选择一份作为真相。先创建冲突记录，判断冲突来源，向负责人确认，再同步更新受影响文档及变更历史。

## 必须人工确认

- 产品目标、目标用户、MVP 范围和需求冲突的最终取舍
- 关键 UX/UI 方向、品牌视觉和核心页面方案
- 技术栈与重大架构决策
- 数据库破坏性迁移、权限模型、安全策略
- 超出任务范围的改动
- 生产环境部署、数据操作和回滚
