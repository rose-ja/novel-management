import { expect, test } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createLocation,
  createSystemCategory,
  createSystemNode,
  createWork,
  uniqueName,
} from "./helpers";

/** 键盘可达性：跳过链接与侧栏导航可用 Tab 聚焦，焦点可见（SPEC-PLAT-004）。 */
test.describe("键盘可达", () => {
  test("首个 Tab 聚焦跳过链接并可跳到主要内容", async ({ page }) => {
    await page.goto("/");

    await page.keyboard.press("Tab");
    const skipLink = page.getByRole("link", { name: "跳到主要内容" });
    await expect(skipLink).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(page.locator("#main-content")).toBeFocused();
  });

  test("侧栏导航可用键盘遍历", async ({ page }) => {
    await page.goto("/");

    const nav = page.getByRole("navigation", { name: "主导航" });
    const overview = nav.getByRole("link", { name: "概览" });
    const search = nav.getByRole("link", { name: "检索" });
    const system = nav.getByRole("link", { name: "体系" });
    const characters = nav.getByRole("link", { name: "人物" });
    const factions = nav.getByRole("link", { name: "势力" });
    const locations = nav.getByRole("link", { name: "地点" });
    const rankings = nav.getByRole("link", { name: "榜单" });
    const trash = nav.getByRole("link", { name: "回收站" });
    const settings = nav.getByRole("link", { name: "设置" });

    await overview.focus();
    await expect(overview).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(search).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(system).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(characters).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(factions).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(locations).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(rankings).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(trash).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(settings).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/settings$/);
    await expect(page.getByRole("heading", { name: "设置 / 备份", level: 1 })).toBeVisible();
  });

  test("作品切换器可用键盘打开、方向键遍历与 Esc 关闭", async ({ page }) => {
    await page.goto("/");

    const trigger = page.locator(".shell__work--trigger");
    await trigger.focus();
    await page.keyboard.press("Enter");

    const menu = page.locator(".work-menu");
    await expect(menu).toBeVisible();
    await expect(menu.locator(":focus")).toHaveCount(1);

    await page.keyboard.press("ArrowDown");
    await expect(menu.locator(":focus")).toHaveCount(1);
    await page.keyboard.press("ArrowUp");
    await expect(menu.locator(":focus")).toHaveCount(1);

    await page.keyboard.press("Escape");
    await expect(menu).toHaveCount(0);
    await expect(trigger).toBeFocused();
  });
});

/** 地点树键盘：展开/收起按钮可用 Enter/空格切换，层级不丢失。 */
test.describe("地点树键盘", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("展开按钮可键盘切换且下级随之显隐", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("键盘地点"));
    const root = await createLocation(request, work.id, uniqueName("键盘东荒"));
    const child = await createLocation(request, work.id, uniqueName("键盘青丘"), {
      parentID: root.id,
    });

    await page.goto("/locations");
    await expect(page.getByRole("heading", { name: "地点", level: 1 })).toBeVisible();
    const toggle = page.getByRole("button", { name: `收起地点：${root.name}` });
    await toggle.focus();
    await expect(toggle).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("button", { name: `展开地点：${root.name}` })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    await expect(page.getByRole("link", { name: child.name })).toHaveCount(0);
    await page.keyboard.press("Enter");
    await expect(page.getByRole("link", { name: child.name })).toBeVisible();
  });
});

