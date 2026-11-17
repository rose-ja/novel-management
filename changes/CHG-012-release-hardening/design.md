# CHG-012 设计

## 1. 跨回收站引用定位

### 1.1 深链协议

`/trash?type=<root_type>&highlight=<root_id>`

- `type`：`work|system_category|system_node|character|faction|location|ranking`，非法值忽略并按「全部」展示。
- `highlight`：回收站条目的 `root_id`。页面加载后：
  1. 以 `type` 初始化类型筛选；
  2. 以 `pageSize=100` 拉取并按页查找该条目（最多 10 页，覆盖 1000 条）；
  3. 命中则切到对应页，给名称单元格加 `data-trash-highlight`，`scrollIntoView({block:"center"})` 并将焦点移到行内「恢复」按钮（仅定位一次，后续列表刷新不再抢焦点）；
  4. 未命中显示「目标可能已被恢复或删除」提示。
- 实体随祖先整组删除时，回收站记录为祖先的 `root_id`，按实体 id 深链会走未命中提示。
- 静态导出兼容：在 `useEffect` 中读取 `window.location.search`，不引入 `useSearchParams` 的 Suspense 约束。

### 1.2 引用对话框链接

| 引用类型 | in_trash=false | in_trash=true |
|---|---|---|
| 榜单席位 | `/rankings?id=<ranking_id>` | `/trash?type=ranking&highlight=<ranking_id>` |
| 势力首领 | `/factions?id=<faction_id>` | `/trash?type=faction&highlight=<faction_id>` |
| 人物/势力的地点关系 | `/characters?id=` / `/factions?id=` | `/trash?type=character|faction&highlight=` |

## 2. 未保存守卫

统一使用 `useUnsavedGuard` + `UnsavedDialog`：

- 势力内联编辑：移除 `window.confirm`，改为守卫 Hook；保存成功后复位。
- 体系类别/节点新建与编辑对话框：关闭时脏状态先确认。
- 榜单新建/容量/调席对话框：关闭时守卫；改名未保存时拦截站内跳转；席位备注取消时确认。
- 人物/地点/作品/导入已覆盖，保持。

## 3. 状态与错误

- 统一文案常量：`loadFailed`/`saveFailed`/`networkError`/`retry`（已存在）。
- `PRECONDITION_REQUIRED`（428）：前端映射「页面数据已过期，请刷新后重试」；集成测试断言缺 `If-Match` 返回 428。
- 列表页 `loading` 使用 Skeleton；错误保留筛选值；空态区分「无数据」与「筛选无结果」（按页审计补齐）。

## 4. 可访问性与键盘

- axe 扩展：人物/势力/榜单详情（有数据）、新建/编辑/删除对话框、回收站有数据。
- 键盘主流程：Tab 到「新建人物」→ Enter → 表单可达 → Enter 保存；Esc 关闭并回焦；删除确认与回收站恢复同理。

## 5. 性能基线

- `cli seed --work <name> --characters 1000 [--factions 50] [--locations 50]`：单事务批量写入，刷新检索。
- `scripts/perf.ps1`：
  1. `docker compose run --rm cli seed ...`；
  2. 20 次 `GET /characters?page=1&page_size=50` 与 `GET /search?q=...`，去掉首请求，计算 P95；
  3. 调用 `npx playwright test e2e/perf.spec.ts`（需 `PERF_WORK_ID`）测量首屏可交互；
  4. 输出 `artifacts/perf/<date>/report.json`（gitignored）。
- 阈值：首屏 ≤2000ms、列表 P95 ≤500ms、检索 P95 ≤150ms；超标即失败。

## 6. 发布演练

- `docker compose up -d` → health；导出 zip → 一次性 `cli` 容器恢复 → 计数一致；`restore-snapshot`；`cli migrate` Up/Down 单元；README 步骤逐条可执行。
