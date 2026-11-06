# Sources — 资料收集与技术调研

> 阶段：项目资料收集与技术调研（本阶段只调研，不编码）
> 调研日期：2026-11-06（下文所有链接的阅读日期同此）
> 输入基线：[PROJECT-SETUP.md](./PROJECT-SETUP.md)、[constitution.md](./constitution.md)、[requirements.md](./requirements.md)
> 状态：调研初稿；涉及范围与决策的条目已按 `requirements.md` v1.0 对齐（2026-11-07）
> 本文件不构成技术选型决策；选型须在架构阶段经批准并记录 ADR。

## 标记约定

| 标记 | 含义 |
|---|---|
| 【事实】 | 直接读取官方文档 / 官方仓库 / 官方页面得到的可核验内容 |
| 【判断】 | 基于事实的个人推断，可能出错，不当作结论 |
| 【待验证】 | 尚未验证，需实验或人工确认 |
| 未直接验证 | 因访问限制未能抓取原文，仅依据官方 URL 的搜索摘要或二手资料 |

## 调研方法与局限

- 优先采用官方文档（sqlite.org、postgresql.org、docs.docker.com、open.larkoffice.com、各框架官网）与官方仓库（GitHub 仓库主页 / API）。
- GitHub 指标：部分仓库的 GitHub API 返回限流（`API rate limit exceeded`），改用 shields.io 徽章 JSON 获取 stars / 最近提交 / open issues，并在表中标注 †。两类来源都可能滞后，指标只作参考，不作为质量结论。
- 竞品页面：World Anvil 官方页面被安全验证拦截，Campfire 的 features 页面返回 404，相关结论均标注「未直接验证」。
- **未做逐行代码审计**。对开源项目的质量判断仅基于仓库结构、文档、活跃度与 Issue 数量，属【判断】。
- 未做任何中文检索基准测试，所有涉及中文分词效果的结论均为【判断】或【待验证】。

---

## 1. 调研目标

调研服务于 [PROJECT-SETUP.md](./PROJECT-SETUP.md) 的 MVP，不解决「用什么框架」这一决策，只为架构阶段提供可核验的输入。

需要回答的问题：

| 编号 | 问题 | 关联 MVP / 原则 |
|---|---|---|
| Q1 | 中文名称、别名、称号的子串检索，哪种方案在零成本本地环境里最可靠？ | MVP-06、MVP-09；第一原则「检索优先」 |
| Q2 | 单用户本地 Docker 场景，SQLite 还是 PostgreSQL？未来公网是否需要迁移？ | MVP-12；架构原则「最简单架构」「不堵死公网」 |
| Q3 | 「作品作用域 + 别名 + 引用关系 + 榜单席位 + 体系层级」应如何建模？ | MVP-01/02/03/04/05/08/14 |
| Q4 | 同类产品如何处理这些对象，哪些值得借鉴、哪些必须回避？ | 全部 MVP；范围克制 |
| Q5 | 飞书《正文》与文档内现有配图的迁移能做到什么保真度，瓶颈在哪里？ | MVP-10；数据原则「迁移不丢数据」 |
| Q6 | 备份、恢复、许可证、Docker 数据持久化有哪些必须提前规避的风险？ | 宪法第 2/7/9 条 |

调研边界：不评估具体云厂商、不做成本报价、不做 UI 视觉方案、不做代码实现。

---

## 2. 技术资料清单

### 2.1 数据库与中文检索（核心议题）

