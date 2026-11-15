# CHG-009 检查清单

## 范围与基线

- [x] 已阅读 `project/AGENTS.md`、`constitution.md`、roadmap §CHG-009、SPEC-SEARCH-001~005、PG-01、API 契约 §2.7/§3.9、data-model §3.15
- [x] 确认无上游冲突；已知偏差（`review_status`/`faction_scope` 参数、`/search` 非默认页、Select 替代 TreeSelect）记录在 `design.md`
- [x] 非目标明确：不做语义/拼音/同义词/跨作品/排序自定义/检索缓存

## 后端

- [ ] 迁移 0009：表、CHECK、唯一键、索引、Up/Down
- [ ] `Refresher.RefreshWork` 覆盖五类、软删除、物理删除、多关系聚合
- [ ] `RebuildAll` 与 `Consistency` 可用
- [ ] 人物/势力写入接入刷新，改名/删除/恢复同步
- [ ] 地点/榜单/体系写入接入刷新
- [ ] 回收站 Restore/Undo/Purge/PurgeAll 与作品恢复接入
- [ ] `GET /api/v1/search` 参数、分词、评分、筛选、分页、DTO
- [ ] 空查询 400、非法参数/失效筛选 422、作品不存在 404
- [ ] `X-Cache: BYPASS`
- [ ] `cli reindex` 与 `--check`
- [ ] 启动时 `RebuildAll` 容错

## 前端

- [ ] `/search` 路由、导航入口、无作品空态
- [ ] 快速搜索：autofocus、1 字符、300ms、Enter、IME、Esc
- [ ] 条件检索：实体类型、势力/分支 scope、境界、地点、榜单、chip、重置
- [ ] 分组结果、组头总数、当前页切片、`<mark>` 高亮、subtitle/url
- [ ] 分页 20、保留状态、返回检索态
- [ ] 空查询/无结果/错误/加载态
- [ ] 失效筛选移除并提示
- [ ] a11y：可见标签、section/aria-live/aria-busy、键盘
- [ ] 1024/1280 响应式

## 测试

- [ ] 后端单元 + 集成（命中矩阵/AND/隔离/改名/删除恢复/筛选/分页/错误/一致性）
- [ ] 前端单测（API 参数、URL 状态、筛选适用性）
- [ ] E2E search + a11y + keyboard + 视觉
- [ ] 1000 人物 P95 ≤150ms 记录
- [ ] 全量门禁与 Node22 构建

## 文档与收尾

- [ ] `verification.md`、`review.md`、`pr.md`、`acceptance.md`
- [ ] 回写 SPEC-SEARCH-001~005、PG-01、API 契约、data-model、roadmap
- [ ] 归档到 `changes/archive/CHG-009-search/`
- [ ] 独立提交与推送
