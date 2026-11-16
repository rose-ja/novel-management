import { expect, test } from "@playwright/test";

import {
  assignRankingSeat,
  cleanupE2E,
  createCharacter,
  createFaction,
  createLocation,
  createRanking,
  createSystemCategory,
  createSystemNode,
  createWork,
  listSystemCategories,
} from "./helpers";

/**
 * 视觉基线：基线图由本机 Windows 环境生成，跨机器对比仅作参考。
 * 数据相关内容（系统信息、空状态、快照卡片）随本机数据变化，统一用遮罩排除。
 */
function dynamicRegions(page: import("@playwright/test").Page) {
  return [page.locator(".info-list"), page.locator(".ui-empty"), page.locator(".ui-card")];
}

test.describe("视觉基线", () => {
  test("概览页 1280", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "概览", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("overview-1280.png", {
      mask: dynamicRegions(page),
      fullPage: true,
    });
  });

  test("概览页 1024", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 800 });
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "概览", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("overview-1024.png", {
      mask: dynamicRegions(page),
      fullPage: true,
    });
  });

  test("设置页 1280", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto("/settings");
    await expect(page.getByRole("heading", { name: "设置 / 备份", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("settings-1280.png", {
      mask: dynamicRegions(page),
      fullPage: true,
    });
  });

  test("回收站页 1280", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto("/trash");
    await expect(page.getByRole("heading", { name: "回收站", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("trash-1280.png", {
      clip: { x: 0, y: 0, width: 1280, height: 320 },
      mask: [page.locator(".ui-page-header__description"), page.locator(".ui-page-header__actions")],
    });
  });

  test("回收站页 1024", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 800 });
    await page.goto("/trash");
    await expect(page.getByRole("heading", { name: "回收站", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("trash-1024.png", {
      clip: { x: 0, y: 0, width: 1024, height: 320 },
      mask: [page.locator(".ui-page-header__description"), page.locator(".ui-page-header__actions")],
    });
  });
});

/**
 * 体系页基线使用固定的 E2E 作品与类别名，避免本机数据影响截图；
 * 每次运行前后只清理 E2E- 前缀数据。
 */
test.describe("视觉基线 · 人物", () => {
  const WORK_NAME = "E2E-视觉人物";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`人物列表 ${width}`, async ({ page, request }) => {
      const work = await createWork(request, WORK_NAME);
      const categories = await listSystemCategories(request, work.id);
      const realm = categories.find((category) => category.is_builtin);
      if (!realm) throw new Error("内置境界类别缺失");
      const node = await createSystemNode(request, realm.id, "E2E-视觉金丹");
      await createCharacter(request, work.id, "E2E-视觉陆沉", {
        names: [
          { kind: "alias", value: "沉舟" },
          { kind: "title", value: "剑主" },
        ],
        systemRefs: [{ category_id: realm.id, node_id: node.id }],
      });
      await createCharacter(request, work.id, "E2E-视觉顾清寒");

      await page.setViewportSize({ width, height });
      await page.goto("/characters");
      await expect(page.getByRole("heading", { name: "人物", level: 1 })).toBeVisible();
      await expect(page.locator(".ui-table tbody tr")).toHaveCount(2);

      await expect(page).toHaveScreenshot(`characters-${width}.png`, {
        fullPage: true,
        mask: [page.locator(".character-time")],
      });
    });
  }

  test("人物详情 1280", async ({ page, request }) => {
    const work = await createWork(request, WORK_NAME);
    const categories = await listSystemCategories(request, work.id);
    const realm = categories.find((category) => category.is_builtin);
    if (!realm) throw new Error("内置境界类别缺失");
    const node = await createSystemNode(request, realm.id, "E2E-视觉金丹");
    const character = await createCharacter(request, work.id, "E2E-视觉陆沉", {
      names: [{ kind: "title", value: "剑主" }],
      description: "E2E 固定简介，用于视觉基线。",
      systemRefs: [{ category_id: realm.id, node_id: node.id }],
    });

    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: "E2E-视觉陆沉", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("character-detail-1280.png", {
      fullPage: true,
      mask: [page.locator(".character-archive-meta")],
    });
  });
});

