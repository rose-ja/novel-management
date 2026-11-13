import { requestJson } from "./client";
import type { FactionBrief } from "./characters";
import type { ReviewStatus } from "./system";

export type FactionLocationBrief = {
  id: string;
  name: string;
};

export type FactionSummary = {
  id: string;
  work_id: string;
  name: string;
  parent: FactionBrief | null;
  leader: FactionBrief | null;
  description: string | null;
  review_status: ReviewStatus;
  reviewed_at: string | null;
  children_count: number;
  member_count: number;
  created_at: string;
  updated_at: string;
  version: number;
  deleted_at: string | null;
};

export type FactionRelations = {
  children: FactionBrief[];
  locations: FactionLocationBrief[];
  chronicle_count: number;
  subtree_member_count: number;
};

export type FactionDetail = FactionSummary & {
  relations: FactionRelations;
};

export type FactionList = {
  items: FactionSummary[];
  total: number;
};

export type ListFactionsOptions = {
  workID: string;
  q?: string;
  reviewStatus?: ReviewStatus;
  parentID?: string;
};

export type CreateFactionInput = {
  workID: string;
  name: string;
  parentID?: string | null;
  leaderCharacterID?: string | null;
  description?: string | null;
};

export type UpdateFactionInput = {
  name?: string;
  /** 传 null 表示清空上级；省略表示保持不变。 */
  parentID?: string | null;
  /** 传 null 表示清空首领；省略表示保持不变。 */
  leaderCharacterID?: string | null;
  /** 传 null 表示清空简介；省略表示保持不变。 */
  description?: string | null;
  version: number;
};

export type DeleteFactionResult = {
  id: string;
  deleted_at: string;
  undo_token: string;
  expires_at: string;
};

export type FactionMemberScope = "direct" | "subtree";

export type FactionMember = {
  id: string;
  name: string;
  faction: FactionBrief | null;
  system_summary: string | null;
  image_thumb_url: string | null;
};

export type FactionMemberPage = {
  items: FactionMember[];
  page: number;
  page_size: number;
  total: number;
};

export function listFactions(options: ListFactionsOptions): Promise<FactionList> {
  const search = new URLSearchParams({ work_id: options.workID });
  if (options.q) search.set("q", options.q);
  if (options.reviewStatus) search.set("review_status", options.reviewStatus);
  if (options.parentID) search.set("parent_id", options.parentID);
  return requestJson<FactionList>(`/api/v1/factions?${search.toString()}`);
}

export function getFaction(id: string): Promise<FactionDetail> {
  return requestJson<FactionDetail>(`/api/v1/factions/${id}`);
}

export function createFaction(input: CreateFactionInput): Promise<FactionDetail> {
  return requestJson<FactionDetail>("/api/v1/factions", {
    method: "POST",
    body: {
      work_id: input.workID,
      name: input.name,
      parent_id: input.parentID ?? null,
      leader_character_id: input.leaderCharacterID ?? null,
      description: input.description ?? null,
    },
  });
}

export function updateFaction(id: string, input: UpdateFactionInput): Promise<FactionDetail> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  // 显式 null 必须原样发送，用于清空上级、首领或简介。
  if (input.parentID !== undefined) body.parent_id = input.parentID;
  if (input.leaderCharacterID !== undefined) body.leader_character_id = input.leaderCharacterID;
  if (input.description !== undefined) body.description = input.description;
  return requestJson<FactionDetail>(`/api/v1/factions/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteFaction(id: string, version: number): Promise<DeleteFactionResult> {
  return requestJson<DeleteFactionResult>(`/api/v1/factions/${id}`, {
    method: "DELETE",
    ifMatch: version,
  });
}

export function listFactionMembers(
  factionID: string,
  options: { scope?: FactionMemberScope; page?: number; pageSize?: number } = {},
): Promise<FactionMemberPage> {
  const search = new URLSearchParams();
  if (options.scope) search.set("scope", options.scope);
  if (options.page) search.set("page", String(options.page));
  if (options.pageSize) search.set("page_size", String(options.pageSize));
  const query = search.toString();
  return requestJson<FactionMemberPage>(
    `/api/v1/factions/${factionID}/members${query ? `?${query}` : ""}`,
  );
}
