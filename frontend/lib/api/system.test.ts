import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
import {
  createCategory,
  deleteCategory,
  deleteNode,
  listCategories,
  listNodes,
  reorderNodes,
  updateCategory,
  updateNode,
} from "./system";

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

describe("system api", () => {
  it("类别列表拼接 work_id 与 include_disabled", async () => {
    const { calls } = mockFetch({ items: [] });

    await listCategories("w1");

    expect(calls[0]?.url).toBe("/api/v1/system-categories?work_id=w1&include_disabled=true");
    expect(calls[0]?.init.method).toBe("GET");
  });

  it("可只拉取启用类别", async () => {
    const { calls } = mockFetch({ items: [] });

    await listCategories("w1", false);

    expect(calls[0]?.url).toBe("/api/v1/system-categories?work_id=w1&include_disabled=false");
  });

  it("创建类别提交 work_id、name 与 category_type", async () => {
    const { calls } = mockFetch(
      { id: "c1", work_id: "w1", name: "纪元", category_type: "era", version: 1 },
      201,
    );

    await createCategory({ workID: "w1", name: "纪元", categoryType: "era" });

    expect(calls[0]?.init.method).toBe("POST");
    expect(headersOf(calls[0]!.init)["X-Requested-With"]).toBe("XMLHttpRequest");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      work_id: "w1",
      name: "纪元",
      category_type: "era",
    });
  });

  it("更新类别携带 If-Match，且 body 只含提供的字段", async () => {
    const { calls } = mockFetch({ id: "c1", work_id: "w1", name: "境界", version: 3 });

    await updateCategory("c1", { isEnabled: false, version: 2 });

    const request = calls[0];
    expect(request?.url).toBe("/api/v1/system-categories/c1");
    expect(request?.init.method).toBe("PATCH");
    expect(headersOf(request!.init)["If-Match"]).toBe('"2"');
    expect(JSON.parse(String(request?.init.body))).toEqual({ is_enabled: false });
  });

  it("删除类别携带 If-Match 并返回删除三件套", async () => {
    const result = {
      deleted: true,
      undo_token: "token",
      affected_nodes: 2,
      expires_at: "2026-11-08T00:00:00Z",
    };
    const { calls } = mockFetch(result);

    await expect(deleteCategory("c1", 4)).resolves.toEqual(result);

    expect(calls[0]?.url).toBe("/api/v1/system-categories/c1");
    expect(calls[0]?.init.method).toBe("DELETE");
    expect(headersOf(calls[0]!.init)["If-Match"]).toBe('"4"');
  });

  it("删除节点携带 If-Match 并返回删除结构", async () => {
    const result = {
      deleted: true,
      undo_token: "token",
      expires_at: "2026-11-08T00:00:00Z",
    };
    const { calls } = mockFetch(result);

    await expect(deleteNode("n1", 5)).resolves.toEqual(result);

    expect(calls[0]?.url).toBe("/api/v1/system-nodes/n1");
    expect(calls[0]?.init.method).toBe("DELETE");
    expect(headersOf(calls[0]!.init)["If-Match"]).toBe('"5"');
  });

  it("节点列表拼接 flat=true 与 review_status=pending", async () => {
    const { calls } = mockFetch({ items: [] });

    await listNodes("c1", { flat: true, reviewStatus: "pending" });

    expect(calls[0]?.url).toBe("/api/v1/system-categories/c1/nodes?flat=true&review_status=pending");
    expect(calls[0]?.init.method).toBe("GET");
  });

  it("节点列表默认不拼接查询参数", async () => {
    const { calls } = mockFetch({ items: [] });

    await listNodes("c1");

    expect(calls[0]?.url).toBe("/api/v1/system-categories/c1/nodes");
  });

  it("更新节点显式发送 era_start_text: null", async () => {
    const { calls } = mockFetch({ id: "n1", name: "炼气", version: 3 });

    await updateNode("n1", { eraStartText: null, version: 2 });

    const request = calls[0];
    expect(request?.init.method).toBe("PATCH");
    expect(headersOf(request!.init)["If-Match"]).toBe('"2"');
    expect(JSON.parse(String(request?.init.body))).toEqual({ era_start_text: null });
  });

  it("重排节点提交 ordered_ids，204 时不解析响应体", async () => {
    const { calls } = mockFetch(undefined, 204);

    await expect(
      reorderNodes({ categoryID: "c1", orderedIDs: ["n2", "n1"] }),
    ).resolves.toBeUndefined();

    expect(calls[0]?.url).toBe("/api/v1/system-nodes/reorder");
    expect(calls[0]?.init.method).toBe("POST");
    expect(headersOf(calls[0]!.init)["X-Requested-With"]).toBe("XMLHttpRequest");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({
      category_id: "c1",
      ordered_ids: ["n2", "n1"],
    });
  });

  it("错误码与 details 原样透传", async () => {
    const details = { items: [{ id: "n1", name: "炼气" }] };
    mockFetch(
      {
        error: {
          code: "REFERENCE_CONFLICT",
          message: "该节点仍被 1 处引用",
          details,
        },
      },
      409,
    );

    const error = await deleteNode("n1", 2).catch((cause: unknown) => cause);

    expect(error).toBeInstanceOf(ApiError);
    const apiError = error as ApiError;
    expect(apiError.code).toBe("REFERENCE_CONFLICT");
    expect(apiError.status).toBe(409);
    expect(apiError.message).toBe("该节点仍被 1 处引用");
    expect(apiError.details).toEqual(details);
  });
});
