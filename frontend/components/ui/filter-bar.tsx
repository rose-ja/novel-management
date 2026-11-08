"use client";

import type { ReactNode } from "react";

export type FilterBarProps = {
  children: ReactNode;
  onReset?: () => void;
  activeCount?: number;
  label?: string;
};

export function FilterBar({ children, onReset, activeCount = 0, label = "筛选条件" }: FilterBarProps) {
  return (
    <div className="ui-filter-bar" role="group" aria-label={label}>
      <div className="ui-filter-bar__fields">{children}</div>
      {onReset ? (
        <button type="button" className="ui-button ui-button--ghost ui-button--sm" onClick={onReset}>
          重置{activeCount > 0 ? `（${activeCount}）` : ""}
        </button>
      ) : null}
    </div>
  );
}
