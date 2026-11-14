import { describe, expect, it } from "vitest";

import type { LocationSummary } from "./api/locations";
import {
  defaultExpandedIDs,
  firstMatchID,
  indexLocations,
  locationDescendantIDs,
  locationPath,
  locationTypeOptions,
  toLocationOptions,
} from "./locations-tree";

function location(
  id: string,
  name: string,
  parent: LocationSummary["parent"] = null,
  typeText: string | null = null,
  reviewStatus: LocationSummary["review_status"] = "confirmed",
): LocationSummary {
  return {
    id,
    work_id: "work-1",
    name,
    parent,
    type_text: typeText,
    description: null,
    review_status: reviewStatus,
    reviewed_at: null,
    children_count: 0,
    characters_count: 0,
    factions_count: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    version: 1,
    deleted_at: null,
  };
}

const items: LocationSummary[] = [
  location("donghuang", "东荒", null, "星域"),
  location("cangwu", "苍梧星城", { id: "donghuang", name: "东荒" }, "星城", "pending"),
  location("guixu", "归墟秘境", { id: "cangwu", name: "苍梧星城" }, "秘境"),
  location("xizhou", "西洲", null, "星域"),
];

describe("indexLocations", () => {
  it("组树并保持顶层顺序", () => {
    const index = indexLocations(items);
    expect(index.roots.map((item) => item.id)).toEqual(["donghuang", "xizhou"]);
    expect(index.children.get("donghuang")?.map((item) => item.id)).toEqual(["cangwu"]);
    expect(index.children.get("cangwu")?.map((item) => item.id)).toEqual(["guixu"]);
  });

  it("搜索命中时补齐祖先", () => {
    const index = indexLocations(items, { query: "归墟" });
    expect([...index.matches]).toEqual(["guixu"]);
    expect(index.visible).toEqual(new Set(["guixu", "cangwu", "donghuang"]));
    expect(index.roots.map((item) => item.id)).toEqual(["donghuang"]);
    expect(firstMatchID(items, index.matches)).toBe("guixu");
  });

  it("类型筛选保留祖先链", () => {
    const index = indexLocations(items, { typeText: "秘境" });
    expect(index.visible).toEqual(new Set(["guixu", "cangwu", "donghuang"]));
    expect(index.children.get("cangwu")?.map((item) => item.id)).toEqual(["guixu"]);
  });

  it("仅看待校对时补齐祖先且只标记命中节点", () => {
    const index = indexLocations(items, { pendingOnly: true });
    expect(index.matches).toEqual(new Set(["cangwu"]));
    expect(index.visible).toEqual(new Set(["cangwu", "donghuang"]));
  });

  it("无匹配时返回空树", () => {
    const index = indexLocations(items, { query: "不存在" });
    expect(index.roots).toHaveLength(0);
    expect(index.matches.size).toBe(0);
  });
});

describe("location helpers", () => {
  it("返回从顶层到目标的路径", () => {
    expect(locationPath(items, "guixu").map((item) => item.name)).toEqual([
      "东荒",
      "苍梧星城",
      "归墟秘境",
    ]);
    expect(locationPath(items, "missing")).toEqual([]);
  });

  it("生成全路径下拉选项", () => {
    expect(toLocationOptions(items)).toEqual([
      { value: "donghuang", label: "东荒" },
      { value: "cangwu", label: "东荒 · 苍梧星城" },
      { value: "guixu", label: "东荒 · 苍梧星城 · 归墟秘境" },
      { value: "xizhou", label: "西洲" },
    ]);
  });

  it("后代集合包含自身", () => {
    expect(locationDescendantIDs(items, "donghuang")).toEqual(new Set(["donghuang", "cangwu", "guixu"]));
    expect(locationDescendantIDs(items, "xizhou")).toEqual(new Set(["xizhou"]));
  });

  it("类型建议去重排序且忽略空值", () => {
    expect(locationTypeOptions(items)).toEqual(["秘境", "星城", "星域"]);
  });

  it("默认展开顶层与第二层", () => {
    expect(defaultExpandedIDs(items)).toEqual(new Set(["donghuang", "xizhou", "cangwu"]));
  });
});
