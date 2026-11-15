import { expect, test } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createFaction,
  createLocation,
  createSystemCategory,
  createSystemNode,
  createWork,
  uniqueName,
} from "./helpers";

const CHARACTER_NAME = "E2E检索陆沉";
const FACTION_NAME = "E2E检索青丘";
const LOCATION_NAME = "E2E检索青丘山";

function resultLink(page: import("@playwright/test").Page, text: string) {
  return page.locator(".search-result").filter({ hasText: text });
}

function trackSearchRequests(page: import("@playwright/test").Page): string[] {
  const seen: string[] = [];
  page.on("request", (request) => {
    if (request.url().includes("/api/v1/search?")) seen.push(request.url());
  });
  return seen;
}

test.describe("检索", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("快速搜索：自动聚焦、单字命中、分组高亮与返回恢复", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("检索快速"));
    const faction = await createFaction(request, work.id, FACTION_NAME);
    const character = await createCharacter(request, work.id, CHARACTER_NAME, {
      names: [
        { kind: "alias", value: "沉舟" },
        { kind: "title", value: "剑主" },
      ],
      description: "青丘之主。",
      factionID: faction.id,
    });
    await createLocation(request, work.id, LOCATION_NAME);

    await page.goto("/search");
    await expect(page.getByRole("heading", { name: "检索", level: 1 })).toBeVisible();

    const input = page.getByLabel("检索关键词");
    await expect(input).toBeFocused();
    await expect(page.getByRole("heading", { name: "输入关键词或选择筛选条件" })).toBeVisible();

    await input.fill("沉");
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();
    await expect(page).toHaveURL(/\/search\?q=%E6%B2%89/);

    const charGroup = page
      .locator(".search-group")
      .filter({ has: page.getByRole("heading", { name: "人物", level: 2 }) });
    await expect(charGroup).toBeVisible();
    await expect(charGroup.locator(".ui-badge").first()).toHaveText("1");
    await expect(charGroup.locator("mark.search-mark").first()).toContainText("沉");
    await expect(charGroup.locator("mark.search-mark").nth(1)).toContainText("沉舟");
    await expect(resultLink(page, CHARACTER_NAME)).toContainText(FACTION_NAME);
    await expect(page.getByText("共 1 条 · 当前作品内")).toBeVisible();

    await resultLink(page, CHARACTER_NAME).click();
    await expect(page).toHaveURL(new RegExp(`/characters\\?id=${character.id}`));
    await expect(page.getByRole("heading", { name: CHARACTER_NAME, level: 1 })).toBeVisible();

    await page.goBack();
    await expect(page).toHaveURL(/\/search\?q=%E6%B2%89/);
    await expect(input).toHaveValue("沉");
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();
  });

  test("300ms 防抖合并连续输入，Enter 立即检索", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("检索防抖"));
    await createCharacter(request, work.id, CHARACTER_NAME, { description: "青丘之主。" });

    const requests = trackSearchRequests(page);
    await page.goto("/search");
    const input = page.getByLabel("检索关键词");

    await input.pressSequentially("陆沉", { delay: 50 });
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();
    expect(requests).toHaveLength(1);
    expect(requests[0]).toContain("q=%E9%99%86%E6%B2%89");

    await input.fill("青丘");
    await input.press("Enter");
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();
    await expect.poll(() => requests.length).toBe(2);
    await page.waitForTimeout(400);
    expect(requests).toHaveLength(2);
    expect(requests[1]).toContain("q=%E9%9D%92%E4%B8%98");
  });

  test("条件检索：仅筛选即可检索，移除 chip 回到空态", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("检索条件"));
    const faction = await createFaction(request, work.id, FACTION_NAME);
    await createCharacter(request, work.id, CHARACTER_NAME, { factionID: faction.id });

    await page.goto("/search");
    await page.getByRole("button", { name: "条件检索" }).click();
    await expect(page).toHaveURL(/mode=conditional/);

    await page.getByLabel("实体类型").selectOption({ label: "人物" });
    await page.getByLabel("势力/分支").selectOption({ label: FACTION_NAME });
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();

    const chip = page.getByRole("button", { name: `移除筛选：势力/分支 ${FACTION_NAME}` });
    await expect(chip).toBeVisible();
    await expect(page).toHaveURL(/faction=/);
    await chip.click();

    await expect(page.getByRole("heading", { name: "输入关键词或选择筛选条件" })).toBeVisible();
    await expect(resultLink(page, CHARACTER_NAME)).toHaveCount(0);
  });

  test("条件检索：仅输入关键词可提交检索", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("检索条件词"));
    await createCharacter(request, work.id, CHARACTER_NAME);

    await page.goto("/search");
    await page.getByRole("button", { name: "条件检索" }).click();
    await page.getByLabel("检索关键词").fill("陆沉");

    const submit = page.getByRole("button", { name: "检索", exact: true });
    await expect(submit).toBeEnabled();
    await submit.click();
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();
  });

  test("空查询不请求、无结果与清除筛选", async ({ page, request }) => {
    await createWork(request, uniqueName("检索空态"));

    const requests = trackSearchRequests(page);
    await page.goto("/search");
    await page.waitForTimeout(500);
    expect(requests).toHaveLength(0);
    await expect(page.getByRole("heading", { name: "输入关键词或选择筛选条件" })).toBeVisible();

    const input = page.getByLabel("检索关键词");
    await input.fill("绝不存在的关键词xyz");
    await expect(page.getByRole("heading", { name: "未找到匹配条目" })).toBeVisible();

    await page.getByRole("button", { name: "清除筛选" }).click();
    await expect(input).toHaveValue("");
    await expect(page.getByRole("heading", { name: "输入关键词或选择筛选条件" })).toBeVisible();
  });

  test("检索失败展示错误并可重试", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("检索错误"));
    await createCharacter(request, work.id, CHARACTER_NAME);

    let fail = true;
    await page.route("**/api/v1/search?*", async (route) => {
      if (!fail) {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: { code: "INTERNAL_ERROR", message: "模拟检索故障" } }),
      });
    });

    await page.goto("/search");
    await page.getByLabel("检索关键词").fill("陆沉");
    await expect(page.locator(".ui-alert--danger")).toContainText("检索失败，请重试");
    await expect(page.locator(".ui-alert--danger")).toContainText("服务内部错误，请稍后重试");

    fail = false;
    await page.getByRole("button", { name: "重试" }).click();
    await expect(resultLink(page, CHARACTER_NAME)).toBeVisible();
  });

  test("体系节点结果深链定位类别与节点", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("检索深链"));
    const category = await createSystemCategory(request, work.id, uniqueName("检索境界"));
    const node = await createSystemNode(request, category.id, "E2E检索金丹境");

    await page.goto("/search");
    await page.getByLabel("检索关键词").fill("金丹");
    await expect(resultLink(page, "E2E检索金丹境")).toBeVisible();

    await resultLink(page, "E2E检索金丹境").click();
    await expect(page).toHaveURL(new RegExp(`/system\\?category=${category.id}&node=${node.id}`));
    await expect(page.locator(".system-detail__title")).toHaveText(category.name);

    const treeitem = page.getByRole("treeitem").filter({ hasText: "E2E检索金丹境" });
    await expect(treeitem).toBeVisible();
    await expect(treeitem).toHaveAttribute("aria-selected", "true");
  });
});
