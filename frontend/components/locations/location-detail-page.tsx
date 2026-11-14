"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import { LocationDeleteDialog } from "@/components/locations/location-delete-dialog";
import { LocationEditForm } from "@/components/locations/location-edit-form";
import { LocationRelations } from "@/components/locations/location-relations";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { listCharacters } from "@/lib/api/characters";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { listFactions } from "@/lib/api/factions";
import {
  getLocation,
  listLocations,
  type DeleteLocationResult,
  type LocationDetail,
  type LocationSummary,
} from "@/lib/api/locations";
import { undoTrash } from "@/lib/api/trash";
import { toFactionOptions } from "@/lib/factions-tree";
import {
  locationDescendantIDs,
  locationPath,
  locationTypeOptions,
  toLocationOptions,
} from "@/lib/locations-tree";
import { messages } from "@/lib/messages";

type DetailStatus = "loading" | "ready" | "notfound" | "error";

export type LocationDetailPageProps = {
  locationID: string;
};

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return messages.locationNoValue;
  return date.toLocaleString("zh-CN", { hour12: false });
}

/** 地点详情（PG-07）：摘要、下级、关联人物/势力、内联编辑与删除阻断。 */
export function LocationDetailPage({ locationID }: LocationDetailPageProps) {
  const { showToast } = useToast();
  const [detail, setDetail] = useState<LocationDetail | null>(null);
  const [all, setAll] = useState<LocationSummary[]>([]);
  const [factionOptions, setFactionOptions] = useState<{ value: string; label: string }[]>([]);
  const [characterOptions, setCharacterOptions] = useState<{ value: string; label: string }[]>([]);
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
    getLocation(locationID)
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
  }, [locationID, reloadTick]);

  const workID = detail?.work_id ?? null;

  useEffect(() => {
    if (!workID) return;
    let active = true;
    (async () => {
      const [locations, factions, characters] = await Promise.all([
        listLocations({ workID }),
        listFactions({ workID }),
        listCharacters({ workID, pageSize: 100, sort: "name" }),
      ]);
      if (!active) return;
      setAll(locations.items);
      setFactionOptions(toFactionOptions(factions.items));
      setCharacterOptions(characters.items.map((item) => ({ value: item.id, label: item.name })));
    })().catch(() => {
      if (active) {
        setAll([]);
        setFactionOptions([]);
        setCharacterOptions([]);
      }
    });
    return () => {
      active = false;
    };
  }, [workID]);

  const parentOptions = useMemo(() => {
    if (!detail) return [];
    const blocked = locationDescendantIDs(all, detail.id);
    return toLocationOptions(all).map((option) => ({
      ...option,
      disabled: blocked.has(option.value),
    }));
  }, [all, detail]);

  const path = useMemo(() => (detail ? locationPath(all, detail.id) : []), [all, detail]);
  const typeSuggestions = useMemo(() => locationTypeOptions(all), [all]);

  function handleDeleted(result: DeleteLocationResult) {
    setDeleted(true);
    showToast({
      tone: "success",
      message: messages.locationMovedToTrash(detail?.name ?? ""),
      actionLabel: messages.undo,
      onAction: async () => {
          try {
            await undoTrash(result.undo_token);
            setDeleted(false);
            setReloadTick((value) => value + 1);
            showToast({ tone: "success", message: messages.locationUndone(detail?.name ?? "") });
          } catch (cause) {
            showToast({ tone: "error", message: toUserMessage(cause, messages.loadFailed) });
          }
      },
    });
  }

  if (status === "loading") {
    return (
      <div className="page" aria-busy="true">
        <Skeleton variant="lines" lines={8} label={messages.characterLoading} />
      </div>
    );
  }

  if (status === "notfound") {
    return (
      <div className="page">
        <EmptyState
          variant="page"
          title={messages.locationNotFoundTitle}
          description={messages.locationNotFoundDescription}
          action={
            <Link className="ui-button ui-button--primary ui-button--md" href="/locations">
              <span className="ui-button__label">{messages.locationBackToList}</span>
            </Link>
          }
        />
      </div>
    );
  }

  if (status === "error" || !detail) {
    return (
      <div className="page">
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

  return (
    <div className="page">
      <PageHeader
        title={detail.name}
        description={messages.locationDetailDescription}
        actions={
          deleted ? null : (
            <>
              <Button variant="secondary" onClick={() => setEditing(true)} disabled={editing}>
                {messages.characterEdit}
              </Button>
              <Button variant="danger" onClick={() => setDeleteOpen(true)}>
                {messages.locationDelete}
              </Button>
            </>
          )
        }
      />

      {path.length > 0 ? (
        <nav className="faction-breadcrumb" aria-label={messages.locationTreeAria}>
          <Link href="/locations">{messages.locationTitle}</Link>
          {path.map((item) => (
            <span key={item.id} className="faction-breadcrumb__item">
              <span aria-hidden="true">/</span>
              <Link href={`/locations?id=${encodeURIComponent(item.id)}`}>{item.name}</Link>
            </span>
          ))}
        </nav>
      ) : null}

      {deleted ? (
        <Alert tone="success" title={messages.locationDeletedTitle}>
          {messages.locationDeletedDescription}
        </Alert>
      ) : null}

      {editing ? (
        <LocationEditForm
          location={detail}
          parentOptions={parentOptions}
          typeSuggestions={typeSuggestions}
          onCancel={() => setEditing(false)}
          onSaved={(updated) => {
            setDetail(updated);
            setEditing(false);
            setReloadTick((value) => value + 1);
            showToast({ tone: "success", message: messages.locationUpdated(updated.name) });
          }}
          onRefresh={() => {
            setEditing(false);
            setReloadTick((value) => value + 1);
          }}
        />
      ) : (
        <div className="character-detail">
          <div className="character-detail__main">
            <section className="character-detail__card" aria-labelledby="location-identity-title">
              <div className="character-detail__title-row">
                <h2 id="location-identity-title" className="character-detail__title">
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
                  <dt>{messages.locationSummaryType}</dt>
                  <dd>{detail.type_text || messages.locationNoValue}</dd>
                </div>
                <div className="faction-summary__item">
                  <dt>{messages.locationSummaryChildren}</dt>
                  <dd>{detail.children_count}</dd>
                </div>
                <div className="faction-summary__item">
                  <dt>{messages.locationSummaryCharacters}</dt>
                  <dd>{detail.characters_count}</dd>
                </div>
                <div className="faction-summary__item">
                  <dt>{messages.locationSummaryFactions}</dt>
                  <dd>{detail.factions_count}</dd>
                </div>
              </dl>
              <p className="character-description-body">
                {detail.description?.trim() ? detail.description : messages.locationDescriptionEmpty}
              </p>
            </section>

            <section className="character-detail__card" aria-labelledby="location-children-title">
              <h2 id="location-children-title" className="character-detail__section-title">
                {messages.locationChildrenTitle}
              </h2>
              {detail.relations.children.length === 0 ? (
                <p className="character-relation-empty">{messages.locationChildrenEmpty}</p>
              ) : (
                <ul className="character-relation-list">
                  {detail.relations.children.map((child) => (
                    <li key={child.id} className="character-relation-item">
                      <Link
                        className="character-name-link"
                        href={`/locations?id=${encodeURIComponent(child.id)}`}
                      >
                        {child.name}
                      </Link>
                      <span className="character-muted">
                        {messages.locationCharactersLabel(child.characters_count)} ·{" "}
                        {messages.locationFactionsLabel(child.factions_count)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <LocationRelations
              locationID={detail.id}
              characters={detail.relations.characters}
              factions={detail.relations.factions}
              characterOptions={characterOptions}
              factionOptions={factionOptions}
              onChanged={() => setReloadTick((value) => value + 1)}
            />
          </div>

          <aside className="character-detail__aside" aria-labelledby="location-archive-title">
            <h2 id="location-archive-title" className="character-detail__section-title">
              {messages.locationArchiveTitle}
            </h2>
            <dl className="character-archive-meta">
              <div>
                <dt>{messages.locationCreatedAt}</dt>
                <dd>{formatTime(detail.created_at)}</dd>
              </div>
              <div>
                <dt>{messages.locationUpdatedAt}</dt>
                <dd>{formatTime(detail.updated_at)}</dd>
              </div>
            </dl>
            {detail.parent ? (
              <p className="character-relation-empty">
                {messages.locationParentValue(detail.parent.name)}
              </p>
            ) : null}
            <Link className="character-back-link" href="/locations">
              {messages.locationBackToList}
            </Link>
          </aside>
        </div>
      )}

      {deleteOpen ? (
        <LocationDeleteDialog
          location={detail}
          onClose={() => setDeleteOpen(false)}
          onDeleted={handleDeleted}
          onVersionConflict={() => setReloadTick((value) => value + 1)}
        />
      ) : null}
    </div>
  );
}
