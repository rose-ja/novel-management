# CHG-012 独立审查与修复记录（F4/F5）

- 审查范围：`7ecb0e6..c34617c`（CHG-012 实现与测试），修复后复审至 `4c54507`
- 审查方式：独立只读审查（非实现者视角），随后由实现者逐条核对、修复并回归
- 结论：发现 1 个高优先级、5 个中优先级、5 个低优先级；阻断/高优先级与可修项已全部处理，其余记录为已知限制

## 1. 高优先级

| # | 问题 | 处置 | 验证 |
|---|---|---|---|
| 1 | 地点删除对话框的回收站引用链接未使用 `trashHrefFor`，仍硬编码 `/trash`，深链定位不生效 | `inTrash` 分支改用 `item.href` | `locations.spec.ts` 删除/回收站用例 PASS；E2E 全量 PASS |

## 2. 中优先级

| # | 问题 | 处置 | 验证 |
|---|---|---|---|
| 1 | 性能用例永久改写 `current_work_id`，结束后应用指向造数作品 | `beforeAll` 记录原值，`afterAll` 恢复 | `scripts/perf.ps1` 复跑 PASS，应用当前作品恢复 |
| 2 | `perf.ps1` 的 `-BaseUrl` 未传给 Playwright，指向其他实例时首屏测错对象 | 设置 `$env:E2E_BASE_URL = $BaseUrl` | perf 复跑 PASS |
| 3 | `cli seed` 只校验负数，无上限，超大参数可能 OOM | 人物上限 100000、势力/地点上限 10000 | `go build`/`go vet` PASS |
| 4 | 体系新增类别 dirty 只看名称，仅切换「普通/纪元」后关闭不确认 | dirty 并入 `categoryType !== "normal"` | `system.spec.ts` 脏状态用例 PASS |
| 5 | 文档声称榜单改名/席位备注/节点树切换已加守卫，代码未覆盖 | 实现榜单改名离开守卫与席位备注取消确认；删除节点树切换的失实描述，并同步 design/tasks | 新增 `rankings.spec.ts`「榜单改名未保存时离开需确认」PASS |

## 3. 低优先级

| # | 问题 | 处置 |
|---|---|---|
| 1 | 回收站定位依赖 `data`，列表刷新会重复滚动并抢焦点 | 增加一次性标记 `scrolledFor`，仅定位一次 |
| 2 | 深链把实体 id 当 `root_id`，祖先整组删除时未命中 | 记录为已知限制（design §1.1、verification §4） |
| 3 | perf 脚本中文注释与 ASCII 自述矛盾；空 stdout 会抛空引用；`-Samples 0` 可假通过 | 注释改英文；空输出显式报错；`Samples < 1` 直接失败 |
| 4 | design 写 `data-highlighted`/`type=import`，实现为 `data-trash-highlight`/无 import | 回写 design，移除 import，修正属性名 |
| 5 | 势力删除对话框 `in_trash` 分支后端不赋值，恒不可达 | 保留现状（无行为影响），记录于 verification §4 |

## 4. 复审结果

- 修复后回归：后端单测、集成、前端类型/lint/单测、镜像构建、受影响 E2E 59/59、性能脚本均 PASS。
- E2E 全量在此前基线上 128 passed / 1 skipped；修复未触及未回归模块。
- 未发现新增阻断或高优先级问题。