| 资料 | 来源（官方链接） | 关键结论 | 标记 |
|---|---|---|---|
| SQLite 适用场景 | https://www.sqlite.org/whentouse.html | 面向本地/嵌入式、单文件、免运维；支持多读者，但同一时刻只有一个写者，写操作排队；多写者或数据与应用分离时应选 client/server 数据库 | 【事实】 |
| SQLite FTS5 | https://sqlite.org/fts5.html | 提供 `unicode61`、`ascii`、`porter`、`trigram` 等 tokenizer；`trigram` 可用于子串匹配 | 【事实】 |
| 同上（中文适用性） | 同上 | 默认 `unicode61` 不按中文词边界切分，整段中文可能被当作一个 token，导致子串查不到；改用 `trigram` 可缓解 | 【判断】+【待验证】 |
| 同上（trigram 代价） | 同上 | trigram 索引体积、短关键词（1–2 字）召回、排序质量、查询延迟均需用真实数据基准测试 | 【待验证】 |
| CJK bigram 分词器 | https://pypi.org/project/sqlite-cjk-fts/0.2.0/ | 社区提供 CJK bigram tokenizer 方案，可作为 FTS5 的替代分词思路 | 【事实】（第三方包，维护状态【待验证】） |
| 实践说明：FTS5 默认 tokenizer 丢失日文子串 | https://dev.to/omochi_dev/why-sqlite-fts5s-default-tokenizer-drops-your-japanese-substrings-and-the-one-line-fix-1k2d | 与中文同属无空格语言，问题同源；给出 `trigram` 的修复思路 | 【事实】（第三方博客，需自行验证） |
| FTS5 外部内容表与触发器 ADR | http://raw.githubusercontent.com/itsacoyote/katra/HEAD/docs/decisions/ADR-013-fts5-external-content-triggers.md | 记录了 FTS5 external content + trigger 同步的工程取舍，可作为索引同步设计参考 | 【事实】（第三方项目文档） |
| SQLite 外键 | https://www.sqlite.org/foreignkeys.html | 外键约束**默认关闭**，必须显式启用；被引用列需有索引；支持 ON DELETE / ON UPDATE | 【事实】 |
| SQLite Backup API | https://www.sqlite.org/backup.html | 提供在线备份 API；直接复制数据库文件存在锁与断电损坏风险 | 【事实】 |
| PostgreSQL 全文搜索 | https://www.postgresql.org/docs/current/textsearch.html | 提供 parser、dictionary、`tsvector` / `tsquery`、ranking、触发器更新等完整能力 | 【事实】 |
| 同上（中文） | 同上 | 默认 parser 对中文分词不理想，需搭配中文扩展 | 【判断】+【待验证】 |
| PostgreSQL `pg_trgm` | https://www.postgresql.org/docs/current/pg_trgm.html | 基于 trigram 的文本相似度；支持 GIN / GiST 索引；可用于模糊与子串匹配 | 【事实】 |
| 同上（中文适用性） | 同上 | 对中文姓名、称号的子串检索可能比默认 FTS 更实用，需真实数据验证 | 【判断】+【待验证】 |
| PostgreSQL 约束 | https://www.postgresql.org/docs/current/ddl-constraints.html | 支持 NOT NULL、UNIQUE、PRIMARY KEY、FOREIGN KEY、CHECK、EXCLUSION 等约束 | 【事实】 |
| 同上（用于本项目） | 同上 | 可实现「同一作品、同一实体类型内主名称唯一」「禁止悬空引用」「必填校验」 | 【判断】 |
| PostgreSQL `pg_dump` | https://www.postgresql.org/docs/current/app-pgdump.html | 官方备份工具 | 【事实】 |
| `pg_jieba` | https://github.com/jaiminpan/pg_jieba | 基于 cppjieba 的 PostgreSQL 中文全文搜索 parser；据第三方文档摘要支持 precise / full / search 模式 | 【事实】（项目存在性）；维护状态与版本兼容【待验证】 |
| `zhparser` | https://github.com/amutu/zhparser | 基于 SCWS 的 PostgreSQL 中文全文搜索插件 | 【事实】（项目存在性，据第三方文档摘要）；维护状态【待验证】 |
| Meilisearch 中文支持 | https://www.meilisearch.com/docs/resources/help/language | 中文采用基于 jieba 词典的分词；支持 decomposition、k-variant 转换 | 【事实】 |
| 同上（代价） | 同上 | 检索体验较好，但引入一个独立搜索服务，增加 Compose 复杂度与数据同步成本 | 【判断】 |
| Typesense 中文支持 | https://typesense.org/docs/27.0/api/collections.html | 支持字段级 `locale` 配置，含 `zh`；还支持 ja / ko / th | 【事实】 |
| 同上（不适用项） | 同上 | 其 Natural Language Search 依赖 LLM，与本项目「零成本、不调大模型」冲突 | 【判断】 |

**阶段性判断（不构成选型）**：本项目检索的典型输入是「人名 / 称号 / 2–4 字片段」，不是长文档全文。这更接近「子串匹配 + 别名命中」，而非「语义搜索」。因此第一版应优先在数据库内解决，而不是一开始就引入独立搜索引擎。

### 2.2 部署、持久化与备份

| 资料 | 来源（官方链接） | 关键结论 | 标记 |
|---|---|---|---|
| Docker Compose | https://docs.docker.com/compose/ | 用单个 YAML 定义 services、networks、volumes；一条命令启动多容器应用 | 【事实】 |
| 同上（本项目） | 同上 | 适合本地单用户部署，避免微服务运维复杂度 | 【判断】 |
| Docker Volumes | https://docs.docker.com/engine/storage/volumes/ | volume 是容器数据的持久化机制，可用于备份、恢复、迁移 | 【事实】 |
| Compose 入门（含 `down -v`） | https://docs.docker.com/compose/gettingstarted/ | `docker compose down -v` 会**永久删除**命名 volume 中的数据 | 【事实】 |
| 同上（风险） | 同上 | 必须纳入备份/恢复设计与用户操作警告，不能只写在文档角落 | 【判断】 |

### 2.3 迁移来源：飞书开放平台

| 资料 | 来源（官方链接） | 关键结论 | 标记 |
|---|---|---|---|
| 获取文档内容（docs v1） | https://open.larkoffice.com/document/docs/docs-v1/get | `GET https://open.feishu.cn/open-apis/docs/v1/content` 可获取升级版文档内容，目前支持 markdown；限流 5 次/秒；需文档内容读取权限 | 【事实】 |
| 获取块内容（docx v1） | https://open.larkoffice.com/document/server-docs/docs/docs/docx-v1/document-block/get | 可获取指定 block 的富文本内容；单应用限流 5 次/秒 | 【事实】 |
| Docx 体系概览 | https://open.larkoffice.com/document/server-docs/docs/docs/docx-v1/docx-overview | Docx 提供 raw content、blocks 等接口 | 【事实】 |
| 图片 / 附件下载 | 未取得明确官方接口页 | 图片批量导出接口未确认；本版不依赖自动导出，由用户整理图片 zip，按模板相对文件名匹配 | 【待验证】 |
| 复杂 block 还原 | 同上 | 标题层级、列表、表格、引用等 block 能否完整还原为结构化字段，需实测 | 【待验证】 |

