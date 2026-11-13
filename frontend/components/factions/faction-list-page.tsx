"use client";

import Link from "next/link";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { FactionInlineCreate } from "@/components/factions/faction-inline-create";
import { FactionTree } from "@/components/factions/faction-tree";
import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { FilterBar } from "@/components/ui/filter-bar";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { toUserMessage } from "@/lib/api/errors";
import { listFactions, type FactionSummary } from "@/lib/api/factions";
import { listWorks } from "@/lib/api/works";
import {
  defaultExpandedIDs,
  factionPath,
  indexFactions,
} from "@/lib/factions-tree";
import { messages } from "@/lib/messages";

type ListStatus = "idle" | "loading" | "ready" | "error";

/** 势力列表（PG-04）：扁平全量取数，前端组树、搜索与默认展开两级。 */
export function FactionListPage() {
  const { settings, status: settingsStatus, error: settingsError, reload } = useSettings();
  const { showToast } = useToast();
  const workID = settings.current_work_id;
  const [workName, setWorkName] = useState<string | null>(null);
  const [items, setItems] = useState<FactionSummary[]>([]);
  const [status, setStatus] = useState<ListStatus>("idle");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [query, setQuery] = useState("");
  const [pendingOnly, setPendingOnly] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [creating, setCreating] = useState<{ parentID: string | null } | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const initializedRef = useRef<string | null>(null);

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
    setQuery("");
    setPendingOnly(false);
    setExpanded(new Set());
    setCreating(null);
    setSelectedID(null);
    initializedRef.current = null;
    setItems([]);
  }, [workID]);

  useEffect(() => {
    if (!workID) {
      setItems([]);
      setStatus("idle");
      setError(null);
      return;
    }
    let active = true;
    setStatus("loading");
    setError(null);
    listFactions({ workID })
      .then((result) => {
        if (!active) return;
        setItems(result.items);
        setStatus("ready");
        if (initializedRef.current !== workID) {
          setExpanded(defaultExpandedIDs(result.items));
          initializedRef.current = workID;
        }
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setItems([]);
        setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [workID, reloadTick]);

  const index = useMemo(
    () => indexFactions(items, { query, pendingOnly }),
    [items, query, pendingOnly],
  );

  // 搜索命中时自动展开祖先，滚动到首个命中。
  useEffect(() => {
    if (!query.trim() || index.matches.size === 0) return;
    setExpanded((current) => {
      const next = new Set(current);
      for (const id of index.matches) {
        for (const ancestor of factionPath(items, id).slice(0, -1)) next.add(ancestor.id);
      }
      return next;
    });
  }, [index, query, items]);

  useEffect(() => {
    if (!query.trim() || index.matches.size === 0) return;
    const handle = window.requestAnimationFrame(() => {
      document.querySelector('[data-match="true"]')?.scrollIntoView({ block: "center" });
    });
    return () => window.cancelAnimationFrame(handle);
  }, [index, query]);

  const toggle = useCallback((id: string) => {
    setExpanded((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const resetFilters = useCallback(() => {
    setQuery("");
    setPendingOnly(false);
  }, []);

  const filtered = query.trim() !== "" || pendingOnly;
  const activeCount = (query.trim() ? 1 : 0) + (pendingOnly ? 1 : 0);
  const description =
    settingsStatus === "ready" && workID
      ? `${messages.factionDescription(workName ?? messages.factionWorkFallback)} · ${messages.factionTotal(items.length)}`
      : undefined;

  function expandAll() {
    setExpanded(new Set(items.map((item) => item.id)));
  }

  function collapseAll() {
    setExpanded(new Set());
  }

  function handleCreated(created: FactionSummary) {
    setCreating(null);
    setSelectedID(created.id);
    setExpanded((current) => {
      const next = new Set(current);
      let parent = created.parent?.id ?? null;
      while (parent) {
        next.add(parent);
        parent = items.find((item) => item.id === parent)?.parent?.id ?? null;
      }
      return next;
    });
    setReloadTick((value) => value + 1);
    showToast({ tone: "success", message: messages.factionCreated(created.name) });
  }

  return (
    <div className="page">
      <PageHeader
        title={messages.factionTitle}
        description={description}
        actions={
          <Button
            onClick={() => setCreating({ parentID: null })}
            disabled={!workID || Boolean(creating)}
          >
            {messages.factionAdd}
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
          <FilterBar onReset={resetFilters} activeCount={activeCount} label={messages.factionTitle}>
            <div className="page-toolbar__search">
              <Input
                value={query}
                maxLength={100}
                placeholder={messages.factionSearchPlaceholder}
                aria-label={messages.factionSearchLabel}
                onChange={(event) => setQuery(event.target.value)}
              />
            </div>
            <div className="character-filter-toggle">
              <input
                id="faction-pending-filter"
                type="checkbox"
                checked={pendingOnly}
                onChange={(event) => setPendingOnly(event.target.checked)}
              />
              <label htmlFor="faction-pending-filter">{messages.factionFilterPending}</label>
            </div>
            <Button variant="ghost" size="sm" onClick={expandAll}>
              {messages.factionExpandAll}
            </Button>
            <Button variant="ghost" size="sm" onClick={collapseAll}>
              {messages.factionCollapseAll}
            </Button>
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

          {creating ? (
            <FactionInlineCreate
              key={creating.parentID ?? "root"}
              workID={workID}
              items={items}
              defaultParentID={creating.parentID}
              onCancel={() => setCreating(null)}
              onCreated={handleCreated}
            />
          ) : null}

          {status === "loading" ? (
            <div aria-busy="true">
              <Skeleton variant="lines" lines={8} label={messages.factionTreeAria} />
            </div>
          ) : null}

          {status === "ready" && items.length === 0 && !creating ? (
            <EmptyState
              variant="page"
              title={messages.factionEmptyTitle}
              description={messages.factionEmptyDescription}
              action={<Button onClick={() => setCreating({ parentID: null })}>{messages.factionAdd}</Button>}
            />
          ) : null}

          {status === "ready" && filtered && index.roots.length === 0 ? (
            <EmptyState
              variant="inline"
              title={messages.factionFilterEmptyTitle}
              description={messages.factionFilterEmptyDescription}
              action={
                <Button variant="secondary" size="sm" onClick={resetFilters}>
                  {messages.factionClearFilter}
                </Button>
              }
            />
          ) : null}

          {status === "ready" && index.roots.length > 0 ? (
            <FactionTree
              index={index}
              expanded={expanded}
              query={query}
              selectedID={selectedID}
              onToggle={toggle}
              onAddChild={(id) => setCreating({ parentID: id })}
            />
          ) : null}
        </>
      ) : null}
    </div>
  );
}
