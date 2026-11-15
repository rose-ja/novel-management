import { expect, test } from "@playwright/test";

import { cleanupE2E, createWork, uniqueName } from "./helpers";

test.describe("设置页备份", () => {
  test.use({ permissions: ["clipboard-read", "clipboard-write"] });

  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("导出 Dialog 默认包含回收站并生成可下载 zip", async ({ page, request }) => {
    test.setTimeout(180_000);
    await createWork(request, uniqueName("导出页"));

    await page.goto("/settings");
    await expect(page.getByRole("heading", { name: "设置 / 备份", level: 1 })).toBeVisible();
    await expect(page.getByRole("heading", { name: "导出全量数据" })).toBeVisible();

    await page.getByRole("button", { name: "导出全量数据" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("checkbox", { name: /包含回收站数据/ })).toBeChecked();

    await dialog.getByRole("button", { name: "生成导出包" }).click();
    await expect(dialog.getByText("导出包已生成", { exact: true })).toBeVisible({ timeout: 150_000 });

    const download = dialog.getByRole("link", { name: "下载 zip" });
    const href = await download.getAttribute("href");
    expect(href).toMatch(/^\/api\/v1\/exports\/.+\/download$/);

    const response = await request.get(href!);
    expect(response.status()).toBe(200);
    expect(response.headers()["content-type"]).toContain("application/zip");
    expect((await response.body()).length).toBeGreaterThan(0);

    await dialog.locator(".ui-dialog__footer").getByRole("button", { name: "关闭" }).click();
    await expect(dialog).toHaveCount(0);
  });

  test("恢复命令可复制并提示", async ({ page }) => {
    await page.goto("/settings");
    const card = page.locator(".ui-card").filter({ hasText: "命令行恢复" });
    await expect(card).toBeVisible();
    await expect(card.getByText("docker compose run --rm cli restore /data/exports/<file>.zip --yes")).toBeVisible();
    await expect(
      card.getByText("docker compose run --rm cli restore-snapshot /data/snapshots/<dir> --yes"),
    ).toBeVisible();

    await card.getByRole("button", { name: "复制恢复命令" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已复制" })).toBeVisible();
  });

  test("无作品时导出按钮禁用并说明原因", async ({ page }) => {
    await page.route(/\/api\/v1\/works\?page=1&page_size=1$/, async (route) => {
      await route.fulfill({ json: { items: [], page: 1, page_size: 1, total: 0 } });
    });
    await page.goto("/settings");
    await expect(page.getByRole("button", { name: "导出全量数据" })).toBeDisabled();
    await expect(page.getByText("暂无作品数据，无法导出")).toBeVisible();
  });
});
