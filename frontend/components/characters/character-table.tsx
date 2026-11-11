"use client";

import Link from "next/link";

import { CharacterImage } from "@/components/characters/character-image";
import { CharacterNameChips } from "@/components/characters/character-name-chips";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Table, type TableColumn } from "@/components/ui/table";
import type { CharacterSummary } from "@/lib/api/characters";
import { messages } from "@/lib/messages";

const columns: TableColumn[] = [
  { key: "image", label: messages.characterColumnImage, width: "56px" },
  { key: "name", label: messages.characterColumnName },
  { key: "names", label: messages.characterColumnNames },
  { key: "realm", label: messages.characterColumnRealm },
  { key: "faction", label: messages.characterColumnFaction },
  { key: "ranking", label: messages.characterColumnRanking },
  { key: "review", label: messages.characterColumnReview },
  { key: "updated_at", label: messages.characterColumnUpdatedAt },
  { key: "actions", label: messages.characterColumnActions, align: "right", width: "96px" },
];

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return messages.characterNoValue;
  return date.toLocaleString("zh-CN", { hour12: false });
}

export type CharacterTableProps = {
  items: CharacterSummary[];
  loading?: boolean;
  filtered?: boolean;
  onCreate: () => void;
  onClearFilters: () => void;
};

/** 人物列表表格；势力与榜单在本轮为占位（CHG-004 D4）。 */
export function CharacterTable({
  items,
  loading = false,
  filtered = false,
  onCreate,
  onClearFilters,
}: CharacterTableProps) {
  const rows = items.map((item) => [
    <CharacterImage
      key="image"
      name={item.name}
      image={item.image_thumb_url ? { asset_id: item.id, url: item.image_thumb_url, width: 40, height: 40 } : null}
      size="sm"
      thumb
    />,
    <Link key="name" className="character-name-link" href={`/characters?id=${encodeURIComponent(item.id)}`}>
      {item.name}
    </Link>,
    <CharacterNameChips key="names" names={item.names} max={3} />,
    <span key="realm">{item.system_summary ?? messages.characterNoValue}</span>,
    <span key="faction" className="character-muted">
      {messages.characterNoValue}
    </span>,
    <span key="ranking" className="character-muted">
      {messages.characterNoValue}
    </span>,
    item.review_status === "pending" ? (
      <Badge key="review" tone="warning" outlined>
        {messages.characterReviewPending}
      </Badge>
    ) : (
      <Badge key="review" tone="success" outlined>
        {messages.characterReviewConfirmed}
      </Badge>
    ),
    <span key="updated" className="character-time">
      {formatTime(item.updated_at)}
    </span>,
    <Link
      key="actions"
      className="ui-button ui-button--ghost ui-button--sm"
      href={`/characters?id=${encodeURIComponent(item.id)}`}
      aria-label={`${messages.characterEdit}「${item.name}」`}
    >
      {messages.characterEdit}
    </Link>,
  ]);

  const empty = filtered ? (
    <EmptyState
      variant="inline"
      title={messages.characterFilterEmptyTitle}
      description={messages.characterFilterEmptyDescription}
      action={
        <Button variant="secondary" size="sm" onClick={onClearFilters}>
          {messages.characterClearFilter}
        </Button>
      }
    />
  ) : (
    <EmptyState
      variant="inline"
      title={messages.characterEmptyTitle}
      description={messages.characterEmptyDescription}
      action={
        <Button size="sm" onClick={onCreate}>
          {messages.characterAdd}
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
      caption={messages.characterTableCaption}
    />
  );
}