test.describe("体系页键盘", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("类别列表可用 ↑/↓/Home/End 切换选中", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("键盘类别"));
    await createSystemCategory(request, work.id, uniqueName("键盘自定义"));

    await page.goto("/system");
    await expect(page.getByRole("heading", { name: "体系", level: 1 })).toBeVisible();

    const options = page.getByRole("listbox", { name: "类别列表" }).getByRole("option");
    await expect(options).toHaveCount(2);

    await options.first().focus();
    await page.keyboard.press("ArrowDown");
    await expect(options.nth(1)).toBeFocused();
    await expect(options.nth(1)).toHaveAttribute("aria-selected", "true");

    await page.keyboard.press("ArrowUp");
    await expect(options.first()).toBeFocused();
    await expect(options.first()).toHaveAttribute("aria-selected", "true");

    await page.keyboard.press("End");
    await expect(options.last()).toBeFocused();
    await expect(options.last()).toHaveAttribute("aria-selected", "true");

    await page.keyboard.press("Home");
    await expect(options.first()).toBeFocused();
    await expect(options.first()).toHaveAttribute("aria-selected", "true");
  });

  test("节点树可用 ↑/↓ 移动焦点、←/→ 展开收起", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("键盘节点"));
    const category = await createSystemCategory(request, work.id, uniqueName("键盘树"));
    const parent = await createSystemNode(request, category.id, uniqueName("键盘父"));
    await createSystemNode(request, category.id, uniqueName("键盘子"), parent.id);
    await createSystemNode(request, category.id, uniqueName("键盘乙"));

    await page.goto("/system");
    await expect(page.getByRole("heading", { name: "体系", level: 1 })).toBeVisible();
    await page.getByRole("option", { name: new RegExp(category.name) }).click();
    await expect(page.locator(".system-detail__title")).toHaveText(category.name);

    const tree = page.getByRole("tree", { name: "节点树" });
    const items = tree.getByRole("treeitem");
    await expect(items).toHaveCount(3);

    const parentRow = items.filter({ hasText: parent.name });
    await parentRow.focus();
    await expect(parentRow).toHaveAttribute("aria-expanded", "true");

    await page.keyboard.press("ArrowDown");
    await expect(items.nth(1)).toBeFocused();

    await page.keyboard.press("ArrowLeft");
    await expect(parentRow).toBeFocused();

    await page.keyboard.press("ArrowRight");
    await expect(items.nth(1)).toBeFocused();

    await page.keyboard.press("ArrowLeft");
    await expect(parentRow).toBeFocused();

    await page.keyboard.press("ArrowLeft");
    await expect(parentRow).toHaveAttribute("aria-expanded", "false");
    await expect(items).toHaveCount(2);
    await expect(parentRow).toBeFocused();

    await page.keyboard.press("ArrowRight");
    await expect(parentRow).toHaveAttribute("aria-expanded", "true");
    await expect(items).toHaveCount(3);
  });
});

/** 设置页键盘：导出对话框焦点锁定、默认勾选与 Esc 关闭。 */
test.describe("设置页键盘", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("导出对话框可用键盘打开、默认勾选并以 Esc 关闭", async ({ page, request }) => {
    await createWork(request, uniqueName("键盘导出"));
    await page.goto("/settings");
    await expect(page.getByRole("heading", { name: "设置 / 备份", level: 1 })).toBeVisible();

    const trigger = page.getByRole("button", { name: "导出全量数据" });
    await trigger.focus();
    await expect(trigger).toBeFocused();
    await page.keyboard.press("Enter");

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("checkbox", { name: /包含回收站数据/ })).toBeChecked();
    await expect(dialog.locator(":focus")).toHaveCount(1);

    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
  });
});

/** 导入页键盘：步骤条语义与拖拽区可用 Enter 触发文件选择（PG-11 §7）。 */
test.describe("导入页键盘", () => {
  test("步骤条标记当前步骤，拖拽区可键盘触发文件选择", async ({ page }) => {
    await page.goto("/import");
    await expect(page.getByRole("heading", { name: "导入设定", level: 1 })).toBeVisible();
    await expect(page.locator('[aria-current="step"]')).toHaveText(/上传/);

    const dropzone = page.getByRole("button", { name: /可拖拽文件到此处，也可使用键盘选择/ });
    await dropzone.focus();
    await expect(dropzone).toBeFocused();

    const chooser = page.waitForEvent("filechooser");
    await page.keyboard.press("Enter");
    expect((await chooser).isMultiple()).toBe(false);
  });
});

