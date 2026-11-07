# Sources — 资料索引

> 阶段：P1 资料整理（模板对齐版）
> 日期：2026-11-06；模板对齐：2026-11-07
> 状态：来源索引与冲突清单已建立；所有业务冲突均已由项目发起人确认并同步上游。
> 本文件只建立索引，不改写原始资料，不复制大段原文，不虚构来源。研究结论见 `research.md`。

## 1. 分类与标记

| 标记 | 含义 |
|---|---|
| 一手 | 官方文档、官方仓库、项目发起人确认记录、原始资料 |
| 二手 | 第三方博客、聚合指标、搜索摘要、二手分析 |
| 推断 | 基于事实的个人判断，不当作结论 |
| 可访问 | 调研时可直接读取原文 |
| 不可访问 | 页面被拦截、404 或需要登录，未能直接验证 |
| 过期 | 内容可能已变化，需重新核对 |
| 待验证 | 尚未实验或人工确认 |

## 2. 来源索引

### 2.1 产品业务

| ID | 来源路径 / URL | 相关章节 / 位置 | 类型 | 可信度 | 状态 | 一句话说明 | 用途 |
|---|---|---|---|---|---|---|---|
| SRC-B-01 | 飞书文档《正文》 https://ycnaqx42w3fc.feishu.cn/wiki/O8BqwUv5tiHI5bkQbgkcn7VynDb | 全文 | 一手 | 高 | 可访问（已读取 2026-11-06） | 世界观设定集：纪元、地理、体系、三张榜单、万族传、人物与配图 | 迁移来源、事实依据 |
| SRC-B-02 | `PROJECT-SETUP.md` | 全文 | 一手（项目发起人确认） | 高 | 可访问（已确认） | 产品、范围、约束、成功标准与 MVP 基线 | P2 输入、范围核对 |
| SRC-B-03 | `requirements.md` | 全文 | 一手（需求基线） | 高 | 可访问（Approved v1.1） | 功能、规则、异常、NFR、验收与追踪 | 产品唯一需求基线 |
| SRC-B-04 | `constitution.md` | 全文 | 一手（项目原则） | 高 | 可访问（已确认） | 检索优先、一致性、数据边界、零成本等长期原则 | 需求与架构约束 |
| SRC-B-05 | `product-brief.md` | 全文 | 整理自一手确认 | 高 | 可访问（按既有确认整理） | 产品定位、用户、价值、非目标与确认记录 | P2 上游产品方向 |
| SRC-B-06 | `mvp-scope.md` | 全文 | 整理自一手确认 | 高 | 可访问（按既有确认整理） | MVP 必须交付、明确不做、V1/V2 候选 | P2 上游范围 |
| SRC-B-07 | `research.md` | 全文 | 研究结论（含推断） | 中高 | 可访问（已更新） | 竞品、开源项目、风险、PoC 与未验证假设 | P2 上游证据 |
| SRC-B-08 | `project-context.md` | 全文 | 摘要 | 高 | 可访问（已更新） | 跨阶段上下文摘要与文档索引 | P2 输入、后续阶段复用 |
| SRC-B-09 | 项目发起人确认记录 | `requirements.md` 第 11、12 章；`product-brief.md` 人工确认记录 | 一手 | 高 | 可访问（已确认） | P-01~P-11、体系类别、榜单、备份、迁移范围等决策 | 冲突裁决依据 |

### 2.2 用户研究与竞品

