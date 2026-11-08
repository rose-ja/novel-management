"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import type { SystemCategory } from "@/lib/api/system";
import { messages } from "@/lib/messages";

export type CategoryDetailProps = {
  category: SystemCategory | null;
  loading?: boolean;
};

function typeLabel(category: SystemCategory): string {
  return category.category_type === "era" ? messages.systemEra : messages.systemNormal;
}

function staticDescription(category: SystemCategory): string {
  if (category.is_builtin) return messages.systemBuiltinDescription;
  return category.category_type === "era" ? messages.systemEraDescription : messages.systemNormalDescription;
}

/**
 * 右栏类别详情：头部、静态说明、待校对筛选占位与节点区容器。
 * 类别写操作与节点树在后续步骤接入；内置类别不显示任何类别写操作入口。
 */
export function CategoryDetail({ category, loading = false }: CategoryDetailProps) {
  const [pendingOnly, setPendingOnly] = useState(false);

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

  return (
    <section className="system-detail" aria-labelledby={headingID}>
      <header className="system-detail__header">
        <div className="system-detail__heading">
          <h2 id={headingID} className="system-detail__title">
            {category.name}
          </h2>
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

      {!category.is_enabled ? (
        <Alert tone="info" title={messages.systemCategoryDisabledNotice} />
      ) : null}

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
          <Button
            variant="secondary"
            size="sm"
            disabled={!category.is_enabled}
            title={!category.is_enabled ? messages.systemAddPrimaryNodeDisabled : undefined}
          >
            {messages.systemAddPrimaryNode}
          </Button>
        </div>
        <div className="system-node-placeholder">
          <EmptyState
            variant="inline"
            title={pendingOnly ? messages.systemNodePendingEmptyTitle : messages.systemNodeEmptyTitle}
            description={
              pendingOnly ? messages.systemNodePendingEmptyDescription : messages.systemNodeEmptyDescription
            }
          />
        </div>
      </section>
    </section>
  );
}
