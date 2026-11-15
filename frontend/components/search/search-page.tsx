"use client";

import { SearchX } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Pagination } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError } from "@/lib/api/client";
import { toUserMessage } from "@/lib/api/errors";
import { listFactions } from "@/lib/api/factions";
import { listLocations } from "@/lib/api/locations";
import { listRankings } from "@/lib/api/rankings";
import { searchEntities, type SearchResult } from "@/lib/api/search";
import { listCategories, listNodes } from "@/lib/api/system";
import { getWork } from "@/lib/api/works";
import { messages } from "@/lib/messages";
import {
  buildSearchParams,
  EMPTY_SEARCH_STATE,
  filtersApplicable,
  hasSearchCriteria,
  parseSearchState,
  toSearchOptions,
  type SearchState,
} from "@/lib/search";
import { useSettings } from "@/components/providers/settings-provider";

import { ConditionalFilterPanel, type SearchFilterOptions } from "./conditional-filter-panel";
import { QuickSearchBar } from "./quick-search-bar";
import { SearchModeSwitch } from "./search-mode-switch";
import { SearchResultGroup } from "./search-result-group";

type Status = "idle" | "loading" | "ready" | "error";

const EMPTY_OPTIONS: SearchFilterOptions = {
  factions: [],
  realms: [],
  locations: [],
  rankings: [],
};

const FILTER_FIELDS: Record<
  string,
  { key: "factionID" | "systemNodeID" | "locationID" | "rankingID"; label: string }
> = {
  faction_id: { key: "factionID", label: messages.searchFilterFaction },
  system_category_id: { key: "systemNodeID", label: messages.searchFilterRealm },
  system_node_id: { key: "systemNodeID", label: messages.searchFilterRealm },
  location_id: { key: "locationID", label: messages.searchFilterLocation },
  ranking_id: { key: "rankingID", label: messages.searchFilterRanking },
};