| ID | 来源路径 / URL | 相关章节 / 位置 | 类型 | 可信度 | 状态 | 一句话说明 | 用途 |
|---|---|---|---|---|---|---|---|
| SRC-C-01 | Kanka https://kanka.io/ ；https://docs.kanka.io/en/latest/overview.html | 产品页、文档概览 | 一手 | 中高 | 可访问 | SaaS + 自托管世界构建；campaign 作用域、实体、关系、导出、API | 领域模型与信息架构参考 |
| SRC-C-02 | World Anvil https://www.worldanvil.com/ ；https://www.worldanvil.com/pricing | 产品页、定价页 | 一手（未直接验证） | 中低 | 不可访问（安全验证） | SaaS 世界构建/写作平台；功能广、付费订阅 | 竞品差异；结论标注未直接验证 |
| SRC-C-03 | LegendKeeper https://www.legendkeeper.com/ ；https://www.legendkeeper.com/features/ | 产品页、功能页 | 一手 | 中 | 可访问 | wiki + 地图 + 时间线 + 搜索 + 离线 + 导出；Open Beta | 检索与导出体验参考 |
| SRC-C-04 | Campfire https://campfirewriting.com/ ；https://campfirewriting.com/pricing | 产品页、定价页 | 一手（部分未验证） | 中低 | 不可访问（features 404） | 模块化写作套件；Characters/Timeline/Maps/Encyclopedia 等 | 竞品差异；不承担写作功能 |
| SRC-C-05 | Notion / 飞书多维表格 / Obsidian 通用产品页 | 产品页 | 一手 | 高 | 可访问 | 通用知识库 / 数据库 / 本地 Markdown 工具 | 现有替代方案与上游工作流 |
| SRC-C-06 | Kanka 仓库 https://github.com/owlchester/kanka | 仓库主页、LICENSE | 一手 | 中高 | 可访问 | PHP + VueJS/jQuery + MySQL；Commons Clause | 开源参考；许可证 |
| SRC-C-07 | Manuskript 仓库 https://github.com/olivierkes/manuskript | 仓库主页 | 一手 | 中 | 可访问 | Python3 + PyQt5；GPLv3；写作工具 | 相邻品类参考 |
| SRC-C-08 | Outline 仓库 https://github.com/outline/outline | 仓库主页 | 一手 | 中高 | 可访问 | TypeScript/React + Node；团队 wiki | 工程实践与 UX 参考 |
| SRC-C-09 | BookStack 仓库 https://github.com/BookStackApp/BookStack | 仓库主页 | 一手 | 中高 | 可访问 | PHP/Laravel；MIT；Books/Chapters/Pages | 自托管与层级信息架构参考 |
| SRC-C-10 | Wiki.js 仓库 https://github.com/requarks/wiki | 仓库主页 | 一手 | 中 | 可访问 | Node.js；多数据库；AGPL-3.0 | 多数据库与搜索架构参考 |
| SRC-C-11 | AppFlowy 仓库 https://github.com/AppFlowy-IO/AppFlowy | 仓库主页 | 一手 | 中 | 可访问 | Dart/Flutter；AGPL-3.0；Notion 替代型工作区 | 仅参考本地优先与编辑体验 |
| SRC-C-12 | GitHub API / shields.io | 仓库指标 | 二手聚合 | 中 | 可访问（部分限流） | Stars、最近更新、Open Issues | 仅作调研快照，不作质量结论 |

### 2.3 UX/UI 设计

| ID | 来源路径 / URL | 相关章节 / 位置 | 类型 | 可信度 | 状态 | 一句话说明 | 用途 |
|---|---|---|---|---|---|---|---|
| SRC-U-01 | LegendKeeper 功能页 https://www.legendkeeper.com/features/ | 功能页 | 一手 | 中 | 可访问 | wiki、搜索、离线、导出、默认私密 | 检索与导出体验参考 |
| SRC-U-02 | Outline 仓库与文档 | README、界面截图 | 一手 | 中 | 可访问 | 文档编辑、搜索与信息架构 | 编辑体验参考 |
| SRC-U-03 | BookStack 仓库 | README、层级模型 | 一手 | 中 | 可访问 | Books / Chapters / Pages 三层结构 | 层级信息架构参考 |
| SRC-U-04 | `PROJECT-SETUP.md` 第 8 章；`requirements.md` 8.5/FR-64~FR-69 | 约束与界面需求 | 一手（项目发起人确认） | 高 | 可访问（已确认） | 浅绿色渐变、简洁、中文、桌面优先、信息优先 | 视觉验收依据 |
| SRC-U-05 | 本项目设计系统文档 `design-system.md` | 全文（后续阶段） | 一手（项目内） | 待定 | 待验证 | 视觉规范与组件约定；当前不替代需求 | 后续 UX/实现阶段参考 |

> 本阶段未建立正式视觉稿；不得把参考产品的界面当作本项目设计决策。

### 2.4 技术 / 代码

