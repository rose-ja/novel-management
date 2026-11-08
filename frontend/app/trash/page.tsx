"use client";

import Link from "next/link";
import { Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { ClearTrashDialog, PurgeTrashDialog, RestoreTrashDialog } from "@/components/trash/trash-dialogs";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { FilterBar } from "@/components/ui/filter-bar";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { Pagination } from "@/components/ui/pagination";
import { Select } from "@/components/ui/select";
import { Table } from "@/components/ui/table";
import { toUserMessage } from "@/lib/api/errors";
import { listTrash, type TrashItem, type TrashPage } from "@/lib/api/trash";
import { formatDateTime } from "@/lib/format";
import { messages } from "@/lib/messages";

const PAGE_SIZE = 20;

type Status = "loading" | "ready" | "error";

type DialogState = { kind: "restore"; item: TrashItem } | { kind: "purge"; item: TrashItem } | { kind: "clear" };

const typeOptions = [
  { value: "all", label: messages.trashTypeAll },
  { value: "work", label: messages.trashTypeWork },
];

export default function TrashPage() {
  const [page, setPage] = useState(1);
  const [data, setData] = useState<TrashPage | null>(null);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [typeFilter, setTypeFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [dialog, setDialog] = useState<DialogState | null>(null);

  const load = useCallback(async () => {
    setStatus("loading");
    setError(null);
    try {
      const result = await listTrash({ page, pageSize: PAGE_SIZE });
      setData(result);
      setStatus("ready");
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
      setStatus("error");
    }
  }, [page]);

  useEffect(() => {
    void load();
  }, [load]);

  const items = useMemo(() => data?.items ?? [], [data]);
  const total = data?.total ?? 0;
  const hasFilter = typeFilter !== "all" || query.trim().length > 0;

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    return items.filter((item) => {
      const typeMatched = typeFilter === "all" || item.root_type === typeFilter;
      const nameMatched = keyword.length === 0 || item.name.toLowerCase().includes(keyword);
      return typeMatched && nameMatched;
    });
  }, [items, typeFilter, query]);

  function resetFilter() {
    setTypeFilter("all");
    setQuery("");
    setPage(1);
  }

  const columns = [
    { key: "type", label: messages.trashColumnType, width: "96px" },
    { key: "name", label: messages.trashColumnName },
    { key: "work", label: messages.trashColumnWork, width: "160px" },
    { key: "deleted_at", label: messages.trashColumnDeletedAt, width: "160px" },
    { key: "affected", label: messages.trashColumnAffected, width: "112px" },
    { key: "actions", label: messages.trashColumnActions, width: "160px", align: "right" as const },
  ];

  const rows = filtered.map((item) => [
    <Badge key="type" tone="neutral">
      {messages.trashTypeWork}
    </Badge>,
    <span key="name">《{item.name}》</span>,
    "—",
    <span key="deleted_at" className="snapshot-time">
      {formatDateTime(item.deleted_at)}
    </span>,
    messages.trashAffected(item.affected_count),
    <div key="actions" className="table-actions">
      <Button
        variant="ghost"
        size="sm"
        aria-label={`${messages.trashRestore}作品《${item.name}》`}
        onClick={() => setDialog({ kind: "restore", item })}
      >
        {messages.trashRestore}
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="table-actions__danger"
        aria-label={`${messages.trashPurge}作品《${item.name}》`}
        onClick={() => setDialog({ kind: "purge", item })}
      >
        {messages.trashPurge}
      </Button>
    </div>,
  ]);

  const emptyNode = hasFilter ? (
    <EmptyState
      variant="inline"
      title={messages.trashFilterEmptyTitle}
      description={messages.trashFilterEmptyDescription}
      action={
        <Button variant="secondary" size="sm" onClick={resetFilter}>
          {messages.trashClearFilter}
        </Button>
      }
    />
  ) : undefined;

  return (
    <div className="page">
      <PageHeader
        title={messages.trashTitle}
        description={status === "ready" ? messages.trashDescription(total) : messages.trashTitle}
        actions={
          <Button
            variant="danger"
            size="sm"
            disabled={status === "loading" || total === 0}
            title={total === 0 ? messages.trashEmptyTitle : undefined}
            onClick={() => setDialog({ kind: "clear" })}
          >
            {messages.trashClear}
          </Button>
        }
      />

      <div className="trash-notice">
        <Alert tone="info" title={messages.trashSnapshotNotice} />
      </div>

      {status === "error" ? (
        <Alert
          tone="danger"
          title={toUserMessage(error, messages.loadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={() => void load()}>
              {messages.retry}
            </Button>
          }
        />
      ) : null}

      {status === "ready" && total === 0 ? (
        <EmptyState
          variant="page"
          icon={<Trash2 aria-hidden="true" size={22} />}
          title={messages.trashEmptyTitle}
          description={messages.trashEmptyDescription}
          action={
            <Link className="ui-button ui-button--secondary ui-button--md" href="/">
              <span className="ui-button__label">{messages.trashEmptyAction}</span>
            </Link>
          }
        />
      ) : null}

      {status === "loading" || total > 0 ? (
        <>
          <FilterBar
            label="回收站筛选"
            activeCount={hasFilter ? (typeFilter !== "all" ? 1 : 0) + (query.trim() ? 1 : 0) : 0}
            onReset={hasFilter ? resetFilter : undefined}
          >
            <div className="page-toolbar__search">
              <Input
                value={query}
                placeholder={messages.trashSearchPlaceholder}
                aria-label={messages.trashSearchPlaceholder}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setPage(1);
                }}
              />
            </div>
            <div className="page-toolbar__select">
              <Select
                aria-label={messages.trashColumnType}
                value={typeFilter}
                options={typeOptions}
                onChange={(event) => {
                  setTypeFilter(event.target.value);
                  setPage(1);
                }}
              />
            </div>
          </FilterBar>

          <Table
            caption="回收站列表"
            columns={columns}
            rows={rows}
            loading={status === "loading"}
            empty={emptyNode}
            rowKey={(index) => filtered[index]?.root_id ?? String(index)}
          />

          {status === "ready" && total > 0 ? (
            <Pagination page={page} pageSize={PAGE_SIZE} total={total} onChange={setPage} />
          ) : null}
        </>
      ) : null}

      {dialog?.kind === "restore" ? (
        <RestoreTrashDialog
          item={dialog.item}
          onClose={() => setDialog(null)}
          onChanged={() => void load()}
        />
      ) : null}
      {dialog?.kind === "purge" ? (
        <PurgeTrashDialog item={dialog.item} onClose={() => setDialog(null)} onChanged={() => void load()} />
      ) : null}
      {dialog?.kind === "clear" ? (
        <ClearTrashDialog count={total} onClose={() => setDialog(null)} onChanged={() => void load()} />
      ) : null}
    </div>
  );
}