**阶段性判断**：飞书提供的是「文档内容 + 块结构」，不是「已结构化的设定数据」。本项目需要的是人物/势力/榜单/体系字段，因此迁移必然是「解析 + 映射 + 人工校对」，不能假设一键完成。

### 2.4 架构与工程实践

| 资料 | 来源（官方链接） | 关键结论 | 标记 |
|---|---|---|---|
| MonolithFirst | https://martinfowler.com/bliki/MonolithFirst.html | 建议新项目先做单体；成功微服务案例多由单体拆分而来 | 【事实】 |
| 同上（本项目） | 同上 | 应采用模块化单体，而非微服务 | 【判断】 |
| Local-first | https://www.inkandswitch.com/local-first/ | 提出本地优先的七项理想，含数据所有权、离线可用、隐私、长期保存、跨设备协作 | 【事实】 |
| 同上（本项目取舍） | 同上 | 不需要 CRDT 实时协作；应借鉴「数据所有权、离线/本地、可导出、长期可读」 | 【判断】 |
| 单一数据源 | 未找到可引用的官方原文（原 bliki 链接返回 404） | 作为项目原则由 [constitution.md](./constitution.md) 自行定义，不引用外部来源 | 【事实】（链接不可用） |

### 2.5 候选技术栈官方文档（仅清单，不做选型）

| 候选 | 官方链接 | 文档层面的事实 | 标记 |
|---|---|---|---|
| Next.js | https://nextjs.org/docs | React 全栈 Web 框架；提供 App Router / Pages Router | 【事实】 |
| SvelteKit | https://svelte.dev/docs/kit | 基于 Svelte 的 Web 应用框架；提供路由、SSR/CSR/预渲染、构建优化等能力 | 【事实】 |
| FastAPI | https://fastapi.tiangolo.com/ | 基于 Python 类型 hints 的 API 框架；支持 OpenAPI / JSON Schema | 【事实】 |
| Django | https://docs.djangoproject.com/en/stable/ | Python Web 框架；自带 ORM、admin、测试能力 | 【事实】 |

说明：以上仅证明「文档存在且描述这些能力」，不代表推荐。技术选型须结合中文检索基准、Docker 镜像体积、单人维护成本在架构阶段决定。

### 2.6 产品与业务资料

| 来源 / 路径 / URL | 相关章节 | 一句话说明 | 可信度 / 状态 |
|---|---|---|---|
| 飞书文档《正文》 https://ycnaqx42w3fc.feishu.cn/wiki/O8BqwUv5tiHI5bkQbgkcn7VynDb | PROJECT-SETUP 2.1 / 2.4 | 世界观设定集，含纪元、地理、体系、三张榜单、万族传、人物配图 | 已读取（项目发起人提供） |
| [PROJECT-SETUP.md](./PROJECT-SETUP.md) | 全文 | 产品、范围、约束、成功标准基线 | 已按 requirements v1.0 对齐 |
| [constitution.md](./constitution.md) | 全文 | 长期有效的产品与工程原则 | 已按 requirements v1.0 对齐 |
| 素材库 | 不适用 | 已确认不纳入本版迁移范围 | 已关闭 |

### 2.7 规范与约束

| 来源 / 路径 / URL | 相关位置 | 一句话说明 | 可信度 / 状态 |
|---|---|---|---|
| [constitution.md](./constitution.md) 第 1、2、11 章 | 数据与检索原则 | 引用而非复制、别名一等检索入口、禁止悬空引用、榜单只做最小配置 | 已确认（requirements v1.0） |
| [PROJECT-SETUP.md](./PROJECT-SETUP.md) 第 7 章 | 非目标范围 | 图谱、正文、协作、移动端、AI 生成、自动排名均为硬边界 | 已确认 |
| [PROJECT-SETUP.md](./PROJECT-SETUP.md) 第 8 章 | 项目约束 | 零成本、本地 Docker、单用户、中文、桌面优先 | 已确认 |

---

## 3. 开源项目分析

### 3.1 指标总览

> Stars / Issues 为调研日快照，会随时间变化。† 表示来自 shields.io 徽章（GitHub API 限流），未经 API 二次核对。**Star 高不等于适合本项目**。

