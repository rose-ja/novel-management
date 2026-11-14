"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { CreateRankingDialog } from "@/components/rankings/create-ranking-dialog";
import { DeleteRankingDialog } from "@/components/rankings/delete-ranking-dialog";
import { RankingTable } from "@/components/rankings/ranking-table";
import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { FilterBar } from "@/components/ui/filter-bar";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { Pagination } from "@/components/ui/pagination";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { toUserMessage } from "@/lib/api/errors";
import {
  listRankings,
  type DeleteRankingResult,
  type RankingSort,
  type RankingSummary,
} from "@/lib/api/rankings";
import { undoTrash } from "@/lib/api/trash";
import { listWorks } from "@/lib/api/works";
import { messages } from "@/lib/messages";

const PAGE_SIZE = 20;

const sortOptions: { value: RankingSort; label: string }[] = [
  { value: "updated_at:desc", label: messages.rankingSortUpdated },
  { value: "name:asc", label: messages.rankingSortName },
];

type ListStatus = "idle" | "loading" | "ready" | "error";

/** 榜单列表（PG-08）：服务端搜索、筛选、排序与分页。 */
export function RankingListPage() {
  const { settings, status: settingsStatus, error: settingsError, reload } = useSettings();
  const { showToast } = useToast();
  const router = useRouter();
  const workID = settings.current_work_id;
  const [workName, setWorkName] = useState<string | null>(null);
  const [items, setItems] = useState<RankingSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<ListStatus>("idle");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [pendingOnly, setPendingOnly] = useState(false);
  const [sort, setSort] = useState<RankingSort>("updated_at:desc");
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<RankingSummary | null>(null);

  useEffect(() => {
    if (!workID) {
      setWorkName(null);
      return;
    }
    let active = true;
    listWorks({ pageSize: 100 })
      .then((result) => {
        if (active) setWorkName(result.items.find((work) => work.id === workID)?.name ?? null);
      })
      .catch(() => {
        if (active) setWorkName(null);
      });
    return () => {
      active = false;
    };
  }, [workID]);

  useEffect(() => {
    setQueryInput("");
    setQuery("");
    setPendingOnly(false);
    setSort("updated_at:desc");
    setPage(1);
    setItems([]);
    setTotal(0);
  }, [workID]);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setQuery(queryInput);
      setPage(1);
    }, 300);
    return () => window.clearTimeout(handle);
  }, [queryInput]);

  useEffect(() => {
    if (!workID) {
      setItems([]);
      setTotal(0);
      setStatus("idle");
      setError(null);
      return;
    }
    let active = true;
    setStatus("loading");
    setError(null);
    listRankings({
      workID,
      q: query.trim() || undefined,
      reviewStatus: pendingOnly ? "pending" : undefined,
      sort,
      page,
      pageSize: PAGE_SIZE,
    })
      .then((result) => {
        if (!active) return;
        setItems(result.items);
        setTotal(result.total);
        setStatus("ready");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setItems([]);
        setTotal(0);
        setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [workID, query, pendingOnly, sort, page, reloadTick]);

  const resetFilters = useCallback(() => {
    setQueryInput("");
    setQuery("");
    setPendingOnly(false);
    setSort("updated_at:desc");
    setPage(1);
  }, []);

  const filtered = query.trim() !== "" || pendingOnly;
  const activeCount = (query.trim() ? 1 : 0) + (pendingOnly ? 1 : 0);
  const description =
    settingsStatus === "ready" && workID
      ? messages.rankingDescription(workName ?? messages.rankingWorkFallback, total)
      : undefined;

  function handleCreated(created: { id: string; name: string }) {
    setCreating(false);
    showToast({ tone: "success", message: messages.rankingCreated(created.name) });
    router.push(`/rankings?id=${encodeURIComponent(created.id)}`);
  }

  function handleDeleted(result: DeleteRankingResult, name: string) {
    setDeleting(null);
    setReloadTick((value) => value + 1);
    showToast({
      tone: "undo",
      message: messages.rankingMovedToTrash(name),
      actionLabel: messages.undo,
      onAction: () => {
        undoTrash(result.undo_token)
          .then(() => {
            showToast({ tone: "success", message: messages.rankingUndone(name) });
            setReloadTick((value) => value + 1);
          })
          .catch((cause: unknown) => {
            showToast({ tone: "error", message: toUserMessage(cause, messages.saveFailed) });
          });
      },
    });
  }

  const showEmpty = status === "ready" && items.length === 0 && !filtered;

  return (
    <div className="page">
      <PageHeader
        title={messages.rankingTitle}
        description={description}
        actions={
          <Button onClick={() => setCreating(true)} disabled={!workID}>
            {messages.rankingAdd}
          </Button>
        }
      />

      {settingsStatus === "loading" ? (
        <div aria-busy="true">
          <Skeleton variant="lines" lines={6} label={messages.characterLoading} />
        </div>
      ) : null}

      {settingsStatus === "error" ? (
        <Alert
          tone="danger"
          title={toUserMessage(settingsError, messages.loadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={reload}>
              {messages.retry}
            </Button>
          }
        />
      ) : null}

      {settingsStatus === "ready" && !workID ? (
        <EmptyState
          variant="page"
          icon={<BookOpen aria-hidden="true" size={22} />}
          title={messages.noWorkTitle}
          description={messages.noWorkDescription}
          action={
            <Link className="ui-button ui-button--primary ui-button--md" href="/">
              <span className="ui-button__label">{messages.trashEmptyAction}</span>
            </Link>
          }
        />
      ) : null}

      {settingsStatus === "ready" && workID ? (
        <>
          <FilterBar onReset={resetFilters} activeCount={activeCount} label={messages.rankingTitle}>
            <div className="page-toolbar__search">
              <Input
                value={queryInput}
                maxLength={100}
                placeholder={messages.rankingSearchPlaceholder}
                aria-label={messages.rankingSearchLabel}
                onChange={(event) => setQueryInput(event.target.value)}
              />
            </div>
            <Select
              aria-label={messages.rankingSortLabel}
              value={sort}
              options={sortOptions}
              onChange={(event) => {
                setSort(event.target.value as RankingSort);
                setPage(1);
              }}
            />
            <div className="character-filter-toggle">
              <input
                id="ranking-pending-filter"
                type="checkbox"
                checked={pendingOnly}
                onChange={(event) => {
                  setPendingOnly(event.target.checked);
                  setPage(1);
                }}
              />
              <label htmlFor="ranking-pending-filter">{messages.rankingFilterPending}</label>
            </div>
          </FilterBar>

          {status === "error" ? (
            <Alert
              tone="danger"
              title={toUserMessage(error, messages.loadFailed)}
              action={
                <Button variant="secondary" size="sm" onClick={() => setReloadTick((value) => value + 1)}>
                  {messages.retry}
                </Button>
              }
            />
          ) : null}

          {status === "loading" ? (
            <div aria-busy="true">
              <Skeleton variant="lines" lines={8} label={messages.rankingTableCaption} />
            </div>
          ) : null}

          {status === "ready" && showEmpty ? (
            <EmptyState
              variant="page"
              title={messages.rankingEmptyTitle}
              description={messages.rankingEmptyDescription}
              action={<Button onClick={() => setCreating(true)}>{messages.rankingAdd}</Button>}
            />
          ) : null}

          {status === "ready" && (!showEmpty || items.length > 0) ? (
            <>
              <RankingTable
                items={items}
                filtered={filtered}
                onCreate={() => setCreating(true)}
                onClearFilters={resetFilters}
                onDelete={setDeleting}
              />
              {total > PAGE_SIZE ? (
                <Pagination page={page} pageSize={PAGE_SIZE} total={total} onChange={setPage} />
              ) : null}
            </>
          ) : null}
        </>
      ) : null}

      {workID ? (
        <CreateRankingDialog
          open={creating}
          workID={workID}
          onOpenChange={setCreating}
          onCreated={handleCreated}
        />
      ) : null}
      <DeleteRankingDialog
        ranking={deleting ? { id: deleting.id, name: deleting.name, version: deleting.version } : null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        onDeleted={handleDeleted}
      />
    </div>
  );
}
