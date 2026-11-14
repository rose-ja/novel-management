import { describe, expect, it } from "vitest";

import { capacityDelta, formatTopSeats, rankingRefLabel, seatPage } from "./rankings";

describe("rankings helpers", () => {
  it("按 50 席/页换算跳席页码", () => {
    expect(seatPage(1)).toBe(1);
    expect(seatPage(50)).toBe(1);
    expect(seatPage(51)).toBe(2);
    expect(seatPage(100, 100)).toBe(1);
    expect(seatPage(0)).toBe(1);
  });

  it("前三席格式化含空席占位", () => {
    expect(
      formatTopSeats([
        { seat_no: 1, character: { id: "c1", name: "甲" } },
        { seat_no: 2, character: { id: "c2", name: "乙" } },
        { seat_no: 3, character: null },
      ]),
    ).toBe("1 甲 · 2 乙 · 3 —");
    expect(formatTopSeats([])).toBe("1 — · 2 — · 3 —");
  });

  it("人物名次标签与容量差值", () => {
    expect(rankingRefLabel({ name: "封神榜", seat_no: 3 })).toBe("封神榜·第 3 席");
    expect(capacityDelta(100, 120)).toBe(20);
    expect(capacityDelta(100, 80)).toBe(-20);
  });
});
