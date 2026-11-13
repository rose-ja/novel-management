"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";

import { ChronicleSection } from "@/components/chronicles/chronicle-section";
import { FactionDeleteDialog } from "@/components/factions/faction-delete-dialog";
import { FactionEditForm } from "@/components/factions/faction-edit-form";
import { MemberTable } from "@/components/factions/member-table";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { listCharacters } from "@/lib/api/characters";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { getFaction, listFactions, type FactionDetail, type FactionSummary } from "@/lib/api/factions";
import { undoTrash } from "@/lib/api/trash";
import { factionDescendantIDs, factionPath, toFactionOptions } from "@/lib/factions-tree";
import { messages } from "@/lib/messages";

type DetailStatus = "loading" | "ready" | "notfound" | "error";

export type FactionDetailPageProps = {
  factionID: string;
};

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return messages.factionNoValue;
  return date.toLocaleString("zh-CN", { hour12: false });
}

/** 势力详情（PG-05）：摘要、下级、成员、势力大事记与内联编辑。 */
export function FactionDetailPage({ factionID }: FactionDetailPageProps) {
  const { showToast } = useToast();
  const [detail, setDetail] = useState<FactionDetail | null>(null);
  const [all, setAll] = useState<FactionSummary[]>([]);
  const [leaderOptions, setLeaderOptions] = useState<{ value: string; label: string }[]>([]);
  const [status, setStatus] = useState<DetailStatus>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [editing, setEditing] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleted, setDeleted] = useState(false);

  useEffect(() => {
    let active = true;
    setStatus("loading");
    setError(null);
    getFaction(factionID)
      .then((result) => {
        if (!active) return;
        setDetail(result);
        setDeleted(false);
        setStatus("ready");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setDetail(null);
        if (isApiCode(cause, "NOT_FOUND")) {
          setStatus("notfound");
          return;
        }
        setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [factionID, reloadTick]);

  const workID = detail?.work_id ?? null;

  useEffect(() => {
    if (!workID) return;
    let active = true;
    (async () => {
      const [factions, characters] = await Promise.all([
        listFactions({ workID }),
        listCharacters({ workID, pageSize: 100, sort: "name" }),
      ]);
      if (!active) return;
      setAll(factions.items);
      setLeaderOptions(characters.items.map((item) => ({ value: item.id, label: item.name })));
    })().catch(() => {
      if (active) {
        setAll([]);
        setLeaderOptions([]);
      }
    });
    return () => {
      active = false;
    };
  }, [workID, reloadTick]);

  const parentOptions = useMemo(() => {
    if (!detail) return [];
    const blocked = factionDescendantIDs(all, detail.id);
    return toFactionOptions(all).map((option) => ({
      ...option,
      disabled: blocked.has(option.value),
    }));
  }, [all, detail]);

  const path = useMemo(() => (detail ? factionPath(all, detail.id) : []), [all, detail]);

  const handleUndo = useCallback(
    async (token: string, name: string) => {
      try {
        await undoTrash(token);
        setDeleted(false);
        setReloadTick((value) => value + 1);
        showToast({ tone: "success", message: messages.factionUndone(name) });
      } catch (cause) {
        showToast({ tone: "error", message: toUserMessage(cause, messages.loadFailed) });
      }
    },
    [showToast],
  );

  if (status === "loading") {
    return (
      <div className="page">
        <PageHeader title={messages.factionTitle} />
        <Skeleton variant="lines" lines={8} label={messages.factionTreeAria} />
      </div>
    );
  }

  if (status === "notfound") {
    return (
      <div className="page">
        <PageHeader title={messages.factionTitle} />
        <EmptyState
          variant="page"
          title={messages.factionNotFoundTitle}
          description={messages.factionNotFoundDescription}
          action={
            <Link className="ui-button ui-button--primary ui-button--md" href="/factions">
              <span className="ui-button__label">{messages.factionBackToList}</span>
            </Link>
          }
        />
      </div>
    );
  }

  if (status === "error" || !detail) {
    return (
      <div className="page">
        <PageHeader title={messages.factionTitle} />
        <Alert
          tone="danger"
          title={toUserMessage(error, messages.loadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={() => setReloadTick((value) => value + 1)}>
              {messages.retry}
            </Button>
          }
        />
      </div>
    );
  }

  if (deleted) {
    return (
      <div className="page">
        <PageHeader
          title={detail.name}
          actions={
            <Link className="ui-button ui-button--secondary ui-button--md" href="/factions">
              <span className="ui-button__label">{messages.factionBackToList}</span>
            </Link>
          }
        />
        <EmptyState
          variant="page"
          title={messages.factionDeletedTitle}
          description={messages.factionDeletedDescription}
          action={
            <Link className="ui-button ui-button--secondary ui-button--md" href="/trash">
              <span className="ui-button__label">{messages.factionGoTrash}</span>
            </Link>
          }
        />
      </div>
    );
  }

  return (
    <div className="page">
      <PageHeader
        title={editing ? messages.factionEditTitle : detail.name}
        description={messages.factionDetailDescription}
        actions={
          editing ? undefined : (
            <>
              <Button variant="secondary" onClick={() => setEditing(true)}>
                {messages.characterEdit}
              </Button>
              <Button variant="danger" onClick={() => setDeleteOpen(true)}>
                {messages.factionDelete}
              </Button>
            </>
          )
        }
      />

      {path.length > 0 ? (
        <nav className="faction-breadcrumb" aria-label={messages.factionTreeAria}>
          <Link href="/factions">{messages.factionTitle}</Link>
          {path.map((item) => (
            <span key={item.id} className="faction-breadcrumb__item">
              <span aria-hidden="true">/</span>
              <Link href={`/factions?id=${encodeURIComponent(item.id)}`}>{item.name}</Link>
            </span>
          ))}
        </nav>
      ) : null}

      {editing ? (
        <FactionEditForm
          faction={detail}
          parentOptions={parentOptions}
          leaderOptions={leaderOptions}
          onCancel={() => setEditing(false)}
          onSaved={(updated) => {
            setDetail(updated);
            setEditing(false);
            setReloadTick((value) => value + 1);
            showToast({ tone: "success", message: messages.factionUpdated(updated.name) });
          }}
          onRefresh={() => {
            setEditing(false);
            setReloadTick((value) => value + 1);
          }}
        />
      ) : (
        <div className="character-detail">
          <div className="character-detail__main">
            <section className="character-detail__card" aria-labelledby="faction-identity-title">
              <div className="character-detail__title-row">
                <h2 id="faction-identity-title" className="character-detail__title">
                  {detail.name}
                </h2>
                {detail.review_status === "pending" ? (
                  <Badge tone="warning" outlined>
                    {messages.characterReviewPending}
                  </Badge>
                ) : (
                  <Badge tone="success" outlined>
                    {messages.characterReviewConfirmed}
                  </Badge>
                )}
              </div>
              <dl className="faction-summary">
                <div className="faction-summary__item">
                  <dt>{messages.factionSummaryLeader}</dt>
                  <dd>
                    {detail.leader ? (
                      <Link className="character-name-link" href={`/characters?id=${encodeURIComponent(detail.leader.id)}`}>
                        {detail.leader.name || messages.characterNoValue}
                      </Link>
                    ) : (
                      messages.factionNoValue
                    )}
                  </dd>
                </div>
                <div className="faction-summary__item">
                  <dt>{messages.factionSummaryDirect}</dt>
                  <dd>{detail.member_count}</dd>
                </div>
                <div className="faction-summary__item">
                  <dt>{messages.factionSummarySubtree}</dt>
                  <dd>{detail.relations.subtree_member_count}</dd>
                </div>
                <div className="faction-summary__item">
                  <dt>{messages.factionSummaryChildren}</dt>
                  <dd>{detail.children_count}</dd>
                </div>
              </dl>
              <p className="character-description-body">
                {detail.description?.trim() ? detail.description : messages.factionDescriptionEmpty}
              </p>
            </section>

            <section className="character-detail__card" aria-labelledby="faction-relations-title">
              <h2 id="faction-relations-title" className="character-detail__section-title">
                {messages.factionRelationsTitle}
              </h2>
              <div className="character-relation-grid">
                <div className="character-relation-group">
                  <h3 className="character-relation-group__title">{messages.factionChildrenTitle}</h3>
                  {detail.relations.children.length === 0 ? (
                    <p className="character-relation-empty">{messages.factionChildrenEmpty}</p>
                  ) : (
                    <ul className="character-relation-list">
                      {detail.relations.children.map((child) => (
                        <li key={child.id} className="character-relation-item">
                          <Link className="character-name-link" href={`/factions?id=${encodeURIComponent(child.id)}`}>
                            {child.name}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
                <div className="character-relation-group">
                  <h3 className="character-relation-group__title">{messages.factionLocationsTitle}</h3>
                  <p className="character-relation-empty">
                    {detail.relations.locations.length === 0
                      ? messages.factionLocationsEmpty
                      : detail.relations.locations.map((item) => item.name).join("、")}
                  </p>
                </div>
              </div>
            </section>

            <MemberTable
              factionID={detail.id}
              hasChildren={detail.children_count > 0}
              refreshKey={detail.version}
            />

            <ChronicleSection
              ownerType="faction"
              ownerID={detail.id}
              title={messages.factionChronicleTitle}
              emptyDescription={messages.factionChronicleEmptyDescription}
            />
          </div>

          <aside className="character-detail__aside" aria-labelledby="faction-archive-title">
            <h2 id="faction-archive-title" className="character-detail__section-title">
              {messages.factionArchiveTitle}
            </h2>
            <dl className="character-archive-meta">
              <div>
                <dt>{messages.factionCreatedAt}</dt>
                <dd>{formatTime(detail.created_at)}</dd>
              </div>
              <div>
                <dt>{messages.factionUpdatedAt}</dt>
                <dd>{formatTime(detail.updated_at)}</dd>
              </div>
            </dl>
            {detail.parent ? (
              <p className="character-relation-empty">
                {messages.factionParentValue(detail.parent.name)}
              </p>
            ) : null}
            <Link className="character-back-link" href="/factions">
              {messages.factionBackToList}
            </Link>
          </aside>
        </div>
      )}

      {deleteOpen ? (
        <FactionDeleteDialog
          faction={detail}
          onClose={() => setDeleteOpen(false)}
          onDeleted={(result) => {
            setDeleted(true);
            showToast({
              tone: "undo",
              message: messages.factionMovedToTrash(detail.name),
              actionLabel: messages.undo,
              onAction: () => void handleUndo(result.undo_token, detail.name),
            });
          }}
          onVersionConflict={() => setReloadTick((value) => value + 1)}
        />
      ) : null}
    </div>
  );
}
