import { describe, expect, it } from "vitest";

import { maxImageBytes, validateImageFile } from "./media";

function fileOf(size: number, type: string): File {
  return new File([new Uint8Array(size)], "test", { type });
}

describe("validateImageFile", () => {
  it("接受 jpg/png/webp 且在 10MB 内", () => {
    expect(validateImageFile(fileOf(1024, "image/jpeg"))).toBeNull();
    expect(validateImageFile(fileOf(1024, "image/png"))).toBeNull();
    expect(validateImageFile(fileOf(1024, "image/webp"))).toBeNull();
    expect(validateImageFile(fileOf(maxImageBytes, "image/png"))).toBeNull();
  });

  it("拒绝不支持的 MIME", () => {
    expect(validateImageFile(fileOf(1024, "image/gif"))).toBe("仅支持 JPG、PNG 或 WebP 图片");
    expect(validateImageFile(fileOf(1024, "text/plain"))).toBe("仅支持 JPG、PNG 或 WebP 图片");
  });

  it("拒绝超过 10MB 的文件", () => {
    expect(validateImageFile(fileOf(maxImageBytes + 1, "image/png"))).toBe("图片不能超过 10MB");
  });
});