| ID | 来源路径 / URL | 相关章节 / 位置 | 类型 | 可信度 | 状态 | 一句话说明 | 用途 |
|---|---|---|---|---|---|---|---|
| SRC-T-01 | SQLite 适用场景 https://www.sqlite.org/whentouse.html | 全文 | 一手 | 高 | 可访问 | 本地/嵌入式、单文件；单写者；多写者应选 client/server | 数据库候选判断 |
| SRC-T-02 | SQLite FTS5 https://sqlite.org/fts5.html | Tokenizer 章节 | 一手 | 高 | 可访问 | 提供 `unicode61`、`trigram` 等 tokenizer；trigram 可做子串匹配 | 中文检索候选 |
| SRC-T-03 | SQLite 外键 https://www.sqlite.org/foreignkeys.html | 全文 | 一手 | 高 | 可访问 | 外键默认关闭，必须显式启用；被引用列需索引 | 引用完整性实现约束 |
| SRC-T-04 | SQLite Backup API https://www.sqlite.org/backup.html | 全文 | 一手 | 高 | 可访问 | 提供在线备份 API；直接复制文件有锁与断电风险 | 备份方案候选 |
| SRC-T-05 | PostgreSQL 全文搜索 https://www.postgresql.org/docs/current/textsearch.html | 全文 | 一手 | 高 | 可访问 | parser、dictionary、tsvector/tsquery、ranking | 检索备选方案 |
| SRC-T-06 | PostgreSQL `pg_trgm` https://www.postgresql.org/docs/current/pg_trgm.html | 全文 | 一手 | 高 | 可访问 | trigram 相似度；GIN/GiST 索引；可用于子串/模糊匹配 | 中文检索备选 |
| SRC-T-07 | PostgreSQL `pg_dump` https://www.postgresql.org/docs/current/app-pgdump.html | 全文 | 一手 | 高 | 可访问 | 官方备份工具 | PostgreSQL 备份候选 |
| SRC-T-08 | `pg_jieba` https://github.com/jaiminpan/pg_jieba | 仓库主页 | 一手 | 中 | 待验证 | 基于 cppjieba 的 PostgreSQL 中文 parser | 仅选 PostgreSQL 时评估 |
| SRC-T-09 | `zhparser` https://github.com/amutu/zhparser | 仓库主页 | 一手 | 中 | 待验证 | 基于 SCWS 的中文全文搜索插件 | 仅选 PostgreSQL 时评估 |
| SRC-T-10 | Meilisearch 中文支持 https://www.meilisearch.com/docs/resources/help/language | Language 章节 | 一手 | 高 | 可访问 | 中文采用 jieba 词典分词；需独立搜索服务 | 外部搜索备选，不作为首选 |
| SRC-T-11 | Typesense 中文支持 https://typesense.org/docs/27.0/api/collections.html | Collections 章节 | 一手 | 高 | 可访问 | 字段级 `locale` 含 `zh`；Natural Language Search 依赖 LLM | 外部搜索备选，与零成本冲突 |
| SRC-T-12 | Docker Compose https://docs.docker.com/compose/ | 全文 | 一手 | 高 | 可访问 | 单 YAML 定义 services/networks/volumes；一条命令启动 | 本地部署约束 |
| SRC-T-13 | Docker Volumes https://docs.docker.com/engine/storage/volumes/ | 全文 | 一手 | 高 | 可访问 | volume 用于持久化、备份、恢复、迁移 | 数据持久化约束 |
| SRC-T-14 | Docker Compose 入门 https://docs.docker.com/compose/gettingstarted/ | `down -v` 说明 | 一手 | 高 | 可访问 | `docker compose down -v` 会永久删除命名 volume 数据 | 备份/操作警告 |
| SRC-T-15 | 飞书获取文档内容（docs v1） https://open.larkoffice.com/document/docs/docs-v1/get | API 文档 | 一手 | 高 | 可访问 | 可获取升级版文档内容，支持 markdown；限流 5 次/秒 | 迁移技术背景 |
| SRC-T-16 | 飞书获取块内容（docx v1） https://open.larkoffice.com/document/server-docs/docs/docs/docx-v1/document-block/get | API 文档 | 一手 | 高 | 可访问 | 获取指定 block 富文本；限流 5 次/秒 | 迁移技术背景 |
| SRC-T-17 | 飞书 Docx 概览 https://open.larkoffice.com/document/server-docs/docs/docs/docx-v1/docx-overview | 概览 | 一手 | 高 | 可访问 | Docx 提供 raw content、blocks 等接口 | 迁移技术背景 |
| SRC-T-18 | MonolithFirst https://martinfowler.com/bliki/MonolithFirst.html | 全文 | 一手 | 高 | 可访问 | 新项目先做单体；成功微服务多由单体拆分 | 架构形态参考 |
| SRC-T-19 | Local-first https://www.inkandswitch.com/local-first/ | 全文 | 一手 | 高 | 可访问 | 本地优先七项理想：数据所有权、离线、隐私、长期保存等 | 本地优先理念 |
| SRC-T-20 | Next.js https://nextjs.org/docs | 文档 | 一手 | 高 | 可访问 | React 全栈 Web 框架 | 候选技术栈清单 |
| SRC-T-21 | SvelteKit https://svelte.dev/docs/kit | 文档 | 一手 | 高 | 可访问 | Svelte Web 应用框架 | 候选技术栈清单 |
| SRC-T-22 | FastAPI https://fastapi.tiangolo.com/ | 文档 | 一手 | 高 | 可访问 | Python 类型 hints API 框架 | 候选技术栈清单 |
| SRC-T-23 | Django https://docs.djangoproject.com/en/stable/ | 文档 | 一手 | 高 | 可访问 | Python Web 框架；ORM/admin/测试 | 候选技术栈清单 |
| SRC-T-24 | `sqlite-cjk-fts` https://pypi.org/project/sqlite-cjk-fts/0.2.0/ | PyPI 页面 | 二手（第三方包） | 低中 | 待验证 | 社区 CJK bigram tokenizer 方案 | FTS5 分词备选 |
| SRC-T-25 | FTS5 external content + trigger ADR http://raw.githubusercontent.com/itsacoyote/katra/HEAD/docs/decisions/ADR-013-fts5-external-content-triggers.md | ADR 文档 | 二手（第三方项目） | 中 | 可访问 | FTS5 外部内容表与触发器同步取舍 | 索引同步参考 |
| SRC-T-26 | FTS5 日文子串问题 https://dev.to/omochi_dev/why-sqlite-fts5s-default-tokenizer-drops-your-japanese-substrings-and-the-one-line-fix-1k2d | 博客 | 二手 | 低中 | 可访问 | 无空格语言默认 tokenizer 可能丢失子串；建议 trigram | 中文问题同源参考 |

