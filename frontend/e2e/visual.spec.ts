import { expect, test } from "@playwright/test";

/**
 * 视觉基线：基线图由本机 Windows 环境生成，跨机器对比仅作参考。
 * 系统信息区含环境相关路径，用遮罩避免无意义差异。
 */
test.describe("视觉基线", () => {
  test("概览页 1280", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "概览", level: 1 })).toBeVisible();
    await expect(page.locator(".info-list")).toBeVisible();

    await expect(page).toHaveScreenshot("overview-1280.png", {
      mask: [page.locator(".info-list")],
      fullPage: true,
    });
  });

  test("概览页 1024", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 800 });
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "概览", level: 1 })).toBeVisible();
    await expect(page.locator(".info-list")).toBeVisible();

    await expect(page).toHaveScreenshot("overview-1024.png", {
      mask: [page.locator(".info-list")],
      fullPage: true,
    });
  });
});