test.describe("视觉基线 · 势力", () => {
  const WORK_NAME = "E2E-视觉势力";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`势力树 ${width}`, async ({ page, request }) => {
      const work = await createWork(request, WORK_NAME);
      const wan = await createFaction(request, work.id, "E2E-视觉万族");
      await createFaction(request, work.id, "E2E-视觉妖族", { parentID: wan.id });
      await createFaction(request, work.id, "E2E-视觉人族");

      await page.setViewportSize({ width, height });
      await page.goto("/factions");
      await expect(page.getByRole("heading", { name: "势力", level: 1 })).toBeVisible();
      await expect(page.locator(".faction-tree__row")).toHaveCount(3);

      await expect(page).toHaveScreenshot(`factions-${width}.png`, { fullPage: true });
    });
  }

  test("势力详情 1280", async ({ page, request }) => {
    const work = await createWork(request, WORK_NAME);
    const wan = await createFaction(request, work.id, "E2E-视觉万族");
    const yao = await createFaction(request, work.id, "E2E-视觉妖族", {
      parentID: wan.id,
      description: "E2E 固定势力简介，用于视觉基线。",
    });

    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(`/factions?id=${yao.id}`);
    await expect(page.getByRole("heading", { name: "E2E-视觉妖族", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("faction-detail-1280.png", {
      fullPage: true,
      mask: [page.locator(".character-archive-meta")],
    });
  });
});

test.describe("视觉基线 · 地点", () => {
  const WORK_NAME = "E2E-视觉地点";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`地点树 ${width}`, async ({ page, request }) => {
      const work = await createWork(request, WORK_NAME);
      const root = await createLocation(request, work.id, "E2E-视觉东荒", { typeText: "星域" });
      await createLocation(request, work.id, "E2E-视觉青丘山", {
        parentID: root.id,
        typeText: "星城",
      });
      await createLocation(request, work.id, "E2E-视觉西洲", { typeText: "星域" });

      await page.setViewportSize({ width, height });
      await page.goto("/locations");
      await expect(page.getByRole("heading", { name: "地点", level: 1 })).toBeVisible();
      await expect(page.locator(".location-tree__row")).toHaveCount(3);

      await expect(page).toHaveScreenshot(`locations-${width}.png`, { fullPage: true });
    });
  }

  test("地点详情 1280", async ({ page, request }) => {
    const work = await createWork(request, WORK_NAME);
    const root = await createLocation(request, work.id, "E2E-视觉东荒", {
      typeText: "星域",
      description: "E2E 固定地点简介，用于视觉基线。",
    });
    const child = await createLocation(request, work.id, "E2E-视觉青丘山", {
      parentID: root.id,
      typeText: "星城",
    });
    const faction = await createFaction(request, work.id, "E2E-视觉狐族");
    await createCharacter(request, work.id, "E2E-视觉涂山氏", {
      locationLinks: [{ locationID: child.id, note: "出生地" }],
    });
    await request.post(`/api/v1/locations/${child.id}/factions`, {
      data: { faction_id: faction.id, note: "狐族驻地" },
    });

    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(`/locations?id=${child.id}`);
    await expect(page.getByRole("heading", { name: "E2E-视觉青丘山", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("location-detail-1280.png", {
      fullPage: true,
      mask: [page.locator(".character-archive-meta")],
    });
  });
});

test.describe("视觉基线 · 榜单", () => {
  const WORK_NAME = "E2E-视觉榜单";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`榜单列表 ${width}`, async ({ page, request }) => {
      const work = await createWork(request, WORK_NAME);
      const alpha = await createCharacter(request, work.id, "E2E-视觉甲");
      const ranking = await createRanking(request, work.id, "E2E-视觉封神榜", 10);
      const seat1 = ranking.seats.items.find((seat) => seat.seat_no === 1);
      if (!seat1) throw new Error("榜单缺少第 1 席");
      await assignRankingSeat(request, ranking.id, 1, alpha.id, seat1.version);

      await page.setViewportSize({ width, height });
      await page.goto("/rankings");
      await expect(page.getByRole("heading", { name: "榜单", level: 1 })).toBeVisible();
      await expect(page.locator(".ui-table tbody tr")).toHaveCount(1);

      await expect(page).toHaveScreenshot(`rankings-${width}.png`, {
        fullPage: true,
        mask: [page.locator(".character-time")],
      });
    });
  }

  test("榜单详情 1280", async ({ page, request }) => {
    const work = await createWork(request, WORK_NAME);
    const alpha = await createCharacter(request, work.id, "E2E-视觉甲");
    const ranking = await createRanking(request, work.id, "E2E-视觉封神榜", 10);
    const seat1 = ranking.seats.items.find((seat) => seat.seat_no === 1);
    if (!seat1) throw new Error("榜单缺少第 1 席");
    await assignRankingSeat(request, ranking.id, 1, alpha.id, seat1.version);

    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(`/rankings?id=${ranking.id}`);
    await expect(page.getByRole("heading", { name: "E2E-视觉封神榜", level: 1 })).toBeVisible();

    await expect(page).toHaveScreenshot("ranking-detail-1280.png", {
      fullPage: true,
      mask: [page.locator(".ranking-rename")],
    });
  });
});

