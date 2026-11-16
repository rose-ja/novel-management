"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog } from "@/components/ui/dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { listCharacters } from "@/lib/api/characters";
import { listFactions } from "@/lib/api/factions";
import { listLocations } from "@/lib/api/locations";
import { listRankings } from "@/lib/api/rankings";
import { batchReview, confirmReview, type ReviewEntityType } from "@/lib/api/reviews";
import { listCategories, listNodes } from "@/lib/api/system";
import { ApiError, NetworkError } from "@/lib/api/client";
import { messages } from "@/lib/messages";

type PendingItem = { id: string; name: string };

type Group = {
  type: ReviewEntityType;
  label: string;
  items: PendingItem[];
  href: (id: string) => string;
};

function detailHref(base: string) {
  return (id: string) => `${base}?id=${encodeURIComponent(id)}`;
}

function errorText(error: unknown): string {
  if (error instanceof ApiError || error instanceof NetworkError) return error.message;
  return messages.importReviewLoadFailed;
}

export function ReviewStep({ workID }: { workID: string }) {
  const { showToast } = useToast();
  const [groups, setGroups] = useState<Group[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<string | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [confirmed, setConfirmed] = useState(0);
  const [batchGroup, setBatchGroup] = useState<Group | null>(null);
  const [batching, setBatching] = useState(false);
  const [busyID, setBusyID] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    setStatus("loading");
    setError(null);
    async function load() {
      try {
        const [characters, factions, locations, rankings, categories] = await Promise.all([
          listCharacters({ workID, reviewStatus: "pending", pageSize: 200 }),
          listFactions({ workID, reviewStatus: "pending" }),
          listLocations({ workID, reviewStatus: "pending" }),
          listRankings({ workID, reviewStatus: "pending", pageSize: 200 }),
          listCategories(workID),
        ]);
        const nodeLists = await Promise.all(
          categories.map(async (category) => {
            const nodes = await listNodes(category.id, { flat: true, reviewStatus: "pending" });
            return nodes.map((node) => ({ id: node.id, name: node.name }));
          }),
        );
        if (!active) return;
        setGroups([
          {
            type: "characters",
            label: messages.importReviewEntityCharacters,
            items: characters.items.map((item) => ({ id: item.id, name: item.name })),
            href: detailHref("/characters"),
          },
          {
            type: "factions",
            label: messages.importReviewEntityFactions,
            items: factions.items.map((item) => ({ id: item.id, name: item.name })),
            href: detailHref("/factions"),
          },
          {
            type: "locations",
            label: messages.importReviewEntityLocations,
            items: locations.items.map((item) => ({ id: item.id, name: item.name })),
            href: detailHref("/locations"),
          },
          {
            type: "rankings",
            label: messages.importReviewEntityRankings,
            items: rankings.items.map((item) => ({ id: item.id, name: item.name })),
            href: detailHref("/rankings"),
          },
          {
            type: "system-nodes",
            label: messages.importReviewEntitySystemNodes,
            items: nodeLists.flat(),
            href: () => "/system",
          },
        ]);
        setStatus("ready");
      } catch (loadError) {
        if (!active) return;
        setStatus("error");
        setError(errorText(loadError));
      }
    }
    void load();
    return () => {
      active = false;
    };
  }, [workID, reloadKey]);

  const total = useMemo(() => groups.reduce((sum, group) => sum + group.items.length, 0), [groups]);

  const confirmOne = useCallback(
    async (group: Group, item: PendingItem) => {
      setBusyID(item.id);
      try {
        await confirmReview(group.type, item.id);
        setConfirmed((value) => value + 1);
        setReloadKey((value) => value + 1);
        showToast({ tone: "success", message: messages.importBatchDone(1) });
      } catch (confirmError) {
        showToast({ tone: "error", message: errorText(confirmError) });
      } finally {
        setBusyID(null);
      }
    },
    [showToast],
  );

  const submitBatch = useCallback(async () => {
    if (!batchGroup) return;
    setBatching(true);
    try {
      const result = await batchReview({
        entity_type: batchGroup.type,
        ids: batchGroup.items.map((item) => item.id),
      });
      setConfirmed((value) => value + result.updated);
      setBatchGroup(null);
      setReloadKey((value) => value + 1);
      showToast({ tone: "success", message: messages.importBatchDone(result.updated) });
    } catch (batchError) {
      showToast({ tone: "error", message: errorText(batchError) });
    } finally {
      setBatching(false);
    }
  }, [batchGroup, showToast]);

  if (status === "loading") {
    return (
      <Card title={messages.importReviewTitle}>
        <Skeleton variant="lines" lines={6} label={messages.importReviewTitle} />
      </Card>
    );
  }

  if (status === "error") {
    return (
      <Alert
        tone="danger"
        title={error ?? messages.importReviewLoadFailed}
        action={
          <Button variant="secondary" size="sm" onClick={() => setReloadKey((value) => value + 1)}>
            {messages.importReviewRetry}
          </Button>
        }
      />
    );
  }

  if (total === 0) {
    return (
      <EmptyState
        title={messages.importReviewAllDone}
        description={messages.importReviewAllDoneDescription}
        action={
          <Link className="ui-button ui-button--secondary ui-button--md" href="/search">
            {messages.importBackSearch}
          </Link>
        }
      />
    );
  }

  return (
    <Card
      title={messages.importReviewTitle}
      description={`${messages.importReviewPending(total)} · ${messages.importReviewConfirmed(confirmed)}`}
    >
      <div className="import-review">
        {groups.map((group) => {
          const isCollapsed = collapsed[group.type];
          return (
            <section className="import-review__group" key={group.type}>
              <header className="import-review__header">
                <button
                  type="button"
                  className="import-review__toggle"
                  aria-expanded={!isCollapsed}
                  onClick={() => setCollapsed((value) => ({ ...value, [group.type]: !value[group.type] }))}
                >
                  {isCollapsed ? <ChevronRight aria-hidden="true" size={16} /> : <ChevronDown aria-hidden="true" size={16} />}
                  <span>{group.label}</span>
                  <span className="import-muted">{messages.importReviewGroup(group.items.length)}</span>
                </button>
                {group.items.length > 0 ? (
                  <Button variant="secondary" size="sm" onClick={() => setBatchGroup(group)}>
                    {messages.importReviewBatch}
                  </Button>
                ) : null}
              </header>
              {!isCollapsed ? (
                group.items.length === 0 ? (
                  <p className="import-muted">{messages.importReviewEmptyDescription}</p>
                ) : (
                  <ul className="import-review__list">
                    {group.items.map((item) => (
                      <li className="import-review__row" key={item.id}>
                        <span>{item.name}</span>
                        <Badge tone="warning" outlined>
                          {messages.searchPendingReview}
                        </Badge>
                        <div className="import-review__actions">
                          <Button
                            variant="secondary"
                            size="sm"
                            loading={busyID === item.id}
                            onClick={() => void confirmOne(group, item)}
                          >
                            {messages.importReviewConfirm}
                          </Button>
                          <Link className="import-link" href={group.href(item.id)}>
                            {messages.importReviewView}
                          </Link>
                        </div>
                      </li>
                    ))}
                  </ul>
                )
              ) : null}
            </section>
          );
        })}
      </div>

      <Dialog
        open={batchGroup !== null}
        onOpenChange={(open) => {
          if (!open && !batching) setBatchGroup(null);
        }}
        title={messages.importBatchTitle}
        description={batchGroup ? messages.importBatchDescription(batchGroup.items.length) : undefined}
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setBatchGroup(null)} disabled={batching}>
              {messages.cancel}
            </Button>
            <Button onClick={() => void submitBatch()} loading={batching}>
              {messages.importBatchSubmit}
            </Button>
          </>
        }
      />
    </Card>
  );
}
