import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
import { conflictItems, isApiCode, toUserMessage } from "./errors";
import { purgeTrashItem } from "./trash";
import { createWork, deleteWork, listWorks, updateWork } from "./works";

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

describe("works api", () => {
  it("列表请求拼接 q/page/page_size", async () => {
    const { calls } = mockFetch({ items: [], page: 2, page_size: 20, total: 0 });

    await listWorks({ q: "山海", page: 2, pageSize: 20 });

    expect(calls[0]?.url).toBe("/api/v1/works?q=%E5%B1%B1%E6%B5%B7&page=2&page_size=20");
    expect(calls[0]?.init.method).toBe("GET");
  });

  it("更新请求携带 If-Match 与 X-Requested-With，并省略未提供的字段", async () => {
    const { calls } = mockFetch({ id: "w1", name: "山海", description: null, version: 4 });

    await updateWork("w1", { name: "山海", version: 3 });

    const request = calls[0];
    expect(request?.url).toBe("/api/v1/works/w1");
    expect(request?.init.method).toBe("PATCH");
    const headers = headersOf(request!.init);
    expect(headers["If-Match"]).toBe('"3"');
    expect(headers["X-Requested-With"]).toBe("XMLHttpRequest");
    expect(headers["Content-Type"]).toBe("application/json");
    expect(JSON.parse(String(request?.init.body))).toEqual({ name: "山海" });
  });

  it("更新时可显式清空简介", async () => {
    const { calls } = mockFetch({ id: "w1", name: "山海", description: null, version: 5 });

    await updateWork("w1", { description: null, version: 4 });

    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({ description: null });
  });

  it("创建请求提交名称与简介", async () => {
    const { calls } = mockFetch({ id: "w1", name: "山海", description: null, version: 1 }, 201);

    await createWork({ name: "山海" });

    expect(calls[0]?.init.method).toBe("POST");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({ name: "山海", description: null });
  });

  it("删除请求提交 confirm_name", async () => {
    const { calls } = mockFetch({
      deleted: true,
      undo_token: "token",
      snapshot_id: "s1",
      expires_at: "2026-11-08T00:00:00Z",
    });

    await deleteWork("w1", "山海", 3);

    expect(calls[0]?.init.method).toBe("DELETE");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({ confirm_name: "山海" });
    expect(headersOf(calls[0]!.init)["If-Match"]).toBe('"3"');
  });

  it("永久删除 204 不尝试解析 JSON", async () => {
    const { calls } = mockFetch(undefined, 204);

    await expect(purgeTrashItem("work", "w1")).resolves.toBeUndefined();

    expect(calls[0]?.url).toBe("/api/v1/trash/work/w1");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({ confirm: true });
  });
});

describe("错误映射", () => {
  it("稳定错误码映射为统一文案", () => {
    const error = new ApiError({ code: "NAME_CONFLICT", message: "名称已存在", status: 409 });
    expect(toUserMessage(error)).toBe("名称已存在，请换一个名称");
    expect(isApiCode(error, "NAME_CONFLICT")).toBe(true);
  });

  it("未覆盖的错误码回退到后端文案", () => {
    const error = new ApiError({ code: "SOMETHING_NEW", message: "后端说明", status: 400 });
    expect(toUserMessage(error)).toBe("后端说明");
  });

  it("RESTORE_CONFLICT 提取 details.items", () => {
    const error = new ApiError({
      code: "RESTORE_CONFLICT",
      message: "恢复目标与现有记录重名",
      status: 422,
      details: { items: [{ id: "c1", name: "白无咎" }, { bad: true }] },
    });

    expect(conflictItems(error)).toEqual([{ id: "c1", name: "白无咎" }]);
  });
});