### 2.5 工程规范与合规约束

| ID | 来源路径 / URL | 相关章节 / 位置 | 类型 | 可信度 | 状态 | 一句话说明 | 用途 |
|---|---|---|---|---|---|---|---|
| SRC-G-01 | `constitution.md` | 全文 | 一手 | 高 | 可访问（已确认） | 产品、数据、架构、编码、测试、安全原则 | 工程与需求约束 |
| SRC-G-02 | `AGENTS.md` | 全文 | 一手 | 高 | 可访问（已确认） | Agent 读取顺序、真相来源、证据与安全要求 | 协作规范 |
| SRC-G-03 | Kanka LICENSE | 仓库 LICENSE | 一手 | 中 | 待验证 | Commons Clause，禁止销售；底层许可证未完整确认 | 许可证合规 |
| SRC-G-04 | Wiki.js LICENSE | 仓库 LICENSE | 一手 | 高 | 可访问 | AGPL-3.0 | 不复制代码 |
| SRC-G-05 | AppFlowy LICENSE | 仓库 LICENSE | 一手 | 高 | 可访问 | AGPL-3.0 | 不复制代码 |
| SRC-G-06 | Manuskript LICENSE | 仓库 LICENSE | 一手 | 高 | 可访问 | GPLv3 | 不复制代码 |
| SRC-G-07 | Outline LICENSE | 仓库 LICENSE | 一手 | 中 | 待验证（需法律审查） | Other / NOASSERTION | 不复制代码 |
| SRC-G-08 | BookStack LICENSE | 仓库 LICENSE | 一手 | 高 | 可访问 | MIT | 可参考模式 |
| SRC-G-09 | `PROJECT-SETUP.md` 第 7、8 章；`requirements.md` 第 8 章 | 非目标与安全约束 | 一手 | 高 | 可访问（已确认） | 零成本、本地、私密、单用户、无云端大模型 | 合规与范围约束 |

## 3. 来源冲突清单

> 以下冲突均已由项目发起人确认，并同步到对应权威文件；**需人工确认的冲突：无**。