| 项目 | Stars | 最近更新 | Open Issues | 技术栈 | License | 指标来源 |
|---|---|---|---|---|---|---|
| Kanka | 363 | 2026-11-05 前后 † | 0 † | PHP + VueJS/jQuery + MySQL | Commons Clause（禁止销售；底层许可证未完整确认） | shields.io |
| Manuskript | 约 2.5k | 2026-09 † | 579 † | Python3 + PyQt5 | GPLv3 | shields.io |
| Outline | 41,157 | 2026-11-03 | 67 | TypeScript / React + Node | Other / NOASSERTION | GitHub API |
| BookStack | 19,131 | 2026-11-06 | 3 | PHP / Laravel | MIT | GitHub API（README 声明已迁至 Codeberg） |
| Wiki.js | 约 29k | 2026-11-06 当天 † | 7 † | Node.js；支持 PostgreSQL / MySQL / MariaDB / MSSQL / SQLite | AGPL-3.0 | shields.io |
| AppFlowy | 78,108 | 2026-10-28 | 1,023 | Dart / Flutter | AGPL-3.0 | GitHub API |

### 3.2 逐项分析

#### Kanka — https://github.com/owlchester/kanka

- 【事实】source available；PHP + VueJS/jQuery + MySQL；支持自托管；免费核心功能、无限 campaign / entry，付费层提供高级 campaign 与更大上传；平均每月发布；产品提供 @mentions、日历、交互地图、时间线、组织/家族/人物/地点、关系、文章、秘密、文件、标签、campaign 分类开关、每日导出、API（https://docs.kanka.io/en/latest/overview.html）。
- 【事实】License 为 Commons Clause 条件，禁止销售该软件；底层许可证全文未完整确认。
- 【判断】**这是与本项目领域模型最接近的项目**：campaign 作用域相当于本项目的「作品作用域」，实体分类 + 关系 + 标签 + 导出 + API 的结构值得研究。
- 【判断】不足：SaaS-first 设计；前端 PHP 模板 + jQuery 混用，工程质量不能按现代 Web 应用标准借鉴；许可证限制商业复用。
- **结论：借鉴领域模型与信息架构，不复制代码。**

#### Manuskript — https://github.com/olivierkes/manuskript

- 【事实】Python3 + PyQt5；GPLv3；约 2.5k stars；最近提交 2026-09 前后；579 open issues；开源写作工具，含人物、情节、大纲、世界构建；本地桌面应用，使用开放的纯文本格式。
- 【判断】属于**相邻品类而非同类**：以「写作」为中心，不是「结构化设定数据库」，缺少别名一等检索、榜单席位、境界层级等本项目必需能力。
- 【判断】579 个 open issues 反映维护压力，不适合作为依赖。
- **结论：只参考「本地优先 + 开放文件格式」理念，不采用。**

#### Outline — https://github.com/outline/outline

- 【事实】41,157 stars；最近推送 2026-11-03；67 open issues；TypeScript / React + Node 生态；License 为 Other / NOASSERTION（需法律审查）；团队 wiki，支持实时协作、Markdown、搜索。
- 【判断】活跃度高、Issue 相对可控，工程实践成熟，可参考其编辑体验、搜索与文档信息架构。
- 【判断】协作型架构较重，且领域模型是「文档集合」，不是「结构化设定实体」。
- **结论：参考工程实践与 UX，不作为底座。**

#### BookStack — https://github.com/BookStackApp/BookStack

- 【事实】19,131 stars；最近推送 2026-11-06；仅 3 open issues；PHP / Laravel；MIT；Books / Chapters / Pages 层级；README 显示「NOW MANAGED ON CODEBERG」，主页已迁至 https://codeberg.org/bookstack/bookstack。
- 【判断】MIT + 低 Issue 数 + 稳定的层级模型，是「轻量自托管知识库」的良好参考样本。
- 【判断】仅有三层页面层级，缺少别名表、引用关系、榜单席位与体系层级，无法满足本项目领域需求。
- **结论：参考其自托管与层级信息架构，不作为底座。**

#### Wiki.js — https://github.com/requarks/wiki

- 【事实】约 29k stars；最近提交为调研日当天；7 open issues；Node.js；支持 PostgreSQL、MySQL、MariaDB、MSSQL、SQLite 多种数据库；AGPL-3.0。
- 【判断】多数据库支持与搜索接入方式值得参考，说明「同一应用兼容 SQLite 与 PostgreSQL」在工程上可行。
- 【判断】AGPL-3.0，若复制代码将带来开源义务；非领域专用。
- **结论：参考多数据库与搜索架构，不复制代码。**

#### AppFlowy — https://github.com/AppFlowy-IO/AppFlowy

- 【事实】78,108 stars；最近推送 2026-10-28；1,023 open issues；Dart / Flutter；AGPL-3.0；Notion 替代型本地/协作工作区，块编辑器。
- 【判断】Star 数最高，但 Issue 数量也最大，且技术栈（Flutter 桌面）与「本地 Docker + 浏览器」的形态不匹配。
- **结论：只参考「本地优先 + Notion 式编辑体验」，不采用。**

### 3.3 开源项目小结

| 维度 | 结论 |
|---|---|
| 是否存在可直接复用的同类开源项目 | 【判断】未发现同时满足「本地优先 + 中文 + 结构化榜单/境界 + 零成本 + 每作品独立体系」的项目 |
| 最接近的领域模型 | Kanka（campaign 作用域 + 实体 + 关系 + 标签 + 导出） |
| 最值得参考的工程质量 | BookStack（MIT、低 Issue）、Outline（活跃、TypeScript） |
| 许可证高风险 | Kanka（Commons Clause）、Wiki.js / AppFlowy（AGPL-3.0）、Manuskript（GPLv3）、Outline（NOASSERTION） |
| 代码复用策略 | 仅借鉴模式与信息架构；不复制代码，避免许可证义务 |

