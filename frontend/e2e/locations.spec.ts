import { expect, test } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createFaction,
  createLocation,
  createWork,
  uniqueName,
} from "./helpers";

function treeRow(page: import("@playwright/test").Page, name: string) {
  return page.locator(".location-tree__row").filter({ hasText: name });
}

test.describe("地点树", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("内联新建三级地点、默认展开两级与搜索/类型筛选", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("地点树"));
    const rootName = uniqueName("东荒");
    const childName = uniqueName("青丘山");
    const leafName = uniqueName("狐族秘境");

    await page.goto("/locations");
    await expect(page.getByRole("heading", { name: "地点", level: 1 })).toBeVisible();
    await page.locator(".ui-page-header").getByRole("button", { name: "新建地点" }).click();
    await page.getByLabel("地点名称").fill(rootName);
    await page.locator("#location-inline-type").fill("星域");
    await page.getByRole("button", { name: "创建地点" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已创建地点「${rootName}」` })).toBeVisible();
    await expect(treeRow(page, rootName)).toBeVisible();

    await treeRow(page, rootName).getByRole("button", { name: `新增下级地点：${rootName}` }).click();
    await expect(page.getByLabel("上级地点")).toHaveValue(/.+/);
    await page.getByLabel("地点名称").fill(childName);
    await page.locator("#location-inline-type").fill("星城");
    await page.getByRole("button", { name: "创建地点" }).click();
    await expect(treeRow(page, childName)).toBeVisible();

    await treeRow(page, childName).getByRole("button", { name: `新增下级地点：${childName}` }).click();
    await page.getByLabel("地点名称").fill(leafName);
    await page.locator("#location-inline-type").fill("秘境");
    await page.getByRole("button", { name: "创建地点" }).click();
    // 默认展开两级：根与第二层可见，第三层需要展开第二层。
    await expect(treeRow(page, rootName)).toBeVisible();
    await expect(treeRow(page, childName)).toBeVisible();
    await expect(treeRow(page, leafName)).toBeVisible();

    // 搜索命中并展开祖先。
    await page.getByLabel("搜索地点").fill("狐族秘境");
    await expect(treeRow(page, leafName)).toBeVisible();
    await expect(page.locator(".location-tree__mark")).toHaveText("狐族秘境");
    await page.getByLabel("搜索地点").fill("");
    await expect(treeRow(page, rootName)).toBeVisible();

    // 类型筛选保留祖先链。
    await page.getByLabel("类型").selectOption({ label: "秘境" });
    await expect(treeRow(page, leafName)).toBeVisible();
    await expect(treeRow(page, rootName)).toBeVisible();
    await page.getByRole("button", { name: "重置", exact: false }).click();

    // 重名提示。
    await page.locator(".ui-page-header").getByRole("button", { name: "新建地点" }).click();
    await page.getByLabel("地点名称").fill(rootName);
    await page.getByRole("button", { name: "创建地点" }).click();
    await expect(page.getByText("名称已存在，请换一个名称")).toBeVisible();
  });
});

test.describe("地点详情与关系", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("详情编辑、人物与势力关系双向可见", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("地点详情"));
    const root = await createLocation(request, work.id, uniqueName("东荒"), { typeText: "星域" });
    const child = await createLocation(request, work.id, uniqueName("青丘山"), {
      parentID: root.id,
      typeText: "星城",
    });
    const faction = await createFaction(request, work.id, uniqueName("青丘狐族"));
    const character = await createCharacter(request, work.id, uniqueName("涂山氏"), {
      locationLinks: [{ locationID: child.id, note: "出生地" }],
    });

    await page.goto(`/locations?id=${child.id}`);
    await expect(page.getByRole("heading", { name: child.name, level: 1 })).toBeVisible();
    await expect(page.getByRole("link", { name: character.name })).toBeVisible();
    await expect(page.getByText("出生地")).toBeVisible();

    // 地点页编辑人物备注，保存后人物详情同步。
    await page.getByRole("button", { name: "编辑关联人物" }).click();
    await page.getByLabel("备注 1").fill("祖地");
    await page.getByRole("button", { name: "保存关联" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "关联已保存" })).toBeVisible();
    await expect(page.getByText("祖地")).toBeVisible();

    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: character.name, level: 1 })).toBeVisible();
    await expect(page.getByRole("link", { name: `查看地点：${child.name}` })).toBeVisible();
    await expect(page.getByText("祖地")).toBeVisible();

    // 地点页新增势力关系并填写备注。
    await page.goto(`/locations?id=${child.id}`);
    await page.getByRole("button", { name: "编辑关联势力" }).click();
    await page.getByRole("button", { name: "添加势力" }).click();
    await page.getByLabel("关联势力 1").selectOption({ label: faction.name });
    await page.getByLabel("备注 1").fill("宗门驻地");
    await page.getByRole("button", { name: "保存关联" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "关联已保存" })).toBeVisible();

    await page.getByRole("link", { name: `查看势力：${faction.name}` }).click();
    await expect(page).toHaveURL(new RegExp(`/factions\\?id=${faction.id}`));
    await expect(page.getByRole("link", { name: `查看地点：${child.name}` })).toBeVisible();
    await expect(page.getByText("宗门驻地")).toBeVisible();

    // 编辑地点：上级选项保留自身但禁用，避免成环。
    await page.goto(`/locations?id=${child.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "编辑" }).click();
    const selfOption = page.getByLabel("上级地点").locator("option").filter({ hasText: child.name });
    await expect(selfOption).toHaveCount(1);
    await expect(selfOption).toBeDisabled();
    await page.getByLabel("简介").fill("狐族世代居住的秘境。");
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已保存地点「${child.name}」` })).toBeVisible();
    await expect(page.getByText("狐族世代居住的秘境。")).toBeVisible();
  });
});

test.describe("地点删除与回收站", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("删除被下级阻挡并列出引用", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("地点删除"));
    const parent = await createLocation(request, work.id, uniqueName("上级地点"));
    const child = await createLocation(request, work.id, uniqueName("下级地点"), {
      parentID: parent.id,
    });

    await page.goto(`/locations?id=${parent.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "删除地点" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除地点" }).click();
    await expect(page.getByRole("dialog")).toContainText("仍有下级地点，无法删除");
    await expect(page.getByRole("dialog").getByRole("link", { name: child.name })).toBeVisible();
  });

  test("删除被人物关系阻挡，移除后可删除并撤销", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("地点关系删除"));
    const location = await createLocation(request, work.id, uniqueName("被引用地点"));
    const character = await createCharacter(request, work.id, uniqueName("引用者"), {
      locationLinks: [{ locationID: location.id, note: "常驻" }],
    });

    await page.goto(`/locations?id=${location.id}`);
    await page.locator(".ui-page-header").getByRole("button", { name: "删除地点" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除地点" }).click();
    await expect(page.getByRole("dialog")).toContainText("地点仍被人物或势力引用，无法删除");
    await expect(page.getByRole("dialog").getByRole("link", { name: character.name })).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: "取消" }).click();

    // 移除人物关系后删除成功，并可从回收站恢复。
    await page.getByRole("button", { name: "编辑关联人物" }).click();
    await page.getByRole("button", { name: `移除关联：${character.name}` }).click();
    await page.getByRole("button", { name: "保存关联" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "关联已保存" })).toBeVisible();
    await page.locator(".ui-page-header").getByRole("button", { name: "删除地点" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除地点" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `地点「${location.name}」已移入回收站` })).toBeVisible();

    await page.goto("/trash");
    await page.getByLabel("类型").selectOption({ label: "地点" });
    const row = page.locator(".ui-table tbody tr").filter({ hasText: location.name });
    await expect(row).toBeVisible();
    await row.getByRole("button", { name: `恢复地点「${location.name}」` }).click();
    await page.getByRole("dialog").getByRole("button", { name: "恢复" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已恢复地点「${location.name}」` })).toBeVisible();
  });
});
