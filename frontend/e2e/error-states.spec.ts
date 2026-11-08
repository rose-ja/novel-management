import { expect, test } from "@playwright/test";

/**
 * 错误态：静态页面由 Go 同源托管，因此「后端整体不可用」时页面本身无法加载；
 * 这里用路由拦截模拟接口失败，验证界面文案与重试入口（SPEC-PLAT-002）。
 */
test.describe("错误态", () => {
  test("接口网络异常时提示检查连接且不显示堆栈", async ({ page }) => {
    await page.route("**/api/v1/settings", (route) => route.abort("failed"));
    await page.goto("/");

    await expect(page.getByText("网络异常，请检查连接")).toBeVisible();
    await expect(page.getByRole("button", { name: "重试" })).toBeVisible();
    await expect(page.locator("body")).not.toContainText("TypeError");
  });

  test("接口 500 时提示加载失败并可手动重试", async ({ page }) => {
    let failed = true;
    await page.route("**/api/v1/settings", async (route) => {
      if (failed) {
        await route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({
            error: { code: "INTERNAL", message: "服务器内部错误", request_id: "e2e-500" },
          }),
        });
        failed = false;
        return;
      }
      await route.continue();
    });

    await page.goto("/");
    await expect(page.getByText("加载失败，请重试")).toBeVisible();

    await page.getByRole("button", { name: "重试" }).click();
    await expect(page.getByText("应用版本")).toBeVisible();
  });
});
