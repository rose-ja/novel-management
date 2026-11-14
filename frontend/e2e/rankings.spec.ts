import { expect, test } from "@playwright/test";

import {
  assignRankingSeat,
  cleanupE2E,
  createCharacter,
  createRanking,
  createWork,
  getRanking,
  uniqueName,
} from "./helpers";

function seatRow(page: import("@playwright/test").Page, seatNo: number) {
  return page.locator(".ui-table tbody tr").filter({ has: page.locator(`[data-seat-no="${seatNo}"]`) });
}

test.describe("榜单", () => {
  test.beforeEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test.afterEach(async ({ request }) => {
    await cleanupE2E(request);
  });

  test("新建榜单、指派、跳席、交换与移出撤销", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("榜单交互"));
    const alpha = await createCharacter(request, work.id, uniqueName("甲"));
    const beta = await createCharacter(request, work.id, uniqueName("乙"));
    const rankingName = uniqueName("天榜");

    await page.goto("/rankings");
    await expect(page.getByRole("heading", { name: "榜单", level: 1 })).toBeVisible();
    await page.locator(".ui-page-header").getByRole("button", { name: "新建榜单" }).click();
    await page.getByRole("dialog").getByLabel("榜单名称").fill(rankingName);
    await page.getByRole("dialog").getByLabel("席位数").fill("10");
    await page.getByRole("dialog").getByRole("button", { name: "创建榜单" }).click();

    await expect(page).toHaveURL(/\/rankings\?id=/);
    await expect(page.getByRole("heading", { name: rankingName, level: 1 })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr")).toHaveCount(10);

    // 指派甲到第 1 席。
    await seatRow(page, 1).getByRole("button", { name: "指派人物" }).click();
    await page.getByRole("dialog").getByRole("option").filter({ hasText: alpha.name }).click();
    await page.getByRole("dialog").getByRole("button", { name: "确认指派" }).click();
    await expect(seatRow(page, 1)).toContainText(alpha.name);

    // 空席移动：第 1 席 → 第 5 席。
    await seatRow(page, 1).getByRole("button", { name: "调整席位" }).click();
    await page.getByRole("dialog").getByLabel("目标席位").fill("5");
    await expect(page.getByText("将移动到第 5 席（空席）")).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: "确认调整" }).click();
    await expect(seatRow(page, 5)).toContainText(alpha.name);
    await expect(seatRow(page, 1)).toContainText("虚位以待");

    // 指派乙到第 1 席后与第 5 席交换。
    await seatRow(page, 1).getByRole("button", { name: "指派人物" }).click();
    await page.getByRole("dialog").getByRole("option").filter({ hasText: beta.name }).click();
    await page.getByRole("dialog").getByRole("button", { name: "确认指派" }).click();
    await expect(seatRow(page, 1)).toContainText(beta.name);

    await seatRow(page, 1).getByRole("button", { name: "调整席位" }).click();
    await page.getByRole("dialog").getByLabel("目标席位").fill("5");
    await expect(page.getByText(new RegExp(`将与第 5 席「${alpha.name}」交换`))).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: "确认调整" }).click();
    await expect(seatRow(page, 1)).toContainText(alpha.name);
    await expect(seatRow(page, 5)).toContainText(beta.name);

    // 移出乙并撤销。
    await seatRow(page, 5).getByRole("button", { name: "移出榜单" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "移出榜单" }).click();
    await expect(seatRow(page, 5)).toContainText("虚位以待");
    await page.locator(".ui-toast").getByRole("button", { name: "撤销" }).click();
    await expect(seatRow(page, 5)).toContainText(beta.name);
  });

  test("缩容被尾席占用阻止，清空后可缩容并删除恢复", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("榜单缩容"));
    const alpha = await createCharacter(request, work.id, uniqueName("丙"));
    const ranking = await createRanking(request, work.id, uniqueName("地榜"), 10);
    const seat8 = ranking.seats.items.find((seat) => seat.seat_no === 8);
    if (!seat8) throw new Error("榜单缺少第 8 席");
    await assignRankingSeat(request, ranking.id, 8, alpha.id, seat8.version);

    await page.goto(`/rankings?id=${ranking.id}`);
    await expect(page.getByRole("heading", { name: ranking.name, level: 1 })).toBeVisible();

    await page.getByRole("button", { name: "调整席位容量" }).click();
    await page.getByRole("dialog").getByLabel("席位总数").fill("5");
    await page.getByRole("dialog").getByRole("button", { name: "保存容量" }).click();
    await expect(page.getByText("尾席仍有人物，无法缩容")).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: /第 8 席/ }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    await seatRow(page, 8).getByRole("button", { name: "移出榜单" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "移出榜单" }).click();
    await expect(seatRow(page, 8)).toContainText("虚位以待");

    await page.getByRole("button", { name: "调整席位容量" }).click();
    await page.getByRole("dialog").getByLabel("席位总数").fill("5");
    await page.getByRole("dialog").getByRole("button", { name: "保存容量" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: "席位数已调整为 5" })).toBeVisible();
    await expect(page.locator(".ui-table tbody tr")).toHaveCount(5);

    // 删除后在回收站恢复。
    await page.getByRole("button", { name: "删除榜单" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "删除榜单" }).click();
    await expect(page).toHaveURL(/\/rankings$/);

    await page.goto("/trash");
    await page.getByLabel("类型").selectOption({ label: "榜单" });
    await page.getByRole("button", { name: `恢复榜单「${ranking.name}」` }).click();
    await page.getByRole("dialog").getByRole("button", { name: "恢复" }).click();
    await expect(page.locator(".ui-toast").filter({ hasText: `已恢复榜单「${ranking.name}」` })).toBeVisible();

    const restored = await getRanking(request, ranking.id);
    expect(restored.seat_count).toBe(5);
  });

  test("名称必填、席位数 0/201 与重名校验", async ({ page, request }) => {
    const work = await createWork(request, uniqueName("榜单校验"));
    const existing = await createRanking(request, work.id, uniqueName("旧榜"), 10);

    await page.goto("/rankings");
    await page.locator(".ui-page-header").getByRole("button", { name: "新建榜单" }).click();
    const dialog = page.getByRole("dialog");

    await dialog.getByRole("button", { name: "创建榜单" }).click();
    await expect(dialog.getByText("请输入榜单名称")).toBeVisible();

    await dialog.getByLabel("榜单名称").fill(uniqueName("新榜"));
    await dialog.getByLabel("席位数").fill("0");
    await dialog.getByRole("button", { name: "创建榜单" }).click();
    await expect(dialog.getByText("席位数需在 1 到 200 之间")).toBeVisible();

    await dialog.getByLabel("席位数").fill("201");
    await dialog.getByRole("button", { name: "创建榜单" }).click();
    await expect(dialog.getByText("席位数需在 1 到 200 之间")).toBeVisible();

    await dialog.getByLabel("席位数").fill("10");
    await dialog.getByLabel("榜单名称").fill(existing.name);
    await dialog.getByRole("button", { name: "创建榜单" }).click();
    await expect(dialog.getByText("名称已存在，请换一个名称")).toBeVisible();
  });
});