/** 检索页键盘：自动聚焦、Esc 清空、Tab 顺序与模式切换（PG-01 §12）。 */
test.describe("检索页键盘", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("输入框自动聚焦，Esc 清空，Tab 顺序可达模式与筛选", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("键盘检索"));
    await createCharacter(request, work.id, uniqueName("陆沉"));

    await page.goto("/search");
    const input = page.getByLabel("检索关键词");
    await expect(input).toBeFocused();

    await input.fill("E2E-键盘检索");
    const clearButton = page.getByRole("button", { name: "清除关键词" });
    await expect(clearButton).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(input).toHaveValue("");
    await expect(input).toBeFocused();

    await input.fill("陆沉");
    await expect(page.locator(".search-result").first()).toBeVisible();

    await page.keyboard.press("Tab");
    await expect(clearButton).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(page.getByRole("button", { name: "快速搜索" })).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(page.getByRole("button", { name: "条件检索" })).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(page.getByRole("button", { name: "条件检索" })).toHaveAttribute("aria-pressed", "true");
    await expect(page.getByLabel("实体类型")).toBeVisible();
  });
});

/** 人物键盘主流程：新建、编辑保存、删除确认与回收站恢复，Esc 关闭后回焦。 */
test.describe("人物键盘主流程", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("键盘完成新建、编辑、删除与恢复，Esc 关闭回焦", async ({ page, request }) => {
    await createWork(request, uniqueName("键盘人物"));
    const name = uniqueName("陆沉");
    const renamed = uniqueName("陆沉改");

    await page.goto("/characters");
    await expect(page.getByRole("heading", { name: "人物", level: 1 })).toBeVisible();

    const createTrigger = page.locator(".ui-page-header").getByRole("button", { name: "新建人物" });
    await createTrigger.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("button", { name: "关闭" })).toBeFocused();

    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
    await expect(createTrigger).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(dialog).toBeVisible();
    await page.keyboard.press("Tab");
    const nameInput = dialog.getByLabel("姓名");
    await expect(nameInput).toBeFocused();
    await page.keyboard.type(name);
    const createSubmit = dialog.getByRole("button", { name: "创建人物" });
    // 从姓名输入框反向 Tab：先到关闭按钮，再环绕到对话框末尾的提交按钮。
    for (let attempt = 0; attempt < 3; attempt += 1) {
      await page.keyboard.press("Shift+Tab");
      if (await createSubmit.evaluate((element) => element === document.activeElement)) break;
    }
    await expect(createSubmit).toBeFocused();
    await page.keyboard.press("Enter");

    await expect(page).toHaveURL(/\/characters\?id=/);
    await expect(page.getByRole("heading", { name, level: 1 })).toBeVisible();

    const editTrigger = page.locator(".ui-page-header").getByRole("button", { name: "编辑" });
    await editTrigger.focus();
    await page.keyboard.press("Enter");
    const editName = page.getByLabel("姓名");
    await editName.focus();
    await page.keyboard.press("Control+A");
    await page.keyboard.type(renamed);
    const save = page.getByRole("button", { name: "保存修改" });
    await save.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("heading", { name: renamed, level: 1 })).toBeVisible();

    const deleteTrigger = page.locator(".ui-page-header").getByRole("button", { name: "删除人物" });
    await deleteTrigger.focus();
    await page.keyboard.press("Enter");
    const deleteDialog = page.getByRole("dialog");
    await expect(deleteDialog).toBeVisible();
    await expect(deleteDialog.getByRole("button", { name: "关闭" })).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(deleteDialog).toHaveCount(0);
    await expect(deleteTrigger).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(deleteDialog).toBeVisible();
    const confirmDelete = deleteDialog.getByRole("button", { name: "删除人物" });
    await confirmDelete.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("heading", { name: "人物已移入回收站" })).toBeVisible();

    await page.goto("/trash");
    const restoreTrigger = page.getByRole("button", { name: `恢复人物「${renamed}」` });
    await restoreTrigger.focus();
    await page.keyboard.press("Enter");
    const restoreDialog = page.getByRole("dialog");
    await expect(restoreDialog).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(restoreDialog).toHaveCount(0);
    await expect(restoreTrigger).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(restoreDialog).toBeVisible();
    const restoreSubmit = restoreDialog.getByRole("button", { name: "恢复" });
    await restoreSubmit.focus();
    await page.keyboard.press("Enter");
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: renamed })).toHaveCount(0);
  });
});
