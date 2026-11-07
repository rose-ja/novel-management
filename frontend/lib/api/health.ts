export type Health = {
  status: "ok" | "degraded";
  version: string;
  db: string;
  cache: string;
  data_dir_writable: boolean;
};

export async function fetchHealth(): Promise<Health> {
  const response = await fetch("/api/v1/health", {
    cache: "no-store",
    headers: { Accept: "application/json" },
  });

  // 503 表示数据库不可用，但响应体仍是合法健康信息。
  if (!response.ok && response.status !== 503) {
    throw new Error(`health request failed: ${response.status}`);
  }
  return (await response.json()) as Health;
}

export function describeHealth(health: Health): string {
  const parts: string[] = [];
  parts.push(health.status === "ok" ? "服务正常" : "服务降级");

  if (health.db !== "ok") {
    parts.push("数据库不可用");
  }
  if (health.cache === "degraded") {
    parts.push("缓存已旁路");
  } else if (health.cache === "disabled") {
    parts.push("缓存未启用");
  }
  if (!health.data_dir_writable) {
    parts.push("数据目录不可写");
  }

  return parts.join("，");
}