---

## 4. 竞品分析

### 4.1 逐项事实与差异

#### Kanka — https://kanka.io/ ，文档 https://docs.kanka.io/en/latest/overview.html

- 【事实】SaaS + 可自托管（source available）；免费核心、无限 campaign / entry；功能覆盖 @mentions、日历、交互地图、时间线、组织、家族、人物、地点、关系、文章、秘密、文件、标签、campaign 分类开关、每日导出、API。
- 【判断】与本项目重合度最高，但它是通用世界构建工具，不针对「中文网文的境界 / 称号 / 榜单席位」建模。
- 【待验证】中文界面与中文检索效果；许可证对个人自用的具体约束。

#### World Anvil — https://www.worldanvil.com/ ，定价 https://www.worldanvil.com/pricing

- 【事实（未直接验证）】SaaS 世界构建/写作平台；提供文章、时间线、地图、家族树、campaign、co-author 等能力。定价搜索摘要：Master 约 $7/月或 $54/年（2GB 存储、4 co-authors、10 campaigns）；Grandmaster 约 $12/月或 $8.25/月年付；Sage 约 $34/月或 $25/月年付、终身 $1,350。
- 访问限制：官方页面被安全验证拦截，**未能直接抓取**，以上为官方 pricing URL 的搜索摘要，需后续人工核价。
- 【判断】功能广度大，但 SaaS 化、英文优先、付费订阅，与本项目「本地优先 / 零成本 / 中文」不符。

#### LegendKeeper — https://www.legendkeeper.com/ ，功能 https://www.legendkeeper.com/features/

- 【事实】SaaS；提供 wiki、地图、时间线、boards、实时协作、离线支持、全文搜索、导出、默认私密、细粒度权限与秘密；无存储或项目数量限制；仍处于 Open Beta。
- 【判断】其「wiki + 搜索 + 离线 + 导出」的体验是最值得参考的 UX 样本；但不做结构化榜单与境界层级。
- 【待验证】Open Beta 阶段的数据保障与长期可持续性。

#### Campfire — https://campfirewriting.com/ ，定价 https://campfirewriting.com/pricing

- 【事实】SaaS；18 个模块，含 Manuscript、Characters、Timeline、Maps、Calendar、Encyclopedia、Magic、Languages、Research、Locations；免费版有限；付费标准档约 $12/月（标价原 $19）。
- 访问限制：features 页面返回 404，**功能页未直接抓取**。
- 【判断】模块化写作套件，思路是「写作 + 设定并重」；本项目刻意只做设定，不承担写作功能。

#### Notion / 飞书多维表格 / Obsidian（通用替代）

- 【事实】均为通用知识库 / 数据库 / 本地 Markdown 工具，不是小说设定专用产品。
- 【判断】它们正是当前工作流的上游（飞书文档），也是「够用就不做系统」的最强替代方案。本项目成立的前提是：通用工具无法同时满足「别名一等检索 + 引用一致性 + 榜单席位 + 每作品独立体系 + 本地零成本」。

### 4.2 能力对比

| 能力 | Kanka | World Anvil | LegendKeeper | Campfire | 飞书 / Notion 类 | 本项目目标 |
|---|---|---|---|---|---|---|
| 人物 / 势力 / 地点结构化 | 有 | 有（未直接验证） | 有 | 有 | 需自建表结构 | 必须 |
| 别名 / 称号一等检索 | 未验证 | 未验证 | 未验证 | 未验证 | 无原生能力 | 必须 |
| 引用式关系与双向跳转 | 有关系与 @mentions | 未验证 | 有 | 未验证 | 手工超链接 | 必须 |
| 榜单 / 席位（可配置席位与空缺） | 未发现（未验证） | 未验证 | 未发现（未验证） | 未验证 | 需自建 | 必须 |
| 每作品独立体系层级（如十五境） | campaign 作用域，但无体系层级概念 | 未验证 | 无 | 无 | 需自建 | 必须 |
| 本地 / 可自托管 | 可自托管 | 否 | 否（有离线） | 否 | 否（Obsidian 本地） | 必须 |
| 中文界面 | 未验证 | 未验证 | 未验证 | 未验证 | 有 | 必须 |
| 零成本 | 免费核心 | 否 | 否 | 否 | 免费档有限 | 必须 |

说明：表中「未验证 / 未发现」不等于「没有」，只表示本次调研未取得证据；不得对外表述为竞品缺陷。

### 4.3 竞品结论

- 【判断】市场存在大量「世界构建」工具，但它们的共同假设是「英文 / SaaS / 通用实体」，没有一个针对「中文网文的多名称人物 + 境界层级 + 可配置席位榜单」建模。
- 【判断】这说明本项目有差异化空间，同时也说明这是小众需求——**因此更不能扩张范围**，否则会向通用世界构建工具演化，直接对上成熟竞品。
- 【判断】最值得借鉴的三点：Kanka 的作品作用域、LegendKeeper 的 wiki 检索与导出体验、World Anvil 的「体系化设定」信息组织。

