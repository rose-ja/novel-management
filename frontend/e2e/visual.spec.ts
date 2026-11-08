import { expect, test } from "@playwright/test";

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
    await expect(page.getByRole("heading", { name: "设置", level: 1 })).toBeVisible();

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
