import { expect, test } from "@playwright/test";

import { cleanupE2E, uniqueName } from "./helpers";

function template(workName: string, extra: string[] = []): string {
  return [
    "# 小说设定导入模板 v1.0",
    "",
    "模板版本：1.0",
    "",
    "## 作品信息",
    "",
    `名称：${workName}`,
    "简介：E2E 导入校验数据。",
    "",
    "## 体系类别",
    "",
    "境界（内置）",
    "",
    ...extra,
    "## 人物",
    "",
    "### E2E-导入人物",
    "",
    "简介：E2E 导入人物。",
    "",
  ].join("\n");
}

/** 示例 zip 中引用了图片，这里单独构造缺图模板用于阻断验证。 */
function missingImageTemplate(workName: string): string {
  return template(workName).replace("简介：E2E 导入人物。", "图片：images/missing.png\n简介：E2E 导入人物。");
}

async function uploadMarkdown(page: import("@playwright/test").Page, markdown: string, fileName = "template.md") {
  await page.locator('input[type="file"]').nth(0).setInputFiles({
    name: fileName,
    mimeType: "text/markdown",
    buffer: Buffer.from(markdown, "utf8"),
  });
}

test.describe("导入向导", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("上传 → 校验错误 → 修正 → 确认 → 报告 → 批量校对", async ({ page }) => {
    test.setTimeout(300_000);
    const workName = uniqueName("导入作品");

    await page.goto("/import");
    await expect(page.getByRole("heading", { name: "导入设定", level: 1 })).toBeVisible();

    // 1. 错误模板：版本不支持，确认按钮必须禁用。
    await uploadMarkdown(page, template(workName).replace("模板版本：1.0", "模板版本：9.9"));
    await page.getByRole("button", { name: "开始校验" }).click();
    await expect(page.locator(".ui-alert--danger").filter({ hasText: /发现 \d+ 个错误/ })).toBeVisible();
    await expect(page.getByRole("button", { name: "确认导入" })).toBeDisabled();

    // 2. 修正后重新上传。
    await page.getByRole("button", { name: "上传" }).click();
    await uploadMarkdown(page, template(workName));
    await page.getByRole("button", { name: "开始校验" }).click();
    await expect(page.getByText("校验通过，可以确认导入")).toBeVisible({ timeout: 60_000 });
    await expect(page.getByText("新作品《" + workName + "》")).toBeVisible();

    // 3. 二次确认后提交。
    await page.getByRole("button", { name: "确认导入" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText(`确认导入《${workName}》？`)).toBeVisible();
    await dialog.getByRole("button", { name: "确认导入" }).click();

    // 4. 报告。
    await expect(page.locator(".ui-alert").filter({ hasText: "导入完成" })).toBeVisible({ timeout: 180_000 });
    await expect(page.getByRole("heading", { name: "导入报告" })).toBeVisible();
    await expect(page.locator(".shell__work-text")).toHaveText(workName);

    // 5. 校对清单：批量确认人物后全部完成。
    await page.getByRole("button", { name: "进入校对" }).click();
    await expect(page.getByRole("heading", { name: "校对清单" })).toBeVisible();
    await page.getByRole("button", { name: "批量确认" }).first().click();
    const batchDialog = page.getByRole("dialog");
    await expect(batchDialog.getByText("批量确认校对")).toBeVisible();
    await batchDialog.getByRole("button", { name: "确认" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已确认 1 项" })).toBeVisible();
    await expect(page.getByText("全部可校对实体已校对")).toBeVisible({ timeout: 30_000 });
  });

  test("缺失图片进入错误清单并禁用确认", async ({ page }) => {
    await page.goto("/import");
    await uploadMarkdown(page, missingImageTemplate(uniqueName("导入缺图")));
    await page.getByRole("button", { name: "开始校验" }).click();

    await expect(page.getByText(/发现 \d+ 个错误，修正后重新上传/)).toBeVisible();
    await expect(page.locator(".import-grid__main p").filter({ hasText: "缺失 1" })).toBeVisible();
    await expect(page.getByRole("button", { name: "确认导入" })).toBeDisabled();
  });
});
