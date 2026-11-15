import type { SearchEntityType } from "@/lib/api/search";

export type SearchMode = "quick" | "conditional";

export type SearchState = {
  mode: SearchMode;
  q: string;
  entityType: SearchEntityType | "all";
  factionID: string;
  factionScope: "direct" | "subtree";
  systemNodeID: string;
  locationID: string;
  rankingID: string;
  page: number;
};

export const SEARCH_GROUP_ORDER: SearchEntityType[] = [
  "character",
  "faction",
  "location",
  "ranking",
  "system_node",
];

export const SEARCH_ENTITY_LABELS: Record<SearchEntityType, string> = {
  character: "人物",
  faction: "势力",
  location: "地点",
  ranking: "榜单",
  system_node: "体系节点",
};

export const SEARCH_MATCHED_FIELD_LABELS = {
  name: "名称",
  alias: "别名",
  title: "称号",
  desc: "简介",
  relation: "关联实体",
} as const;

export const EMPTY_SEARCH_STATE: SearchState = {
  mode: "quick",
  q: "",
  entityType: "all",
  factionID: "",
  factionScope: "direct",
  systemNodeID: "",
  locationID: "",
  rankingID: "",
  page: 1,
};

const ENTITY_TYPES: SearchEntityType[] = ["character", "faction", "location", "ranking", "system_node"];

export function parseSearchState(params: URLSearchParams): SearchState {
  const mode = params.get("mode") === "conditional" ? "conditional" : "quick";
  const rawType = params.get("type");
  const entityType = ENTITY_TYPES.includes(rawType as SearchEntityType)
    ? (rawType as SearchEntityType)
    : "all";
  const rawScope = params.get("scope");
  const page = Number.parseInt(params.get("page") ?? "", 10);
  return {
    mode,
    q: params.get("q") ?? "",
    entityType,
    factionID: params.get("faction") ?? "",
    factionScope: rawScope === "subtree" ? "subtree" : "direct",
    systemNodeID: params.get("realm") ?? "",
    locationID: params.get("location") ?? "",
    rankingID: params.get("ranking") ?? "",
    page: Number.isFinite(page) && page > 1 ? page : 1,
  };
}

export function buildSearchParams(state: SearchState): URLSearchParams {
  const params = new URLSearchParams();
  const q = state.q.trim();
  if (q) params.set("q", q);
  if (state.mode === "conditional") params.set("mode", "conditional");
  if (state.entityType !== "all") params.set("type", state.entityType);
  if (state.factionID) {
    params.set("faction", state.factionID);
    if (state.factionScope === "subtree") params.set("scope", "subtree");
  }
  if (state.systemNodeID) params.set("realm", state.systemNodeID);
  if (state.locationID) params.set("location", state.locationID);
  if (state.rankingID) params.set("ranking", state.rankingID);
  if (state.page > 1) params.set("page", String(state.page));
  return params;
}

/** 人物专属筛选只在条件模式且实体类型为人物时生效。 */
export function filtersApplicable(state: SearchState): boolean {
  return state.mode === "conditional" && state.entityType === "character";
}

export function hasSearchCriteria(state: SearchState): boolean {
  if (state.q.trim() !== "") return true;
  if (!filtersApplicable(state)) return false;
  return Boolean(state.factionID || state.systemNodeID || state.locationID || state.rankingID);
}

/** 把 UI 状态转换为 API 查询参数；不适用的人物筛选不发送。 */
export function toSearchOptions(state: SearchState): {
  q: string;
  entityTypes: SearchEntityType[];
  factionID?: string;
  factionScope?: "direct" | "subtree";
  systemNodeID?: string;
  locationID?: string;
  rankingID?: string;
  page: number;
} {
  const applicable = filtersApplicable(state);
  return {
    q: state.q.trim(),
    entityTypes: state.entityType === "all" ? [] : [state.entityType],
    factionID: applicable ? state.factionID || undefined : undefined,
    factionScope: applicable && state.factionID ? state.factionScope : undefined,
    systemNodeID: applicable ? state.systemNodeID || undefined : undefined,
    locationID: applicable ? state.locationID || undefined : undefined,
    rankingID: applicable ? state.rankingID || undefined : undefined,
    page: state.page,
  };
}

export function entityTypeLabel(entityType: SearchEntityType): string {
  return SEARCH_ENTITY_LABELS[entityType];
}

export function matchedFieldLabel(field: keyof typeof SEARCH_MATCHED_FIELD_LABELS): string {
  return SEARCH_MATCHED_FIELD_LABELS[field];
}
