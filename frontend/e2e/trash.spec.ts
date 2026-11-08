import { expect, test } from "@playwright/test";

import { cleanupE2E, createWork, deleteWork, listTrash, uniqueName } from "./helpers";

test.describe("回收站", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("恢复作品后从回收站消失", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("恢复"));
    await deleteWork(request, work);

    await page.goto("/trash");
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: work.name })).toBeVisible();

    await page.getByRole("button", { name: `恢复作品《${work.name}》` }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText(`恢复作品《${work.name}》？`)).toBeVisible();
    await dialog.getByRole("button", { name: "恢复" }).click();

    await expect(page.locator(".ui-toast").filter({ hasText: `已恢复《${work.name}》` })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: work.name })).toHaveCount(0);
  });

  test("重名时改名恢复", async ({ page, request }) => {
    const name = uniqueName("重名");
    const deleted = await createWork(request, name);
    await deleteWork(request, deleted);
    await createWork(request, name);
    const renamed = `${name}（恢复）`;

    await page.goto("/trash");
    await page.getByRole("button", { name: `恢复作品《${name}》` }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "恢复" }).click();

    await expect(dialog.getByText("恢复目标与现有记录重名")).toBeVisible();
    const nameInput = dialog.getByLabel("恢复后名称");
    await expect(nameInput).toHaveValue(renamed);
    await dialog.getByRole("button", { name: "恢复" }).click();

    await expect(page.locator(".ui-toast").filter({ hasText: `已恢复《${renamed}》` })).toBeVisible();
  });

  test("永久删除需要确认并生成快照", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("永久删除"));
    await deleteWork(request, work);

    await page.goto("/trash");
    await page.getByRole("button", { name: `永久删除作品《${work.name}》` }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("此操作不可撤销。删除前会自动创建快照。")).toBeVisible();
    await dialog.getByRole("button", { name: "永久删除" }).click();

    await expect(page.locator(".ui-toast").filter({ hasText: `已永久删除《${work.name}》` })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: work.name })).toHaveCount(0);
  });

  test("清空回收站仅清理测试数据时验证", async ({ page, request }) => {
    const first = await createWork(request, uniqueName("清空A"));
    const second = await createWork(request, uniqueName("清空B"));
    await deleteWork(request, first);
    await deleteWork(request, second);

    const items = await listTrash(request);
    test.skip(
      items.some((item) => !item.name.startsWith("E2E-")),
      "回收站中存在非测试数据，跳过清空操作",
    );

    await page.goto("/trash");
    await page.getByRole("button", { name: "清空回收站" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("将永久删除全部 2 项")).toBeVisible();
    await dialog.getByRole("button", { name: "清空回收站" }).click();

    await expect(page.locator(".ui-toast").filter({ hasText: "回收站已清空" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "回收站是空的" })).toBeVisible();
  });

  for (const width of [1280, 1024]) {
    test(`${width} 宽度无横向溢出`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto("/trash");
      await expect(page.getByRole("heading", { name: "回收站", level: 1 })).toBeVisible();

      const overflow = await page.evaluate(() => {
        const element = document.scrollingElement ?? document.documentElement;
        return element.scrollWidth - element.clientWidth;
      });
      expect(overflow).toBeLessThanOrEqual(1);
    });
  }
});
