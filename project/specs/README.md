# Specs

`project/specs/` 保存行为规格及其状态。状态定义：

| 状态 | 含义 |
|---|---|
| `planned` | P6 定义的目标行为，尚未实现；不得当作当前系统能力 |
| `implemented` | 已合并到主分支，尚未完成发布验收 |
| `verified` | 已通过人工/自动化验收，并在目标环境发布 |

按业务域组织，例如：
- `works/`（作品与切换）
- `characters/`（人物、别名、称号、图片、大事记）
- `factions/`（势力树与成员）
- `locations/`（地点树与关联）
- `rankings/`（榜单与席位）
- `systems/`（体系类别与节点）
- `search/`（当前作品检索）
- `migration/`（飞书《正文》与图片迁移）
- `backup/`（导出、还原、快照）
- `trash/`（回收站与撤销）
- `platform/`（应用壳层、全局状态、数据完整性、部署与性能等跨域约束）

> `platform/` 为 P6 新增的跨域目录：README 原列表为示例，P6 用它承载 PG-00 与 EX/NFR 类跨域行为。

每条 Spec 使用稳定 ID，包含来源 Requirement/User Story/Page Spec ID、行为描述、边界、验收场景（Given/When/Then）、状态和变更历史。P6 新建的 Spec 一律标记 `planned`；F10 只把已合并且验收的行为更新为 `implemented` 或 `verified`，未实现行为不得写成当前能力。
