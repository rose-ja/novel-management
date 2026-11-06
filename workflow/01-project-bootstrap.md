# 从零项目启动执行手册（P0～P8）

每个阶段调用同编号 `prompts/` 提示词。提示词是执行指令，本文档是流程与门禁；二者必须一致。

## P0：产品探索

**提示词：** `prompts/P0-product-discovery.md`

1. 人提供产品想法、目标用户、场景和约束。
2. Product Agent 复述产品定位、痛点、核心价值、关键假设。
3. 对竞品和市场信息进行有来源的初步研究，区分事实/推断。
4. 划分 MVP / V1 / Later / 明确不做。
5. 对关键不确定性提出问题；需要验证的技术风险列为 PoC。
6. 人确认产品目标和 MVP。

**文件：** `project/product-brief.md`、`project/mvp-scope.md`、`project/research.md`

**门禁：** 能用一句话说明为谁解决什么问题；核心价值和 MVP 已获人工确认。

## P1：资料整理

**提示词：** `prompts/P1-source-index.md`

1. 收集 PRD、会议纪要、用户反馈、竞品、官方文档、历史实现和团队规范。
2. 建立来源索引，记录路径/URL、章节/位置、可信度、状态和用途。
3. 标记不可访问、过期、互相矛盾和缺失资料。
4. 汇总项目上下文，但摘要不得取代原始权威文档。
5. 人确认关键约束和资料缺口。

**文件：** `project/sources.md`、`project/project-context.md`

**门禁：** 关键结论可追溯；未知与假设显式标注。

## P2：需求定义

**提示词：** `prompts/P2-requirements.md`

1. 复述目标、用户、场景和 MVP。
2. 按影响排序提问，每轮最多 5 个关键问题。
3. 检查范围、非目标、业务规则、角色权限、异常、边界、NFR 和验收。
4. 检查 P0/P1 文档之间是否存在冲突；发现冲突先暂停，按总手册第 5 节处理。
5. 用户确认决策后，先更新对应上游权威文件，再生成需求基线。
6. 将核心场景写成 Given/When/Then。

**文件：** `project/requirements.md`、`project/user-stories.md`、`project/glossary.md`；必要时同步更新 `product-brief.md`、`mvp-scope.md`、`research.md`、`project-context.md`。

**门禁：** 阻断性未决问题清零；需求状态标记 Approved 前必须有验收人确认。

## P3：UX/UI 设计与视觉规范

**提示词：** `prompts/P3-ux-design.md`

1. 建立 Sitemap、页面清单和角色/权限视图。
2. 绘制核心 User Flow 和关键任务路径。
3. 确定视觉方向：产品气质、设计参考、色彩、字体、密度、布局、圆角、阴影、动效。
4. 定义 Design Tokens、组件规范、组件状态和交互反馈。
5. 为每个核心页面编写 Page Spec：布局、组件、字段、交互、状态、响应式、无障碍。
6. 为关键页面提供可供实现的高保真参考：设计稿、截图、原型或明确的结构化描述；若使用生成式 UI/图片，必须人工确认后作为参考资产。
7. 人工确认关键页面和视觉基线。

**文件：** `project/user-flow.md`、`project/design-system.md`、`project/page-specs/*.md`；设计资产放 `project/design-assets/`（如适用）。

**门禁：** 核心页面、主要交互状态和视觉方向明确；设计不能静默增加业务需求。

## P4：全局架构

**提示词：** `prompts/P4-architecture.md`

1. 明确质量属性和工程约束。
2. 比较可行技术方案及其代价。
3. 确定系统边界、模块职责和依赖方向。
4. 定义数据模型、API 契约、认证授权、错误模型。
5. 定义环境、日志、测试、部署、监控和回滚。
6. 评估 UI 设计对组件、前端架构和数据接口的约束。
7. 重大决策写 ADR；高风险技术先做 PoC。

**文件：** `project/architecture.md`、`project/tech-stack.md`、`project/api-contract.md`、`project/data-model.md`、`project/testing-strategy.md`、`project/deployment.md`、`project/adr/`

**门禁：** 方案可落地；关键风险有验证或兜底；不以未经验证的假设支撑设计。

## P5：独立红队审查与修订

**提示词：** `prompts/P5-red-team.md`，修订使用 `prompts/P5b-review-resolution.md`。

1. 独立 Reviewer 审查 P2～P4 的完整文档，不参与原方案撰写。
2. 按 P0/P1/P2 输出具体问题、依据、影响和建议。
3. 方案作者逐条给出接受/部分接受/拒绝/延期的决定。
4. 任何涉及产品取舍或重大架构的拒绝/豁免必须由人批准。
5. 修改权威源文件，并同步受影响的派生文件。
6. Reviewer 复审 P0/P1；必要时重新执行 P2/P3/P4。

**文件：** `project/reviews/P5-red-team-review.md` 及被修订的权威文档。

**门禁：** P0 清零；P1 已修复或有人工批准的豁免；复审结论通过。

## P6：Spec 与 MVP 增量

**提示词：** `prompts/P6-spec-and-roadmap.md`

1. 将已确认需求写成带稳定 ID 的行为规格。
2. 将 MVP 拆成垂直切片，优先交付端到端用户价值。
3. 为每个变更定义 Spec ID、范围、非目标、依赖、DoD、测试和风险。
4. 检查依赖图无环，需求→Spec→变更→验收可追踪。
5. 人确认路线图和增量顺序。

**文件：** `project/specs/`、`project/roadmap.md`

**门禁：** 每个 MVP 需求都有对应 Spec 和变更计划；没有无法验收的模糊增量。

## P7：工程初始化

**提示词：** `prompts/P7-bootstrap.md`

1. 初始化 Git、目录、脚手架和依赖锁定。
2. 建立环境变量示例，禁止提交真实密钥。
3. 配置 lint、format、typecheck、unit、E2E、build。
4. 建立仓库 Agent 规则、日志、错误处理、健康检查和 CI。
5. 只实现工程基础能力，不提前开发未排期的业务功能。
6. 实际执行适用的安装、启动和检查命令。

**文件：** 工程代码、`project/AGENTS.md`、`project/bootstrap-verification.md`。

**门禁：** 工程可以启动，关键质量检查真实执行；未执行项明确标记。

## P8：启动验收

**提示词：** `prompts/P8-bootstrap-verification.md`

1. 独立 QA 按 `testing-strategy.md` 执行安装、启动、健康检查、最小 API/页面链路、lint、typecheck、unit、build 和 CI。
2. 对照架构和工程初始化范围检查目录、配置、依赖和安全。
3. 输出 PASS/FAIL/BLOCKED 及证据。
4. FAIL/BLOCKED 的阻断项返回 P7 修复后重验。

**文件：** `project/bootstrap-verification.md`

**门禁：** 只有 PASS 才进入功能变更循环。
