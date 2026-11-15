import { afterEach, describe, expect, it, vi } from "vitest";

import { searchEntities } from "./search";

const originalFetch = globalThis.fetch;

type Captured = { url: string; init: RequestInit };

function mockFetch(payload: unknown, status = 200): { calls: Captured[] } {
  const calls: Captured[] = [];
  globalThis.fetch = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} });
    return new Response(JSON.stringify(payload), {
      status,
      headers: { "Content-Type": "application/json" },
    });
  }) as unknown as typeof fetch;
  return { calls };
}

afterEach(() => {
  globalThis.fetch = originalFetch;
  vi.restoreAllMocks();
});

describe("search api", () => {
  it("拼接关键词、实体类型与人物筛选", async () => {
    const { calls } = mockFetch({ groups: [], total: 0, page: 1, page_size: 20 });

    await searchEntities({
      workID: "w1",
      q: "山海 帝君",
      entityTypes: ["character"],
      factionID: "f1",
      factionScope: "subtree",
      systemNodeID: "n1",
      locationID: "l1",
      rankingID: "r1",
      page: 2,
    });

    const url = calls[0]?.url ?? "";
    expect(url).toContain("work_id=w1");
    expect(url).toContain("q=%E5%B1%B1%E6%B5%B7+%E5%B8%9D%E5%90%9B");
    expect(url).toContain("entity_types=character");
    expect(url).toContain("faction_id=f1");
    expect(url).toContain("faction_scope=subtree");
    expect(url).toContain("system_node_id=n1");
    expect(url).toContain("location_id=l1");
    expect(url).toContain("ranking_id=r1");
    expect(url).toContain("page=2");
  });

  it("省略空筛选与第 1 页", async () => {
    const { calls } = mockFetch({ groups: [], total: 0, page: 1, page_size: 20 });

    await searchEntities({ workID: "w1", q: "小圣" });

    const url = calls[0]?.url ?? "";
    expect(url).not.toContain("entity_types");
    expect(url).not.toContain("faction_id");
    expect(url).not.toContain("page=");
  });

  it("透传 AbortSignal", async () => {
    const { calls } = mockFetch({ groups: [], total: 0, page: 1, page_size: 20 });
    const controller = new AbortController();

    await searchEntities({ workID: "w1", q: "x", signal: controller.signal });

    expect(calls[0]?.init.signal).toBe(controller.signal);
  });
});
