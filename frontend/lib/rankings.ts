import type { RankingTopSeat } from "./api/rankings";

/** 席位号所在分页（50 席/页）。 */
export function seatPage(seatNo: number, pageSize = 50): number {
  if (!Number.isFinite(seatNo) || seatNo < 1) return 1;
  return Math.floor((seatNo - 1) / pageSize) + 1;
}

/** 列表前三席格式化：`1 姓名 · 2 姓名 · 3 —`。 */
export function formatTopSeats(topSeats: RankingTopSeat[], limit = 3): string {
  return topSeats
    .slice(0, limit)
    .map((seat) => `${seat.seat_no} ${seat.character?.name ?? "—"}`)
    .join(" · ");
}

/** 人物侧名次标签：`封神榜·第 3 席`。 */
export function rankingRefLabel(ref: { name: string; seat_no: number }): string {
  return `${ref.name}·第 ${ref.seat_no} 席`;
}

/** 扩容/缩容差值；正数为扩容，负数为缩容。 */
export function capacityDelta(from: number, to: number): number {
  return to - from;
}
