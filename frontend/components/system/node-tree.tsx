"use client";

import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, Pencil, Plus, Trash2 } from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { IconButton } from "@/components/ui/icon-button";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { toUserMessage } from "@/lib/api/errors";
import { reorderNodes, type SystemCategory, type SystemNode } from "@/lib/api/system";
import { messages } from "@/lib/messages";

export type NodeTreeProps = {
  category: SystemCategory;
  nodes: SystemNode[];
  loading: boolean;
  error: Error | null;
  /** 仅看待校对时后端返回平铺列表，此时同级顺序不完整，禁用排序入口。 */
  pendingOnly: boolean;
  readOnly: boolean;
  onRetry: () => void;
  onReload: () => Promise<void> | void;
  onCreateChild: (parent: SystemNode) => void;
  onRename: (node: SystemNode) => void;
  onDelete: (node: SystemNode) => void;
};

type TreeRow = {
  node: SystemNode;
  level: 1 | 2;
  parent: SystemNode | null;
  siblings: SystemNode[];
  index: number;
};

/** 树形响应直接使用 children；平铺响应按 parent_id 归组，保持后端已有顺序。 */
function normalizeNodes(items: SystemNode[]): SystemNode[] {
  const isTree = items.some((item) => Array.isArray(item.children));
  if (isTree) {
    return items.map((item) => ({ ...item, children: item.children ?? [] }));
  }

  const byID = new Map<string, SystemNode>();
  for (const item of items) {
    byID.set(item.id, { ...item, children: [] });
  }
  const roots: SystemNode[] = [];
  for (const item of byID.values()) {
    const parent = item.parent_id ? byID.get(item.parent_id) : undefined;
    if (parent) {
      parent.children = [...(parent.children ?? []), item];
    } else {
      roots.push(item);
    }
  }
  return roots;
}

function buildRows(roots: SystemNode[], collapsed: Set<string>): TreeRow[] {
  const rows: TreeRow[] = [];
  roots.forEach((node, index) => {
    rows.push({ node, level: 1, parent: null, siblings: roots, index });
    const children = node.children ?? [];
    if (children.length === 0 || collapsed.has(node.id)) return;
    children.forEach((child, childIndex) => {
      rows.push({ node: child, level: 2, parent: node, siblings: children, index: childIndex });
    });
  });
  return rows;
}

function replaceSiblings(roots: SystemNode[], parentID: string | null, ordered: SystemNode[]): SystemNode[] {
  if (parentID === null) {
    return ordered;
  }
  return roots.map((root) => (root.id === parentID ? { ...root, children: ordered } : root));
}

