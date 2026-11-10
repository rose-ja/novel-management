import AxeBuilder from "@axe-core/playwright";
import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createSystemNode,
  createWork,
  listSystemCategories,
  uniqueName,
  updateCharacter,
} from "./helpers";

type SeededWork = {
  work: Awaited<ReturnType<typeof createWork>>;
  categoryID: string;
  realmID: string;
  realmName: string;
};

async function seedWork(request: APIRequestContext, name = uniqueName("人物")): Promise<SeededWork> {
  const work = await createWork(request, name);
  const categories = await listSystemCategories(request, work.id);
  const realm = categories.find((category) => category.is_builtin);
  if (!realm) throw new Error("内置境界类别缺失");
  const realmName = "金丹期";
  const node = await createSystemNode(request, realm.id, realmName);
  return { work, categoryID: realm.id, realmID: node.id, realmName };
}

async function openCharacters(page: Page): Promise<void> {
  await page.goto("/characters");
  await expect(page.getByRole("heading", { name: "人物", level: 1 })).toBeVisible();
}

function rowByName(page: Page, name: string) {
  return page.locator(".ui-table tbody tr").filter({ hasText: name });
}

async function createCharacterViaUI(
  page: Page,
  options: {
    name: string;
    alias?: string;
    title?: string;
    realm?: string;
    description?: string;
  },
): Promise<void> {
  await page.locator(".ui-page-header").getByRole("button", { name: "新建人物" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("heading", { name: "新建人物" })).toBeVisible();
  await dialog.getByLabel("姓名").fill(options.name);
  if (options.alias) {
    await dialog.getByRole("button", { name: "添加别名 / 称号" }).click();
    await dialog.getByLabel("第 1 条内容").fill(options.alias);
  }
  if (options.title) {
    await dialog.getByRole("button", { name: "添加别名 / 称号" }).click();
    await dialog.getByLabel("第 2 条类型").selectOption("title");
    await dialog.getByLabel("第 2 条内容").fill(options.title);
  }
  if (options.description) {
    await dialog.getByLabel("简介").fill(options.description);
  }
  if (options.realm) {
    await expect(dialog.getByLabel("境界")).toBeVisible();
    await dialog.getByLabel("境界").selectOption({ label: options.realm });
  }
  await dialog.getByRole("button", { name: "创建人物" }).click();
  await expect(page.locator(".ui-toast").filter({ hasText: `已创建人物「${options.name}」` })).toBeVisible();
  await expect(dialog).toBeHidden();
  await expect(page).toHaveURL(/\/characters\?id=/);
  await expect(page.getByRole("heading", { name: options.name, level: 1 })).toBeVisible();
  await page.getByRole("link", { name: "返回人物列表" }).first().click();
  await expect(page).toHaveURL(/\/characters$/);
  await expect(page.locator(".ui-table")).toBeVisible();
}

test.describe("人物列表", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("新建人物后可搜索别名、筛选境界与待校对", async ({ page, request }) => {
    const seeded = await seedWork(request);
    await openCharacters(page);

    await expect(page.locator(".ui-empty")).toBeVisible();
    await createCharacterViaUI(page, {
      name: "陆沉",
      alias: "沉舟",
      title: "剑主",
      realm: seeded.realmName,
      description: "北境剑修。",
    });
    await createCharacterViaUI(page, { name: "顾清寒" });

    const row = rowByName(page, "陆沉");
    await expect(row).toBeVisible();
    await expect(row).toContainText("沉舟");
    await expect(row).toContainText("剑主");
    await expect(row).toContainText(seeded.realmName);
    await expect(row).toContainText("已校对");
    await expect(rowByName(page, "顾清寒")).toContainText("—");

    await page.getByLabel("搜索").fill("沉舟");
    await expect(rowByName(page, "陆沉")).toBeVisible();
    await expect(rowByName(page, "顾清寒")).toHaveCount(0);

    await page.getByLabel("搜索").fill("");
    await page.getByLabel("境界").selectOption({ label: seeded.realmName });
    await expect(rowByName(page, "陆沉")).toBeVisible();
    await expect(rowByName(page, "顾清寒")).toHaveCount(0);

    await page.getByLabel("仅看待校对").check();
    await expect(page.getByText("没有符合筛选的人物")).toBeVisible();
    await expect(rowByName(page, "陆沉")).toHaveCount(0);

    await page.locator(".ui-empty").getByRole("button", { name: "清除筛选" }).click();
    await expect(rowByName(page, "顾清寒")).toBeVisible();
  });

  test("分页每页 20 条并可切换下一页", async ({ page, request }) => {
    const seeded = await seedWork(request);
    for (let index = 1; index <= 22; index += 1) {
      await createCharacter(request, seeded.work.id, `人物${String(index).padStart(2, "0")}`);
    }

    await openCharacters(page);
    const pagination = page.getByRole("navigation", { name: "分页" });
    await expect(pagination).toContainText("共 22 条 · 第 1 / 2 页");
    await expect(page.locator(".ui-table tbody tr")).toHaveCount(20);

    await page.getByRole("button", { name: "下一页" }).click();
    await expect(pagination).toContainText("第 2 / 2 页");
    await expect(page.locator(".ui-table tbody tr")).toHaveCount(2);
    await expect(rowByName(page, "人物01")).toBeVisible();
    await expect(rowByName(page, "人物22")).toHaveCount(0);
  });
});

