import type { FactionSummary } from "./api/factions";

export type FactionTreeFilter = {
  query?: string;
  pendingOnly?: boolean;
};

export type FactionTreeIndex = {
  /** 顶层节点（父级缺失或未设置）。 */
  roots: FactionSummary[];
  byID: Map<string, FactionSummary>;
  /** parentID → 直接下级；顶层用空字符串。 */
  children: Map<string, FactionSummary[]>;
  /** 过滤后应展示的节点；搜索时含命中节点的祖先。 */
  visible: Set<string>;
  /** 直接命中搜索/待校对条件的节点。 */
  matches: Set<string>;
};

function rootKey(): string {
  return "";
}

/** 一次索引扁平列表：组树、搜索命中与祖先展开都在前端完成（设计 D3）。 */
export function indexFactions(
  items: FactionSummary[],
  filter: FactionTreeFilter = {},
): FactionTreeIndex {
  const byID = new Map(items.map((item) => [item.id, item]));
  const children = new Map<string, FactionSummary[]>();
  for (const item of items) {
    const parentID = item.parent && byID.has(item.parent.id) ? item.parent.id : rootKey();
    const bucket = children.get(parentID);
    if (bucket) {
      bucket.push(item);
    } else {
      children.set(parentID, [item]);
    }
  }

  const query = filter.query?.trim().toLowerCase() ?? "";
  const matches = new Set<string>();
  for (const item of items) {
    if (filter.pendingOnly && item.review_status !== "pending") continue;
    if (query && !item.name.toLowerCase().includes(query)) continue;
    matches.add(item.id);
  }

  const visible = new Set<string>();
  if (!query && !filter.pendingOnly) {
    for (const item of items) visible.add(item.id);
  } else {
    for (const id of matches) visible.add(id);
    // 任意过滤命中都补齐祖先，避免组树丢父链（搜索与「仅看待校对」一致）。
    for (const id of matches) {
      let parent = byID.get(id)?.parent ?? null;
      while (parent && byID.has(parent.id) && !visible.has(parent.id)) {
        visible.add(parent.id);
        parent = byID.get(parent.id)?.parent ?? null;
      }
    }
  }

  return {
    roots: (children.get(rootKey()) ?? []).filter((item) => visible.has(item.id)),
    byID,
    children,
    visible,
    matches,
  };
}

/** 返回从顶层到目标节点的路径；目标缺失时返回空数组。 */
export function factionPath(items: FactionSummary[], id: string): FactionSummary[] {
  const byID = new Map(items.map((item) => [item.id, item]));
  const path: FactionSummary[] = [];
  let current = byID.get(id);
  while (current) {
    path.unshift(current);
    const parentID = current.parent?.id;
    current = parentID ? byID.get(parentID) : undefined;
  }
  return path;
}

/** 生成「上级 · 下级」全路径下拉选项，供上级选择与人物所属势力使用。 */
export function toFactionOptions(items: FactionSummary[]): { value: string; label: string }[] {
  const index = indexFactions(items);
  const options: { value: string; label: string }[] = [];
  const walk = (nodes: FactionSummary[], prefix: string) => {
    for (const node of nodes) {
      const label = prefix ? `${prefix} · ${node.name}` : node.name;
      options.push({ value: node.id, label });
      walk(index.children.get(node.id) ?? [], label);
    }
  };
  walk(index.roots, "");
  return options;
}

/** 自身及全部后代 ID，用于编辑时禁用非法上级。 */
export function factionDescendantIDs(items: FactionSummary[], id: string): Set<string> {
  const index = indexFactions(items);
  const out = new Set<string>();
  const walk = (current: string) => {
    if (out.has(current)) return;
    out.add(current);
    for (const child of index.children.get(current) ?? []) walk(child.id);
  };
  walk(id);
  return out;
}

/** 返回第一个命中的节点 ID，用于搜索后滚动定位。 */
export function firstMatchID(items: FactionSummary[], matches: Set<string>): string | null {
  for (const item of items) {
    if (matches.has(item.id)) return item.id;
  }
  return null;
}

/** 返回默认展开的节点：顶层与第二层（默认展开两级）。 */
export function defaultExpandedIDs(items: FactionSummary[]): Set<string> {
  const index = indexFactions(items);
  const out = new Set<string>();
  for (const root of index.roots) {
    out.add(root.id);
    for (const child of index.children.get(root.id) ?? []) {
      out.add(child.id);
    }
  }
  return out;
}
