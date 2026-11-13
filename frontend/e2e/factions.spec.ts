import { expect, test } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createFaction,
  createWork,
  deleteFaction,
  listFactions,
  uniqueName,
} from "./helpers";

function treeRow(page: import("@playwright/test").Page, name: string) {
  return page.locator(".faction-tree__row").filter({ hasText: name });
}

test.describe("势力树", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("三级势力默认展开两级，搜索命中并展开祖先", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("势力树"));
    const wan = await createFaction(request, work.id, uniqueName("万族"));
    const yao = await createFaction(request, work.id, uniqueName("妖族"), { parentID: wan.id });
    const qingqiu = await createFaction(request, work.id, uniqueName("青丘"), { parentID: yao.id });
    const ren = await createFaction(request, work.id, uniqueName("人族"));

    await page.goto("/factions");
    await expect(page.getByRole("heading", { name: "势力", level: 1 })).toBeVisible();
    await expect(treeRow(page, wan.name)).toBeVisible();
    await expect(treeRow(page, yao.name)).toBeVisible();
    await expect(treeRow(page, qingqiu.name)).toBeVisible();
    await expect(treeRow(page, ren.name)).toBeVisible();

    // 搜索命中青丘时只保留命中与祖先。
    await page.getByLabel("搜索势力").fill("青丘");
    await expect(treeRow(page, qingqiu.name)).toBeVisible();
    await expect(treeRow(page, wan.name)).toBeVisible();
    await expect(treeRow(page, yao.name)).toBeVisible();
    await expect(treeRow(page, ren.name)).toHaveCount(0);
    await expect(page.locator(".faction-tree__mark")).toHaveText("青丘");

    await page.getByLabel("搜索势力").fill("不存在的势力");
    await expect(page.getByText("没有匹配的势力节点")).toBeVisible();
    await page.getByRole("button", { name: "清除筛选" }).click();
    await expect(treeRow(page, qingqiu.name)).toBeVisible();
  });

  test("内联新建与详情编辑保存后同步", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("势力编辑"));
    const leader = await createCharacter(request, work.id, uniqueName("首领"));
    const rootName = uniqueName("万族");
    const childName = uniqueName("妖族");

    await page.goto("/factions");
    await page.locator(".ui-page-header").getByRole("button", { name: "新建势力" }).click();
    await page.getByLabel("势力名称").fill(rootName);
    await page.getByRole("button", { name: "创建势力" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已创建势力「${rootName}」` })).toBeVisible();
    await expect(treeRow(page, rootName)).toBeVisible();

    // 通过行内「下级」按钮以内联方式创建子势力。
    await treeRow(page, rootName).getByRole("button", { name: `新增下级势力：${rootName}` }).click();
    await expect(page.getByLabel("上级势力")).toHaveValue(/.+/);
    await page.getByLabel("势力名称").fill(childName);
    await page.getByRole("button", { name: "创建势力" }).click();
    await expect(treeRow(page, childName)).toBeVisible();

    await treeRow(page, childName).getByRole("link", { name: childName }).click();
    await expect(page.getByRole("heading", { name: childName, level: 1 })).toBeVisible();
    await expect(page.getByText("暂无直属成员")).toBeVisible();

    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    // 首领按名称服务端搜索后再选择（PG-05 §6.2.3）。
    await page.getByLabel("搜索首领人物").fill(leader.name);
    await page.getByLabel("首领", { exact: true }).selectOption({ label: leader.name });
    await page.getByLabel("简介").fill("势力视觉简介。");
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已保存势力「${childName}」` })).toBeVisible();
    await expect(page.locator(".faction-summary")).toContainText(leader.name);
    await expect(page.getByText("势力视觉简介。")).toBeVisible();

    // 上级选项保留自身但在下拉中禁用，避免误选成环。
    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    const selfOption = page.getByLabel("上级势力").locator("option").filter({ hasText: childName });
    await expect(selfOption).toHaveCount(1);
    await expect(selfOption).toBeDisabled();
    await page.getByRole("button", { name: "取消" }).click();
  });
});