test.describe("视觉基线 · 体系", () => {
  const WORK_NAME = "E2E-视觉体系";
  const CATEGORY_NAME = "E2E-视觉血脉";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`体系页 ${width}`, async ({ page, request }) => {
      const work = await createWork(request, WORK_NAME);
      const category = await createSystemCategory(request, work.id, CATEGORY_NAME);
      const root = await createSystemNode(request, category.id, "E2E-视觉-人阶");
      await createSystemNode(request, category.id, "E2E-视觉-初期", root.id);
      await createSystemNode(request, category.id, "E2E-视觉-地阶");

      await page.setViewportSize({ width, height });
      await page.goto("/system");
      await expect(page.getByRole("heading", { name: "体系", level: 1 })).toBeVisible();
      await page.getByRole("option", { name: new RegExp(CATEGORY_NAME) }).click();
      await expect(page.locator(".system-detail__title")).toHaveText(CATEGORY_NAME);
      await expect(page.getByRole("treeitem")).toHaveCount(3);

      await expect(page).toHaveScreenshot(`system-${width}.png`, { fullPage: true });
    });
  }
});

test.describe("视觉基线 · 导入", () => {
  const WORK_NAME = "E2E-视觉导入";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  const errorMarkdown = [
    "# 小说设定导入模板 v1.0",
    "",
    "模板版本：9.9",
    "",
    "## 作品信息",
    "",
    `名称：${WORK_NAME}`,
    "",
    "## 体系类别",
    "",
    "境界（内置）",
    "",
    "## 人物",
    "",
    "### E2E-视觉导入人物",
    "",
    "图片：images/missing.png",
    "",
  ].join("\n");

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`导入校验错误态 ${width}`, async ({ page }) => {
      await page.setViewportSize({ width, height });
      await page.goto("/import");
      await expect(page.getByRole("heading", { name: "导入设定", level: 1 })).toBeVisible();

      await page.locator('input[type="file"]').nth(0).setInputFiles({
        name: "template.md",
        mimeType: "text/markdown",
        buffer: Buffer.from(errorMarkdown, "utf8"),
      });
      await page.getByRole("button", { name: "开始校验" }).click();
      await expect(page.locator(".ui-alert--danger").filter({ hasText: /发现 \d+ 个错误/ })).toBeVisible();
      await expect(page.locator(".import-error-list")).toBeVisible();

      await expect(page).toHaveScreenshot(`import-error-${width}.png`, { fullPage: true });
    });
  }
});

test.describe("视觉基线 · 检索", () => {
  const WORK_NAME = "E2E-视觉检索";

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  for (const { width, height } of [
    { width: 1280, height: 900 },
    { width: 1024, height: 800 },
  ]) {
    test(`检索结果 ${width}`, async ({ page, request }) => {
      const work = await createWork(request, WORK_NAME);
      const faction = await createFaction(request, work.id, "E2E-视觉狐族");
      await createCharacter(request, work.id, "E2E-视觉陆沉", {
        names: [
          { kind: "alias", value: "沉舟" },
          { kind: "title", value: "剑主" },
        ],
        description: "青丘之主。",
        factionID: faction.id,
      });
      await createCharacter(request, work.id, "E2E-视觉顾清寒");

      await page.setViewportSize({ width, height });
      await page.goto("/search?q=沉");
      await expect(page.getByRole("heading", { name: "检索", level: 1 })).toBeVisible();
      await expect(page.locator(".search-result")).toHaveCount(1);
      await expect(page.locator(".search-result")).toContainText("E2E-视觉陆沉");

      await expect(page).toHaveScreenshot(`search-${width}.png`, { fullPage: true });
    });
  }
});
