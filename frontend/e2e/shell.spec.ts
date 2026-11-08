import { expect, test } from "@playwright/test";

test.describe("应用壳层", () => {
  test("渲染顶栏、导航与内容区", async ({ page }) => {
    await page.goto("/");

    await expect(page.locator(".shell__brand-name")).toHaveText("小说设定管理系统");
    await expect(page.getByText("本机 · 127.0.0.1")).toBeVisible();

    const nav = page.getByRole("navigation", { name: "主导航" });
    await expect(nav.getByRole("link", { name: "概览" })).toHaveAttribute("aria-current", "page");
    await expect(nav.getByRole("link", { name: "设置" })).toBeVisible();
    await expect(nav.getByText("资料")).toBeVisible();
    await expect(nav.getByText("工具")).toBeVisible();
    await expect(nav.getByText("数据仅存本机")).toBeVisible();

    await expect(page.getByRole("heading", { name: "概览", level: 1 })).toBeVisible();
    await expect(page.getByRole("main")).toBeVisible();
  });

  test("无作品时展示空状态说明", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "还没有作品" })).toBeVisible();
    await expect(page.getByText("作品用于隔离每本小说的世界观、体系与榜单。")).toBeVisible();
    // 本阶段不提供创建入口，避免出现不可用按钮。
    await expect(page.getByRole("main").getByRole("button", { name: /新建作品/ })).toHaveCount(0);
  });

  test("设置页只读展示系统信息", async ({ page }) => {
    await page.goto("/settings");

    await expect(page.getByRole("heading", { name: "设置", level: 1 })).toBeVisible();
    await expect(page.getByText("应用版本")).toBeVisible();
    await expect(page.getByText("数据目录")).toBeVisible();
    await expect(page.getByText("自动快照保留")).toBeVisible();
    await expect(page.getByRole("button", { name: /保存/ })).toHaveCount(0);
  });

  test("未知路由返回可返回的 404 页", async ({ page }) => {
    const response = await page.goto("/no-such-page");

    expect(response?.status()).toBe(404);
    await expect(page.getByRole("heading", { name: "页面不存在" })).toBeVisible();
    await expect(page.getByRole("link", { name: "返回首页" })).toBeVisible();
  });

  for (const width of [1280, 1024]) {
    test(`${width} 宽度无横向溢出`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto("/settings");
      await expect(page.getByRole("heading", { name: "设置", level: 1 })).toBeVisible();

      const overflow = await page.evaluate(() => {
        const element = document.scrollingElement ?? document.documentElement;
        return element.scrollWidth - element.clientWidth;
      });
      expect(overflow).toBeLessThanOrEqual(1);
    });
  }

  test("<1024 显示桌面端提示且不提供移动导航", async ({ page }) => {
    await page.setViewportSize({ width: 900, height: 800 });
    await page.goto("/");

    await expect(page.getByText("本系统按桌面端设计，建议使用 1024px 以上窗口")).toBeVisible();
    // 侧栏始终可见，不切换到抽屉导航。
    await expect(page.getByRole("navigation", { name: "主导航" })).toBeVisible();
    await expect(page.getByRole("button", { name: "打开导航" })).toHaveCount(0);
  });

  test("壳层加载无 console 错误", async ({ page }) => {
    const errors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text());
    });
    page.on("pageerror", (error) => errors.push(error.message));

    await page.goto("/");
    await expect(page.getByRole("heading", { name: "概览", level: 1 })).toBeVisible();
    await expect(page.getByText("应用版本")).toBeVisible();

    expect(errors).toEqual([]);
  });
});
