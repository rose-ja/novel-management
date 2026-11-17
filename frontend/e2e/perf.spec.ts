import { mkdirSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

import { expect, test } from "@playwright/test";

import { JSON_HEADERS } from "./helpers";

/**
 * 性能基线：仅在显式设置 PERF_WORK_ID 时运行，由 scripts/perf.ps1 调用。
 * 首屏判定为人物列表首个数据行可见，阈值 2000ms。
 */
const workID = process.env.PERF_WORK_ID;

test.describe("性能基线", () => {
  test.skip(!workID, "设置 PERF_WORK_ID 后运行性能基线");

  let previousWorkID: string | null = null;

  test.beforeAll(async ({ request }) => {
    const current = await request.get("/api/v1/settings");
    expect(current.ok()).toBe(true);
    previousWorkID = ((await current.json()) as { current_work_id?: string | null }).current_work_id ?? null;

    const response = await request.patch("/api/v1/settings", {
      data: { current_work_id: workID },
      headers: JSON_HEADERS,
    });
    expect(response.ok()).toBe(true);
  });

  test.afterAll(async ({ request }) => {
    const response = await request.patch("/api/v1/settings", {
      data: { current_work_id: previousWorkID },
      headers: JSON_HEADERS,
    });
    expect(response.ok()).toBe(true);
  });

  test("人物列表首屏可交互时间不超过 2000ms", async ({ page }) => {
    const started = Date.now();
    await page.goto("/characters");
    await expect(page.getByRole("heading", { name: "人物", level: 1 })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr").first()).toBeVisible({ timeout: 10_000 });
    const elapsed = Date.now() - started;

    console.log(`首屏可交互：${elapsed}ms`);
    const resultFile = process.env.PERF_RESULT_FILE;
    if (resultFile) {
      mkdirSync(dirname(resultFile), { recursive: true });
      writeFileSync(
        resultFile,
        JSON.stringify({ work_id: workID, first_screen_ms: elapsed, measured_at: new Date().toISOString() }),
      );
    }
    expect(elapsed).toBeLessThanOrEqual(2000);
  });
});