test.describe("势力成员与人物侧", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("直属/含下级切换与人物势力链接", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("势力成员"));
    const parent = await createFaction(request, work.id, uniqueName("万族"));
    const child = await createFaction(request, work.id, uniqueName("妖族"), { parentID: parent.id });
    const directMember = await createCharacter(request, work.id, uniqueName("直属甲"), { factionID: parent.id });
    const childMember = await createCharacter(request, work.id, uniqueName("下级乙"), { factionID: child.id });

    await page.goto(`/factions?id=${parent.id}`);
    const stats = page.locator(".faction-summary__item dd");
    await expect(stats.nth(1)).toHaveText("1");
    await expect(stats.nth(2)).toHaveText("2");
    await expect(stats.nth(3)).toHaveText("1");
    await expect(page.getByRole("link", { name: child.name }).first()).toBeVisible();
    await expect(page.getByRole("link", { name: directMember.name })).toBeVisible();
    await expect(page.getByRole("link", { name: childMember.name })).toHaveCount(0);

    await page.getByRole("button", { name: "含下级" }).click();
    await expect(page.getByRole("link", { name: childMember.name })).toBeVisible();
    await expect(page.getByRole("link", { name: directMember.name })).toBeVisible();

    await page.goto(`/characters?id=${directMember.id}`);
    await expect(page.getByRole("heading", { name: directMember.name, level: 1 })).toBeVisible();
    await page.getByRole("link", { name: `查看势力「${parent.name}」` }).first().click();
    await expect(page).toHaveURL(new RegExp(`/factions\\?id=${parent.id}`));
    await expect(page.getByRole("heading", { name: parent.name, level: 1 })).toBeVisible();
  });
});

test.describe("势力删除与回收站", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("删除被下级阻止并列出引用", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("势力删除"));
    const parent = await createFaction(request, work.id, uniqueName("上级"));
    const child = await createFaction(request, work.id, uniqueName("下级"), { parentID: parent.id });

    await page.goto(`/factions?id=${parent.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "删除势力" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除势力" }).click();
    await expect(page.getByRole("dialog")).toContainText("仍有下级势力，无法删除");
    await expect(page.getByRole("dialog").getByRole("link", { name: child.name })).toBeVisible();
  });

  test("软删下级在删除阻断中标出回收站", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("回收站下级"));
    const parent = await createFaction(request, work.id, uniqueName("回收站上级"));
    const child = await createFaction(request, work.id, uniqueName("已删下级"), { parentID: parent.id });
    await deleteFaction(request, child);

    await page.goto(`/factions?id=${parent.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "删除势力" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除势力" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toContainText("仍有下级势力，无法删除");
    await expect(dialog.locator("li").filter({ hasText: child.name })).toContainText("（回收站）");
    await expect(dialog.getByRole("link", { name: child.name })).toHaveAttribute("href", "/trash");
  });

  test("回收站恢复与永久删除势力根", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("势力回收站"));
    const faction = await createFaction(request, work.id, uniqueName("待删势力"));
    await deleteFaction(request, faction);

    await page.goto("/trash");
    await page.getByLabel("类型", { exact: true }).selectOption("faction");
    const row = page.locator(".ui-table tbody tr").filter({ hasText: faction.name });
    await expect(row).toBeVisible();
    await expect(row).toContainText(work.name);

    await page.getByRole("button", { name: `恢复势力「${faction.name}」` }).click();
    await page.getByRole("dialog").getByRole("button", { name: "恢复" }).click();
    await expect(page.locator(".ui-toast")).toContainText("已恢复势力");
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: faction.name })).toHaveCount(0);

    const [restored] = await listFactions(request, work.id);
    await deleteFaction(request, restored);
    await page.reload();
    await page.getByLabel("类型", { exact: true }).selectOption("faction");
    await page.getByRole("button", { name: `永久删除势力「${faction.name}」` }).click();
    await page.getByRole("dialog").getByRole("button", { name: "永久删除" }).click();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: faction.name })).toHaveCount(0);
  });
});
