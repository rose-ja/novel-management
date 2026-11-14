"use client";

import Link from "next/link";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { LocationInlineCreate } from "@/components/locations/location-inline-create";
import { LocationTree } from "@/components/locations/location-tree";
import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { FilterBar } from "@/components/ui/filter-bar";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { toUserMessage } from "@/lib/api/errors";
import { listLocations, type LocationSummary } from "@/lib/api/locations";
import { listWorks } from "@/lib/api/works";
import {
  defaultExpandedIDs,
  indexLocations,
  locationPath,
  locationTypeOptions,
} from "@/lib/locations-tree";
import { messages } from "@/lib/messages";

type ListStatus = "idle" | "loading" | "ready" | "error";

/** 地点列表（PG-06）：扁平全量取数，前端组树、搜索与默认展开两级。 */
export function LocationListPage() {
  const { settings, status: settingsStatus, error: settingsError, reload } = useSettings();
  const { showToast } = useToast();
  const workID = settings.current_work_id;
  const [workName, setWorkName] = useState<string | null>(null);
  const [items, setItems] = useState<LocationSummary[]>([]);
  const [status, setStatus] = useState<ListStatus>("idle");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
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
    setQueryInput("");
    setQuery("");
    setTypeFilter("");
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
    listLocations({ workID })
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

  // 详情页撤销删除后通知列表刷新，避免停留在旧快照。
  useEffect(() => {
    const handle = () => setReloadTick((value) => value + 1);
    window.addEventListener("locations:changed", handle);
    return () => window.removeEventListener("locations:changed", handle);
  }, []);

  // 本地过滤做 300ms 防抖，避免逐键重建树。
  useEffect(() => {
    const handle = window.setTimeout(() => setQuery(queryInput), 300);
    return () => window.clearTimeout(handle);
  }, [queryInput]);

  const index = useMemo(
    () => indexLocations(items, { query, typeText: typeFilter, pendingOnly }),
    [items, query, typeFilter, pendingOnly],
  );

  // 搜索命中时自动展开祖先，滚动到首个命中。
  useEffect(() => {
    if (!query.trim() || index.matches.size === 0) return;
    setExpanded((current) => {
      const next = new Set(current);
      for (const id of index.matches) {
        for (const ancestor of locationPath(items, id).slice(0, -1)) next.add(ancestor.id);
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
    setQueryInput("");
    setQuery("");
    setTypeFilter("");
    setPendingOnly(false);
  }, []);

  const typeOptions = useMemo(
    () => [
      { value: "", label: messages.locationTypeAll },
      ...locationTypeOptions(items).map((value) => ({ value, label: value })),
    ],
    [items],
  );

  const filtered = query.trim() !== "" || typeFilter !== "" || pendingOnly;
  const activeCount =
    (query.trim() ? 1 : 0) + (typeFilter ? 1 : 0) + (pendingOnly ? 1 : 0);
  const description =
    settingsStatus === "ready" && workID
      ? `${messages.locationDescription(workName ?? messages.locationWorkFallback)} · ${messages.locationTotal(items.length)}`
      : undefined;

  function expandAll() {
    setExpanded(new Set(items.map((item) => item.id)));
  }

  function collapseAll() {
    setExpanded(new Set());
  }

  function handleCreated(created: LocationSummary) {
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
    showToast({ tone: "success", message: messages.locationCreated(created.name) });
  }

  return (
    <div className="page">
      <PageHeader
        title={messages.locationTitle}
        description={description}
        actions={
          <Button
            onClick={() => setCreating({ parentID: null })}
            disabled={!workID || Boolean(creating)}
          >
            {messages.locationAdd}
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
          <FilterBar onReset={resetFilters} activeCount={activeCount} label={messages.locationTitle}>
            <div className="page-toolbar__search">
              <Input
                value={queryInput}
                maxLength={100}
                placeholder={messages.locationSearchPlaceholder}
                aria-label={messages.locationSearchLabel}
                onChange={(event) => setQueryInput(event.target.value)}
              />
            </div>
            <Select
              aria-label={messages.locationTypeLabel}
              value={typeFilter}
              options={typeOptions}
              onChange={(event) => setTypeFilter(event.target.value)}
            />
            <div className="character-filter-toggle">
              <input
                id="location-pending-filter"
                type="checkbox"
                checked={pendingOnly}
                onChange={(event) => setPendingOnly(event.target.checked)}
              />
              <label htmlFor="location-pending-filter">{messages.locationFilterPending}</label>
            </div>
            <Button variant="ghost" size="sm" onClick={expandAll}>
              {messages.locationExpandAll}
            </Button>
            <Button variant="ghost" size="sm" onClick={collapseAll}>
              {messages.locationCollapseAll}
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
            <LocationInlineCreate
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
              <Skeleton variant="lines" lines={8} label={messages.locationTreeAria} />
            </div>
          ) : null}

          {status === "ready" && items.length === 0 && !creating ? (
            <EmptyState
              variant="page"
              title={messages.locationEmptyTitle}
              description={messages.locationEmptyDescription}
              action={<Button onClick={() => setCreating({ parentID: null })}>{messages.locationAdd}</Button>}
            />
          ) : null}

          {status === "ready" && filtered && index.roots.length === 0 ? (
            <EmptyState
              variant="inline"
              title={messages.locationFilterEmptyTitle}
              description={messages.locationFilterEmptyDescription}
              action={
                <Button variant="secondary" size="sm" onClick={resetFilters}>
                  {messages.locationClearFilter}
                </Button>
              }
            />
          ) : null}

          {status === "ready" && index.roots.length > 0 ? (
            <LocationTree
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