| Conflict ID | 涉及文件 / 章节 | 冲突内容 | 来源 | 影响 | 可能选项 | 解决方案 | 决策人 / 日期 |
|---|---|---|---|---|---|---|---|
| CL-01 | 飞书《正文》体系内容 vs `PROJECT-SETUP.md` 早期四类体系 vs `requirements.md` FR-21 | 星曜/星命/十五境是否为固定类别 | SRC-B-01、SRC-B-02、SRC-B-03 | 若写死，其他作品无法使用「圣体」等类别 | A 固定四类；B 每作品自定义；C 固定 + 少量扩展 | B：每作品自定义类别（≤10 类，含内置境界）；名称不得写死 | 项目发起人 / 2026-11-06 |
| CL-02 | 飞书榜单现状 1–100 vs `requirements.md` FR-17 | 产品席位是固定 100 还是可配置 | SRC-B-01、SRC-B-03 | 影响榜单录入与迁移 | A 固定 100；B 1–200 默认 100；C 任意可配 | B：产品支持 1–200，默认 100；迁移按现状 100 | 项目发起人 / 2026-11-06 |
| CL-03 | `PROJECT-SETUP.md` 早期迁移范围 vs `requirements.md` P-02 | 素材库是否迁移 | SRC-B-02、SRC-B-03 | 迁移工作量与验收范围 | A 全部迁移；B 仅《正文》+配图；C 不迁移 | B：仅迁移《正文》+文档内现有配图；素材库不纳入本版 | 项目发起人 / 2026-11-06 |
| CL-04 | `sources.md` 早期 R-03 vs `requirements.md` FR-52~FR-56 | 备份/导出是否为 MVP | SRC-B-02、SRC-B-03 | 数据可靠性底线 | A 后补；B 进入 Must；C 仅手工备份 | B：确认 Must：全量导出、CLI 还原、快照保留 3 份 | 项目发起人 / 2026-11-06 |
| CL-05 | `PROJECT-SETUP.md` 早期「规则化问答」vs 产品实际能力 | 是否承诺自然语言问答 | SRC-B-02、SRC-B-03、SRC-B-04 | 用户预期落差与验收口径 | A 保留「问答」；B 统一为「检索」 | B：统一为「检索 + 摘要 + 出处」，不承诺推理 | 项目发起人 / 2026-11-07 |
| CL-06 | `requirements.md` 输入基线的 `project/docs/sources.md` vs 实际文件 | 来源路径不存在 | SRC-B-03 | 文档引用失效 | A 保留错误路径；B 修正为实际路径 | B：修正为 `project/sources.md` | 项目发起人 / 2026-11-07 |
| CL-07 | 早期迁移设想（兼容飞书原始排版）vs `requirements.md` FR-39 | 是否承诺自动解析任意排版 | SRC-B-01、SRC-B-03 | 迁移预期与工作量 | A 兼容任意排版；B 固定模板 + 人工校对；C 放弃迁移 | B：只解析固定模板；模板 + 图片 zip + 人工校对 | 项目发起人 / 2026-11-06 |
| CL-08 | 早期唯一性规则 vs `requirements.md` FR-48/EX-21 | 英文名称是否区分大小写 | SRC-B-03 | 重复数据风险与检索不一致 | A 区分大小写；B 不区分 | B：唯一性校验与检索不区分大小写 | 项目发起人 / 2026-11-06 |

## 4. 资料缺口

| 缺口 | 影响 | 状态 | 下一步 |
|---|---|---|---|
| V-01 中文子串检索基准 | 影响数据库/搜索选型 | 待验证（P0 技术项） | 用 20–30 条真实人物样例做基准脚本 |
| P-03a 固定模板与《正文》真实片段核对 | 影响模板定稿与迁移 | 待验证（P0 技术项） | 用真实片段走解析/校验/预览 |
| V-06 Kanka 许可证全文 | 影响是否需要法律审查 | 待验证（P1） | 读取仓库 LICENSE 全文 |
| V-07 GitHub 指标补齐 | 仅影响调研准确性 | 待验证（P1） | 恢复 API 配额后重新拉取 |
| V-08 World Anvil / Campfire 功能页与定价 | 仅影响竞品结论准确性 | 待验证（P1） | 人工浏览器访问 |
| V-09 `pg_jieba` / `zhparser` 维护状态 | 仅选 PostgreSQL 时相关 | 待验证（P2） | 查看仓库最近提交与发行版 |
| 飞书图片/附件批量下载接口 | 本版不依赖自动导出 | 已关闭（非阻塞） | 用户整理图片 zip，按模板匹配 |
| 飞书复杂 block 还原能力 | 本版只解析固定模板 | 已关闭（非阻塞） | 不承诺兼容任意原始排版 |
| 数据规模估算 | 已按 requirements 6.7 基线确认 | 已关闭 | 不再重复调研 |

## 5. 引用与维护规则

- 引用来源时必须写明路径/URL 与章节位置；不得只写「官方文档」。
- 事实、推断、建议、未知必须区分；推断不得写成结论。
- 不复制大段原文；只保留可核验结论与一句话说明。
- 新来源先登记到本索引，再更新 `research.md`；不得覆盖项目发起人已确认的产品决策。
- 来源冲突不自行裁决；记录冲突双方、影响与决策人，确认后同步权威文件与变更记录。
- 不可访问来源必须标注「未直接验证」；不得用搜索摘要冒充原文。
