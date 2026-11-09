import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createSystemCategory,
  createSystemNode,
  createWork,
  deleteSystemNode,
  JSON_HEADERS,
  listSystemCategories,
  listSystemNodes,
  listTrash,
  restoreTrashItem,
  undoTrash,
  uniqueName,
  type SystemCategoryJson,
  type SystemNodeJson,
} from "./helpers";

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

async function openSystem(page: Page): Promise<void> {
  await page.goto("/system");
  await expect(page.getByRole("heading", { name: "体系", level: 1 })).toBeVisible();
  await expect(page.locator(".system-category-item").first()).toBeVisible();
}

async function selectCategory(page: Page, name: string): Promise<void> {
  await page.getByRole("option", { name: new RegExp(escapeRegExp(name)) }).click();
  await expect(page.locator(".system-detail__title")).toHaveText(name);
}

async function renameCategoryViaUI(page: Page, from: string, to: string): Promise<void> {
  await page.locator(".system-detail").getByRole("button", { name: "改名", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("类别名称").fill(to);
  await dialog.getByRole("button", { name: "保存" }).click();
  await expect(page.locator(".ui-toast").filter({ hasText: `已保存类别「${to}」` })).toBeVisible();
  await expect(page.locator(".system-detail__title")).toHaveText(to);
  await expect(
    page.getByRole("option", { name: new RegExp(`^${escapeRegExp(from)}(?:\\s|$)`) }),
  ).toHaveCount(0);
}

async function createCategoryViaUI(page: Page, name: string): Promise<void> {
  await page.getByRole("button", { name: "新增类别" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("类别名称").fill(name);
  await dialog.getByRole("button", { name: "创建类别" }).click();
  await expect(page.locator(".ui-toast").filter({ hasText: `已创建类别「${name}」` })).toBeVisible();
  await expect(page.getByRole("option", { name: new RegExp(escapeRegExp(name)) })).toBeVisible();
}

async function createRootNodeViaUI(page: Page, name: string): Promise<void> {
  await page.getByRole("button", { name: "新增主层级" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("节点名称").fill(name);
  await dialog.getByRole("button", { name: "创建节点" }).click();
  await expect(page.locator(".ui-toast").filter({ hasText: `已新增节点「${name}」` })).toBeVisible();
  await expect(page.getByRole("treeitem").filter({ hasText: name })).toBeVisible();
}

async function deleteCategoryViaUI(page: Page, name: string): Promise<void> {
  await page.locator(".system-detail").getByRole("button", { name: "删除", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText(`删除类别「${name}」？`)).toBeVisible();
  await dialog.getByRole("button", { name: "删除" }).click();
  await expect(page.locator(".ui-toast").filter({ hasText: `类别「${name}」已移入回收站` })).toBeVisible();
}

async function bootWork(
  request: APIRequestContext,
  label: string,
): Promise<{ workID: string; builtin: SystemCategoryJson }> {
  const work = await createWork(request, uniqueName(label));
  const categories = await listSystemCategories(request, work.id);
  const builtin = categories.find((item) => item.is_builtin);
  expect(builtin).toBeTruthy();
  return { workID: work.id, builtin: builtin as SystemCategoryJson };
}

function nodeNames(nodes: SystemNodeJson[]): string[] {
  return nodes.map((node) => node.name);
}

test.describe("体系配置", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("内置「境界」无改名/停用/删除入口，API 同样拒绝", async ({ page, request }) => {
    const { builtin } = await bootWork(request, "内置");
    expect(builtin.name).toBe("境界");

    await openSystem(page);
    await selectCategory(page, builtin.name);
    await expect(page.getByRole("option", { name: /境界/ })).toContainText("内置");
    await expect(page.getByRole("option", { name: /境界/ })).toContainText("必开");

    const detail = page.locator(".system-detail");
    await expect(detail.getByRole("button", { name: "改名", exact: true })).toHaveCount(0);
    await expect(detail.getByRole("button", { name: "停用", exact: true })).toHaveCount(0);
    await expect(detail.getByRole("button", { name: "删除", exact: true })).toHaveCount(0);

    const ifMatch = { ...JSON_HEADERS, "If-Match": `"${builtin.version}"` };
    const rename = await request.patch(`/api/v1/system-categories/${builtin.id}`, {
      data: { name: "E2E-改名" },
      headers: ifMatch,
    });
    expect(rename.status()).toBe(409);
    expect((await rename.json()).error.code).toBe("CATEGORY_BUILTIN_IMMUTABLE");

    const disable = await request.patch(`/api/v1/system-categories/${builtin.id}`, {
      data: { is_enabled: false },
      headers: ifMatch,
    });
    expect(disable.status()).toBe(409);
    expect((await disable.json()).error.code).toBe("CATEGORY_BUILTIN_IMMUTABLE");

    const remove = await request.delete(`/api/v1/system-categories/${builtin.id}`, { headers: ifMatch });
    expect(remove.status()).toBe(409);
    expect((await remove.json()).error.code).toBe("CATEGORY_BUILTIN_IMMUTABLE");
  });

  test("类别创建、改名、停用（节点只读）与启用", async ({ page, request }) => {
    await bootWork(request, "类别");
    const original = uniqueName("类别");
    const renamed = `${original}-改`;

    await openSystem(page);
    await createCategoryViaUI(page, original);
    await selectCategory(page, original);

    await renameCategoryViaUI(page, original, renamed);

    await page.locator(".system-detail").getByRole("button", { name: "停用", exact: true }).click();
    let dialog = page.getByRole("dialog");
    await expect(dialog.getByText(`停用类别「${renamed}」？`)).toBeVisible();
    await dialog.getByRole("button", { name: "停用" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已停用类别「${renamed}」` })).toBeVisible();

    await expect(page.getByRole("option", { name: new RegExp(escapeRegExp(renamed)) })).toContainText("已停用");
    const detail = page.locator(".system-detail");
    await expect(detail.getByText("该类别已停用，人物表单不再显示该字段；已有数据保留。")).toBeVisible();
    await expect(detail.getByRole("button", { name: "新增主层级" })).toHaveCount(0);
    await expect(detail.getByText("该类别已停用，节点仅供查看；启用后可新增、改名、删除或调整顺序。")).toBeVisible();

    await detail.getByRole("button", { name: "启用", exact: true }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已启用类别「${renamed}」` })).toBeVisible();
    await expect(detail.getByRole("button", { name: "新增主层级" })).toBeVisible();
  });

  test("类别上限 10：第 10 个成功，第 11 个被拒 CATEGORY_LIMIT", async ({ page, request }) => {
    const { workID } = await bootWork(request, "上限");
    for (let index = 0; index < 8; index += 1) {
      await createSystemCategory(request, workID, uniqueName(`限${index}`));
    }

    await openSystem(page);
    await expect(page.locator(".system-category-list__count")).toHaveText("9/10");

    const tenth = uniqueName("第十");
    await createCategoryViaUI(page, tenth);
    await expect(page.locator(".system-category-list__count")).toHaveText("10/10");

    const addButton = page.locator(".system-category-list__add");
    await expect(addButton).toBeDisabled();
    await expect(page.getByText("最多 10 类（含内置境界）")).toBeVisible();

    const eleventh = await request.post("/api/v1/system-categories", {
      data: { work_id: workID, name: uniqueName("第十一"), category_type: "normal" },
      headers: JSON_HEADERS,
    });
    expect(eleventh.status()).toBe(409);
    expect((await eleventh.json()).error.code).toBe("CATEGORY_LIMIT");
  });

  test("两级节点：新增主/子层级，子节点无新增子层级入口", async ({ page, request }) => {
    const { workID } = await bootWork(request, "两级");
    const category = await createSystemCategory(request, workID, uniqueName("层级"));

    await openSystem(page);
    await selectCategory(page, category.name);

    const parent = uniqueName("父");
    await createRootNodeViaUI(page, parent);
    await expect(page.getByRole("treeitem").filter({ hasText: parent })).toHaveAttribute("aria-level", "1");

    await page.getByRole("button", { name: `新增子层级：${parent}` }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText(`新增子层级 · ${parent}`)).toBeVisible();
    const child = uniqueName("子");
    await dialog.getByLabel("节点名称").fill(child);
    await dialog.getByRole("button", { name: "创建节点" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已新增节点「${child}」` })).toBeVisible();

    const childRow = page.getByRole("treeitem").filter({ hasText: child });
    await expect(childRow).toHaveAttribute("aria-level", "2");
    await expect(childRow.getByRole("button", { name: `新增子层级：${child}` })).toHaveCount(0);

    const nodes = await listSystemNodes(request, category.id);
    const childNode = nodes.flatMap((node) => node.children ?? []).find((node) => node.name === child);
    expect(childNode).toBeTruthy();
    const grandchild = await request.post("/api/v1/system-nodes", {
      data: { category_id: category.id, parent_id: childNode?.id, name: uniqueName("孙") },
      headers: JSON_HEADERS,
    });
    expect(grandchild.status()).toBe(422);
    expect((await grandchild.json()).error.code).toBe("NODE_LEVEL_LIMIT");
  });

  test("同级重名返回 NAME_CONFLICT 并在弹窗内提示", async ({ page, request }) => {
    const { workID } = await bootWork(request, "重名");
    const category = await createSystemCategory(request, workID, uniqueName("重名类"));
    const name = uniqueName("节点");
    await createSystemNode(request, category.id, name);

    const duplicate = await request.post("/api/v1/system-nodes", {
      data: { category_id: category.id, name },
      headers: JSON_HEADERS,
    });
    expect(duplicate.status()).toBe(409);
    expect((await duplicate.json()).error.code).toBe("NAME_CONFLICT");

    await openSystem(page);
    await selectCategory(page, category.name);
    await page.getByRole("button", { name: "新增主层级" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("节点名称").fill(name);
    await dialog.getByRole("button", { name: "创建节点" }).click();
    await expect(dialog.getByText("名称已存在，请换一个名称")).toBeVisible();
    await dialog.getByRole("button", { name: "取消" }).click();
  });

  test("上移排序后刷新页面顺序保持不变", async ({ page, request }) => {
    const { workID } = await bootWork(request, "排序");
    const category = await createSystemCategory(request, workID, uniqueName("排序类"));
    const first = uniqueName("甲");
    const second = uniqueName("乙");
    const third = uniqueName("丙");
    await createSystemNode(request, category.id, first);
    await createSystemNode(request, category.id, second);
    await createSystemNode(request, category.id, third);

    await openSystem(page);
    await selectCategory(page, category.name);
    await expect(page.locator(".system-node-row__name")).toHaveText([first, second, third]);

    await page.getByRole("button", { name: `上移节点：${second}` }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已调整顺序" })).toBeVisible();
    await expect(page.locator(".system-node-row__name")).toHaveText([second, first, third]);

    await page.reload();
    await expect(page.getByRole("heading", { name: "体系", level: 1 })).toBeVisible();
    await selectCategory(page, category.name);
    await expect(page.locator(".system-node-row__name")).toHaveText([second, first, third]);
  });

  test("引用阻断：停用与删除均弹出引用清单", async ({ page, request }) => {
    const { workID } = await bootWork(request, "引用");
    const category = await createSystemCategory(request, workID, uniqueName("引用类"));
    const node = await createSystemNode(request, category.id, uniqueName("引用节点"));
    const character = await createCharacter(request, workID, uniqueName("引用人物"), {
      systemRefs: [{ category_id: category.id, node_id: node.id }],
    });

    await openSystem(page);
    await selectCategory(page, category.name);
    await expect(page.getByRole("option", { name: new RegExp(escapeRegExp(category.name)) })).toContainText(
      "被引用 1 人",
    );

    await page.locator(".system-detail").getByRole("button", { name: "停用", exact: true }).click();
    let dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "停用" }).click();
    await expect(page.getByRole("dialog")).toContainText(
      `无法停用「${category.name}」：仍有 1 人引用该类别`,
    );
    await expect(page.getByRole("dialog")).toContainText(node.name);
    await expect(page.getByRole("dialog")).toContainText(character.name);
    await expect(page.getByRole("dialog")).toContainText("1 人");
    await page.getByRole("dialog").getByRole("button", { name: "知道了" }).click();

    await page.locator(".system-detail").getByRole("button", { name: "删除", exact: true }).click();
    dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "删除" }).click();
    await expect(page.getByRole("dialog")).toContainText(
      `无法删除「${category.name}」：仍有 1 人引用该类别`,
    );
    await expect(page.getByRole("dialog")).toContainText(node.name);
    await page.getByRole("dialog").getByRole("button", { name: "知道了" }).click();

    const remove = await request.delete(`/api/v1/system-categories/${category.id}`, {
      headers: { ...JSON_HEADERS, "If-Match": `"${category.version}"` },
    });
    expect(remove.status()).toBe(409);
    const body = await remove.json();
    expect(body.error.code).toBe("REFERENCE_CONFLICT");
    expect(body.error.details).toHaveLength(1);
    expect(body.error.details[0]).toMatchObject({
      type: "character_system_ref",
      entity_id: character.id,
      name: character.name,
      field: "node",
      value: node.name,
      in_trash: false,
    });
  });

  test("整组删除进入回收站，并用 new_name 整组恢复", async ({ page, request }) => {
    const { workID } = await bootWork(request, "整组");
    const category = await createSystemCategory(request, workID, uniqueName("整组类"));
    const nodeA = await createSystemNode(request, category.id, uniqueName("整组甲"));
    const nodeB = await createSystemNode(request, category.id, uniqueName("整组乙"));

    await openSystem(page);
    await selectCategory(page, category.name);
    await deleteCategoryViaUI(page, category.name);

    const deleted = (await listTrash(request, "system_category")).find((item) => item.root_id === category.id);
    expect(deleted).toBeTruthy();
    expect(deleted?.name).toBe(category.name);
    expect(deleted?.affected_count).toBe(2);

    const renamed = `${category.name}-恢复`;
    const restored = await restoreTrashItem(request, "system_category", category.id, { new_name: renamed });
    expect(restored.name).toBe(renamed);
    expect(restored.restored_count).toBe(3);

    const categories = await listSystemCategories(request, workID);
    expect(categories.map((item) => item.name)).toContain(renamed);

    const nodes = await listSystemNodes(request, category.id);
    expect(nodeNames(nodes)).toEqual([nodeA.name, nodeB.name]);

    const trash = await listTrash(request, "system_category");
    expect(trash.some((item) => item.root_id === category.id)).toBe(false);
  });

  test("节点删除进入回收站，可通过撤销恢复", async ({ page, request }) => {
    const { workID } = await bootWork(request, "节点撤销");
    const category = await createSystemCategory(request, workID, uniqueName("撤销类"));
    const node = await createSystemNode(request, category.id, uniqueName("撤销节点"));

    const { undo_token: undoToken } = await deleteSystemNode(request, node.id, node.version);

    const root = (await listTrash(request, "system_node")).find((item) => item.root_id === node.id);
    expect(root).toBeTruthy();
    expect(root?.name).toBe(node.name);

    await undoTrash(request, undoToken);
    const nodes = await listSystemNodes(request, category.id);
    expect(nodeNames(nodes)).toContain(node.name);

    const trash = await listTrash(request, "system_node");
    expect(trash.some((item) => item.root_id === node.id)).toBe(false);
  });

  test("父级在回收站时恢复子节点返回 PARENT_DELETED", async ({ request }) => {
    const { workID } = await bootWork(request, "父级删除");
    const category = await createSystemCategory(request, workID, uniqueName("父级类"));
    const parent = await createSystemNode(request, category.id, uniqueName("父节点"));
    const child = await createSystemNode(request, category.id, uniqueName("子节点"), parent.id);

    await deleteSystemNode(request, child.id, child.version);
    await deleteSystemNode(request, parent.id, parent.version);

    const blocked = await request.post(`/api/v1/trash/system_node/${child.id}/restore`, {
      data: {},
      headers: JSON_HEADERS,
    });
    expect(blocked.status()).toBe(409);
    expect((await blocked.json()).error.code).toBe("PARENT_DELETED");

    await restoreTrashItem(request, "system_node", parent.id);
    await restoreTrashItem(request, "system_node", child.id);

    const nodes = await listSystemNodes(request, category.id);
    const restoredParent = nodes.find((item) => item.id === parent.id);
    expect(restoredParent).toBeTruthy();
    expect(restoredParent?.children?.map((item) => item.id)).toContain(child.id);
  });

  test("1024 视口无横向溢出", async ({ page, request }) => {
    const { workID } = await bootWork(request, "窄屏");
    const category = await createSystemCategory(request, workID, uniqueName("窄屏类"));
    const longName = `${uniqueName("很长的节点名称")}-用于验证窄屏布局不会横向溢出`;
    await createSystemNode(request, category.id, longName);
    await createSystemNode(request, category.id, uniqueName("窄屏乙"));

    await page.setViewportSize({ width: 1024, height: 800 });
    await openSystem(page);
    await selectCategory(page, category.name);
    await expect(page.getByRole("treeitem").filter({ hasText: longName })).toBeVisible();

    const overflow = await page.evaluate(() => {
      const element = document.scrollingElement ?? document.documentElement;
      return element.scrollWidth - element.clientWidth;
    });
    expect(overflow).toBeLessThanOrEqual(1);
  });
});
