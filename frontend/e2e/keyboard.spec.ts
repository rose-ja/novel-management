import { expect, test } from "@playwright/test";

/** 键盘可达性：跳过链接与侧栏导航可用 Tab 聚焦，焦点可见（SPEC-PLAT-004）。 */
test.describe("键盘可达", () => {
  test("首个 Tab 聚焦跳过链接并可跳到主要内容", async ({ page }) => {
    await page.goto("/");

    await page.keyboard.press("Tab");
    const skipLink = page.getByRole("link", { name: "跳到主要内容" });
    await expect(skipLink).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(page.locator("#main-content")).toBeFocused();
  });

  test("侧栏导航可用键盘遍历", async ({ page }) => {
    await page.goto("/");

    const nav = page.getByRole("navigation", { name: "主导航" });
    const overview = nav.getByRole("link", { name: "概览" });
    const trash = nav.getByRole("link", { name: "回收站" });
    const settings = nav.getByRole("link", { name: "设置" });

    await overview.focus();
    await expect(overview).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(trash).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(settings).toBeFocused();

    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/settings$/);
    await expect(page.getByRole("heading", { name: "设置", level: 1 })).toBeVisible();
  });

  test("作品切换器可用键盘打开、方向键遍历与 Esc 关闭", async ({ page }) => {
    await page.goto("/");

    const trigger = page.locator(".shell__work--trigger");
    await trigger.focus();
    await page.keyboard.press("Enter");

    const menu = page.locator(".work-menu");
    await expect(menu).toBeVisible();
    await expect(menu.locator(":focus")).toHaveCount(1);

    await page.keyboard.press("ArrowDown");
    await expect(menu.locator(":focus")).toHaveCount(1);
    await page.keyboard.press("ArrowUp");
    await expect(menu.locator(":focus")).toHaveCount(1);

    await page.keyboard.press("Escape");
    await expect(menu).toHaveCount(0);
    await expect(trigger).toBeFocused();
  });
});
