"use client";

import { useRef } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import type { SystemCategory } from "@/lib/api/system";
import { messages } from "@/lib/messages";

const CATEGORY_LIMIT = 10;

export type CategoryListProps = {
  categories: SystemCategory[];
  selectedID: string | null;
  loading?: boolean;
  onSelect: (id: string) => void;
  onCreate: () => void;
};

function typeLabel(category: SystemCategory): string {
  return category.category_type === "era" ? messages.systemEra : messages.systemNormal;
}

/** 左栏类别列表：内置置顶、listbox 语义与上下键切换（PG-10 §4.1、§10）。 */
export function CategoryList({ categories, selectedID, loading = false, onSelect, onCreate }: CategoryListProps) {
  const itemRefs = useRef(new Map<string, HTMLButtonElement>());
  const atLimit = categories.length >= CATEGORY_LIMIT;

  function selectAndFocus(id: string) {
    onSelect(id);
    itemRefs.current.get(id)?.focus();
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    if (categories.length === 0) return;
    const currentIndex = categories.findIndex((category) => category.id === selectedID);
    let nextIndex: number;
    if (event.key === "ArrowDown") {
      nextIndex = currentIndex < 0 ? 0 : Math.min(currentIndex + 1, categories.length - 1);
    } else if (event.key === "ArrowUp") {
      nextIndex = currentIndex < 0 ? categories.length - 1 : Math.max(currentIndex - 1, 0);
    } else if (event.key === "Home") {
      nextIndex = 0;
    } else if (event.key === "End") {
      nextIndex = categories.length - 1;
    } else {
      return;
    }
    event.preventDefault();
    const next = categories[nextIndex];
    if (next) selectAndFocus(next.id);
  }

  return (
    <section className="system-category-list" aria-labelledby="system-category-list-title">
      <div className="system-category-list__header">
        <h2 id="system-category-list-title" className="system-category-list__title">
          {messages.systemCategory}
        </h2>
        <span className="system-category-list__count">
          {categories.length}/{CATEGORY_LIMIT}
        </span>
      </div>

      {loading ? (
        <div aria-busy="true">
          <Skeleton variant="lines" lines={6} label={messages.systemLoadingCategories} />
        </div>
      ) : categories.length === 0 ? (
        <EmptyState
          variant="inline"
          title={messages.systemCategoryEmptyTitle}
          description={messages.systemCategoryEmptyDescription}
        />
      ) : (
        <div
          role="listbox"
          aria-label={messages.systemCategoryListAria}
          className="system-category-list__items"
          onKeyDown={handleKeyDown}
        >
          {categories.map((category) => {
            const active = category.id === selectedID;
            return (
              <button
                key={category.id}
                ref={(node) => {
                  if (node) {
                    itemRefs.current.set(category.id, node);
                  } else {
                    itemRefs.current.delete(category.id);
                  }
                }}
                type="button"
                role="option"
                aria-selected={active}
                tabIndex={active ? 0 : -1}
                className={`system-category-item${active ? " system-category-item--active" : ""}`}
                onClick={() => onSelect(category.id)}
              >
                <span className="system-category-item__top">
                  <span className="system-category-item__name">{category.name}</span>
                  {category.is_builtin ? (
                    <Badge tone="neutral">
                      {messages.systemBuiltin} · {messages.systemRequired}
                    </Badge>
                  ) : null}
                </span>
                <span className="system-category-item__meta">
                  <Badge tone="brand">{typeLabel(category)}</Badge>
                  <Badge tone={category.is_enabled ? "success" : "warning"}>
                    {category.is_enabled ? messages.systemStateEnabled : messages.systemStateDisabled}
                  </Badge>
                  <span className="system-category-item__count">
                    {messages.systemNodeCount(category.node_count)}
                  </span>
                  <span className="system-category-item__count">
                    {messages.systemReferenceCount(category.reference_count)}
                  </span>
                </span>
              </button>
            );
          })}
        </div>
      )}

      <div className="system-category-list__footer" hidden={loading}>
        <Button
          variant="primary"
          size="sm"
          className="system-category-list__add"
          disabled={atLimit}
          title={atLimit ? messages.errorByCode.CATEGORY_LIMIT : undefined}
          onClick={onCreate}
        >
          {messages.systemAddCategory}
        </Button>
        {atLimit ? (
          <p className="system-category-list__limit" role="note">
            {messages.errorByCode.CATEGORY_LIMIT}
          </p>
        ) : null}
      </div>
    </section>
  );
}
