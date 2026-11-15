import { requestJson } from "./client";
import type { ReviewStatus } from "./system";

export type SearchEntityType = "character" | "faction" | "location" | "ranking" | "system_node";

export type SearchMatchedField = {
  field: "name" | "alias" | "title" | "desc" | "relation";
  matched_text: string;
};

export type SearchItem = {
  entity_id: string;
  entity_type: SearchEntityType;
  title: string;
  subtitle: string;
  matched_fields: SearchMatchedField[];
  snippet: string;
  score: number;
  updated_at: string;
  url: string;
  review_status: ReviewStatus;
  image_thumb_url?: string;
};

export type SearchGroup = {
  entity_type: SearchEntityType;
  total: number;
  items: SearchItem[];
};

export type SearchQueryEcho = {
  q: string;
  tokens: string[];
  filters: {
    entity_types: SearchEntityType[];
    faction_id: string | null;
    faction_scope: "direct" | "subtree";
    system_category_id: string | null;
    system_node_id: string | null;
    location_id: string | null;
    ranking_id: string | null;
    review_status: ReviewStatus | null;
  };
};

export type SearchResult = {
  groups: SearchGroup[];
  total: number;
  page: number;
  page_size: number;
  query: SearchQueryEcho;
};

export type SearchOptions = {
  workID: string;
  q?: string;
  entityTypes?: SearchEntityType[];
  factionID?: string;
  factionScope?: "direct" | "subtree";
  systemCategoryID?: string;
  systemNodeID?: string;
  locationID?: string;
  rankingID?: string;
  reviewStatus?: ReviewStatus;
  page?: number;
  signal?: AbortSignal;
};

export function searchEntities(options: SearchOptions): Promise<SearchResult> {
  const search = new URLSearchParams({ work_id: options.workID });
  if (options.q) search.set("q", options.q);
  if (options.entityTypes && options.entityTypes.length > 0) {
    search.set("entity_types", options.entityTypes.join(","));
  }
  if (options.factionID) search.set("faction_id", options.factionID);
  if (options.factionScope) search.set("faction_scope", options.factionScope);
  if (options.systemCategoryID) search.set("system_category_id", options.systemCategoryID);
  if (options.systemNodeID) search.set("system_node_id", options.systemNodeID);
  if (options.locationID) search.set("location_id", options.locationID);
  if (options.rankingID) search.set("ranking_id", options.rankingID);
  if (options.reviewStatus) search.set("review_status", options.reviewStatus);
  if (options.page && options.page > 1) search.set("page", String(options.page));
  return requestJson<SearchResult>(`/api/v1/search?${search.toString()}`, { signal: options.signal });
}
