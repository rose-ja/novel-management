import { describe, expect, it } from "vitest";

import {
  buildSearchParams,
  EMPTY_SEARCH_STATE,
  filtersApplicable,
  hasSearchCriteria,
  matchedFieldLabel,
  parseSearchState,
  toSearchOptions,
} from "./search";

describe("search state", () => {
  it("URL 参数往返保持模式、关键词、筛选与页码", () => {
    const params = buildSearchParams({
      ...EMPTY_SEARCH_STATE,
      mode: "conditional",
      q: "  青丘  ",
      entityType: "character",
      factionID: "f1",
      factionScope: "subtree",
      systemNodeID: "n1",
      locationID: "l1",
      rankingID: "r1",
      page: 3,
    });

    const state = parseSearchState(params);
    expect(state.mode).toBe("conditional");
    expect(state.q).toBe("青丘");
    expect(state.entityType).toBe("character");
    expect(state.factionID).toBe("f1");
    expect(state.factionScope).toBe("subtree");
    expect(state.systemNodeID).toBe("n1");
    expect(state.locationID).toBe("l1");
    expect(state.rankingID).toBe("r1");
    expect(state.page).toBe(3);
  });

  it("默认进入快速搜索", () => {
    const state = parseSearchState(new URLSearchParams());
    expect(state.mode).toBe("quick");
    expect(state.entityType).toBe("all");
    expect(state.page).toBe(1);
  });

  it("非法类型与页码回退默认值", () => {
    const state = parseSearchState(new URLSearchParams("type=unknown&page=-2&scope=nope"));
    expect(state.entityType).toBe("all");
    expect(state.page).toBe(1);
    expect(state.factionScope).toBe("direct");
  });

  it("人物筛选只在条件模式且实体类型为人物时生效", () => {
    const conditional = { ...EMPTY_SEARCH_STATE, mode: "conditional" as const, entityType: "character" as const };
    expect(filtersApplicable(conditional)).toBe(true);
    expect(filtersApplicable({ ...conditional, mode: "quick" })).toBe(false);
    expect(filtersApplicable({ ...conditional, entityType: "faction" })).toBe(false);
  });

  it("不适用筛选不发送到 API", () => {
    const options = toSearchOptions({
      ...EMPTY_SEARCH_STATE,
      mode: "conditional",
      entityType: "faction",
      factionID: "f1",
      systemNodeID: "n1",
    });
    expect(options.factionID).toBeUndefined();
    expect(options.systemNodeID).toBeUndefined();
  });

  it("仅条件模式下的人物筛选构成检索条件", () => {
    expect(hasSearchCriteria({ ...EMPTY_SEARCH_STATE, q: "  " })).toBe(false);
    expect(hasSearchCriteria({ ...EMPTY_SEARCH_STATE, q: "a" })).toBe(true);
    expect(hasSearchCriteria({ ...EMPTY_SEARCH_STATE, mode: "conditional", entityType: "character", factionID: "f1" })).toBe(
      true,
    );
    expect(hasSearchCriteria({ ...EMPTY_SEARCH_STATE, mode: "conditional", entityType: "faction", factionID: "f1" })).toBe(
      false,
    );
  });

  it("命中字段使用中文标签", () => {
    expect(matchedFieldLabel("relation")).toBe("关联实体");
  });
});
