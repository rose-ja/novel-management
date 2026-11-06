# 执行总手册

## 1. 每个阶段都使用同一张执行卡

开始阶段前，明确：

| 项目 | 必须填写 |
|---|---|
| 阶段编号 | 例如 P2 / F4 |
| 负责人 | 人 / Product Agent / Architect Agent / Coding Agent / QA Agent |
| 输入 | 具体文件路径，不写“参考上下文” |
| 动作 | 可执行步骤 |
| 输出 | 具体文件路径或代码变更 |
| 通过标准 | 可观察、可判定 |
| 失败回退 | 返回哪个阶段 |
| 人工确认 | 是 / 否，确认什么 |

禁止只说“分析一下”“优化一下”“确保没问题”。

## 2. 三种流程档位

### Level 1：轻量任务
适用：文案、样式、小配置、局部 Bug，通常不改变模块边界和数据契约。

流程：理解任务 → 定位代码 → 实施 → 相关测试 → Diff Review → 提交。

必需产出：简短任务说明、测试结果、变更摘要。

### Level 2：标准变更
适用：新增页面、API、业务逻辑、数据库字段、模块内功能。

流程：需求澄清 → 变更设计 → Spec/Checklist/Tasks → 实现 → 自测 → 独立审查 → E2E/集成验证 → 人工验收 → PR → Spec 同步归档。

必需产出：变更目录中的 proposal、design、tasks、checklist、verification。

### Level 3：完整新项目或重大重构
适用：从零开发、核心架构变更、复杂 Agent 系统、跨多个业务域的改造。

流程：P0 产品探索 → P1 资料整理 → P2 需求定义 → P3 UX/UI → P4 架构 → P5 红队审查 → P6 Spec 与路线图 → P7 工程初始化 → P8 启动验收 → Level 2 变更循环。

## 3. 项目级阶段执行表

| 阶段 | 负责人 | 输入 | 输出 | 通过标准 | 失败回退 |
|---|---|---|---|---|---|
| P0 产品探索 | 人 + Product Agent | 产品想法、目标用户 | product-brief、mvp-scope、research | 用户、痛点、价值、MVP 明确 | 继续探索 |
| P1 资料整理 | 人 + Research Agent | PRD、竞品、代码、规范 | sources、project-context | 来源可访问，事实与假设区分 | 补资料 |
| P2 需求定义 | Product Agent + 人 | P0/P1 产物 | requirements、user-stories | 范围、非目标、边界、验收明确 | 继续澄清 |
| P3 UX/UI | UX Agent + 人 | PRD、需求 | user-flow、design-system、page-specs | 核心流程和状态完整，人工确认 | 返回 P2/P3 |
| P4 架构设计 | Architect Agent | 需求、设计、约束 | architecture、ADR、API/DB 设计 | 文件/模块级方案、风险、测试、回滚齐全 | 返回 P2/P4 |
| P5 对抗审查 | 独立 Reviewer | P2-P4 文档 | review report | P0 清零，P1 有处理决定 | 返回对应阶段 |
| P6 规格与路线图 | Planner Agent + 人 | 已通过文档 | specs、MVP increments | 每个增量可验证、依赖清楚 | 返回 P2/P4/P6 |
| P7 工程初始化 | Coding Agent | 架构、规范、命令 | 可运行工程、AGENTS、CI | install/lint/test/build/smoke 实际通过 | 修复初始化 |
| P8 启动验收 | QA Agent + 人 | 工程、测试策略 | bootstrap verification | 核心技术链路真实跑通 | 返回 P7 |

## 4. 变更级阶段执行表

| 阶段 | 负责人 | 输入 | 输出 | 通过标准 | 失败回退 |
|---|---|---|---|---|---|
| F1 任务拆解 | 主 Agent | 当前 Spec、需求、代码库 | proposal/design/tasks/checklist | 文件范围、步骤、DoD 明确 | 重新拆解 |
| F2 实现 | Coding Agent | 已批准任务卡 | 代码变更 | 范围受控、代码可运行 | 停止并请求决策 |
| F3 自测 | Coding Agent | Diff、测试策略 | 自测报告 | 相关 lint/type/unit/build 通过 | F2 |
| F4 独立审查 | Reviewer Agent | Diff、Spec、任务卡、测试结果 | review report | 无未处理 P0 | F5 |
| F5 修复复审 | Coding Agent + Reviewer | 审查问题 | 修复记录、复审结论 | P0 清零，P1 有决定，复测通过 | F2/F5 |
| F6 集成/E2E | QA Agent | DoD、验收场景 | verification | 主链路、异常路径真实通过 | F2/F5 |
| F7 人工验收 | 人 | 页面/接口/证据/风险 | acceptance decision | 业务正确、范围正确、可回滚 | 返回对应阶段 |
| F8 PR/MR | 主 Agent | Diff、报告、证据 | PR/MR | 描述完整、一个交付单元 | 补充材料 |
| F9 合并/发布 | 人 + Agent | 已批准 PR、发布方案 | 发布记录 | CI、部署后 Smoke Test 通过 | 回滚/修复 |
| F10 Spec 同步归档 | 主 Agent + 人 | 已合并实现、验证结果 | 更新后的 Spec、归档目录 | 长期规格反映真实系统 | 补齐文档 |

## 5. 质量门禁

- G0 产品门禁：产品目标、目标用户、MVP 获得人工确认。
- G1 需求门禁：关键需求可判定，无阻断性未决问题。
- G2 设计门禁：用户流程、状态和视觉基线确认。
- G3 架构门禁：技术方案、模块边界、风险和回滚明确。
- G4 工程门禁：工程启动、检查、构建和 CI 可运行。
- G5 实现门禁：任务卡范围内完成，必要自测通过。
- G6 审查门禁：无未处理 P0，P1 有明确结论。
- G7 验证门禁：真实 E2E/集成证据通过。
- G8 人工门禁：业务结果和风险由人确认。
- G9 发布门禁：合并、部署和 Smoke Test 完成。
- G10 归档门禁：Spec 同步、变更归档完成。

## 6. 变更范围控制

当 Agent 发现任务卡范围外的必要改动时，必须暂停并输出：

1. 发现的问题。
2. 不修改会造成什么后果。
3. 涉及文件和模块。
4. 最小修改建议。
5. 是否需要修改 Spec/Architecture。
6. 等待人工批准。

未经批准不得扩大范围。

## 7. 测试证据规则

每项测试记录：
- 实际执行命令或操作步骤
- 执行时间
- 实际结果
- 预期结果
- 通过/失败
- 日志、截图、报告路径

没有执行的测试标记为 `NOT RUN`，不得写成 PASS。环境阻塞标记为 `BLOCKED`，不得伪造证据。
