import { requestJson, requestVoid } from "./client";
import type { FactionBrief } from "./characters";
import type { ReviewStatus } from "./system";

export type LocationBrief = {
  id: string;
  name: string;
};

export type LocationSummary = {
  id: string;
  work_id: string;
  name: string;
  parent: LocationBrief | null;
  type_text: string | null;
  description: string | null;
  review_status: ReviewStatus;
  reviewed_at: string | null;
  children_count: number;
  characters_count: number;
  factions_count: number;
  created_at: string;
  updated_at: string;
  version: number;
  deleted_at: string | null;
};

export type LocationChild = {
  id: string;
  name: string;
  type_text: string | null;
  characters_count: number;
  factions_count: number;
};

export type LocationCharacter = {
  character_id: string;
  name: string;
  faction: FactionBrief | null;
  system_summary: string | null;
  image_thumb_url: string | null;
  note: string | null;
  in_trash: boolean;
};

export type LocationFaction = {
  faction_id: string;
  name: string;
  note: string | null;
  in_trash: boolean;
};

export type LocationRelationItem = LocationCharacter | LocationFaction;

export type LocationRelations = {
  children: LocationChild[];
  characters: LocationCharacter[];
  factions: LocationFaction[];
};

export type LocationDetail = LocationSummary & {
  relations: LocationRelations;
};

export type LocationList = {
  items: LocationSummary[];
  total: number;
};

export type ListLocationsOptions = {
  workID: string;
  q?: string;
  parentID?: string;
  typeText?: string;
  reviewStatus?: ReviewStatus;
};

export type CreateLocationInput = {
  workID: string;
  name: string;
  parentID?: string | null;
  typeText?: string | null;
  description?: string | null;
};

export type UpdateLocationInput = {
  name?: string;
  /** 传 null 表示清空上级；省略表示保持不变。 */
  parentID?: string | null;
  /** 传 null 表示清空类型；省略表示保持不变。 */
  typeText?: string | null;
  /** 传 null 表示清空简介；省略表示保持不变。 */
  description?: string | null;
  version: number;
};

export type DeleteLocationResult = {
  deleted: boolean;
  id: string;
  deleted_at: string;
  undo_token: string;
  expires_at: string;
};

export type AddLocationFactionInput = {
  factionID: string;
  note?: string | null;
};

export function listLocations(options: ListLocationsOptions): Promise<LocationList> {
  const search = new URLSearchParams({ work_id: options.workID });
  if (options.q) search.set("q", options.q);
  if (options.parentID) search.set("parent_id", options.parentID);
  if (options.typeText) search.set("type_text", options.typeText);
  if (options.reviewStatus) search.set("review_status", options.reviewStatus);
  return requestJson<LocationList>(`/api/v1/locations?${search.toString()}`);
}

export function getLocation(id: string): Promise<LocationDetail> {
  return requestJson<LocationDetail>(`/api/v1/locations/${id}`);
}

export function createLocation(input: CreateLocationInput): Promise<LocationDetail> {
  return requestJson<LocationDetail>("/api/v1/locations", {
    method: "POST",
    body: {
      work_id: input.workID,
      name: input.name,
      parent_id: input.parentID ?? null,
      type_text: input.typeText ?? null,
      description: input.description ?? null,
    },
  });
}

export function updateLocation(id: string, input: UpdateLocationInput): Promise<LocationDetail> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  // 显式 null 必须原样发送，用于清空上级、类型或简介。
  if (input.parentID !== undefined) body.parent_id = input.parentID;
  if (input.typeText !== undefined) body.type_text = input.typeText;
  if (input.description !== undefined) body.description = input.description;
  return requestJson<LocationDetail>(`/api/v1/locations/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteLocation(id: string, version: number): Promise<DeleteLocationResult> {
  return requestJson<DeleteLocationResult>(`/api/v1/locations/${id}`, {
    method: "DELETE",
    ifMatch: version,
  });
}

export function listLocationCharacters(id: string): Promise<{ items: LocationCharacter[] }> {
  return requestJson<{ items: LocationCharacter[] }>(`/api/v1/locations/${id}/characters`);
}

export function listLocationFactions(id: string): Promise<{ items: LocationFaction[] }> {
  return requestJson<{ items: LocationFaction[] }>(`/api/v1/locations/${id}/factions`);
}

export function addLocationFaction(
  locationID: string,
  input: AddLocationFactionInput,
): Promise<LocationFaction> {
  return requestJson<LocationFaction>(`/api/v1/locations/${locationID}/factions`, {
    method: "POST",
    body: {
      faction_id: input.factionID,
      note: input.note ?? null,
    },
  });
}

export function updateLocationFactionNote(
  locationID: string,
  factionID: string,
  note: string | null,
): Promise<LocationFaction> {
  return requestJson<LocationFaction>(
    `/api/v1/locations/${locationID}/factions/${factionID}`,
    {
      method: "PATCH",
      body: { note },
    },
  );
}

export function removeLocationFaction(locationID: string, factionID: string): Promise<void> {
  return requestVoid(`/api/v1/locations/${locationID}/factions/${factionID}`, {
    method: "DELETE",
  });
}
