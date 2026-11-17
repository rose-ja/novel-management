import { expect, test } from "@playwright/test";

import { cleanupE2E, createCharacter, createWork, uniqueName } from "./helpers";

/**
 * E2E-13：CSP 生效时检索与详情仍可用，且控制台不出现 CSP 违规。
 * 静态页面由 Go 同源托管，安全响应头由 httpapi 中间件统一注入。
 */
test.describe("CSP 安全基线", () => {
  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("响应头包含 CSP，详情与检索可用且无控制台违规", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("CSP作品"));
    const character = await createCharacter(request, work.id, uniqueName("陆沉"));

    const violations: string[] = [];
    page.on("console", (message) => {
      const text = message.text();
      if (/Content Security Policy|Refused to (load|execute|apply)/i.test(text)) {
        violations.push(text);
      }
    });
    page.on("pageerror", (error) => {
      violations.push(error.message);
    });

    const home = await page.goto("/");
    const csp = home?.headers()["content-security-policy"] ?? "";
    expect(csp).toContain("default-src 'self'");
    expect(csp).toContain("script-src");

    await page.goto(`/characters?id=${character.id}`);
    await expect(page.getByRole("heading", { name: character.name, level: 1 })).toBeVisible();

    await page.goto("/search");
    await page.getByLabel("检索关键词").fill(character.name);
    await expect(page.locator(".search-result").filter({ hasText: character.name }).first()).toBeVisible();

    expect(violations).toEqual([]);
  });
});