---

## 5. 可借鉴的架构与实践

以下均为【判断】/ 建议，不是已确认技术决策。

### 5.1 架构形态：模块化单体

- 依据：MonolithFirst（https://martinfowler.com/bliki/MonolithFirst.html）。
- 做法：单仓库、单进程、单数据库；按领域模块划分（作品、人物、势力与地点、榜单、体系、搜索、迁移、备份），模块间通过明确接口调用，不跨模块直接读写对方数据表。
- 反向约束：不引入微服务、消息队列、独立搜索集群、K8s。

### 5.2 数据边界：作品作用域

- 依据：Kanka 的 campaign 模式（https://docs.kanka.io/en/latest/overview.html）。
- 做法：所有业务表带 `work_id`；查询默认按作品过滤；跨作品引用在第一版被禁止（与宪法第 2 条一致）。
- 收益：多本小说天然隔离；未来若做多用户，`work_id` 是可复用的隔离维度。

### 5.3 检索策略：数据库原生优先

- 依据：SQLite FTS5 / PostgreSQL FTS / `pg_trgm` 官方文档；Meilisearch、Typesense 会引入独立服务。
- 建议路径：
  1. 先用真实样例集做基准测试（本名、别名、称号、2–4 字片段）。
  2. 若 SQLite `trigram` 达标，则不引入任何外部搜索服务。
  3. 若不达标，再评估 PostgreSQL + `pg_trgm` / `pg_jieba`，最后才考虑 Meilisearch。
- 反模式：一开始就上「数据库 + 搜索引擎 + 同步管道」，对单用户项目是纯负担。

### 5.4 迁移管线：固定模板 + 产品内导入向导

建议流程（对应 MVP-10）：

```
按固定模板整理《正文》+ 图片 zip → 产品内导入向导（解析 / 校验 / 预览 / 确认）
→ 原子写入 → 数量核对（实体数/图片数/层级数） → 全部实体待校对 → 项目发起人签收
```

- 导入报告与校对记录必须留存，作为「迁移证据」。
- 图片由用户整理为 zip，按模板相对文件名匹配；本版不依赖飞书批量下载接口。
- 只解析固定模板，不做「全自动解析任意飞书排版」承诺。

### 5.5 备份与恢复

- SQLite：使用官方 Backup API，而不是直接复制 `.db` 文件（https://www.sqlite.org/backup.html）。
- PostgreSQL：使用 `pg_dump`（https://www.postgresql.org/docs/current/app-pgdump.html）。
- Docker：数据必须放在命名 volume；`docker compose down -v` 会删除数据（https://docs.docker.com/compose/gettingstarted/），文档与操作确认必须显式警告。
- 恢复演练：迁移前、公网部署前各执行一次「空环境恢复」，与宪法第 9 条一致。

### 5.6 领域模型草案（已按 requirements v1.0 对齐）

> 仅为调研阶段的建模建议，字段命名与实现方式在架构阶段确定；与 `requirements.md` 冲突时以需求文档为准。

| 实体 | 关键字段 | 约束 |
|---|---|---|
| work | 名称、简介 | 数据边界 |
| alias | 实体类型、实体 ID、别名文本、类型（别名 / 称号） | 一等检索入口，参与搜索索引 |
| character | 主名称、境界、子层级、势力、1 张主图、简介、手工大事记；别名/称号通过 alias 表 | 主名称必填；同一作品内唯一；无状态/性别/年龄/外貌字段 |
| faction | 名称、上级、首领 | 单棵树、深度不限；节点不区分类型；禁止悬空引用 |
| location | 名称、类型（自由文本）、上级 | 单棵树、深度不限；类型带已有值建议；可关联人物与势力 |
| ranking | 名称、席位数（1–200，默认 100） | 每作品独立；名称同作品唯一；只可扩容 |
| ranking_seat | 榜单、席位序号、人物、备注 | 人物可空（允许空缺）；同一人物同榜仅 1 席；不做并列 |
| system_category | 作品、名称、类型（普通 / 纪元）、启用、是否内置 | 名称同作品唯一；境界内置必开；最多 10 类 |
| system_node | 类别、名称、顺序、上级、起止年份文本 | 同类别同父唯一；最多两级；纪元年份不做日期计算 |
| relation | 源实体、目标实体、关系类型 | 必须定义反向展示 |
| chronicle | 归属实体、文本、时间标签 | 手工大事记，不做全局事件库 |

关键点：
- 姓名、称号不进备注字段，进 `alias` 表；
- 榜单席位与人物是「引用」，人物改名后席位显示自动更新；
- 「帝境五重天」拆为「主境界 + 子层级」，不存自由文本（宪法第 1 条）；
- 体系类别按作品自定义，星曜 / 星命 / 圣体等名称不得写死；境界为内置必开类别，人物每类 0..1，势力不引用；
- 英文名称唯一性校验与检索不区分大小写。

### 5.7 明确不借鉴

