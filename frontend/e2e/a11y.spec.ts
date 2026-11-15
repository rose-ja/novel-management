import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

import { cleanupE2E, createCharacter, createFaction, createLocation, createWork, uniqueName } from "./helpers";

const pages = [
  { path: "/", name: "概览" },
  { path: "/search", name: "检索" },
  { path: "/system", name: "体系" },
  { path: "/characters", name: "人物" },
  { path: "/factions", name: "势力" },
  { path: "/locations", name: "地点" },
  { path: "/rankings", name: "榜单" },
  { path: "/settings", name: "设置" },
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
