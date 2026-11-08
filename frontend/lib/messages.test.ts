import { describe, expect, it } from "vitest";

import { messages } from "./messages";

describe("messages", () => {
  it("同一语义只有一种说法", () => {
    expect(messages.loadFailed).toBe("加载失败，请重试");
    expect(messages.networkError).toBe("网络异常，请检查连接");
    expect(messages.retry).toBe("重试");
  });

  it("空状态文案说明原因与下一步", () => {
    expect(messages.noWorkTitle).toBe("还没有作品");
    expect(messages.noWorkDescription).toContain("当前作品");
  });
});