| 实践 | 不采纳原因 |
|---|---|
| CRDT / 实时协作 | 单用户，无并发冲突需求；成本极高 |
| 交互地图、时间线可视化 | 与「检索优先」冲突；MVP 非目标 |
| 通用低代码实体建模（体系类别自定义除外） | 范围膨胀风险；四类对象已足够 |
| 语义/向量检索 | 需要嵌入模型或付费 API；与零成本、不可解释冲突 |
| 微服务拆分 | 单人维护，运维成本不成比例 |

---

## 6. 技术风险

| 编号 | 风险 | 等级 | 触发条件 | 影响 | 缓解方向 | 状态 |
|---|---|---|---|---|---|---|
| R-01 | 中文检索召回不足 | 高 | 使用默认 `unicode61` 或短关键词查询 | 别名、称号查不到，直接违背第一原则 | 先做基准测试；必要时 `trigram` / `pg_trgm` / 外部引擎 | 【待验证】 |
| R-02 | 迁移保真度不足 | 高 | 飞书排版不完全符合固定模板、图片 zip 缺失 | 人物、层级、图片丢失，迁移验收失败 | 固定模板 + 校验预览 + 数量核对 + 人工校对 + 签收；不承诺兼容任意飞书原始排版 | 【待验证】 |
| R-03 | 备份 / 导出风险 | 高 | 误删、清空回收站、覆盖恢复 | 无审计无任意时点回滚，数据永久丢失 | 已确认进入 MVP：全量导出、CLI 还原、自动快照保留 3 份、恢复演练 | 已关闭（Must） |
| R-04 | 范围膨胀 | 中高 | 「榜单可配置」「详情页功能齐全」被不断扩展 | 演化为低代码平台，进度失控 | 最小配置边界；新增项须人工批准（宪法第 8 条） | 持续监控 |
| R-05 | SQLite 单写者限制 | 中 | 未来公网多用户并发写 | 写锁排队，体验下降 | 第一版限定单用户；公网阶段评估 PostgreSQL | 【事实】+【判断】 |
| R-06 | 外键默认关闭导致悬空引用 | 中 | 未显式启用 `PRAGMA foreign_keys` | 删除后留下死链，违背数据原则 | 启动时强制开启；删除走统一完整性校验 | 【事实】 |
| R-07 | Docker volume 误删 | 中 | 用户执行 `docker compose down -v` | 数据永久丢失 | 文档显著警告；操作前备份；提供导出入口 | 【事实】 |
| R-08 | 规则化问答预期落差 | 中 | 用户期待自然语言推理 | 满意度下降，被认为「不好用」 | 明确「检索问答」定位；预设查询集验收 | 【判断】 |
| R-09 | 许可证风险 | 中 | 复制 Kanka / Wiki.js / AppFlowy 代码 | 开源义务或商业限制 | 只借鉴模式；建立许可证白名单 | 【事实】+【判断】 |
| R-10 | Windows Docker Desktop 文件性能 | 中低 | 数据卷跨文件系统挂载、大量小图片 | 迁移或图片加载变慢 | 使用命名 volume；避免绑定宿主机目录做数据库存储 | 【待验证】 |
| R-11 | 未来公网安全 | 中 | 直接暴露端口 | 私密创作数据泄露 | 第一版只绑本机；公网必须加访问控制（宪法第 7 条） | 已识别 |
| R-12 | 数据规模超基线 | 中 | 实际数据大于基线 | 分页、索引、检索性能判断失真 | 按 requirements 6.7 基线设计：1000 人物、200 势力、200 地点、5 榜 ×100 席、10 类体系、150 节点、500 图、作品 ≤10 | 已确认基线 |

---

## 7. 待验证问题

| 编号 | 问题 | 优先级 | 验证方式 | 阻塞的决策 |
|---|---|---|---|---|
| V-01 | 中文子串检索在 SQLite FTS5 `trigram` 下的召回、排序、延迟、索引体积 | P0 | 用 20–30 条真实人物样例（本名 / 别名 / 称号 / 2 字片段）做基准脚本；错字仅作对照，不纳入产品验收 | 数据库与搜索选型 |
| V-02 | 素材库迁移 | — | 已确认不纳入本版 | 不阻塞 |
| V-03 | 备份 / 导出是否作为 MVP 必须交付 | — | 已确认进入 MVP（Must） | 不阻塞 |
| V-04 | 飞书 Docx 图片 / 附件批量下载能力 | — | 本版不依赖自动导出，由用户整理图片 zip | 不阻塞 |
| V-05 | 飞书复杂 block 还原能力 | — | 本版只解析固定模板，不承诺兼容任意排版 | 不阻塞 |
| V-06 | Kanka 许可证底层全文与个人自用边界 | P1 | 读取仓库 LICENSE 全文 | 是否需要法律审查 |
| V-07 | GitHub 指标（Kanka / Manuskript / Wiki.js）补齐 | P1 | 恢复 API 配额后重新拉取 | 仅影响报告准确性 |
| V-08 | World Anvil、Campfire 功能页与定价 | P1 | 人工浏览器访问 | 竞品结论准确性 |
| V-09 | `pg_jieba` / `zhparser` 当前维护状态与 Docker 安装方式 | P2 | 查看仓库最近提交与发行版 | 仅在选 PostgreSQL 时相关 |
| V-10 | 每本小说的人物、势力、地点、榜单条目量级 | — | 已按 requirements 6.7 基线确认 | 不阻塞 |
| V-11 | 详情页视觉验收参考 | — | 按 requirements 8.5 与 AC-30 验收 | 不阻塞 |

