import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

import {
  cleanupE2E,
  createCharacter,
  createWork,
  uniqueName,
} from "./helpers";

const png = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
  "base64",
);

async function seedCharacter(request: APIRequestContext) {
  const work = await createWork(request, uniqueName("媒体"));
  const character = await createCharacter(request, work.id, uniqueName("陆沉"));
  return { work, character };
}

async function openDetail(page: Page, id: string) {
  await page.goto(`/characters?id=${id}`);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await expect(page.locator(".character-detail")).toBeVisible();
}

test.describe("人物主图与大事记", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("上传、展示与移除主图", async ({ page, request }) => {
    const { character } = await seedCharacter(request);
    await openDetail(page, character.id);

    await page.getByRole("button", { name: "编辑" }).click();
    const uploader = page.locator(".image-uploader");
    await uploader.locator('input[type="file"]').setInputFiles({
      name: "avatar.png",
      mimeType: "image/png",
      buffer: png,
    });
    await expect(uploader.locator(".character-image").first()).toBeVisible();
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已保存人物" })).toBeVisible();

    await expect(page.locator(".character-detail__identity img.character-image")).toBeVisible();
    await expect(page.getByText("已上传").first()).toBeVisible();

    await page.getByRole("link", { name: "返回人物列表" }).first().click();
    await expect(page.locator(".ui-table tbody tr").filter({ hasText: character.name }).locator("img.character-image")).toBeVisible();

    await page.locator(".ui-table tbody tr").filter({ hasText: character.name }).getByRole("link").first().click();
    await expect(page.locator(".character-detail")).toBeVisible();
    await page.getByRole("button", { name: "编辑" }).click();
    await page.getByRole("button", { name: "移除主图" }).click();
    await expect(page.getByRole("dialog").getByText("移除主图？")).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: "移除" }).click();
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "已保存人物" })).toBeVisible();
    await expect(page.getByRole("img", { name: `人物主图：${character.name}` })).toBeVisible();
    await expect(page.getByText("未上传").first()).toBeVisible();
  });

  test("非法图片在字段级被拒绝", async ({ page, request }) => {
    const { character } = await seedCharacter(request);
    await openDetail(page, character.id);

    await page.getByRole("button", { name: "编辑" }).click();
    await page.locator('.image-uploader input[type="file"]').setInputFiles({
      name: "fake.png",
      mimeType: "image/png",
      buffer: Buffer.from("not an image"),
    });
    await page.getByRole("button", { name: "保存修改" }).click();
    await expect(page.getByText("仅支持 JPG、PNG 或 WebP 图片")).toBeVisible();
    await expect(page.getByRole("button", { name: "保存修改" })).toBeVisible();
  });

  test("大事记新增、编辑、删除与撤销", async ({ page, request }) => {
    const { character } = await seedCharacter(request);
    await openDetail(page, character.id);

    const section = page.locator(".chronicle");
    await expect(section.getByText("还没有大事记")).toBeVisible();
    await section.getByRole("button", { name: "添加第一条" }).click();
    await section.getByLabel("时间").fill("元启三年");
    await section.getByLabel("内容").fill("初入宗门");
    await section.getByRole("button", { name: "保存" }).click();
    await expect(section.getByText("初入宗门")).toBeVisible();
    await expect(section.getByText("01")).toBeVisible();
    await expect(section.getByText("元启三年")).toBeVisible();

    await section.getByRole("button", { name: "添加" }).click();
    await section.getByLabel("时间").fill("元启五年");
    await section.getByLabel("内容").fill("突破金丹");
    await section.getByRole("button", { name: "保存" }).click();
    await expect(section.getByText("突破金丹")).toBeVisible();
    await expect(section.getByText("02")).toBeVisible();

    const first = section.locator(".chronicle__item").filter({ hasText: "初入宗门" });
    await first.getByRole("button", { name: "编辑" }).click();
    await section.getByLabel("内容").fill("初入宗门（修订）");
    await section.getByRole("button", { name: "保存" }).click();
    await expect(section.getByText("初入宗门（修订）")).toBeVisible();

    const second = section.locator(".chronicle__item").filter({ hasText: "突破金丹" });
    await second.getByRole("button", { name: "删除" }).click();
    await section.locator(".chronicle__confirm").getByRole("button", { name: "删除" }).click();
    await expect(section.getByText("突破金丹")).toHaveCount(0);
    const toast = page.locator(".ui-toast").filter({ hasText: "已删除大事记" });
    await expect(toast).toBeVisible();
    await toast.getByRole("button", { name: "撤销" }).click();
    await expect(section.getByText("突破金丹")).toBeVisible();
  });
});
