"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { NodeTree } from "@/components/system/node-tree";
import { ReferenceDialog, type ReferenceItem } from "@/components/system/reference-dialog";
import {
  CreateNodeDialog,
  DeleteNodeDialog,
  RenameNodeDialog,
} from "@/components/system/system-dialogs";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { listNodes, type SystemCategory, type SystemNode } from "@/lib/api/system";
import { messages } from "@/lib/messages";

export type CategoryDetailProps = {
  category: SystemCategory | null;
  loading?: boolean;
  onRename: (category: SystemCategory) => void;
  onDisable: (category: SystemCategory) => void;
  onEnable: (category: SystemCategory) => void;
  onDelete: (category: SystemCategory) => void;
  /** 节点增删改后通知父级刷新类别统计（节点数/引用数）。 */
  onCategoryChanged?: () => Promise<void> | void;
};

type NodeStatus = "idle" | "loading" | "ready" | "error";

type NodeDialogState =
  | { mode: "create"; parent: SystemNode | null }
  | { mode: "rename"; node: SystemNode }
  | { mode: "delete"; node: SystemNode };

type NodeReferenceState = {
  items: ReferenceItem[];
  nodeName: string;
};

function typeLabel(category: SystemCategory): string {
  return category.category_type === "era" ? messages.systemEra : messages.systemNormal;
}

function staticDescription(category: SystemCategory): string {
  if (category.is_builtin) return messages.systemBuiltinDescription;
  return category.category_type === "era" ? messages.systemEraDescription : messages.systemNormalDescription;
}

function findNodeVersion(items: SystemNode[], id: string): number | null {
  for (const item of items) {
    if (item.id === id) return item.version;
    const child = item.children?.find((candidate) => candidate.id === id);
    if (child) return child.version;
  }
  return null;
}

/**
 * 右栏类别详情：头部、静态说明、待校对筛选与两级节点树。
 * 停用类别只读；节点写操作全部经由本节点的弹窗完成。
 */