**信息缺口汇总**

| 缺失资料 | 影响 | 状态 |
|---|---|---|
| 素材库位置 / 格式 / 数量 | 已确认不纳入本版 | 已关闭 |
| 数据规模估算 | 已按 requirements 6.7 基线确认 | 已关闭 |
| 飞书图片接口细节 | 本版不依赖自动导出，图片由用户整理 zip | 已关闭 |
| 备份 / 导出是否进 MVP | 已确认 Must | 已关闭 |
| 正式项目名称 | 已确认「小说设定管理系统」 | 已关闭 |

> 仍待验证的仅有技术选型项（V-01、V-06~V-09），不阻塞需求与架构。

---

## 8. 对本项目的建议

> 以下为调研建议；与 `requirements.md` v1.0 冲突时以需求文档为准。

| 编号 | 建议 | 理由 | 依据 |
|---|---|---|---|
| R1 | 先做中文检索基准测试，再定数据库与框架 | 检索是本项目第一原则；反过来会让技术选型受框架限制 | V-01；FTS5 / `pg_trgm` 文档 |
| R2 | 第一版默认「SQLite + FTS5（优先 `trigram`）」；把 PostgreSQL 作为公网/多用户阶段的迁移路径 | 零成本、单文件、Docker 内零运维；SQLite 官方明确适配本地单用户 | https://www.sqlite.org/whentouse.html |
| R3 | 数据模型以「作品作用域 + 别名表 + 引用关系」为骨架，先建模再写界面 | 一致性与检索都依赖这个骨架；后期补别名表代价极高 | 宪法第 2 条；Kanka campaign 模式 |
| R4 | 榜单只做「名称 / 席位 1–200（默认 100）/ 席位顺序 / 上榜人物 / 备注」，允许空缺、一人同榜一席、不做并列 | 防止向排名引擎与战力计算膨胀 | MVP-04；范围风险 R-04 |
| R5 | 备份 / 导出纳入 MVP（Must），并强制「自动快照 + 空环境恢复演练」 | 无审计无任意时点回滚，备份与回收站是主要恢复手段 | 宪法第 9 条；Backup API / `pg_dump` |
| R6 | 采用模块化单体 + 单仓库 + Docker Compose；不引入独立搜索服务与消息队列 | 单人维护成本必须最小化 | MonolithFirst；Compose 文档 |
| R7 | 迁移采用「固定模板 + 产品内导入向导 + 人工校对 + 用户签收」，不承诺全自动解析 | 飞书提供的是文档结构，不是领域数据；本版只解析固定模板 | 飞书 Docx 文档；MVP-10 |
| R8 | 建立一套 20–30 条真实样例的验收集，检索 / 一致性 / 迁移共用 | 三类验收都需要同一份真实数据，避免各写一套 | 成功标准第 9 章 |
| R9 | 规则化问答限定为「检索 + 摘要 + 出处」，无匹配必须明示 | 与用户预期对齐，避免被当作 AI 助手 | MVP-09；R-08 |
| R10 | 建立许可证白名单：优先 MIT / Apache-2.0；AGPL / GPL / Commons Clause 只读不复制 | 个人项目也应避免不必要的法律义务 | 开源项目分析 3.3 |

**明确不建议做的事**

- 不建议在第一版引入 Meilisearch / Typesense / Elasticsearch：在检索基准出结果前，这是成本而非收益。
- 不建议做关系图谱、时间线可视化、地图：与「查找与一致性」这一首要痛点无关。
- 不建议先写界面再定数据模型：别名、引用、席位三类结构一旦缺失，界面越漂亮返工越贵。
- 不建议把飞书当作运行时数据源或做双向同步：迁移即归档（宪法第 2 条）。

**建议的下一步顺序**

1. 项目发起人确认本文件与 [PROJECT-SETUP.md](./PROJECT-SETUP.md)、[constitution.md](./constitution.md)、[requirements.md](./requirements.md) 已对齐。
2. 执行 V-01 中文检索基准测试，产出可核验的数据（不写业务代码，只写一次性基准脚本）。
3. 进入架构阶段：数据模型、技术栈、部署方案、ADR；批准后才编码。

---

## 附录：链接可用性说明

| 链接 | 状态 |
|---|---|
| https://martinfowler.com/bliki/SingleSourceOfTruth.html | 返回 404，未引用 |
| https://www.worldanvil.com/features | 被安全验证拦截，未直接抓取 |
| https://campfirewriting.com/features | 返回 404，未直接抓取 |
| GitHub API（Kanka / Manuskript / Wiki.js / BookStack 部分请求） | 触发限流，改用 shields.io，表中已标 † |
| 飞书图片 / 附件下载接口页 | 未取得明确页面，列为待验证 |
