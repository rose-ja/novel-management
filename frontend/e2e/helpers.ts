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

export type SystemCategoryJson = {
  id: string;
  work_id: string;
  name: string;
  category_type: "normal" | "era";
  is_builtin: boolean;
  is_enabled: boolean;
  sort_order: number;
  node_count: number;
  reference_count: number;
  version: number;
};

export type SystemNodeJson = {
  id: string;
  work_id: string;
  category_id: string;
  parent_id: string | null;
  name: string;
  sort_order: number;
  era_start_text: string | null;
  era_end_text: string | null;
  review_status: "confirmed" | "pending";
  version: number;
  reference_count: number;
  children?: SystemNodeJson[];
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

export async function listTrash(
  request: APIRequestContext,
  rootType?: "work" | "system_category" | "system_node" | "character" | "faction",
): Promise<TrashItemJson[]> {
  const search = new URLSearchParams({ page: "1", page_size: "100" });
  if (rootType) search.set("root_type", rootType);
  const res = await request.get(`/api/v1/trash?${search.toString()}`);
  await ensure(res, "读取回收站");
  const body = (await res.json()) as { items: TrashItemJson[] };
  return body.items;
}

export async function restoreTrashItem(
  request: APIRequestContext,
  rootType: TrashItemJson["root_type"],
  rootID: string,
  body: Record<string, unknown> = {},
): Promise<{ name: string; restored_count: number }> {
  const res = await request.post(`/api/v1/trash/${rootType}/${rootID}/restore`, {
    data: body,
    headers: JSON_HEADERS,
  });
  await ensure(res, `恢复${rootType}`);
  return (await res.json()) as { name: string; restored_count: number };
}

export async function purgeTrashItem(
  request: APIRequestContext,
  rootType: TrashItemJson["root_type"],
  rootID: string,
): Promise<void> {
  const res = await request.delete(`/api/v1/trash/${rootType}/${rootID}`, {
    data: { confirm: true },
    headers: JSON_HEADERS,
  });
  await ensure(res, `永久删除${rootType}`);
}

export async function listSystemCategories(
  request: APIRequestContext,
  workID: string,
): Promise<SystemCategoryJson[]> {
  const res = await request.get(`/api/v1/system-categories?work_id=${encodeURIComponent(workID)}`);
  await ensure(res, "读取体系类别");
  const body = (await res.json()) as { items: SystemCategoryJson[] };
  return body.items;
}

export async function createSystemCategory(
  request: APIRequestContext,
  workID: string,
  name: string,
  categoryType: "normal" | "era" = "normal",
): Promise<SystemCategoryJson> {
  const res = await request.post("/api/v1/system-categories", {
    data: { work_id: workID, name, category_type: categoryType },
    headers: JSON_HEADERS,
  });
  await ensure(res, `创建体系类别「${name}」`);
  return (await res.json()) as SystemCategoryJson;
}

export async function deleteSystemCategory(
  request: APIRequestContext,
  categoryID: string,
  version: number,
): Promise<void> {
  const res = await request.delete(`/api/v1/system-categories/${categoryID}`, {
    headers: { ...JSON_HEADERS, "If-Match": `"${version}"` },
  });
  await ensure(res, "删除体系类别");
}

export async function listSystemNodes(
  request: APIRequestContext,
  categoryID: string,
): Promise<SystemNodeJson[]> {
  const res = await request.get(`/api/v1/system-categories/${categoryID}/nodes`);
  await ensure(res, "读取体系节点");
  const body = (await res.json()) as { items: SystemNodeJson[] };
  return body.items;
}

export async function createSystemNode(
  request: APIRequestContext,
  categoryID: string,
  name: string,
  parentID?: string,
): Promise<SystemNodeJson> {
  const data: Record<string, unknown> = { category_id: categoryID, name };
  if (parentID) data.parent_id = parentID;
  const res = await request.post("/api/v1/system-nodes", { data, headers: JSON_HEADERS });
  await ensure(res, `创建体系节点「${name}」`);
  return (await res.json()) as SystemNodeJson;
}

export async function deleteSystemNode(
  request: APIRequestContext,
  nodeID: string,
  version: number,
): Promise<{ undo_token: string }> {
  const res = await request.delete(`/api/v1/system-nodes/${nodeID}`, {
    headers: { ...JSON_HEADERS, "If-Match": `"${version}"` },
  });
  await ensure(res, "删除体系节点");
  return (await res.json()) as { undo_token: string };
}

export async function undoTrash(request: APIRequestContext, undoToken: string): Promise<void> {
  const res = await request.post("/api/v1/trash/undo", {
    data: { undo_token: undoToken },
    headers: JSON_HEADERS,
  });
  await ensure(res, "撤销删除");
}

export type CharacterNameJson = {
  kind: "alias" | "title";
  value: string;
};

export type CharacterJson = {
  id: string;
  work_id: string;
  name: string;
  names: CharacterNameJson[];
  description: string | null;
  system_refs: {
    category_id: string;
    category_name: string;
    node: { id: string; name: string; parent: { id: string; name: string } | null };
  }[];
  review_status: "confirmed" | "pending";
  version: number;
};

export type CreateCharacterOptions = {
  names?: CharacterNameJson[];
  description?: string | null;
  factionID?: string | null;
  systemRefs?: { category_id: string; node_id: string }[];
};

export async function createCharacter(
  request: APIRequestContext,
  workID: string,
  name: string,
  options: CreateCharacterOptions = {},
): Promise<CharacterJson> {
  const res = await request.post("/api/v1/characters", {
    data: {
      work_id: workID,
      name,
      names: options.names ?? [],
      description: options.description ?? null,
      faction_id: options.factionID ?? null,
      system_refs: options.systemRefs ?? [],
      location_links: [],
    },
    headers: JSON_HEADERS,
  });
  await ensure(res, `创建人物「${name}」`);
  return (await res.json()) as CharacterJson;
}

export async function updateCharacter(
  request: APIRequestContext,
  character: Pick<CharacterJson, "id" | "version">,
  data: Record<string, unknown>,
): Promise<CharacterJson> {
  const res = await request.patch(`/api/v1/characters/${character.id}`, {
    data,
    headers: { ...JSON_HEADERS, "If-Match": `"${character.version}"` },
  });
  await ensure(res, `更新人物「${character.id}」`);
  return (await res.json()) as CharacterJson;
}

export async function deleteCharacter(
  request: APIRequestContext,
  character: Pick<CharacterJson, "id" | "version">,
): Promise<{ undo_token: string }> {
  const res = await request.delete(`/api/v1/characters/${character.id}`, {
    headers: { ...JSON_HEADERS, "If-Match": `"${character.version}"` },
  });
  await ensure(res, `删除人物「${character.id}」`);
  return (await res.json()) as { undo_token: string };
}

export async function listCharacters(
  request: APIRequestContext,
  workID: string,
  params: { q?: string; page?: number; pageSize?: number } = {},
): Promise<CharacterJson[]> {
  const search = new URLSearchParams({ work_id: workID, page: String(params.page ?? 1) });
  search.set("page_size", String(params.pageSize ?? 100));
  if (params.q) search.set("q", params.q);
  const res = await request.get(`/api/v1/characters?${search.toString()}`);
  await ensure(res, "读取人物列表");
  const body = (await res.json()) as { items: CharacterJson[] };
  return body.items;
}

export type FactionJson = {
  id: string;
  work_id: string;
  name: string;
  parent: { id: string; name: string } | null;
  leader: { id: string; name: string } | null;
  description: string | null;
  review_status: "confirmed" | "pending";
  children_count: number;
  member_count: number;
  version: number;
  relations?: {
    children: { id: string; name: string }[];
    chronicle_count: number;
    subtree_member_count: number;
  };
};

export type CreateFactionOptions = {
  parentID?: string | null;
  leaderID?: string | null;
  description?: string | null;
};

export async function createFaction(
  request: APIRequestContext,
  workID: string,
  name: string,
  options: CreateFactionOptions = {},
): Promise<FactionJson> {
  const res = await request.post("/api/v1/factions", {
    data: {
      work_id: workID,
      name,
      parent_id: options.parentID ?? null,
      leader_character_id: options.leaderID ?? null,
      description: options.description ?? null,
    },
    headers: JSON_HEADERS,
  });
  await ensure(res, `创建势力「${name}」`);
  return (await res.json()) as FactionJson;
}

export async function listFactions(
  request: APIRequestContext,
  workID: string,
): Promise<FactionJson[]> {
  const res = await request.get(`/api/v1/factions?work_id=${encodeURIComponent(workID)}`);
  await ensure(res, "读取势力列表");
  const body = (await res.json()) as { items: FactionJson[] };
  return body.items;
}

export async function deleteFaction(
  request: APIRequestContext,
  faction: Pick<FactionJson, "id" | "version">,
): Promise<{ undo_token: string }> {
  const res = await request.delete(`/api/v1/factions/${faction.id}`, {
    headers: { ...JSON_HEADERS, "If-Match": `"${faction.version}"` },
  });
  await ensure(res, `删除势力「${faction.id}」`);
  return (await res.json()) as { undo_token: string };
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
    await purgeTrashItem(request, item.root_type, item.root_id);
  }
}

/** 打开顶栏作品菜单。 */
export async function openWorkMenu(page: import("@playwright/test").Page): Promise<void> {
  await page.locator(".shell__work--trigger").click();
}
