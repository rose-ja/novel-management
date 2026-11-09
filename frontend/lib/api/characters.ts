import { requestJson } from "./client";
import type { CategoryType, ReviewStatus } from "./system";

export type CharacterNameKind = "alias" | "title";

/** 别名或称号；与主名称分开提交。 */
export type CharacterName = {
  kind: CharacterNameKind;
  value: string;
};

/** 势力摘要占位；CHG-006 前恒为 null。 */
export type FactionBrief = {
  id: string;
  name: string;
};

export type CharacterNodeRef = {
  id: string;
  name: string;
  parent: { id: string; name: string } | null;
};

/** 详情页按类别分组的体系引用，每类别至多一条。 */
export type CharacterSystemRef = {
  category_id: string;
  category_name: string;
  category_type: CategoryType;
  node: CharacterNodeRef;
};

/** 地点关联占位；CHG-007 前恒为空数组。 */
export type CharacterLocationLink = {
  location_id: string;
  name: string;
  note: string | null;
};

/** 主图占位；CHG-005 前恒为 null。 */
export type CharacterImage = {
  asset_id: string;
  url: string;
  width: number;
  height: number;
};

export type CharacterSummary = {
  id: string;
  name: string;
  names: CharacterName[];
  faction: FactionBrief | null;
  /** 形如「境界 · 父层级 · 子层级」；无引用时为 null。 */
  system_summary: string | null;
  image_thumb_url: string | null;
  review_status: ReviewStatus;
  reviewed_at: string | null;
  updated_at: string;
  version: number;
};

export type CharacterDetail = {
  id: string;
  work_id: string;
  name: string;
  names: CharacterName[];
  description: string | null;
  faction: FactionBrief | null;
  system_refs: CharacterSystemRef[];
  location_links: CharacterLocationLink[];
  image: CharacterImage | null;
  review_status: ReviewStatus;
  reviewed_at: string | null;
  reviewed_by: string | null;
  created_at: string;
  updated_at: string;
  version: number;
  deleted_at: string | null;
};

export type CharacterPage = {
  items: CharacterSummary[];
  page: number;
  page_size: number;
  total: number;
};

export type CharacterSort = "updated_at" | "-updated_at" | "name" | "-name";

export type ListCharactersOptions = {
  workID: string;
  q?: string;
  systemNodeID?: string;
  reviewStatus?: ReviewStatus;
  sort?: CharacterSort;
  page?: number;
  pageSize?: number;
};

export type CharacterNameInput = {
  kind: CharacterNameKind;
  value: string;
};

export type CharacterSystemRefInput = {
  category_id: string;
  node_id: string;
};

export type CharacterLocationLinkInput = {
  location_id: string;
  note?: string | null;
};

export type CreateCharacterInput = {
  workID: string;
  name: string;
  names?: CharacterNameInput[];
  description?: string | null;
  /** 势力暂未开放，仅接受 null。 */
  factionID?: string | null;
  systemRefs?: CharacterSystemRefInput[];
  locationLinks?: CharacterLocationLinkInput[];
};

export type UpdateCharacterInput = {
  name?: string;
  /** 省略表示保持不变；传空数组表示清空别名/称号。 */
  names?: CharacterNameInput[];
  /** 传 null 表示清空简介；省略表示保持不变。 */
  description?: string | null;
  /** 传 null 表示清空势力；省略表示保持不变。 */
  factionID?: string | null;
  /** 省略表示保持不变；传空数组表示清空体系引用。 */
  systemRefs?: CharacterSystemRefInput[];
  /** 省略表示保持不变；传空数组表示清空地点关联。 */
  locationLinks?: CharacterLocationLinkInput[];
  version: number;
};

export type DeleteCharacterResult = {
  id: string;
  deleted_at: string;
  undo_token: string;
  expires_at: string;
};

function listQuery(options: ListCharactersOptions): string {
  const search = new URLSearchParams();
  search.set("work_id", options.workID);
  if (options.q) search.set("q", options.q);
  if (options.systemNodeID) search.set("system_node_id", options.systemNodeID);
  if (options.reviewStatus) search.set("review_status", options.reviewStatus);
  if (options.sort) search.set("sort", options.sort);
  if (options.page) search.set("page", String(options.page));
  if (options.pageSize) search.set("page_size", String(options.pageSize));
  return `?${search.toString()}`;
}

export function listCharacters(options: ListCharactersOptions): Promise<CharacterPage> {
  return requestJson<CharacterPage>(`/api/v1/characters${listQuery(options)}`);
}

export function getCharacter(id: string): Promise<CharacterDetail> {
  return requestJson<CharacterDetail>(`/api/v1/characters/${id}`);
}

export function createCharacter(input: CreateCharacterInput): Promise<CharacterDetail> {
  return requestJson<CharacterDetail>("/api/v1/characters", {
    method: "POST",
    body: {
      work_id: input.workID,
      name: input.name,
      names: input.names ?? [],
      description: input.description ?? null,
      faction_id: input.factionID ?? null,
      system_refs: input.systemRefs ?? [],
      location_links: input.locationLinks ?? [],
    },
  });
}

export function updateCharacter(id: string, input: UpdateCharacterInput): Promise<CharacterDetail> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.names !== undefined) body.names = input.names;
  // 显式 null 必须原样发送，用于清空简介或势力。
  if (input.description !== undefined) body.description = input.description;
  if (input.factionID !== undefined) body.faction_id = input.factionID;
  if (input.systemRefs !== undefined) body.system_refs = input.systemRefs;
  if (input.locationLinks !== undefined) body.location_links = input.locationLinks;
  return requestJson<CharacterDetail>(`/api/v1/characters/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteCharacter(id: string, version: number): Promise<DeleteCharacterResult> {
  return requestJson<DeleteCharacterResult>(`/api/v1/characters/${id}`, {
    method: "DELETE",
    ifMatch: version,
  });
}
