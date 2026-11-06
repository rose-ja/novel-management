# 执行总手册

## 1. 使用原则

本模板分为项目级（P）和变更级（F）两层。`workflow/` 定义流程，`prompts/` 提供可复制提示词，`project/` 和 `changes/` 保存产物。三者必须使用相同阶段编号和文件路径。

每次调用提示词前，补齐占位符，并明确 Agent 可读、可写的文件范围。不要让 Agent 只凭聊天上下文工作。

## 2. 三种流程档位

| 档位 | 适用范围 | 流程 |
|---|---|---|
| Level 1 轻量 | 文案、样式、小配置、局部 Bug，不改模块边界/API/数据契约 | 理解 → 定位 → 实施 → 相关测试 → Diff Review → 提交 |
| Level 2 标准 | 新页面、API、业务逻辑、数据库字段、模块内功能 | F1～F10；按风险裁剪，但必须保留自测、独立审查、验证和人工验收 |
| Level 3 完整 | 从零项目、核心架构变化、复杂 Agent 系统、跨域重构 | P0～P8，再对每个交付增量执行 F1～F10 |

## 3. 项目级阶段执行表

| 阶段 | 负责人 | 主要输入 | 必须输出 | 通过标准 | 回退/循环 |
|---|---|---|---|---|---|
| P0 产品探索 | 人 + Product Agent | 想法、用户、场景 | `product-brief.md`、`mvp-scope.md`、`research.md` | 用户、痛点、价值、MVP 明确并获确认 | 继续探索 |
| P1 资料整理 | Research Agent | 原始资料、P0 结论 | `sources.md`、`project-context.md` | 来源可追溯，事实/假设/冲突分开 | 补资料；必要时返回 P0 |
| P2 需求定义 | Product Agent + 人 | P0/P1 文档 | `requirements.md`、`user-stories.md`、`glossary.md` | 范围、规则、边界、验收明确；冲突已裁决 | 更新 P0/P1 文档后重审 P2 |
| P3 UX/UI | UX Agent + 人 | 已确认需求 | `user-flow.md`、`design-system.md`、`page-specs/` | 流程、页面、状态、视觉规范完整并获确认 | 需求缺口返回 P2；设计迭代留在 P3 |
| P4 架构设计 | Architect Agent | 需求、UX/UI、约束 | architecture、tech-stack、API、data、testing、deployment、ADR | 方案可落地，风险/测试/回滚明确 | 需求冲突返回 P2；UX 影响返回 P3 |
| P5 红队审查与修订 | 独立 Reviewer + 方案作者 | P2～P4 全部基线 | `reviews/P5-red-team-review.md`、修订后的文档 | P0 清零，P1 已修复或人工豁免，复审通过 | 回到对应上游阶段修订并复审 |
| P6 Spec 与路线图 | Planner Agent + 人 | 已批准需求/设计/架构 | `specs/`、`roadmap.md` | 需求→Spec→增量→验收可追踪，依赖无环 | 返回对应文档修订 |
| P7 工程初始化 | Coding Agent | 架构、技术栈、工程规范 | 可运行工程、`AGENTS.md`、CI | 安装、lint、typecheck、test、build 等实际验证 | 修复后重跑 |
| P8 启动验收 | QA Agent + 人 | 工程、测试策略 | `bootstrap-verification.md` | 最小技术链路真实跑通，阻断项清零 | 返回 P7 |

## 4. 变更级阶段执行表

| 阶段 | 负责人 | 必须输出 | 通过标准 | 对应提示词 |
|---|---|---|---|---|
| F1 拆解 | 主 Agent / Planner | proposal、design、tasks、checklist | 范围、文件、DoD、风险明确 | `F1-task-breakdown.md` |
| F2 实现 | Coding Agent | 代码 Diff | 仅实现批准任务 | `F2-implementation.md` |
| F3 自测 | Coding Agent | verification | 相关静态检查/单测/构建真实执行 | `F3-self-test.md` |
| F4 独立审查 | 独立 Reviewer | review findings | 无未处理 P0；P1 有决定 | `F4-code-review.md` |
| F5 修复复审 | Coding Agent + Reviewer | 修复记录、复审结论 | P0 清零、P1 修复或豁免、复测通过 | `F5-fix-and-rereview.md` |
| F6 集成/E2E | QA Agent | verification 中的 E2E 证据 | 主链路、异常路径、回归验证通过 | `F6-e2e.md` |
| F7 人工验收 | 人 + 验收 Agent | acceptance | 业务、UI、范围和风险通过 | `F7-human-acceptance.md` |
| F8 PR/MR | 主 Agent | PR/MR 描述与证据 | 变更、Spec、测试、风险、回滚清晰 | `F8-pr.md` |
| F9 合并/发布 | 人 + 发布 Agent | 发布记录、Smoke Test | 授权、CI、部署后检查通过 | `F9-release.md` |
| F10 Spec 同步归档 | 主 Agent + 人 | 更新 Spec、归档变更 | 文档反映已实现且已验证行为 | `F10-archive.md` |

## 5. 冲突与变更传播规则（强制）

当任一阶段发现信息冲突、需求缺口或上游假设不成立时：

1. **暂停受影响的下游工作**，不要继续设计或编码。
2. 创建冲突记录：冲突 ID、涉及文件/章节、各自内容、来源、影响、待决策问题。
3. 判断冲突类型：产品目标/MVP、需求规则、UX/UI、架构/技术、实现与文档不一致。
4. 指出需要由谁决策；涉及产品取舍、预算、用户体验优先级时必须询问人。
5. 得到明确决定后，更新权威源文件，而不只是更新当前阶段文档。
6. 同步更新所有受影响的上游/下游文件，并在文档的变更历史或决策记录中注明原因、日期和影响。
7. 对受影响阶段重新执行检查；重大变化重新走红队审查。

**示例：** P2 需求澄清发现 MVP 与产品目标冲突，不能只改 `requirements.md`。应先请人确认，再同步 `product-brief.md` / `mvp-scope.md`，必要时更新 `research.md` / `project-context.md`，最后重新生成 `requirements.md` 和 `user-stories.md`。

若多个文档冲突，按“人工确认的最新决策 > 对应领域权威基线 > 下游派生文档 > Agent 推断”处理；但 Agent 不得自行替人做产品或架构决策。

## 6. 质量门禁

- G0 产品门禁：目标用户、痛点、价值、MVP 获得人工确认。
- G1 需求门禁：关键需求可判定，阻断性未决问题清零。
- G2 UX/UI 门禁：核心流程、页面结构、设计系统和关键页面获得确认。
- G3 架构门禁：技术方案、模块边界、风险、测试和回滚明确。
- G4 规划门禁：Spec、增量、依赖和验收可追踪。
- G5 工程门禁：工程启动、检查、构建和 CI 可运行。
- G6 实现门禁：任务卡范围内完成，开发者自测通过。
- G7 审查门禁：无未处理 P0，P1 有明确结论。
- G8 验证门禁：集成/E2E 有真实证据。
- G9 人工门禁：业务和 UI/UX 验收由人确认。
- G10 发布归档门禁：发布检查完成，Spec 同步、变更归档完成。

## 7. 测试证据规则

每项测试记录实际命令/操作、环境、预期、实际结果、PASS/FAIL、证据路径。未执行写 `NOT RUN`；环境阻塞写 `BLOCKED`。不得用代码阅读或 Agent 自述代替实际执行证据。
