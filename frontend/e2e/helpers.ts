import type { APIRequestContext } from "@playwright/test";

/** E2E 数据统一前缀；清理时只处理本前缀的数据，避免影响本机其他作品。 */
export const E2E_PREFIX = "E2E-";

export const JSON_HEADERS = {
  "Content-Type": "application/json",
  "X-Requested-With": "XMLHttpRequest",
};

export type WorkJson = {
  id: string;
  name: string;
  version: number;
};

export type TrashItemJson = {
  root_type: string;
  root_id: string;
  name: string;
  affected_count: number;
};

export function uniqueName(label: string): string {
  return `${E2E_PREFIX}${label}-${Date.now().toString(36)}`;
}

async function ensure(res: { ok: () => boolean; status: () => number; text: () => Promise<string> }, action: string) {
  if (!res.ok()) {
    throw new Error(`${action}失败: ${res.status()} ${await res.text()}`);
  }
}

export async function createWork(
  request: APIRequestContext,
  name: string,
  description: string | null = null,
): Promise<WorkJson> {
  const res = await request.post("/api/v1/works", {
    data: { name, description },
    headers: JSON_HEADERS,
  });
  await ensure(res, `创建作品「${name}」`);
  return (await res.json()) as WorkJson;
}

export async function deleteWork(request: APIRequestContext, work: WorkJson): Promise<void> {
  const res = await request.delete(`/api/v1/works/${work.id}`, {
    data: { confirm_name: work.name },
    headers: { ...JSON_HEADERS, "If-Match": `"${work.version}"` },
  });
  await ensure(res, `删除作品「${work.name}」`);
}

export async function listWorks(request: APIRequestContext): Promise<WorkJson[]> {
  const res = await request.get("/api/v1/works?page=1&page_size=100");
  await ensure(res, "读取作品列表");
  const body = (await res.json()) as { items: WorkJson[] };
  return body.items;
}

export async function listTrash(request: APIRequestContext): Promise<TrashItemJson[]> {
  const res = await request.get("/api/v1/trash?page=1&page_size=100");
  await ensure(res, "读取回收站");
  const body = (await res.json()) as { items: TrashItemJson[] };
  return body.items;
}

/**
 * 清理本前缀的测试数据：先删活跃作品，再永久删除回收站中的同前缀项。
 * 不触碰用户的其他作品与回收站记录。
 */
export async function cleanupE2E(request: APIRequestContext): Promise<void> {
  const works = await listWorks(request);
  for (const work of works.filter((item) => item.name.startsWith(E2E_PREFIX))) {
    await deleteWork(request, work);
  }

  const trash = await listTrash(request);
  for (const item of trash.filter((entry) => entry.name.startsWith(E2E_PREFIX))) {
    const res = await request.delete(`/api/v1/trash/${item.root_type}/${item.root_id}`, {
      data: { confirm: true },
      headers: JSON_HEADERS,
    });
    await ensure(res, `永久删除「${item.name}」`);
  }
}

/** 打开顶栏作品菜单。 */
export async function openWorkMenu(page: import("@playwright/test").Page): Promise<void> {
  await page.locator(".shell__work--trigger").click();
}
