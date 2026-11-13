import { describe, expect, it } from "vitest";

import type { FactionSummary } from "./api/factions";
import {
  defaultExpandedIDs,
  factionDescendantIDs,
  factionPath,
  firstMatchID,
  indexFactions,
  toFactionOptions,
} from "./factions-tree";

function faction(
  id: string,
  name: string,
  parent: FactionSummary["parent"] = null,
  reviewStatus: FactionSummary["review_status"] = "confirmed",
): FactionSummary {
  return {
    id,
    work_id: "work-1",
    name,
    parent,
    leader: null,
    description: null,
    review_status: reviewStatus,
    reviewed_at: null,
    children_count: 0,
    member_count: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    version: 1,
    deleted_at: null,
  };
}

const items: FactionSummary[] = [
  faction("wan", "万族"),
  faction("yao", "妖族", { id: "wan", name: "万族" }, "pending"),
  faction("qingqiu", "青丘", { id: "yao", name: "妖族" }),
  faction("ren", "人族"),
];

describe("indexFactions", () => {
  it("组树并保持顶层顺序", () => {
    const index = indexFactions(items);
    expect(index.roots.map((item) => item.id)).toEqual(["wan", "ren"]);
    expect(index.children.get("wan")?.map((item) => item.id)).toEqual(["yao"]);
    expect(index.children.get("yao")?.map((item) => item.id)).toEqual(["qingqiu"]);
  });

  it("搜索命中时补齐祖先", () => {
    const index = indexFactions(items, { query: "青丘" });
    expect([...index.matches]).toEqual(["qingqiu"]);
    expect(index.visible).toEqual(new Set(["qingqiu", "yao", "wan"]));
    expect(index.roots.map((item) => item.id)).toEqual(["wan"]);
    expect(firstMatchID(items, index.matches)).toBe("qingqiu");
  });

  it("仅看待校对时不补祖先", () => {
    const index = indexFactions(items, { pendingOnly: true });
    expect(index.visible).toEqual(new Set(["yao"]));
    expect(index.roots).toHaveLength(0);
  });

  it("无匹配时返回空树", () => {
    const index = indexFactions(items, { query: "不存在" });
    expect(index.roots).toHaveLength(0);
    expect(index.matches.size).toBe(0);
  });
});

describe("faction helpers", () => {
  it("返回从顶层到目标的路径", () => {
    expect(factionPath(items, "qingqiu").map((item) => item.name)).toEqual(["万族", "妖族", "青丘"]);
    expect(factionPath(items, "missing")).toEqual([]);
  });

  it("生成全路径下拉选项", () => {
    const options = toFactionOptions(items);
    expect(options).toEqual([
      { value: "wan", label: "万族" },
      { value: "yao", label: "万族 · 妖族" },
      { value: "qingqiu", label: "万族 · 妖族 · 青丘" },
      { value: "ren", label: "人族" },
    ]);
  });

  it("后代集合包含自身", () => {
    expect(factionDescendantIDs(items, "wan")).toEqual(new Set(["wan", "yao", "qingqiu"]));
    expect(factionDescendantIDs(items, "ren")).toEqual(new Set(["ren"]));
  });

  it("默认展开顶层与第二层", () => {
    expect(defaultExpandedIDs(items)).toEqual(new Set(["wan", "ren", "yao"]));
  });
});
