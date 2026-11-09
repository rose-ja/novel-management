import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
import {
  createCharacter,
  deleteCharacter,
  getCharacter,
  listCharacters,
  updateCharacter,
} from "./characters";

const originalFetch = globalThis.fetch;

type Captured = { url: string; init: RequestInit };

function mockFetch(payload: unknown, status = 200): { calls: Captured[] } {
  const calls: Captured[] = [];
  globalThis.fetch = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} });
    const body = payload === undefined ? null : JSON.stringify(payload);
    return new Response(body, {
      status,
      headers: { "Content-Type": "application/json" },
    });
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

describe("characters api", () => {
  it("列表拼接全部筛选、排序与分页参数", async () => {
    const { calls } = mockFetch({ items: [], page: 2, page_size: 50, total: 0 });

    await listCharacters({
      workID: "w1",
      q: "陆沉",
      systemNodeID: "n1",
      reviewStatus: "pending",
      sort: "name",
      page: 2,
      pageSize: 50,
    });

    expect(calls[0]?.init.method).toBe("GET");
    expect(calls[0]?.url).toBe(
      "/api/v1/characters?work_id=w1&q=%E9%99%86%E6%B2%89&system_node_id=n1&review_status=pending&sort=name&page=2&page_size=50",
    );
  });

  it("列表只传 work_id 时不带空参数", async () => {
    const { calls } = mockFetch({ items: [], page: 1, page_size: 20, total: 0 });

    await listCharacters({ workID: "w1" });

    expect(calls[0]?.url).toBe("/api/v1/characters?work_id=w1");
  });

  it("详情按 ID 读取", async () => {
    const { calls } = mockFetch({ id: "c1", name: "陆沉", version: 2 });

    await getCharacter("c1");

    expect(calls[0]?.url).toBe("/api/v1/characters/c1");
    expect(calls[0]?.init.method).toBe("GET");
  });

  it("创建提交名称、别名、简介与体系引用，省略项补空值", async () => {
    const { calls } = mockFetch({ id: "c1", name: "陆沉", version: 1 }, 201);

    await createCharacter({
      workID: "w1",
      name: "陆沉",
      names: [{ kind: "alias", value: "沉舟" }],
      description: "   ",
      systemRefs: [{ category_id: "cat1", node_id: "n1" }],
    });

    expect(calls[0]?.init.method).toBe("POST");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      work_id: "w1",
      name: "陆沉",
      names: [{ kind: "alias", value: "沉舟" }],
      description: "   ",
      faction_id: null,
      system_refs: [{ category_id: "cat1", node_id: "n1" }],
      location_links: [],
    });
  });

  it("创建时未提供的可选字段提交 null 或空数组", async () => {
    const { calls } = mockFetch({ id: "c1", name: "陆沉", version: 1 }, 201);

    await createCharacter({ workID: "w1", name: "陆沉" });

    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      work_id: "w1",
      name: "陆沉",
      names: [],
      description: null,
      faction_id: null,
      system_refs: [],
      location_links: [],
    });
  });

  it("更新携带 If-Match 且只提交提供的字段", async () => {
    const { calls } = mockFetch({ id: "c1", name: "陆沉", version: 4 });

    await updateCharacter("c1", { name: "陆沉", version: 3 });

    const request = calls[0];
    expect(request?.url).toBe("/api/v1/characters/c1");
    expect(request?.init.method).toBe("PATCH");
    const headers = headersOf(request!.init);
    expect(headers["If-Match"]).toBe('"3"');
    expect(headers["X-Requested-With"]).toBe("XMLHttpRequest");
    expect(JSON.parse(String(request?.init.body))).toEqual({ name: "陆沉" });
  });

  it("更新时可显式清空简介与势力", async () => {
    const { calls } = mockFetch({ id: "c1", name: "陆沉", version: 5 });

    await updateCharacter("c1", { description: null, factionID: null, version: 4 });

    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      description: null,
      faction_id: null,
    });
  });

  it("更新时传空数组表示整组清空别名与体系引用", async () => {
    const { calls } = mockFetch({ id: "c1", name: "陆沉", version: 6 });

    await updateCharacter("c1", { names: [], systemRefs: [], version: 5 });

    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      names: [],
      system_refs: [],
    });
  });

  it("删除携带 If-Match 并解析撤销令牌", async () => {
    const { calls } = mockFetch({
      id: "c1",
      deleted_at: "2026-11-09T00:00:00Z",
      undo_token: "token",
      expires_at: "2026-11-09T00:00:08Z",
    });

    const result = await deleteCharacter("c1", 7);

    expect(calls[0]?.init.method).toBe("DELETE");
    expect(headersOf(calls[0]!.init)["If-Match"]).toBe('"7"');
    expect(calls[0]?.init.body).toBeUndefined();
    expect(result.undo_token).toBe("token");
    expect(result.expires_at).toBe("2026-11-09T00:00:08Z");
  });
});

describe("characters 错误码解析", () => {
  it("重名 409 保留 NAME_CONFLICT 与 details.items", async () => {
    mockFetch(
      {
        error: {
          code: "NAME_CONFLICT",
          message: "同一作品下已存在同名人物",
          details: { items: [{ id: "c9", name: "陆沉" }] },
        },
      },
      409,
    );

    await expect(createCharacter({ workID: "w1", name: "陆沉" })).rejects.toMatchObject({
      code: "NAME_CONFLICT",
      status: 409,
      details: { items: [{ id: "c9", name: "陆沉" }] },
    });
  });

  it("版本冲突 409 暴露 current_version", async () => {
    mockFetch(
      {
        error: {
          code: "VERSION_CONFLICT",
          message: "记录已被修改",
          details: { current_version: 5 },
        },
      },
      409,
    );

    const error = await updateCharacter("c1", { name: "陆沉", version: 4 }).catch(
      (caught: unknown) => caught,
    );
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).details).toEqual({ current_version: 5 });
  });

  it("跨作品引用 422 暴露 details.field", async () => {
    mockFetch(
      {
        error: {
          code: "CROSS_WORK_REFERENCE",
          message: "只能引用同一作品下的体系节点",
          details: { field: "system_refs" },
        },
      },
      422,
    );

    await expect(
      updateCharacter("c1", {
        systemRefs: [{ category_id: "cat1", node_id: "n1" }],
        version: 1,
      }),
    ).rejects.toMatchObject({
      code: "CROSS_WORK_REFERENCE",
      status: 422,
      details: { field: "system_refs" },
    });
  });

  it("缺少 If-Match 428 解析为 PRECONDITION_REQUIRED", async () => {
    mockFetch(
      { error: { code: "PRECONDITION_REQUIRED", message: "缺少 If-Match 请求头" } },
      428,
    );

    await expect(updateCharacter("c1", { name: "陆沉", version: 1 })).rejects.toMatchObject({
      code: "PRECONDITION_REQUIRED",
      status: 428,
    });
  });
});
