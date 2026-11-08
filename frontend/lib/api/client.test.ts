import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, NetworkError, fetchJson } from "./client";

const originalFetch = globalThis.fetch;

afterEach(() => {
  globalThis.fetch = originalFetch;
  vi.restoreAllMocks();
});

describe("fetchJson", () => {
  it("成功时返回解析后的 JSON", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ) as unknown as typeof fetch;

    await expect(fetchJson<{ status: string }>("/api/v1/health")).resolves.toEqual({ status: "ok" });
  });

  it("非 2xx 时按错误契约抛出 ApiError", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: { code: "NOT_FOUND", message: "页面不存在", request_id: "req-1" },
        }),
        { status: 404, headers: { "Content-Type": "application/json" } },
      ),
    ) as unknown as typeof fetch;

    const error = await fetchJson("/api/v1/unknown").catch((cause: unknown) => cause);
    expect(error).toBeInstanceOf(ApiError);
    const apiError = error as ApiError;
    expect(apiError.code).toBe("NOT_FOUND");
    expect(apiError.status).toBe(404);
    expect(apiError.requestId).toBe("req-1");
    expect(apiError.message).toBe("页面不存在");
  });

  it("响应体不可解析时回退到状态码", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response("<!doctype html>", { status: 502 }),
    ) as unknown as typeof fetch;

    const error = (await fetchJson("/api/v1/health").catch((cause: unknown) => cause)) as ApiError;
    expect(error.code).toBe("HTTP_502");
  });

  it("网络失败时抛出 NetworkError", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch")) as unknown as typeof fetch;

    await expect(fetchJson("/api/v1/health")).rejects.toBeInstanceOf(NetworkError);
  });
});
