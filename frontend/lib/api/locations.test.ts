import { afterEach, describe, expect, it, vi } from "vitest";

import {
  addLocationFaction,
  createLocation,
  deleteLocation,
  listLocations,
  removeLocationFaction,
  updateLocation,
  updateLocationFactionNote,
} from "./locations";

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

describe("locations api", () => {
  it("列表拼接搜索、类型与待校对筛选", async () => {
    const { calls } = mockFetch({ items: [], total: 0 });

    await listLocations({
      workID: "w1",
      q: "东荒",
      typeText: "星域",
      reviewStatus: "pending",
    });

    expect(calls[0]?.init.method).toBe("GET");
    expect(calls[0]?.url).toBe(
      "/api/v1/locations?work_id=w1&q=%E4%B8%9C%E8%8D%92&type_text=%E6%98%9F%E5%9F%9F&review_status=pending",
    );
  });

  it("列表只传 work_id 时不带空参数", async () => {
    const { calls } = mockFetch({ items: [], total: 0 });
    await listLocations({ workID: "w1" });
    expect(calls[0]?.url).toBe("/api/v1/locations?work_id=w1");
  });

  it("创建时未填写的父级、类型与简介发送 null", async () => {
    const { calls } = mockFetch({ id: "l1" }, 201);
    await createLocation({ workID: "w1", name: "东荒" });
    expect(calls[0]?.init.method).toBe("POST");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      work_id: "w1",
      name: "东荒",
      parent_id: null,
      type_text: null,
      description: null,
    });
  });

  it("更新使用 If-Match，并原样发送显式 null 以清空字段", async () => {
    const { calls } = mockFetch({ id: "l1", version: 3 });

    await updateLocation("l1", {
      name: "东荒",
      parentID: null,
      typeText: null,
      description: "简介",
      version: 2,
    });

    expect(calls[0]?.init.method).toBe("PATCH");
    expect(headersOf(calls[0]?.init)["If-Match"]).toBe('"2"');
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      name: "东荒",
      parent_id: null,
      type_text: null,
      description: "简介",
    });
  });

  it("删除携带 If-Match 并返回撤销令牌", async () => {
    const { calls } = mockFetch({ deleted: true, id: "l1", undo_token: "u1" });

    const result = await deleteLocation("l1", 4);

    expect(calls[0]?.init.method).toBe("DELETE");
    expect(headersOf(calls[0]?.init)["If-Match"]).toBe('"4"');
    expect(result.undo_token).toBe("u1");
  });

  it("地点-势力关系新增、备注更新与移除使用对应方法", async () => {
    const { calls } = mockFetch({ faction_id: "f1" }, 201);

    await addLocationFaction("l1", { factionID: "f1", note: "驻地" });
    expect(calls[0]?.url).toBe("/api/v1/locations/l1/factions");
    expect(calls[0]?.init.method).toBe("POST");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({ faction_id: "f1", note: "驻地" });

    await updateLocationFactionNote("l1", "f1", null);
    expect(calls[1]?.url).toBe("/api/v1/locations/l1/factions/f1");
    expect(calls[1]?.init.method).toBe("PATCH");
    expect(JSON.parse(String(calls[1]?.init.body))).toEqual({ note: null });

    const removed: Captured[] = [];
    globalThis.fetch = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
      removed.push({ url: String(url), init: init ?? {} });
      return new Response(null, { status: 204 });
    }) as unknown as typeof fetch;
    await removeLocationFaction("l1", "f1");
    expect(removed[0]?.url).toBe("/api/v1/locations/l1/factions/f1");
    expect(removed[0]?.init.method).toBe("DELETE");
  });
});
