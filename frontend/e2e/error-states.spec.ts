import { expect, test } from "@playwright/test";

import { cleanupE2E, createCharacter, createWork, uniqueName } from "./helpers";

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

/** E2E-12：列表错误、筛选无结果与未选择作品时的空态。 */
test.describe("列表状态", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("列表接口 500 时保留筛选并可重试", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("错误态人物"));
    const created = await createCharacter(request, work.id, uniqueName("甲"));
    const query = uniqueName("无匹配");
    let failFiltered = true;

    await page.route(/\/api\/v1\/characters\?/, async (route) => {
      const url = new URL(route.request().url());
      if (route.request().method() === "GET" && failFiltered && url.searchParams.get("q") === query) {
        await route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({
            error: { code: "INTERNAL", message: "服务器内部错误", request_id: "e2e-list-500" },
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.goto("/characters");
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: created.name })).toBeVisible();

    await page.getByLabel("搜索").fill(query);
    await expect(page.getByRole("alert").filter({ hasText: "服务器内部错误" })).toBeVisible();
    await expect(page.getByLabel("搜索")).toHaveValue(query);

    failFiltered = false;
    await page.getByRole("button", { name: "重试" }).click();
    await expect(page.getByText("没有符合筛选的人物")).toBeVisible();
    await expect(page.getByLabel("搜索")).toHaveValue(query);

    await page.locator(".ui-empty").getByRole("button", { name: "清除筛选" }).click();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: created.name })).toBeVisible();
  });

  test("筛选无结果时展示可清除的筛选空态", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("筛选空态"));
    await createCharacter(request, work.id, uniqueName("乙"));

    await page.goto("/characters");
    await page.getByLabel("搜索").fill(uniqueName("无结果"));

    await expect(page.getByText("没有符合筛选的人物")).toBeVisible();
    await expect(page.getByText("调整或清除筛选条件后再试。")).toBeVisible();
    await expect(page.getByRole("button", { name: "清除筛选" })).toBeVisible();
  });

  test("未选择作品时主页面展示创建作品空态", async ({ page }) => {
    await page.route("**/api/v1/settings", async (route) => {
      if (route.request().method() !== "GET") {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          current_work_id: null,
          last_export_at: null,
          last_restore_at: null,
          data_dir: "/data",
          snapshot_keep: 3,
          app_version: "e2e",
        }),
      });
    });

    await page.goto("/characters");
    await expect(page.getByRole("heading", { name: "还没有作品" })).toBeVisible();
    await expect(page.getByRole("button", { name: "新建人物" })).toBeDisabled();
  });
});
