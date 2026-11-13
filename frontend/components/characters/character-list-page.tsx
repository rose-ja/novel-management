"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { CharacterCreateDrawer } from "@/components/characters/character-create-drawer";
import { flattenNodeOptions } from "@/components/characters/character-form-fields";
import { CharacterFilters, type CharacterFilterValues } from "@/components/characters/character-filters";
import { CharacterTable } from "@/components/characters/character-table";
import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Pagination } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { listCharacters, type CharacterPage } from "@/lib/api/characters";
import { toUserMessage } from "@/lib/api/errors";
import { listCategories, listNodes } from "@/lib/api/system";
import { listWorks } from "@/lib/api/works";
import { listFactions } from "@/lib/api/factions";
import { toFactionOptions } from "@/lib/factions-tree";
import { messages } from "@/lib/messages";

type ListStatus = "idle" | "loading" | "ready" | "error";

const DEFAULT_FILTERS: CharacterFilterValues = {
  q: "",
  systemNodeID: "",
  factionID: "",
  pendingOnly: false,
  sort: "-updated_at",
  pageSize: 20,
};

/** 人物列表页：工作区隔离、筛选/排序/分页与新建 Drawer（PG-11 §6.1）。 */
export function CharacterListPage() {
  const router = useRouter();
  const { settings, status: settingsStatus, error: settingsError, reload } = useSettings();
  const workID = settings.current_work_id;
  const [workName, setWorkName] = useState<string | null>(null);
  const [filters, setFilters] = useState<CharacterFilterValues>(DEFAULT_FILTERS);
  const [page, setPage] = useState(1);
  const [reloadTick, setReloadTick] = useState(0);
  const [data, setData] = useState<CharacterPage | null>(null);
  const [status, setStatus] = useState<ListStatus>("idle");
  const [error, setError] = useState<Error | null>(null);
  const [realmOptions, setRealmOptions] = useState<{ value: string; label: string }[]>([]);
  const [factionOptions, setFactionOptions] = useState<{ value: string; label: string }[]>([]);
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    if (!workID) {
      setWorkName(null);
      return;
    }
    let active = true;
    listWorks({ pageSize: 100 })
      .then((result) => {
        if (!active) return;
        setWorkName(result.items.find((work) => work.id === workID)?.name ?? null);
      })
      .catch(() => {
        if (active) setWorkName(null);
      });
    return () => {
      active = false;
    };
  }, [workID]);

  useEffect(() => {
    if (!workID) {
      setRealmOptions([]);
      return;
    }
    let active = true;
    (async () => {
      const categories = await listCategories(workID, false);
      const builtin = categories.find((category) => category.is_builtin);
      const nodes = builtin ? await listNodes(builtin.id) : [];
      if (active) setRealmOptions(flattenNodeOptions(nodes));
    })().catch(() => {
      if (active) setRealmOptions([]);
    });
    return () => {
      active = false;
    };
  }, [workID]);

  useEffect(() => {
    if (!workID) {
      setFactionOptions([]);
      return;
    }
    let active = true;
    listFactions({ workID })
      .then((result) => {
        if (active) setFactionOptions(toFactionOptions(result.items));
      })
      .catch(() => {
        if (active) setFactionOptions([]);
      });
    return () => {
      active = false;
    };
  }, [workID]);

  useEffect(() => {
    setFilters(DEFAULT_FILTERS);
    setPage(1);
    setData(null);
    setDrawerOpen(false);
  }, [workID]);

  useEffect(() => {
    if (!workID) {
      setData(null);
      setStatus("idle");
      setError(null);
      return;
    }
    let active = true;
    setStatus("loading");
    setError(null);
    listCharacters({
      workID,
      q: filters.q,
      systemNodeID: filters.systemNodeID || undefined,
      factionID: filters.factionID || undefined,
      reviewStatus: filters.pendingOnly ? "pending" : undefined,
      sort: filters.sort,
      page,
      pageSize: filters.pageSize,
    })
      .then((result) => {
        if (!active) return;
        setData(result);
        setStatus("ready");
        if (result.page !== page) setPage(result.page);
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setData(null);
        setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [workID, filters, page, reloadTick]);

  const updateFilters = useCallback((patch: Partial<CharacterFilterValues>) => {
    setFilters((current) => ({ ...current, ...patch }));
    setPage(1);
  }, []);

  const resetFilters = useCallback(() => {
    setFilters(DEFAULT_FILTERS);
    setPage(1);
  }, []);

  const activeCount = useMemo(() => {
    let count = 0;
    if (filters.q.trim()) count += 1;
    if (filters.systemNodeID) count += 1;
    if (filters.factionID) count += 1;
    if (filters.pendingOnly) count += 1;
    if (filters.sort !== DEFAULT_FILTERS.sort) count += 1;
    if (filters.pageSize !== DEFAULT_FILTERS.pageSize) count += 1;
    return count;
  }, [filters]);

  const filtered =
    filters.q.trim() !== "" || filters.systemNodeID !== "" || filters.factionID !== "" || filters.pendingOnly;
  const description =
    settingsStatus === "ready" && workID
      ? messages.characterDescription(workName ?? messages.characterWorkFallback)
      : undefined;

  return (
    <div className="page">
      <PageHeader
        title={messages.characterTitle}
        description={description}
        actions={
          <Button onClick={() => setDrawerOpen(true)} disabled={!workID}>
            {messages.characterAdd}
          </Button>
        }
      />

      {settingsStatus === "loading" ? (
        <div aria-busy="true">
          <Skeleton variant="lines" lines={5} label={messages.characterLoading} />
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
          <CharacterFilters
            values={filters}
            realmOptions={realmOptions}
            factionOptions={factionOptions}
            onChange={updateFilters}
            onReset={resetFilters}
            activeCount={activeCount}
          />

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
          ) : (
            <>
              <CharacterTable
                items={data?.items ?? []}
                loading={status === "loading"}
                filtered={filtered}
                onCreate={() => setDrawerOpen(true)}
                onClearFilters={resetFilters}
              />

              {data && data.total > 0 ? (
                <Pagination
                  page={data.page}
                  pageSize={data.page_size}
                  total={data.total}
                  onChange={setPage}
                />
              ) : null}
            </>
          )}

          <CharacterCreateDrawer
            open={drawerOpen}
            workID={workID}
            onClose={() => setDrawerOpen(false)}
            onCreated={(created) => {
              router.push(`/characters?id=${created.id}`);
            }}
          />
        </>
      ) : null}
    </div>
  );
}
