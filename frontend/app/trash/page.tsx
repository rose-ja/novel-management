"use client";

import Link from "next/link";
import { Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

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
import { listTrash, type TrashItem, type TrashPage, type TrashRootType } from "@/lib/api/trash";
import { formatDateTime } from "@/lib/format";
import { messages } from "@/lib/messages";

const PAGE_SIZE = 20;
/** 定位深链时按更大页幅查找目标条目，避免目标不在第 1 页。 */
const HIGHLIGHT_PAGE_SIZE = 100;
const MAX_HIGHLIGHT_PAGES = 10;

type Status = "loading" | "ready" | "error";

type DialogState = { kind: "restore"; item: TrashItem } | { kind: "purge"; item: TrashItem } | { kind: "clear" };

type TypeFilter = "all" | TrashRootType;

const typeOptions = [
  { value: "all", label: messages.trashTypeAll },
  { value: "work", label: messages.trashTypeWork },
  { value: "system_category", label: messages.trashTypeSystemCategory },
  { value: "system_node", label: messages.trashTypeSystemNode },
  { value: "character", label: messages.trashTypeCharacter },
  { value: "faction", label: messages.trashTypeFaction },
  { value: "location", label: messages.trashTypeLocation },
  { value: "ranking", label: messages.trashTypeRanking },
];

export default function TrashPage() {
  const [page, setPage] = useState(1);
  const [data, setData] = useState<TrashPage | null>(null);
  const [allTotal, setAllTotal] = useState(0);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const [query, setQuery] = useState("");
  const [dialog, setDialog] = useState<DialogState | null>(null);
  const [pendingHighlight, setPendingHighlight] = useState<string | null>(null);
  const [highlightId, setHighlightId] = useState<string | null>(null);
  const [highlightMissing, setHighlightMissing] = useState(false);
  const locatedFor = useRef<string | null>(null);

  // 深链参数在客户端解析，兼容静态导出；非法类型忽略。
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const type = params.get("type");
    if (type && typeOptions.some((option) => option.value === type && option.value !== "all")) {
      setTypeFilter(type as TypeFilter);
      setPage(1);
    }
    const highlight = params.get("highlight");
    if (highlight) setPendingHighlight(highlight);
  }, []);

  // 按 100 条/页跨页查找目标；命中后切回 20 条/页并高亮。
  useEffect(() => {
    if (!pendingHighlight || locatedFor.current === pendingHighlight) return;
    locatedFor.current = pendingHighlight;
    let cancelled = false;
    const rootType = typeFilter === "all" ? undefined : typeFilter;
    void (async () => {
      try {
        for (let current = 1; current <= MAX_HIGHLIGHT_PAGES; current++) {
          const result = await listTrash({ rootType, page: current, pageSize: HIGHLIGHT_PAGE_SIZE });
          const index = result.items.findIndex((item) => item.root_id === pendingHighlight);
          if (index >= 0) {
            if (cancelled) return;
            setPage(Math.floor(index / PAGE_SIZE) + 1);
            setHighlightId(pendingHighlight);
            setHighlightMissing(false);
            return;
          }
          if (current * HIGHLIGHT_PAGE_SIZE >= result.total) break;
        }
        if (!cancelled) setHighlightMissing(true);
      } catch {
        if (!cancelled) setHighlightMissing(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [pendingHighlight, typeFilter]);

  // 定位后滚动到目标行并把焦点交给「恢复」按钮，键盘用户可直接操作。
  useEffect(() => {
    if (!highlightId || status !== "ready") return;
    const marker = document.querySelector<HTMLElement>(`[data-trash-highlight="${highlightId}"]`);
    if (!marker) return;
    marker.scrollIntoView({ block: "center" });
    const row = marker.closest("tr");
    row?.querySelector<HTMLElement>("button")?.focus();
  }, [highlightId, status, data]);

  function clearHighlight() {
    setPendingHighlight(null);
    setHighlightId(null);
    setHighlightMissing(false);
    locatedFor.current = null;
  }

  const load = useCallback(async () => {
    setStatus("loading");
    setError(null);
    try {
      const rootType = typeFilter === "all" ? undefined : typeFilter;
      // 类型筛选走服务端 root_type；同时取全量总数，供页头/清空/空态使用。
      const [result, all] = await Promise.all([
        listTrash({ rootType, page, pageSize: PAGE_SIZE }),
        rootType ? listTrash({ page: 1, pageSize: 1 }) : null,
      ]);
      setData(result);
      setAllTotal(all ? all.total : result.total);
      setStatus("ready");
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
      setStatus("error");
    }
  }, [page, typeFilter]);

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
    { key: "parent", label: messages.trashColumnParent, width: "160px" },
    { key: "deleted_at", label: messages.trashColumnDeletedAt, width: "160px" },
    { key: "affected", label: messages.trashColumnAffected, width: "112px" },
    { key: "actions", label: messages.trashColumnActions, width: "160px", align: "right" as const },
  ];

  const rows = filtered.map((item) => [
    <Badge key="type" tone="neutral">
      {messages.trashTypeLabel(item.root_type)}
    </Badge>,
    <span
      key="name"
      className={item.root_id === highlightId ? "trash-highlight" : undefined}
      data-trash-highlight={item.root_id === highlightId ? item.root_id : undefined}
    >
      {messages.trashDisplayName(item.root_type, item.name)}
    </span>,
    item.parent_name ?? "—",
    <span key="deleted_at" className="snapshot-time">
      {formatDateTime(item.deleted_at)}
    </span>,
    messages.trashAffected(item.root_type, item.affected_count),
    <div key="actions" className="table-actions">
      <Button
        variant="ghost"
        size="sm"
        aria-label={messages.trashRestoreAria(item.root_type, item.name)}
        onClick={() => setDialog({ kind: "restore", item })}
      >
        {messages.trashRestore}
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className="table-actions__danger"
        aria-label={messages.trashPurgeAria(item.root_type, item.name)}
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
        description={status === "ready" ? messages.trashDescription(allTotal) : messages.trashTitle}
        actions={
          <Button
            variant="danger"
            size="sm"
            disabled={status === "loading" || allTotal === 0}
            title={allTotal === 0 ? messages.trashEmptyTitle : undefined}
            onClick={() => setDialog({ kind: "clear" })}
          >
            {messages.trashClear}
          </Button>
        }
      />

      <div className="trash-notice">
        <Alert tone="info" title={messages.trashSnapshotNotice} />
      </div>

      {highlightMissing ? (
        <Alert
          tone="warning"
          title={messages.trashHighlightMissing}
          action={
            <Button variant="secondary" size="sm" onClick={clearHighlight}>
              {messages.trashClearHighlight}
            </Button>
          }
        />
      ) : null}

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

      {status === "ready" && allTotal === 0 ? (
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

      {status === "loading" || allTotal > 0 ? (
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
                  setTypeFilter(event.target.value as TypeFilter);
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
        <ClearTrashDialog count={allTotal} onClose={() => setDialog(null)} onChanged={() => void load()} />
      ) : null}
    </div>
  );
}
