"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";

export type PaginationProps = {
  page: number;
  pageSize: number;
  total: number;
  onChange: (page: number) => void;
};

/** 分页：只渲染当前页附近窗口，首尾页始终可达。 */
export function Pagination({ page, pageSize, total, onChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const current = Math.min(Math.max(page, 1), totalPages);
  const windowStart = Math.max(1, Math.min(current - 2, totalPages - 4));
  const pages = Array.from({ length: Math.min(5, totalPages) }, (_, index) => windowStart + index);

  return (
    <nav className="ui-pagination" aria-label="分页">
      <span className="ui-pagination__summary">
        共 {total} 条 · 第 {current} / {totalPages} 页
      </span>
      <div className="ui-pagination__controls">
        <button
          type="button"
          className="ui-icon-button ui-icon-button--sm"
          aria-label="上一页"
          disabled={current <= 1}
          onClick={() => onChange(current - 1)}
        >
          <ChevronLeft aria-hidden="true" size={16} />
        </button>
        {pages.map((value) => (
          <button
            key={value}
            type="button"
            className={value === current ? "ui-pagination__page ui-pagination__page--active" : "ui-pagination__page"}
            aria-current={value === current ? "page" : undefined}
            onClick={() => onChange(value)}
          >
            {value}
          </button>
        ))}
        <button
          type="button"
          className="ui-icon-button ui-icon-button--sm"
          aria-label="下一页"
          disabled={current >= totalPages}
          onClick={() => onChange(current + 1)}
        >
          <ChevronRight aria-hidden="true" size={16} />
        </button>
      </div>
    </nav>
  );
}
