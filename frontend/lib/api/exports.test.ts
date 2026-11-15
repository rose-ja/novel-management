import { afterEach, describe, expect, it, vi } from "vitest";

import { truncateMiddle } from "../format";
import { createExport, exportDownloadUrl, fetchExport, fetchExportStatus, listExports, resolveExportPath } from "./exports";

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

afterEach(() => {
  globalThis.fetch = originalFetch;
  vi.restoreAllMocks();
});

describe("exports api", () => {
  it("创建导出时提交 include_trash 与同源写请求头", async () => {
    const { calls } = mockFetch({ export_id: "e1", status: "queued" }, 202);

    await createExport(false);

    expect(calls[0]?.url).toBe("/api/v1/exports");
    expect(calls[0]?.init.method).toBe("POST");
    const headers = (calls[0]?.init.headers ?? {}) as Record<string, string>;
    expect(headers["X-Requested-With"]).toBe("XMLHttpRequest");
    expect(JSON.parse(String(calls[0]?.init.body))).toEqual({ include_trash: false });
  });

  it("读取状态、详情与列表使用约定路径", async () => {
    mockFetch({ active: null, last: null });
    await fetchExportStatus();
    expect(globalThis.fetch).toHaveBeenLastCalledWith("/api/v1/export/status", expect.anything());

    mockFetch({ id: "e1" });
    await fetchExport("e1");
    expect(globalThis.fetch).toHaveBeenLastCalledWith("/api/v1/exports/e1", expect.anything());

    mockFetch([]);
    await listExports();
    expect(globalThis.fetch).toHaveBeenLastCalledWith("/api/v1/exports", expect.anything());
  });

  it("下载 URL 对 ID 编码", () => {
    expect(exportDownloadUrl("a b/c")).toBe("/api/v1/exports/a%20b%2Fc/download");
  });
});

describe("export path helpers", () => {
  it("拼接数据目录时避免重复斜杠", () => {
    expect(resolveExportPath("/data/", "exports/a.zip")).toBe("/data/exports/a.zip");
    expect(resolveExportPath("/data", "exports/a.zip")).toBe("/data/exports/a.zip");
    expect(resolveExportPath("", "exports/a.zip")).toBe("exports/a.zip");
  });

  it("长路径中间省略且保留头尾", () => {
    const value = "/data/exports/novel-settings-20261115-143012.zip";
    const result = truncateMiddle(value, 20);
    expect(result.length).toBe(20);
    expect(result).toContain("…");
    expect(result.startsWith("/data/")).toBe(true);
    expect(result.endsWith(".zip")).toBe(true);
  });
});
