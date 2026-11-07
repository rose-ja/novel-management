import { describe, expect, it } from "vitest";

import { describeHealth, type Health } from "./health";

const base: Health = {
  status: "ok",
  version: "0.1.0",
  db: "ok",
  cache: "disabled",
  data_dir_writable: true,
};

describe("describeHealth", () => {
  it("健康时只报告服务正常", () => {
    expect(describeHealth(base)).toBe("服务正常，缓存未启用");
  });

  it("数据库不可用时报告降级", () => {
    expect(describeHealth({ ...base, status: "degraded", db: "unavailable" })).toBe(
      "服务降级，数据库不可用，缓存未启用",
    );
  });

  it("缓存不可用时追加旁路说明", () => {
    expect(describeHealth({ ...base, cache: "degraded" })).toBe("服务正常，缓存已旁路");
  });

  it("数据目录不可写时明确提示", () => {
    expect(describeHealth({ ...base, data_dir_writable: false })).toBe(
      "服务正常，缓存未启用，数据目录不可写",
    );
  });
});
