import { expect, test } from "@playwright/test";

import {
  cleanupE2E,
  createWork,
  openWorkMenu,
  uniqueName,
} from "./helpers";

test.describe("作品生命周期", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("创建作品并成为当前作品", async ({ page }) => {
    const name = uniqueName("创建");

    await page.goto("/");
    await openWorkMenu(page);
    await page.getByRole("button", { name: "新建作品" }).click();
    await page.getByLabel("作品名称").fill(name);
    await page.getByLabel("简介").fill("用于验证创建流程");
    await page.getByRole("button", { name: "创建作品" }).click();

    await expect(page.locator(".shell__work-text")).toHaveText(name);
    await expect(page.locator(".ui-toast").filter({ hasText: `已创建《${name}》` })).toBeVisible();

    await page.goto("/settings");
    await expect(page.getByText(`《${name}》`)).toBeVisible();
  });

  test("切换作品并持久化到设置页", async ({ page, request }) => {
    const first = await createWork(request, uniqueName("切换A"));
    const second = await createWork(request, uniqueName("切换B"));

    await page.goto("/");
    await expect(page.locator(".shell__work-text")).toHaveText(second.name);

    await openWorkMenu(page);
    await page.locator(".work-menu__select").filter({ hasText: first.name }).click();

    await expect(page.locator(".shell__work-text")).toHaveText(first.name);
    await expect(page.locator(".ui-toast").filter({ hasText: `已切换到《${first.name}》` })).toBeVisible();

    await page.goto("/settings");
    await expect(page.getByText(`《${first.name}》`)).toBeVisible();
  });

  test("重命名作品后顶栏同步", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("重命名"));
    const renamed = `${work.name}-新`;

    await page.goto("/");
    await openWorkMenu(page);
    await page.getByRole("button", { name: `编辑作品《${work.name}》` }).click();
    await page.getByLabel("作品名称").fill(renamed);
    await page.getByRole("button", { name: "保存修改" }).click();

    await expect(page.locator(".shell__work-text")).toHaveText(renamed);
    await expect(page.locator(".ui-toast").filter({ hasText: `已保存《${renamed}》` })).toBeVisible();
  });

  test("删除作品需要名称确认，撤销后回到当前作品", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("删除"));

    await page.goto("/");
    await openWorkMenu(page);
    await page.getByRole("button", { name: `删除作品《${work.name}》` }).click();

    const submit = page.getByRole("button", { name: "删除作品" });
    await expect(submit).toBeDisabled();

    await page.getByLabel("输入作品名称以确认").fill(work.name);
    await expect(submit).toBeEnabled();
    await submit.click();

    await expect(page.locator(".ui-toast").filter({ hasText: `《${work.name}》已移入回收站` })).toBeVisible();
    await expect(page.locator(".shell__work-text")).toHaveText("未选择作品");

    await page.locator(".ui-toast").getByRole("button", { name: "撤销" }).click();
    await expect(page.locator(".shell__work-text")).toHaveText(work.name);
  });

  test("名称冲突在弹窗内提示", async ({ page, request }) => {
    const existing = await createWork(request, uniqueName("冲突"));

    await page.goto("/");
    await openWorkMenu(page);
    await page.getByRole("button", { name: "新建作品" }).click();
    await page.getByLabel("作品名称").fill(existing.name);
    await page.getByRole("button", { name: "创建作品" }).click();

    await expect(page.getByText("名称已存在，请换一个名称")).toBeVisible();
  });
});
