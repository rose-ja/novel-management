import { defineConfig, devices } from "@playwright/test";

/**
 * E2E 针对真实容器栈运行：
 * 1. `docker compose up -d`
 * 2. `npm run build`（产物由 Go 托管）
 * 3. `npm run test:e2e`
 * 可通过 E2E_BASE_URL 指向其他地址。
 *
 * 浏览器：默认使用本机已安装的 Chrome（`channel: "chrome"`），
 * 因为本机到 Playwright CDN 的下载不可用。需要固定版本时可设置
 * `E2E_CHANNEL=chromium` 并执行 `npx playwright install chromium`。
 */
const baseURL = process.env.E2E_BASE_URL ?? "http://127.0.0.1:17860";
const channel = process.env.E2E_CHANNEL ?? "chrome";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: !!process.env.CI,
  reporter: [["list"]],
  expect: {
    toHaveScreenshot: { maxDiffPixelRatio: 0.02 },
  },
  use: {
    baseURL,
    locale: "zh-CN",
    timezoneId: "Asia/Shanghai",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        channel,
        viewport: { width: 1280, height: 900 },
      },
    },
  ],
});
