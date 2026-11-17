import { expect, test } from "@playwright/test";

import {
  assignRankingSeat,
  cleanupE2E,
  createCharacter,
  createFaction,
  createRanking,
  createSystemNode,
  createWork,
  listSystemCategories,
  uniqueName,
} from "./helpers";

/**
 * E2E-08 一致性：人物改名、换势力、改境界后，
 * 人物列表、势力详情、榜单席位与检索结果必须同步。
 */
test.describe("跨页面一致性", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("改名、换势力与改境界后列表、势力详情、榜单与检索同步", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("一致性"));
    const oldFaction = await createFaction(request, work.id, uniqueName("旧势力"));
    const newFaction = await createFaction(request, work.id, uniqueName("新势力"));
    const categories = await listSystemCategories(request, work.id);
    const realm = categories.find((category) => category.is_builtin);
    if (!realm) throw new Error("内置境界类别缺失");
    const oldRealm = await createSystemNode(request, realm.id, uniqueName("金丹期"));
    const newRealm = await createSystemNode(request, realm.id, uniqueName("元婴期"));

    const oldName = uniqueName("陆沉");
    const newName = uniqueName("陆沉改");
    const character = await createCharacter(request, work.id, oldName, {
      factionID: oldFaction.id,
      systemRefs: [{ category_id: realm.id, node_id: oldRealm.id }],
    });
    const ranking = await createRanking(request, work.id, uniqueName("天榜"), 5);
    await assignRankingSeat(request, ranking.id, 1, character.id, ranking.seats.items[0].version);

    // 改名：人物列表、势力详情、榜单席位与检索同步。
    await page.goto(`/characters?id=${character.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    await page.getByLabel("姓名").fill(newName);
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.getByRole("heading", { name: newName, level: 1 })).toBeVisible();

    await page.goto("/characters");
    const listRow = page.locator(".ui-table tbody tr").filter({ hasText: newName });
    await expect(listRow).toBeVisible();
    await expect(listRow).toContainText(oldFaction.name);
    await expect(listRow).toContainText(oldRealm.name);

    await page.goto(`/factions?id=${oldFaction.id}`);
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: newName })).toBeVisible();

    await page.goto(`/rankings?id=${ranking.id}`);
    const seatRow = page.locator(".ui-table tbody tr").filter({ hasText: newName });
    await expect(seatRow).toBeVisible();

    await page.goto("/search");
    await page.getByLabel("检索关键词").fill(newName);
    await expect(page.locator(".search-result").filter({ hasText: newName }).filter({ hasText: "人物" })).toHaveCount(1);

    await page.getByLabel("检索关键词").fill(oldName);
    await expect(page.getByText("未找到匹配条目")).toBeVisible();

    // 换势力与改境界：势力详情、人物列表与检索同步。
    await page.goto(`/characters?id=${character.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    await page.getByLabel("所属势力").selectOption({ label: newFaction.name });
    await page.getByLabel("境界").selectOption({ label: newRealm.name });
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.getByRole("heading", { name: newName, level: 1 })).toBeVisible();

    await page.goto(`/factions?id=${oldFaction.id}`);
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: newName })).toHaveCount(0);

    await page.goto(`/factions?id=${newFaction.id}`);
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: newName })).toBeVisible();

    await page.goto("/characters");
    const updatedRow = page.locator(".ui-table tbody tr").filter({ hasText: newName });
    await expect(updatedRow).toContainText(newFaction.name);
    await expect(updatedRow).toContainText(newRealm.name);
    await expect(updatedRow).not.toContainText(oldRealm.name);

    await page.goto("/search");
    await page.getByLabel("检索关键词").fill(newName);
    const searchResult = page.locator(".search-result").filter({ hasText: newName }).filter({ hasText: "人物" });
    await expect(searchResult).toHaveCount(1);
    await expect(searchResult).toContainText(newFaction.name);
    await expect(searchResult).toContainText(newRealm.name);
  });
});