test.describe("人物详情", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("内联编辑、未保存守卫与版本冲突", async ({ page, request }) => {
    const seeded = await seedWork(request);
    const character = await createCharacter(request, seeded.work.id, "陆沉", {
      names: [{ kind: "alias", value: "沉舟" }],
      description: "北境剑修。",
      systemRefs: [{ category_id: seeded.categoryID, node_id: seeded.realmID }],
    });

    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: "陆沉", level: 1 })).toBeVisible();
    await expect(page.getByText("北境剑修。")).toBeVisible();

    await page.getByRole("button", { name: "编辑" }).click();
    await page.getByLabel("姓名").fill("陆沉·改");
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已保存人物「陆沉·改」" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "陆沉·改", level: 1 })).toBeVisible();

    // 外部更新推高版本，页面仍持有旧版本 → 保存触发 409。
    await updateCharacter(
      request,
      { id: character.id, version: 2 },
      { name: "站外同步" },
    );
    await page.getByRole("button", { name: "编辑" }).click();
    await page.getByLabel("姓名").fill("陆沉·草稿");
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.getByText("数据已被其他页面修改，请刷新后重试")).toBeVisible();
    await page.getByRole("button", { name: "刷新最新内容" }).click();
    const discard = page.getByRole("dialog");
    await expect(discard.getByRole("heading", { name: "放弃未保存的修改？" })).toBeVisible();
    await discard.getByRole("button", { name: "放弃并离开" }).click();
    await expect(page.getByRole("heading", { name: "站外同步", level: 1 })).toBeVisible();
  });

  test("未保存守卫拦截站内链接", async ({ page, request }) => {
    const seeded = await seedWork(request);
    const character = await createCharacter(request, seeded.work.id, "陆沉");

    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: "陆沉", level: 1 })).toBeVisible();
    await page.getByRole("button", { name: "编辑" }).click();
    await page.getByLabel("姓名").fill("陆沉·未保存");

    await page.getByRole("link", { name: "体系" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("heading", { name: "放弃未保存的修改？" })).toBeVisible();
    await dialog.getByRole("button", { name: "继续编辑" }).click();
    await expect(page.getByRole("heading", { name: "编辑人物", level: 1 })).toBeVisible();

    await page.getByRole("button", { name: "取消" }).click();
    await expect(dialog.getByRole("heading", { name: "放弃未保存的修改？" })).toBeVisible();
    await dialog.getByRole("button", { name: "放弃并离开" }).click();
    await expect(page.getByRole("heading", { name: "陆沉", level: 1 })).toBeVisible();
    await page.getByRole("link", { name: "体系" }).click();
    await expect(page.getByRole("heading", { name: "体系", level: 1 })).toBeVisible();
  });

  test("删除后可撤销，回收站按人物类型恢复", async ({ page, request }) => {
    const seeded = await seedWork(request);
    const character = await createCharacter(request, seeded.work.id, "陆沉");

    await page.goto(`/characters?id=${character.id}`);
    await page.getByRole("button", { name: "删除人物" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("heading", { name: "删除人物「陆沉」？" })).toBeVisible();
    await dialog.getByRole("button", { name: "删除人物" }).click();

    const toast = page.locator(".ui-toast").filter({ hasText: "已移入回收站" });
    await expect(toast).toBeVisible();
    await expect(page.getByRole("heading", { name: "人物已移入回收站" })).toBeVisible();
    await toast.getByRole("button", { name: "撤销" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已撤销删除人物「陆沉」" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "陆沉", level: 1 })).toBeVisible();

    await page.getByRole("button", { name: "删除人物" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除人物" }).click();
    await expect(page.getByRole("heading", { name: "人物已移入回收站" })).toBeVisible();

    await page.goto("/trash");
    await expect(page.getByRole("heading", { name: "回收站", level: 1 })).toBeVisible();
    await page.getByLabel("类型", { exact: true }).selectOption("character");
    const trashRow = page.locator(".ui-table tbody tr").filter({ hasText: "陆沉" });
    await expect(trashRow).toBeVisible();
    await expect(trashRow).toContainText("人物");

    await page.getByRole("button", { name: "恢复人物「陆沉」" }).click();
    const restore = page.getByRole("dialog");
    await restore.getByRole("button", { name: "恢复" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已恢复人物「陆沉」" })).toBeVisible();

    await page.goto("/characters");
    await expect(rowByName(page, "陆沉")).toBeVisible();
  });

  test("列表与详情无 serious/critical 无障碍问题", async ({ page, request }) => {
    const seeded = await seedWork(request);
    const character = await createCharacter(request, seeded.work.id, "陆沉", {
      names: [{ kind: "title", value: "剑主" }],
      systemRefs: [{ category_id: seeded.categoryID, node_id: seeded.realmID }],
    });

    await page.goto("/characters");
    await expect(rowByName(page, "陆沉")).toBeVisible();
    const listResults = await new AxeBuilder({ page }).analyze();
    expect(
      listResults.violations
        .filter((violation) => violation.impact === "serious" || violation.impact === "critical")
        .map((violation) => `${violation.id} @ ${violation.nodes.flatMap((node) => node.target).join(" | ")}`),
    ).toEqual([]);

    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: "陆沉", level: 1 })).toBeVisible();
    const detailResults = await new AxeBuilder({ page }).analyze();
    expect(
      detailResults.violations
        .filter((violation) => violation.impact === "serious" || violation.impact === "critical")
        .map((violation) => `${violation.id} @ ${violation.nodes.flatMap((node) => node.target).join(" | ")}`),
    ).toEqual([]);
  });

  test("1024 宽度无横向溢出", async ({ page, request }) => {
    const seeded = await seedWork(request);
    await createCharacter(request, seeded.work.id, "陆沉", {
      names: [{ kind: "alias", value: "沉舟" }],
      description: "北境剑修。",
    });

    await page.setViewportSize({ width: 1024, height: 800 });
    await openCharacters(page);
    await expect(rowByName(page, "陆沉")).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth > window.innerWidth + 1,
    );
    expect(overflow).toBe(false);
  });
});
