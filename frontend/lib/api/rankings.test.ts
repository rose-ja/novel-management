import { afterEach, describe, expect, it, vi } from "vitest";

import { listRankings, putSeat, updateRanking } from "./rankings";

const originalFetch = globalThis.fetch;

type Captured = { url: string; init: RequestInit };

function mockFetch(payload: unknown, status = 200): { calls: Captured[] } {
  const calls: Captured[] = [];
  globalThis.fetch = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} });
    const body = payload === undefined ? null : JSON.stringify(payload);
    return new Response(body, { status, headers: { "Content-Type": "application/json" } });
  }) as unknown as typeof fetch;
  return { calls };
}

function headersOf(init: RequestInit): Record<string, string> {
  return (init.headers ?? {}) as Record<string, string>;
}

afterEach(() => {
  globalThis.fetch = originalFetch;
  vi.restoreAllMocks();
});

describe("rankings api", () => {
  it("列表拼接搜索、筛选、排序与分页", async () => {
    const { calls } = mockFetch({ items: [], total: 0, page: 2, page_size: 20 });

    await listRankings({
      workID: "w1",
      q: "天榜",
      reviewStatus: "pending",
      sort: "name:asc",
      page: 2,
      pageSize: 20,
    });

    expect(calls[0]?.url).toBe(
      "/api/v1/rankings?work_id=w1&q=%E5%A4%A9%E6%A6%9C&review_status=pending&sort=name%3Aasc&page=2&page_size=20",
    );
  });

  it("更新榜单携带 If-Match 版本", async () => {
    const { calls } = mockFetch({ id: "r1" });

    await updateRanking("r1", { seatCount: 120, version: 3 });

    expect(calls[0]?.init.method).toBe("PATCH");
    expect(headersOf(calls[0]!.init)["If-Match"]).toBe('"3"');
    expect(calls[0]?.init.body).toBe(JSON.stringify({ seat_count: 120 }));
  });

  it("席位写入省略人物时不清空，显式 null 时发送 null", async () => {
    const { calls } = mockFetch({ seat_no: 1, version: 2 });

    await putSeat("r1", 1, { note: "并列", version: 1 });
    expect(calls[0]?.init.body).toBe(JSON.stringify({ note: "并列" }));
    expect(calls[0]?.url).toBe("/api/v1/rankings/r1/seats/1");
    expect(headersOf(calls[0]!.init)["If-Match"]).toBe('"1"');

    await putSeat("r1", 1, { characterID: null, version: 2 });
    expect(calls[1]?.init.body).toBe(JSON.stringify({ character_id: null }));
  });
});
