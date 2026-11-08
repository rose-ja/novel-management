import { expect, test } from "@playwright/test";

import { cleanupE2E, createWork, deleteWork, uniqueName } from "./helpers";

test.describe("设置页快照", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("删除作品后快照列表只读展示", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("快照"));
    await deleteWork(request, work);

    await page.goto("/settings");
    await expect(page.getByRole("heading", { name: "自动快照" })).toBeVisible();

    const card = page.locator(".ui-card").filter({ hasText: "自动快照" });
    const row = card.locator(".ui-table tbody tr").filter({ hasText: "删除作品" }).first();
    await expect(row).toBeVisible();
    await expect(row.getByText("成功")).toBeVisible();
    // 本变更不提供页面内恢复入口。
    await expect(card.getByRole("button", { name: /恢复|还原/ })).toHaveCount(0);
  });
});
