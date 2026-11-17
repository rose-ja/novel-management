import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

import {
  assignRankingSeat,
  cleanupE2E,
  createCharacter,
  createFaction,
  createLocation,
  createRanking,
  createWork,
  deleteCharacter,
  getRanking,
  uniqueName,
} from "./helpers";

const pages = [
  { path: "/", name: "概览" },
  { path: "/search", name: "检索" },
  { path: "/system", name: "体系" },
  { path: "/characters", name: "人物" },
  { path: "/factions", name: "势力" },
  { path: "/locations", name: "地点" },
  { path: "/rankings", name: "榜单" },
  { path: "/settings", name: "设置" },
  { path: "/import", name: "导入" },
  { path: "/trash", name: "回收站" },
  { path: "/no-such-page", name: "404" },
];

for (const target of pages) {
  test(`${target.name} 无 serious/critical 无障碍问题`, async ({ page }) => {
    await page.goto(target.path);

    const results = await new AxeBuilder({ page }).analyze();
    const blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );

    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);
  });
}

/** 有数据的地点树与详情需要覆盖树行、关系区与删除对话框。 */
test.describe("地点：有数据无障碍", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("地点树与详情无 serious/critical 问题", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("a11y地点"));
    const root = await createLocation(request, work.id, uniqueName("东荒"), { typeText: "星域" });
    const child = await createLocation(request, work.id, uniqueName("青丘山"), {
      parentID: root.id,
      typeText: "星城",
    });

    await page.goto("/locations");
    await expect(page.locator(".location-tree__row")).toHaveCount(2);
    let results = await new AxeBuilder({ page }).analyze();
    let blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );
    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);

    await page.goto(`/locations?id=${child.id}`);
    await expect(page.getByRole("heading", { name: child.name, level: 1 })).toBeVisible();
    results = await new AxeBuilder({ page }).analyze();
    blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );
    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);

    await page.locator(".ui-page-header").getByRole("button", { name: "删除地点" }).click();
    results = await new AxeBuilder({ page }).include('[role="dialog"]').analyze();
    blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );
    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);
  });
});

/** 有数据的检索结果与条件筛选需要覆盖分组、高亮与筛选 chip。 */
test.describe("检索：有数据无障碍", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("检索结果与条件筛选无 serious/critical 问题", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("a11y检索"));
    const faction = await createFaction(request, work.id, uniqueName("狐族"));
    await createCharacter(request, work.id, uniqueName("陆沉"), {
      names: [{ kind: "alias", value: "沉舟" }],
      description: "青丘之主。",
      factionID: faction.id,
    });
    await createLocation(request, work.id, uniqueName("青丘山"));

    await page.goto("/search");
    await page.getByLabel("检索关键词").fill("沉");
    await expect(page.locator(".search-result")).toHaveCount(1);

    let results = await new AxeBuilder({ page }).analyze();
    let blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );
    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);

    await page.getByRole("button", { name: "条件检索" }).click();
    await page.getByLabel("实体类型").selectOption({ label: "人物" });
    await page.getByLabel("势力/分支").selectOption({ label: faction.name });
    await expect(page.locator(".search-chip")).toHaveCount(1);

    results = await new AxeBuilder({ page }).analyze();
    blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );
    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);
  });
});

/** 统一 axe 断言：可限定扫描范围，只拦截 serious/critical。 */
async function expectNoBlockingViolations(page: Page, include?: string): Promise<void> {
  const builder = new AxeBuilder({ page });
  if (include) builder.include(include);
  const results = await builder.analyze();
  const blocking = results.violations.filter(
    (violation) => violation.impact === "serious" || violation.impact === "critical",
  );
  expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);
}

/** 人物详情有数据，覆盖编辑内联表单与新建/删除对话框。 */
test.describe("人物：有数据无障碍", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("人物详情、编辑与新建/删除对话框无 serious/critical 问题", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("a11y人物"));
    const faction = await createFaction(request, work.id, uniqueName("青云宗"));
    const character = await createCharacter(request, work.id, uniqueName("陆沉"), {
      names: [{ kind: "alias", value: "沉舟" }],
      description: "青云宗首徒。",
      factionID: faction.id,
    });

    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: character.name, level: 1 })).toBeVisible();
    await expectNoBlockingViolations(page);

    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    await expect(page.getByRole("button", { name: "保存修改" })).toBeVisible();
    await expectNoBlockingViolations(page);

    await page.goto("/characters");
    await page.locator(".ui-page-header").getByRole("button", { name: "新建人物" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
    await page.getByRole("dialog").getByRole("button", { name: "关闭" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    await page.goto(`/characters?id=${character.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "删除人物" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
  });
});

/** 势力详情有数据，覆盖编辑表单与删除对话框。 */
test.describe("势力：有数据无障碍", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("势力详情、编辑表单与删除对话框无 serious/critical 问题", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("a11y势力"));
    const parent = await createFaction(request, work.id, uniqueName("天门"));
    const faction = await createFaction(request, work.id, uniqueName("青云峰"), {
      parentID: parent.id,
      description: "天门下属山峰。",
    });

    await page.goto(`/factions?id=${faction.id}`);
    await expect(page.getByRole("heading", { name: faction.name, level: 1 })).toBeVisible();
    await expectNoBlockingViolations(page);

    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    await expect(page.getByRole("button", { name: "保存修改" })).toBeVisible();
    await expectNoBlockingViolations(page);

    await page.goto(`/factions?id=${faction.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "删除势力" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
  });
});

/** 榜单详情有数据，覆盖新建、容量与删除对话框。 */
test.describe("榜单：有数据无障碍", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("榜单详情、容量与新建/删除对话框无 serious/critical 问题", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("a11y榜单"));
    const character = await createCharacter(request, work.id, uniqueName("甲"));
    const ranking = await createRanking(request, work.id, uniqueName("天榜"), 5);
    const seatVersion = ranking.seats.items[0].version;
    await assignRankingSeat(request, ranking.id, 1, character.id, seatVersion);

    await page.goto(`/rankings?id=${ranking.id}`);
    await expect(page.getByRole("heading", { name: ranking.name, level: 1 })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr")).toHaveCount(5);
    await expectNoBlockingViolations(page);

    await page.locator(".ui-page-header").getByRole("button", { name: "调整席位容量" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);

    await page.goto("/rankings");
    await page.locator(".ui-page-header").getByRole("button", { name: "新建榜单" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);

    const row = page.locator(".ui-table tbody tr").filter({ hasText: ranking.name });
    await row.getByRole("button", { name: "删除榜单" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
  });
});

/** 回收站有数据，覆盖列表行与恢复对话框。 */
test.describe("回收站：有数据无障碍", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("回收站列表与恢复对话框无 serious/critical 问题", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("a11y回收站"));
    const character = await createCharacter(request, work.id, uniqueName("陆沉"));
    await deleteCharacter(request, character);

    await page.goto("/trash");
    await expect(page.getByRole("heading", { name: "回收站", level: 1 })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: character.name })).toBeVisible();
    await expectNoBlockingViolations(page);

    await page.getByRole("button", { name: `恢复人物「${character.name}」` }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expectNoBlockingViolations(page, '[role="dialog"]');
  });
});