/** 两级节点树：展开/收起、同级排序、行内新增子层级/改名/删除（PG-10 §4.3、§10）。 */
export function NodeTree({
  category,
  nodes,
  loading,
  error,
  pendingOnly,
  readOnly,
  onRetry,
  onReload,
  onCreateChild,
  onRename,
  onDelete,
}: NodeTreeProps) {
  const { showToast } = useToast();
  const normalized = useMemo(() => normalizeNodes(nodes), [nodes]);
  const [items, setItems] = useState<SystemNode[]>(normalized);
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [reordering, setReordering] = useState(false);
  const rowRefs = useRef(new Map<string, HTMLDivElement>());

  useEffect(() => {
    setItems(normalized);
  }, [normalized]);

  useEffect(() => {
    setCollapsed(new Set());
    setSelectedID(null);
  }, [category.id]);

  const rows = useMemo(() => buildRows(items, collapsed), [items, collapsed]);
  const activeID =
    selectedID && rows.some((row) => row.node.id === selectedID) ? selectedID : (rows[0]?.node.id ?? null);

  const focusRow = useCallback((id: string) => {
    setSelectedID(id);
    rowRefs.current.get(id)?.focus();
  }, []);

  function toggleCollapse(node: SystemNode) {
    setCollapsed((current) => {
      const next = new Set(current);
      if (next.has(node.id)) {
        next.delete(node.id);
      } else {
        next.add(node.id);
      }
      return next;
    });
  }

  async function move(row: TreeRow, delta: -1 | 1) {
    const target = row.index + delta;
    if (reordering || target < 0 || target >= row.siblings.length) {
      return;
    }
    const ordered = [...row.siblings];
    [ordered[row.index], ordered[target]] = [ordered[target], ordered[row.index]];
    const previous = items;
    setItems((current) => replaceSiblings(current, row.parent?.id ?? null, ordered));
    setReordering(true);
    try {
      await reorderNodes({
        categoryID: category.id,
        parentID: row.parent ? row.parent.id : undefined,
        orderedIDs: ordered.map((item) => item.id),
      });
      await onReload();
      showToast({ tone: "success", message: messages.systemNodeReordered });
    } catch (cause) {
      setItems(previous);
      showToast({ tone: "error", message: toUserMessage(cause, messages.systemNodeReorderFailed) });
    } finally {
      setReordering(false);
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    const target = event.target as HTMLElement;
    if (target.closest("button")) {
      return;
    }
    const currentIndex = rows.findIndex((row) => row.node.id === activeID);
    if (currentIndex < 0) {
      return;
    }
    const row = rows[currentIndex];

    if (event.key === "ArrowDown") {
      event.preventDefault();
      const next = rows[Math.min(currentIndex + 1, rows.length - 1)];
      if (next) focusRow(next.node.id);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      const previous = rows[Math.max(currentIndex - 1, 0)];
      if (previous) focusRow(previous.node.id);
      return;
    }
    if (event.key === "Home") {
      event.preventDefault();
      const first = rows[0];
      if (first) focusRow(first.node.id);
      return;
    }
    if (event.key === "End") {
      event.preventDefault();
      const last = rows[rows.length - 1];
      if (last) focusRow(last.node.id);
      return;
    }
    if (event.key === "ArrowRight") {
      const children = row.node.children ?? [];
      if (row.level !== 1 || children.length === 0) return;
      event.preventDefault();
      if (collapsed.has(row.node.id)) {
        toggleCollapse(row.node);
      } else {
        focusRow(children[0].id);
      }
      return;
    }
    if (event.key === "ArrowLeft") {
      if (row.level === 2 && row.parent) {
        event.preventDefault();
        focusRow(row.parent.id);
        return;
      }
      const children = row.node.children ?? [];
      if (children.length > 0 && !collapsed.has(row.node.id)) {
        event.preventDefault();
        toggleCollapse(row.node);
      }
      return;
    }
    if (event.key === "Enter" && !readOnly) {
      event.preventDefault();
      onRename(row.node);
    }
  }

  if (loading) {
    return (
      <div className="system-node-loading" aria-busy="true">
        <Skeleton variant="lines" lines={5} label={messages.systemNodeLoading} />
      </div>
    );
  }

  if (error) {
    return (
      <Alert
        tone="danger"
        title={toUserMessage(error, messages.loadFailed)}
        action={
          <Button variant="secondary" size="sm" onClick={onRetry}>
            {messages.retry}
          </Button>
        }
      />
    );
  }

  if (items.length === 0) {
    return (
      <div className="system-node-placeholder">
        <EmptyState
          variant="inline"
          title={pendingOnly ? messages.systemNodePendingEmptyTitle : messages.systemNodeEmptyTitle}
          description={
            pendingOnly
              ? messages.systemNodePendingEmptyDescription
              : readOnly
                ? messages.systemNodeEmptyReadOnlyDescription
                : messages.systemNodeEmptyDescription
          }
        />
      </div>
    );
  }

  return (
    <div
      className="system-node-tree"
      role="tree"
      aria-label={messages.systemNodeTreeAria}
      aria-busy={reordering || undefined}
      onKeyDown={handleKeyDown}
    >
      {rows.map((row) => {
        const children = row.node.children ?? [];
        const expanded = children.length > 0 && !collapsed.has(row.node.id);
        const selected = row.node.id === activeID;
        return (
          <div
            key={row.node.id}
            ref={(element) => {
              if (element) {
                rowRefs.current.set(row.node.id, element);
              } else {
                rowRefs.current.delete(row.node.id);
              }
            }}
            role="treeitem"
            aria-level={row.level}
            aria-selected={selected}
            aria-expanded={children.length > 0 ? expanded : undefined}
            tabIndex={selected ? 0 : -1}
            className={[
              "system-node-row",
              `system-node-row--level-${row.level}`,
              selected ? "system-node-row--selected" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            onClick={() => setSelectedID(row.node.id)}
            onFocus={() => setSelectedID(row.node.id)}
          >
            {children.length > 0 ? (
              <span className="system-node-row__toggle">
                <IconButton
                  size="sm"
                  label={expanded ? messages.systemCollapseNode(row.node.name) : messages.systemExpandNode(row.node.name)}
                  onClick={() => toggleCollapse(row.node)}
                >
                  {expanded ? (
                    <ChevronDown aria-hidden="true" size={16} />
                  ) : (
                    <ChevronRight aria-hidden="true" size={16} />
                  )}
                </IconButton>
              </span>
            ) : (
              <span className="system-node-row__toggle" aria-hidden="true" />
            )}

            <span className="system-node-row__name" title={row.node.name}>
              {row.node.name}
            </span>

            {row.node.review_status === "pending" ? (
              <Badge tone="warning" outlined>
                {messages.systemReviewPending}
              </Badge>
            ) : null}

            <span className="system-node-row__reference">
              {messages.systemReferenceCount(row.node.reference_count)}
            </span>

            {!readOnly ? (
              <span className="system-node-row__actions">
                <IconButton
                  size="sm"
                  label={
                    pendingOnly
                      ? messages.systemMoveNodeUpFiltered(row.node.name)
                      : row.index === 0
                        ? messages.systemMoveNodeUpAtStart(row.node.name)
                        : messages.systemMoveNodeUp(row.node.name)
                  }
                  disabled={pendingOnly || row.index === 0 || reordering}
                  onClick={() => void move(row, -1)}
                >
                  <ArrowUp aria-hidden="true" size={16} />
                </IconButton>
                <IconButton
                  size="sm"
                  label={
                    pendingOnly
                      ? messages.systemMoveNodeDownFiltered(row.node.name)
                      : row.index === row.siblings.length - 1
                        ? messages.systemMoveNodeDownAtEnd(row.node.name)
                        : messages.systemMoveNodeDown(row.node.name)
                  }
                  disabled={pendingOnly || row.index === row.siblings.length - 1 || reordering}
                  onClick={() => void move(row, 1)}
                >
                  <ArrowDown aria-hidden="true" size={16} />
                </IconButton>
                {row.level === 1 ? (
                  <IconButton
                    size="sm"
                    label={messages.systemAddChildNodeAria(row.node.name)}
                    onClick={() => onCreateChild(row.node)}
                  >
                    <Plus aria-hidden="true" size={16} />
                  </IconButton>
                ) : null}
                <IconButton
                  size="sm"
                  label={messages.systemRenameNodeAria(row.node.name)}
                  onClick={() => onRename(row.node)}
                >
                  <Pencil aria-hidden="true" size={16} />
                </IconButton>
                <IconButton
                  size="sm"
                  tone="danger"
                  label={messages.systemDeleteNodeAria(row.node.name)}
                  onClick={() => onDelete(row.node)}
                >
                  <Trash2 aria-hidden="true" size={16} />
                </IconButton>
              </span>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
