# 从零项目启动执行手册

## P0：产品探索

1. 人提供一句话想法、目标用户、预期使用场景。
2. Product Agent 输出产品定位、用户痛点、核心价值和待确认问题。
3. 对竞品进行有来源的研究；区分事实、推断和建议。
4. 对核心技术不确定项建立 PoC 列表，先验证高风险项。
5. 划分 MVP / V1 / Later，明确非目标。
6. 人确认 Product Brief 和 MVP 范围。

文件：
- `project/product-brief.md`
- `project/mvp-scope.md`
- `project/research.md`

停止条件：产品目标仍无法用一句话说明，或核心价值与目标用户不明确。

## P1：资料整理

1. 收集 PRD、会议纪要、用户反馈、竞品、官方文档、历史实现和团队规范。
2. 在 `project/sources.md` 中只记录来源、路径/URL、相关章节、可信度和用途。
3. 标记不可访问、过期、相互矛盾的资料。
4. 写入 `project/project-context.md`。
5. 人确认资料索引和关键约束。

## P2：需求定义

1. 复述目标、用户、场景。
2. 按影响排序提出最多 5 个关键问题。
3. 对“大概、可能、以后再说”等模糊回答追问。
4. 明确范围、非目标、业务规则、权限、异常、边界、非功能要求。
5. 将核心场景写成 Given/When/Then。
6. 人确认需求基线。

文件：
- `project/requirements.md`
- `project/user-stories.md`
- `project/glossary.md`

## P3：UX/UI 设计

1. 建立 Sitemap 和页面清单。
2. 绘制核心 User Flow。
3. 定义 Design Tokens 和组件规范。
4. 为核心页面定义布局、交互、加载、空、错误、成功状态。
5. 人确认关键页面和交互。
6. 设计变更回写需求，不得静默增加功能。

文件：
- `project/user-flow.md`
- `project/design-system.md`
- `project/page-specs/`

## P4：全局架构

1. 先列出质量属性和工程约束。
2. 比较至少两个可行技术方案。
3. 确定系统边界、模块职责、依赖方向。
4. 定义数据模型、API 契约、认证授权和错误模型。
5. 定义环境配置、日志、测试、部署和回滚策略。
6. 重要决策写 ADR。
7. 对高风险技术做 PoC，不用未经验证的假设支撑架构。

文件：
- `project/architecture.md`
- `project/tech-stack.md`
- `project/api-contract.md`
- `project/data-model.md`
- `project/testing-strategy.md`
- `project/deployment.md`
- `project/adr/`

## P5：独立红队审查

1. Reviewer 不参与原方案撰写。
2. 输入需求、UX、架构和技术调研。
3. 按 P0/P1/P2 输出问题，必须提供依据和影响。
4. 作者逐条修订或提交豁免理由。
5. Reviewer 复核 P0/P1。
6. 人确认通过。

## P6：Spec 与 MVP 增量

1. 把确认后的系统行为写入 `project/specs/`。
2. 将 MVP 拆为垂直切片优先的变更。
3. 每个变更有覆盖的 Spec、依赖、范围、DoD、测试方式。
4. 依赖图无环；无法独立部署的增量必须有独立验证方式。
5. 人确认增量顺序。

## P7：工程初始化

1. 初始化 Git、目录和项目脚手架。
2. 固定依赖并提交锁文件。
3. 建立 `.env.example`，禁止提交真实密钥。
4. 配置 lint、格式化、类型检查、单测和 E2E。
5. 编写仓库级 Agent 规则。
6. 配置 CI；先在本地执行相同命令。
7. 建立健康检查、日志和错误处理基础能力。
8. 不提前实现未进入当前增量的业务功能。

## P8：启动验收

执行：
- 安装依赖
- 启动前端/后端
- 数据库连接和迁移（如适用）
- 健康检查
- 最小 API 请求
- 最小端到端链路
- lint / typecheck / unit / build
- CI

输出 `project/bootstrap-verification.md`。所有未执行项明确标注。