export function SearchPage({ initial }: { initial: SearchState }) {
  const router = useRouter();
  const { settings, status: settingsStatus, error: settingsError, reload: reloadSettings } = useSettings();
  const workID = settings.current_work_id;

  const [state, setState] = useState<SearchState>(initial);
  const [queryInput, setQueryInput] = useState(initial.q);
  const [status, setStatus] = useState<Status>("idle");
  const [result, setResult] = useState<SearchResult | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [notice, setNotice] = useState("");
  const [reloadTick, setReloadTick] = useState(0);
  const [options, setOptions] = useState<SearchFilterOptions>(EMPTY_OPTIONS);
  const [optionsError, setOptionsError] = useState(false);
  const [workName, setWorkName] = useState("");

  const sequence = useRef(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);

  const criteria = hasSearchCriteria(state);
  const applyingFilters = filtersApplicable(state);

  // URL 固化检索态，浏览器返回时由初始参数恢复。
  useEffect(() => {
    const query = buildSearchParams(state).toString();
    router.replace(query ? `/search?${query}` : "/search", { scroll: false });
  }, [router, state]);

  // 快速搜索 300ms 防抖；条件模式等待「检索」。
  useEffect(() => {
    if (state.mode !== "quick") return;
    const timer = window.setTimeout(() => {
      const trimmed = queryInput.trim();
      setState((prev) => (prev.q === trimmed ? prev : { ...prev, q: trimmed, page: 1 }));
    }, 300);
    return () => window.clearTimeout(timer);
  }, [queryInput, state.mode]);

  // 切换作品时清空结果，保留关键词与模式。
  useEffect(() => {
    setResult(null);
    setStatus("idle");
    setError(null);
    setNotice("");
    setState((prev) => ({ ...prev, page: 1 }));
    if (workID) inputRef.current?.focus();
  }, [workID]);

  // 加载筛选选项。
  useEffect(() => {
    if (!workID) {
      setOptions(EMPTY_OPTIONS);
      setOptionsError(false);
      return;
    }
    let active = true;
    setOptionsError(false);
    Promise.all([
      listFactions({ workID }),
      listLocations({ workID }),
      listRankings({ workID, pageSize: 100 }),
      listCategories(workID, false),
    ])
      .then(async ([factions, locations, rankings, categories]) => {
        const builtin = categories.find((category) => category.is_builtin) ?? categories[0];
        const nodes = builtin ? await listNodes(builtin.id, { flat: true }) : [];
        if (!active) return;
        const nodeNames = new Map(nodes.map((node) => [node.id, node.name]));
        setOptions({
          factions: factions.items.map((item) => ({ value: item.id, label: item.name })),
          locations: locations.items.map((item) => ({ value: item.id, label: item.name })),
          rankings: rankings.items.map((item) => ({ value: item.id, label: item.name })),
          realms: nodes.map((node) => ({
            value: node.id,
            label: node.parent_id
              ? `${nodeNames.get(node.parent_id) ?? ""} / ${node.name}`.trim()
              : node.name,
          })),
        });
      })
      .catch(() => {
        if (!active) return;
        setOptionsError(true);
      });
    return () => {
      active = false;
    };
  }, [workID]);

  // 当前作品名称用于页头副标题。
  useEffect(() => {
    if (!workID) {
      setWorkName("");
      return;
    }
    let active = true;
    getWork(workID)
      .then((work) => {
        if (active) setWorkName(work.name);
      })
      .catch(() => {
        if (active) setWorkName("");
      });
    return () => {
      active = false;
    };
  }, [workID]);

  // 执行检索；丢弃过期响应，失效筛选自动移除后重试一次。
  useEffect(() => {
    if (!workID || !criteria) {
      setStatus("idle");
      setResult(null);
      return;
    }
    const controller = new AbortController();
    const id = sequence.current + 1;
    sequence.current = id;
    setStatus("loading");
    setError(null);

    searchEntities({ workID, ...toSearchOptions(state), signal: controller.signal })
      .then((value) => {
        if (sequence.current !== id) return;
        setResult(value);
        setStatus("ready");
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted || sequence.current !== id) return;
        if (cause instanceof ApiError && cause.code === "VALIDATION_ERROR") {
          const field = (cause.details as { field?: unknown } | null | undefined)?.field;
          const target = typeof field === "string" ? FILTER_FIELDS[field] : undefined;
          if (target) {
            setNotice(messages.searchFilterInvalid(target.label));
            setState((prev) => ({ ...prev, [target.key]: "", page: 1 }));
            return;
          }
        }
        setError(cause);
        setStatus("error");
      });

    return () => controller.abort();
  }, [workID, state, criteria, reloadTick]);

  const updateState = useCallback((patch: Partial<SearchState>) => {
    setState((prev) => ({ ...prev, ...patch, page: 1 }));
  }, []);

  const handleClear = useCallback(() => {
    setQueryInput("");
    setState((prev) => ({ ...prev, q: "", page: 1 }));
    inputRef.current?.focus();
  }, []);

  const handleReset = useCallback(() => {
    setQueryInput("");
    setState((prev) => ({ ...EMPTY_SEARCH_STATE, mode: prev.mode }));
    inputRef.current?.focus();
  }, []);

  const handleEnter = useCallback(() => {
    setState((prev) => ({ ...prev, q: queryInput.trim(), page: 1 }));
    setReloadTick((value) => value + 1);
  }, [queryInput]);

  const handleSubmit = useCallback(() => {
    setState((prev) => ({ ...prev, q: queryInput.trim(), page: 1 }));
    setReloadTick((value) => value + 1);
  }, [queryInput]);

  const handlePage = useCallback((page: number) => {
    setState((prev) => ({ ...prev, page }));
    window.requestAnimationFrame(() => {
      resultsRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
    });
  }, []);

  const handleMode = useCallback((mode: SearchState["mode"]) => {
    setState((prev) => ({ ...prev, mode, page: 1 }));
  }, []);

  const summary = useMemo(() => (result ? messages.searchSummary(result.total) : ""), [result]);

  return (
    <div className="page search-page">
      <PageHeader title={messages.searchTitle} description={messages.searchDescription(workName)} />

      {settingsStatus === "loading" ? (
        <Skeleton variant="lines" lines={4} label={messages.workLoading} />
      ) : settingsStatus === "error" ? (
        <Alert
          tone="danger"
          title={messages.loadFailed}
          action={
            <Button variant="secondary" size="sm" onClick={reloadSettings}>
              {messages.retry}
            </Button>
          }
        >
          {toUserMessage(settingsError, messages.loadFailed)}
        </Alert>
      ) : (
        <>
          <QuickSearchBar
            value={queryInput}
            disabled={!workID}
            inputRef={inputRef}
            onChange={setQueryInput}
            onEnter={handleEnter}
            onClear={handleClear}
          />
          <SearchModeSwitch mode={state.mode} disabled={!workID} onChange={handleMode} />

          {!workID ? (
            <EmptyState
              title={messages.searchNoWorkTitle}
              description={messages.searchNoWorkDescription}
              action={
                <Button onClick={() => router.push("/")} variant="secondary">
                  {messages.backHome}
                </Button>
              }
            />
          ) : (
            <>
              {state.mode === "conditional" ? (
                <ConditionalFilterPanel
                  state={state}
                  options={options}
                  optionsError={optionsError}
                  submitting={status === "loading"}
                  submitDisabled={!criteria && queryInput.trim() === ""}
                  onChange={updateState}
                  onReset={handleReset}
                  onSubmit={handleSubmit}
                />
              ) : null}

              {notice ? <Alert tone="warning" title={notice} /> : null}

              <div ref={resultsRef} className="search-results" aria-busy={status === "loading" || undefined}>
                {status === "ready" && result ? (
                  <div className="search-summary" aria-live="polite">
                    <span>{summary}</span>
                    <details className="search-sort-note">
                      <summary>{messages.searchSortLabel}</summary>
                      <p>{messages.searchSortDescription}</p>
                    </details>
                  </div>
                ) : null}

                {!criteria ? (
                  <EmptyState
                    variant="inline"
                    title={messages.searchEmptyTitle}
                    description={messages.searchEmptyDescription}
                  />
                ) : status === "loading" ? (
                  <div className="search-skeleton" aria-label={messages.searchPageResultLabel}>
                    {Array.from({ length: 6 }, (_, index) => (
                      <div key={index} className="search-skeleton__row">
                        <Skeleton variant="block" width="40px" height="40px" />
                        <Skeleton variant="lines" lines={2} />
                      </div>
                    ))}
                  </div>
                ) : status === "error" ? (
                  <Alert
                    tone="danger"
                    title={messages.searchErrorTitle}
                    action={
                      <Button variant="secondary" size="sm" onClick={() => setReloadTick((value) => value + 1)}>
                        {messages.searchRetry}
                      </Button>
                    }
                  >
                    {toUserMessage(error, messages.searchErrorTitle)}
                  </Alert>
                ) : result && result.total === 0 ? (
                  <EmptyState
                    title={messages.searchNoResultTitle}
                    description={messages.searchNoResultDescription}
                    icon={<SearchX aria-hidden="true" size={28} />}
                    action={
                      applyingFilters || state.q ? (
                        <Button variant="secondary" onClick={handleReset}>
                          {messages.searchClearFilters}
                        </Button>
                      ) : undefined
                    }
                  />
                ) : result ? (
                  <div className="search-groups">
                    {result.groups.map((group) => (
                      <SearchResultGroup key={group.entity_type} group={group} />
                    ))}
                  </div>
                ) : null}
              </div>

              {status === "ready" && result && result.total > result.page_size ? (
                <Pagination
                  page={result.page}
                  pageSize={result.page_size}
                  total={result.total}
                  onChange={handlePage}
                />
              ) : null}
            </>
          )}
        </>
      )}
    </div>
  );
}
