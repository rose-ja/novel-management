import type { LocationSummary } from "./api/locations";

export type LocationTreeFilter = {
  query?: string;
  typeText?: string;
  pendingOnly?: boolean;
};

export type LocationTreeIndex = {
  /** 顶层节点（父级缺失或未设置）。 */
  roots: LocationSummary[];
  byID: Map<string, LocationSummary>;
  /** parentID → 直接下级；顶层用空字符串。 */
  children: Map<string, LocationSummary[]>;
  /** 过滤后应展示的节点；筛选时含命中节点的祖先。 */
  visible: Set<string>;
  /** 直接命中搜索/类型/待校对条件的节点。 */
  matches: Set<string>;
};

function rootKey(): string {
  return "";
}

/** 一次索引扁平列表：组树、搜索命中与祖先补齐都在前端完成。 */
export function indexLocations(
  items: LocationSummary[],
  filter: LocationTreeFilter = {},
): LocationTreeIndex {
  const byID = new Map(items.map((item) => [item.id, item]));
  const children = new Map<string, LocationSummary[]>();
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
  const typeText = filter.typeText?.trim() ?? "";
  const matches = new Set<string>();
  for (const item of items) {
    if (filter.pendingOnly && item.review_status !== "pending") continue;
    if (typeText && (item.type_text ?? "") !== typeText) continue;
    if (query && !item.name.toLowerCase().includes(query)) continue;
    matches.add(item.id);
  }

  const filtering = Boolean(query) || Boolean(typeText) || Boolean(filter.pendingOnly);
  const visible = new Set<string>();
  if (!filtering) {
    for (const item of items) visible.add(item.id);
  } else {
    for (const id of matches) visible.add(id);
    // 任意筛选命中都补齐祖先，避免组树丢父链。
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

/** 返回列表中第一个命中筛选的节点，供滚动定位。 */
export function firstMatchID(items: LocationSummary[], matches: Set<string>): string | null {
  for (const item of items) {
    if (matches.has(item.id)) return item.id;
  }
  return null;
}

/** 返回从顶层到目标节点的路径；目标缺失时返回空数组。 */
export function locationPath(items: LocationSummary[], id: string): LocationSummary[] {
  const byID = new Map(items.map((item) => [item.id, item]));
  const path: LocationSummary[] = [];
  let current = byID.get(id);
  while (current) {
    path.unshift(current);
    const parentID = current.parent?.id;
    current = parentID ? byID.get(parentID) : undefined;
  }
  return path;
}

/** 生成「上级 · 下级」全路径下拉选项。 */
export function toLocationOptions(items: LocationSummary[]): { value: string; label: string }[] {
  const index = indexLocations(items);
  const options: { value: string; label: string }[] = [];
  const walk = (nodes: LocationSummary[], prefix: string) => {
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
export function locationDescendantIDs(items: LocationSummary[], id: string): Set<string> {
  const index = indexLocations(items);
  const out = new Set<string>();
  const walk = (current: string) => {
    if (out.has(current)) return;
    out.add(current);
    for (const child of index.children.get(current) ?? []) walk(child.id);
  };
  walk(id);
  return out;
}

/** 已有类型去重排序，作为输入建议（允许新值）。 */
export function locationTypeOptions(items: LocationSummary[]): string[] {
  const values = new Set<string>();
  for (const item of items) {
    const value = item.type_text?.trim();
    if (value) values.add(value);
  }
  return Array.from(values).sort((a, b) => a.localeCompare(b, "zh-CN"));
}

/** 返回默认展开的节点：顶层与第二层（默认展开两级）。 */
export function defaultExpandedIDs(items: LocationSummary[]): Set<string> {
  const index = indexLocations(items);
  const out = new Set<string>();
  for (const root of index.roots) {
    out.add(root.id);
    for (const child of index.children.get(root.id) ?? []) {
      out.add(child.id);
    }
  }
  return out;
}