export function CategoryDetail({
  category,
  loading = false,
  onRename,
  onDisable,
  onEnable,
  onDelete,
  onCategoryChanged,
}: CategoryDetailProps) {
  const categoryID = category?.id ?? null;
  const [pendingOnly, setPendingOnly] = useState(false);
  const [nodes, setNodes] = useState<SystemNode[]>([]);
  const [nodeStatus, setNodeStatus] = useState<NodeStatus>("idle");
  const [nodeError, setNodeError] = useState<Error | null>(null);
  const [nodeDialog, setNodeDialog] = useState<NodeDialogState | null>(null);
  const [nodeReference, setNodeReference] = useState<NodeReferenceState | null>(null);
  const loadSequence = useRef(0);

  const loadNodes = useCallback(
    async (pending: boolean, silent = false) => {
      if (!categoryID) return;
      const sequence = ++loadSequence.current;
      if (!silent) setNodeStatus("loading");
      try {
        const items = await listNodes(categoryID, pending ? { reviewStatus: "pending" } : {});
        if (sequence !== loadSequence.current) return;
        setNodes(items);
        setNodeError(null);
        setNodeStatus("ready");
      } catch (cause) {
        if (sequence !== loadSequence.current) return;
        setNodeError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setNodeStatus("error");
      }
    },
    [categoryID],
  );

  const reloadNodes = useCallback(async () => {
    await loadNodes(pendingOnly, true);
  }, [loadNodes, pendingOnly]);

  useEffect(() => {
    setNodeDialog(null);
    setNodeReference(null);
    setNodes([]);
    setNodeError(null);
    if (!categoryID) {
      setNodeStatus("idle");
      return;
    }
    void loadNodes(pendingOnly);
  }, [categoryID, pendingOnly, loadNodes]);

  const refreshNodeVersion = useCallback(
    async (nodeID: string): Promise<number | null> => {
      if (!categoryID) return null;
      try {
        const items = await listNodes(categoryID, pendingOnly ? { reviewStatus: "pending" } : {});
        setNodes(items);
        setNodeError(null);
        setNodeStatus("ready");
        return findNodeVersion(items, nodeID);
      } catch (cause) {
        setNodeError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setNodeStatus("error");
        return null;
      }
    },
    [categoryID, pendingOnly],
  );

  async function handleNodeChanged() {
    await reloadNodes();
    await onCategoryChanged?.();
  }

  if (loading) {
    return (
      <section className="system-detail" aria-busy="true">
        <Skeleton variant="lines" lines={8} label={messages.systemLoadingDetail} />
      </section>
    );
  }

  if (!category) {
    return (
      <section className="system-detail">
        <EmptyState
          variant="section"
          title={messages.systemCategoryEmptyTitle}
          description={messages.systemCategoryEmptyDescription}
        />
      </section>
    );
  }

  const headingID = `system-category-title-${category.id}`;
  const nodesHeadingID = `system-nodes-title-${category.id}`;
  const pendingID = `system-pending-only-${category.id}`;
  const readOnly = !category.is_enabled;

  return (
    <section className="system-detail" aria-labelledby={headingID}>
      <header className="system-detail__header">
        <div className="system-detail__heading">
          <div className="system-detail__title-row">
            <h2 id={headingID} className="system-detail__title">
              {category.name}
            </h2>
            {!category.is_builtin ? (
              <div className="system-detail__actions">
                <Button variant="secondary" size="sm" onClick={() => onRename(category)}>
                  {messages.systemRenameCategory}
                </Button>
                {category.is_enabled ? (
                  <Button variant="secondary" size="sm" onClick={() => onDisable(category)}>
                    {messages.systemDisableCategory}
                  </Button>
                ) : (
                  <Button variant="secondary" size="sm" onClick={() => onEnable(category)}>
                    {messages.systemEnableCategory}
                  </Button>
                )}
                <Button variant="danger" size="sm" onClick={() => onDelete(category)}>
                  {messages.systemDeleteCategory}
                </Button>
              </div>
            ) : null}
          </div>
          <div className="system-detail__badges">
            <Badge tone="brand">{typeLabel(category)}</Badge>
            {category.is_builtin ? (
              <Badge tone="neutral">
                {messages.systemBuiltin} · {messages.systemRequired}
              </Badge>
            ) : null}
            <Badge tone={category.is_enabled ? "success" : "warning"}>
              {category.is_enabled ? messages.systemStateEnabled : messages.systemStateDisabled}
            </Badge>
          </div>
        </div>
        <dl className="system-detail__stats">
          <div className="system-detail__stat">
            <dt>{messages.systemNode}</dt>
            <dd>{category.node_count}</dd>
          </div>
          <div className="system-detail__stat">
            <dt>{messages.systemReferenceLabel}</dt>
            <dd>{category.reference_count} 人</dd>
          </div>
        </dl>
      </header>

      <p className="system-detail__description">{staticDescription(category)}</p>

      {readOnly ? <Alert tone="info" title={messages.systemCategoryDisabledNotice} /> : null}

      <div className="system-detail__filter">
        <label className="system-detail__checkbox" htmlFor={pendingID}>
          <input
            id={pendingID}
            type="checkbox"
            checked={pendingOnly}
            onChange={(event) => setPendingOnly(event.target.checked)}
          />
          <span>{messages.systemPendingOnly}</span>
        </label>
      </div>

      <section className="system-detail__nodes" aria-labelledby={nodesHeadingID}>
        <div className="system-detail__nodes-header">
          <h3 id={nodesHeadingID} className="system-detail__nodes-title">
            {messages.systemNodeSectionTitle}
          </h3>
          {readOnly ? null : (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setNodeDialog({ mode: "create", parent: null })}
            >
              {messages.systemAddPrimaryNode}
            </Button>
          )}
        </div>

        {readOnly ? (
          <p className="system-detail__nodes-readonly" role="note">
            {messages.systemNodeReadOnlyNotice}
          </p>
        ) : null}

        <NodeTree
          category={category}
          nodes={nodes}
          loading={nodeStatus === "loading" || nodeStatus === "idle"}
          error={nodeStatus === "error" ? nodeError : null}
          pendingOnly={pendingOnly}
          readOnly={readOnly}
          onRetry={() => void loadNodes(pendingOnly)}
          onReload={reloadNodes}
          onCreateChild={(parent) => setNodeDialog({ mode: "create", parent })}
          onRename={(node) => setNodeDialog({ mode: "rename", node })}
          onDelete={(node) => setNodeDialog({ mode: "delete", node })}
        />
      </section>

      {nodeDialog?.mode === "create" ? (
        <CreateNodeDialog
          category={category}
          parent={nodeDialog.parent}
          onClose={() => setNodeDialog(null)}
          onCreated={handleNodeChanged}
        />
      ) : null}

      {nodeDialog?.mode === "rename" ? (
        <RenameNodeDialog
          category={category}
          node={nodeDialog.node}
          onClose={() => setNodeDialog(null)}
          onChanged={handleNodeChanged}
          onVersionConflict={() => refreshNodeVersion(nodeDialog.node.id)}
        />
      ) : null}

      {nodeDialog?.mode === "delete" ? (
        <DeleteNodeDialog
          node={nodeDialog.node}
          onClose={() => setNodeDialog(null)}
          onChanged={handleNodeChanged}
          onVersionConflict={() => refreshNodeVersion(nodeDialog.node.id)}
          onReferenceConflict={(items) => {
            setNodeDialog(null);
            setNodeReference({ items, nodeName: nodeDialog.node.name });
          }}
        />
      ) : null}

      {nodeReference ? (
        <ReferenceDialog
          subject="node"
          action="delete"
          categoryName={nodeReference.nodeName}
          items={nodeReference.items}
          onClose={() => setNodeReference(null)}
        />
      ) : null}
    </section>
  );
}
