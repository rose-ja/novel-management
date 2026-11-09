import { expect, test } from "@playwright/test";

import {
  cleanupE2E,
  createSystemCategory,
  createSystemNode,
  createWork,
  deleteSystemCategory,
  deleteSystemNode,
  deleteWork,
  listSystemCategories,
  listSystemNodes,
  listTrash,
  uniqueName,
} from "./helpers";

/** 回收站中的体系类别/节点根：类型 Badge、名称、父级、筛选、恢复与永久删除。 */
test.describe("回收站：体系根", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("类别整组显示为「体系类别」，筛选后整组恢复", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("体系根"));
    const otherWork = await createWork(request, uniqueName("对照作品"));
    const category = await createSystemCategory(request, work.id, uniqueName("回收类别"));
    const parent = await createSystemNode(request, category.id, uniqueName("父层级"));
    await createSystemNode(request, category.id, uniqueName("子层级"), parent.id);

    await deleteWork(request, otherWork);
    await deleteSystemCategory(request, category.id, category.version);

    await page.goto("/trash");
    const categoryRow = page.locator(".ui-table tbody tr").filter({ hasText: category.name });
    await expect(categoryRow).toBeVisible();
    await expect(categoryRow.locator(".ui-badge")).toHaveText("体系类别");
    await expect(categoryRow).not.toContainText(`《${category.name}》`);
    await expect(categoryRow).toContainText(work.name);
    await expect(categoryRow).toContainText("共 2 个节点");
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: otherWork.name })).toBeVisible();

    await page.getByLabel("类型").selectOption("system_category");
    await expect(categoryRow).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: otherWork.name })).toHaveCount(0);

    await page.getByRole("button", { name: `恢复体系类别「${category.name}」` }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText(`整组恢复类别「${category.name}」？`)).toBeVisible();
    await expect(dialog).toContainText("整组恢复类别及其节点");
    await dialog.getByRole("button", { name: "恢复" }).click();

    await expect(page.locator(".ui-toast").filter({ hasText: `已恢复体系类别「${category.name}」` })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: category.name })).toHaveCount(0);

    const categories = await listSystemCategories(request, work.id);
    expect(categories.some((item) => item.id === category.id)).toBe(true);
    const nodes = await listSystemNodes(request, category.id);
    expect(nodes.map((node) => node.name)).toContain(parent.name);
  });

  test("节点显示为「体系节点」，筛选后可永久删除", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("节点根"));
    const category = await createSystemCategory(request, work.id, uniqueName("节点类别"));
    const node = await createSystemNode(request, category.id, uniqueName("叶子节点"));

    await deleteSystemNode(request, node.id, node.version);

    await page.goto("/trash");
    const nodeRow = page.locator(".ui-table tbody tr").filter({ hasText: node.name });
    await expect(nodeRow).toBeVisible();
    await expect(nodeRow.locator(".ui-badge")).toHaveText("体系节点");
    await expect(nodeRow).not.toContainText(`《${node.name}》`);
    await expect(nodeRow).toContainText(category.name);

    await page.getByLabel("类型").selectOption("system_node");
    await expect(nodeRow).toBeVisible();

    await page.getByRole("button", { name: `永久删除体系节点「${node.name}」` }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText(`永久删除体系节点「${node.name}」？`)).toBeVisible();
    await expect(dialog).toContainText("不可恢复");
    await dialog.getByRole("button", { name: "永久删除" }).click();

    await expect(page.locator(".ui-toast").filter({ hasText: `已永久删除体系节点「${node.name}」` })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: node.name })).toHaveCount(0);

    const remaining = await listTrash(request, "system_node");
    expect(remaining.some((item) => item.root_id === node.id)).toBe(false);
  });
});
