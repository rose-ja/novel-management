"use client";

import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Table, type TableColumn } from "@/components/ui/table";
import type { RankingSummary } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";
import { formatTopSeats } from "@/lib/rankings";

const columns: TableColumn[] = [
  { key: "name", label: messages.rankingColumnName },
  { key: "seats", label: messages.rankingColumnSeats, align: "right", width: "96px" },
  { key: "occupied", label: messages.rankingColumnOccupied, align: "right", width: "80px" },
  { key: "empty", label: messages.rankingColumnEmpty, align: "right", width: "80px" },
  { key: "top", label: messages.rankingColumnTop },
  { key: "updated_at", label: messages.rankingColumnUpdatedAt, width: "180px" },
  { key: "actions", label: messages.rankingColumnActions, align: "right", width: "120px" },
];

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return messages.rankingNoValue;
  return date.toLocaleString("zh-CN", { hour12: false });
}

export type RankingTableProps = {
  items: RankingSummary[];
  loading?: boolean;
  filtered?: boolean;
  onCreate: () => void;
  onClearFilters: () => void;
  onDelete: (item: RankingSummary) => void;
};

/** 榜单列表表格：前三席展示与行内删除。 */
export function RankingTable({
  items,
  loading = false,
  filtered = false,
  onCreate,
  onClearFilters,
  onDelete,
}: RankingTableProps) {
  const rows = items.map((item) => [
    <div key="name" className="ranking-name-cell">
      <Link className="character-name-link" href={`/rankings?id=${encodeURIComponent(item.id)}`}>
        {item.name}
      </Link>
      {item.review_status === "pending" ? (
        <Badge tone="warning" outlined>
          {messages.characterReviewPending}
        </Badge>
      ) : null}
    </div>,
    <span key="seats">{item.seat_count}</span>,
    <span key="occupied">{item.occupied_count}</span>,
    <span key="empty">{item.empty_count}</span>,
    <span key="top" className="ranking-top-seats">
      {formatTopSeats(item.top_seats)}
    </span>,
    <span key="updated" className="character-time">
      {formatTime(item.updated_at)}
    </span>,
    <Button key="actions" variant="ghost" size="sm" onClick={() => onDelete(item)}>
      {messages.rankingDelete}
    </Button>,
  ]);

  const empty = filtered ? (
    <EmptyState
      variant="inline"
      title={messages.rankingFilterEmptyTitle}
      description={messages.rankingFilterEmptyDescription}
      action={
        <Button variant="secondary" size="sm" onClick={onClearFilters}>
          {messages.rankingClearFilter}
        </Button>
      }
    />
  ) : (
    <EmptyState
      variant="inline"
      title={messages.rankingEmptyTitle}
      description={messages.rankingEmptyDescription}
      action={
        <Button size="sm" onClick={onCreate}>
          {messages.rankingAdd}
        </Button>
      }
    />
  );

  return (
    <Table
      columns={columns}
      rows={rows}
      loading={loading}
      skeletonRows={5}
      empty={empty}
      rowKey={(index) => items[index]?.id ?? String(index)}
      caption={messages.rankingTableCaption}
    />
  );
}
