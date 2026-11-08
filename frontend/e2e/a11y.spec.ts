import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

const pages = [
  { path: "/", name: "概览" },
  { path: "/settings", name: "设置" },
  { path: "/no-such-page", name: "404" },
];

for (const target of pages) {
  test(`${target.name} 无 serious/critical 无障碍问题`, async ({ page }) => {
    await page.goto(target.path);

    const results = await new AxeBuilder({ page }).analyze();
    const blocking = results.violations.filter(
      (violation) => violation.impact === "serious" || violation.impact === "critical",
    );

    expect(blocking.map((violation) => `${violation.id}: ${violation.nodes.length}`)).toEqual([]);
  });
}
