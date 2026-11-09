"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

import { CharacterDeleteDialog } from "@/components/characters/character-delete-dialog";
import {
  CharacterFormFields,
  createCharacterFormValues,
  toCharacterNameInputs,
  toCharacterSystemRefs,
  useCharacterSystemFields,
  type CharacterFormValues,
} from "@/components/characters/character-form-fields";
import { CharacterNameChips } from "@/components/characters/character-name-chips";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { ApiError } from "@/lib/api/client";
import { getCharacter, updateCharacter, type CharacterDetail, type CharacterSystemRef } from "@/lib/api/characters";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { undoTrash } from "@/lib/api/trash";
import { useUnsavedGuard } from "@/lib/hooks/use-unsaved-guard";
import { messages } from "@/lib/messages";

type DetailStatus = "loading" | "ready" | "notfound" | "error";

export type CharacterDetailPageProps = {
  characterID: string;
};

function detailToForm(detail: CharacterDetail): CharacterFormValues {
  return {
    name: detail.name,
    names: detail.names.map((name) => ({ kind: name.kind, value: name.value })),
    description: detail.description ?? "",
    systemRefs: Object.fromEntries(
      detail.system_refs.map((ref) => [ref.category_id, ref.node.id]),
    ),
  };
}

function versionFromError(error: unknown): number | null {
  if (!(error instanceof ApiError)) return null;
  const details = error.details as { current_version?: unknown } | null | undefined;
  return typeof details?.current_version === "number" ? details.current_version : null;
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return messages.characterNoValue;
  return date.toLocaleString("zh-CN", { hour12: false });
}

function nodePath(ref: CharacterSystemRef): string {
  return ref.node.parent ? `${ref.node.parent.name} · ${ref.node.name}` : ref.node.name;
}

function isNotFound(error: unknown): boolean {
  return isApiCode(error, "NOT_FOUND");
}

/** 详情页：只读档案、内联编辑、乐观锁冲突与删除撤销（PG-11 §6.2）。 */
export function CharacterDetailPage({ characterID }: CharacterDetailPageProps) {
  const router = useRouter();
  const { showToast } = useToast();
  const guard = useUnsavedGuard();
  const { dirty, markDirty, requestLeave, reset, confirmOpen, keepEditing, discardAndLeave } = guard;
  const [detail, setDetail] = useState<CharacterDetail | null>(null);
  const [status, setStatus] = useState<DetailStatus>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [editing, setEditing] = useState(false);
  const [values, setValues] = useState<CharacterFormValues>(createCharacterFormValues);
  const [initialValues, setInitialValues] = useState<CharacterFormValues | null>(null);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [hasConflict, setHasConflict] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleted, setDeleted] = useState(false);

  const formDirty = useMemo(
    () => Boolean(initialValues) && JSON.stringify(values) !== JSON.stringify(initialValues),
    [values, initialValues],
  );

  const { categories, status: systemStatus } = useCharacterSystemFields(
    detail?.work_id ?? null,
    Boolean(detail),
  );

  useEffect(() => {
    let active = true;
    setStatus("loading");
    setError(null);
    getCharacter(characterID)
      .then((result) => {
        if (!active) return;
        setDetail(result);
        setDeleted(false);
        setStatus("ready");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setDetail(null);
        if (isNotFound(cause)) {
          setStatus("notfound");
          return;
        }
        setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [characterID, reloadTick]);

  useEffect(() => {
    markDirty(editing && formDirty);
  }, [editing, formDirty, markDirty]);

  useEffect(() => {
    if (!dirty) return;
    function handleBeforeUnload(event: BeforeUnloadEvent) {
      event.preventDefault();
      event.returnValue = "";
    }
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [dirty]);

  useEffect(() => {
    if (!dirty) return;
    function handleLinkClick(event: MouseEvent) {
      if (event.defaultPrevented || event.button !== 0) return;
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      const anchor = (event.target as Element | null)?.closest("a[href]");
      if (!(anchor instanceof HTMLAnchorElement)) return;
      const href = anchor.getAttribute("href");
      if (!href || href.startsWith("#")) return;
      if (/^[a-z]+:/i.test(href)) return;
      event.preventDefault();
      requestLeave(() => router.push(href));
    }
    document.addEventListener("click", handleLinkClick, true);
    return () => document.removeEventListener("click", handleLinkClick, true);
  }, [dirty, requestLeave, router]);

  const realmRef = useMemo(() => {
    if (!detail) return null;
    const builtin = categories.find((category) => category.isBuiltin);
    if (!builtin) return null;
    return detail.system_refs.find((ref) => ref.category_id === builtin.id) ?? null;
  }, [detail, categories]);

  const startEdit = useCallback(() => {
    if (!detail) return;
    const next = detailToForm(detail);
    setInitialValues(next);
    setValues(next);
    setFormError(null);
    setHasConflict(false);
    setEditing(true);
  }, [detail]);

  function cancelEdit() {
    setEditing(false);
    setInitialValues(null);
    setFormError(null);
    setHasConflict(false);
    reset();
  }

  function refreshAfterConflict() {
    requestLeave(() => {
      cancelEdit();
      setReloadTick((value) => value + 1);
    });
  }

  async function save() {
    if (!detail) return;
    const name = values.name.trim();
    if (!name) {
      setFormError(messages.characterNameRequired);
      return;
    }
    if (values.names.some((item) => !item.value.trim())) {
      setFormError(messages.characterNameValueRequired);
      return;
    }
    setSaving(true);
    setFormError(null);
    setHasConflict(false);
    try {
      const updated = await updateCharacter(detail.id, {
        name,
        names: toCharacterNameInputs(values),
        description: values.description.trim() ? values.description : null,
        systemRefs: toCharacterSystemRefs(values),
        version: detail.version,
      });
      setDetail(updated);
      setEditing(false);
      setInitialValues(null);
      reset();
      showToast({ tone: "success", message: messages.characterUpdated(updated.name) });
    } catch (cause) {
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        const current = versionFromError(cause);
        setHasConflict(true);
        setFormError(messages.characterVersionConflict(current));
      } else {
        setFormError(toUserMessage(cause, messages.saveFailed));
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleUndo(token: string, name: string) {
    try {
      await undoTrash(token);
      setDeleted(false);
      setReloadTick((value) => value + 1);
      showToast({ tone: "success", message: messages.characterUndone(name) });
    } catch (cause) {
      showToast({ tone: "error", message: toUserMessage(cause, messages.loadFailed) });
    }
  }

  if (status === "loading") {
    return (
      <div className="page">
        <PageHeader title={messages.characterTitle} />
        <Skeleton variant="lines" lines={8} label={messages.characterLoading} />
      </div>
    );
  }

  if (status === "notfound") {
    return (
      <div className="page">
        <PageHeader title={messages.characterTitle} />
        <EmptyState
          variant="page"
          title={messages.characterNotFoundTitle}
          description={messages.characterNotFoundDescription}
          action={
            <Link className="ui-button ui-button--primary ui-button--md" href="/characters">
              <span className="ui-button__label">{messages.characterBackToList}</span>
            </Link>
          }
        />
      </div>
    );
  }

  if (status === "error" || !detail) {
    return (
      <div className="page">
        <PageHeader title={messages.characterTitle} />
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
            <Link className="ui-button ui-button--secondary ui-button--md" href="/characters">
              <span className="ui-button__label">{messages.characterBackToList}</span>
            </Link>
          }
        />
        <EmptyState
          variant="page"
          title={messages.characterDeletedTitle}
          description={messages.characterDeletedDescription}
          action={
            <Link className="ui-button ui-button--secondary ui-button--md" href="/trash">
              <span className="ui-button__label">{messages.characterGoTrash}</span>
            </Link>
          }
        />
      </div>
    );
  }

  return (
    <div className="page">
      <PageHeader
        title={editing ? messages.characterEditTitle : detail.name}
        description={messages.characterDetailDescription}
        actions={
          editing ? undefined : (
            <>
              <Button variant="secondary" onClick={startEdit}>
                {messages.characterEdit}
              </Button>
              <Button variant="danger" onClick={() => setDeleteOpen(true)}>
                {messages.characterDelete}
              </Button>
            </>
          )
        }
      />

      {editing ? (
        <section className="character-detail__card" aria-label={messages.characterEditTitle}>
          {formError ? (
            <Alert
              tone="danger"
              title={formError}
              action={
                hasConflict ? (
                  <Button variant="secondary" size="sm" onClick={refreshAfterConflict}>
                    {messages.characterRefresh}
                  </Button>
                ) : undefined
              }
            />
          ) : null}
          <CharacterFormFields
            values={values}
            onChange={setValues}
            disabled={saving}
            categories={categories}
            systemStatus={systemStatus}
          />
          <div className="character-edit-actions">
            <Button
              variant="secondary"
              onClick={() => requestLeave(cancelEdit)}
              disabled={saving}
            >
              {messages.cancel}
            </Button>
            <Button onClick={save} loading={saving} disabled={!formDirty || !values.name.trim()}>
              {messages.characterSaveSubmit}
            </Button>
          </div>
        </section>
      ) : (
        <div className="character-detail">
          <div className="character-detail__main">
            <section className="character-detail__card" aria-labelledby="character-identity-title">
              <div className="character-detail__identity">
                <div className="character-detail__title-row">
                  <h2 id="character-identity-title" className="character-detail__title">
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
                <CharacterNameChips names={detail.names} />
              </div>

              <dl className="character-detail__facts">
                <div className="character-detail__fact">
                  <dt>{messages.characterColumnRealm}</dt>
                  <dd>{realmRef ? nodePath(realmRef) : messages.characterNoValue}</dd>
                </div>
                <div className="character-detail__fact">
                  <dt>{messages.characterColumnFaction}</dt>
                  <dd>{messages.characterNoValue}</dd>
                </div>
                <div className="character-detail__fact">
                  <dt>{messages.characterColumnRanking}</dt>
                  <dd>{messages.characterNoValue}</dd>
                </div>
                <div className="character-detail__fact">
                  <dt>{messages.characterLocationTitle}</dt>
                  <dd>{messages.characterNoValue}</dd>
                </div>
              </dl>

              <div className="character-image-placeholder" role="img" aria-label={messages.characterImagePlaceholder}>
                {messages.characterImagePlaceholder}
              </div>
            </section>

            <section className="character-detail__card" aria-labelledby="character-relations-title">
              <h2 id="character-relations-title" className="character-detail__section-title">
                {messages.characterRelationsTitle}
              </h2>

              <div className="character-relation-group">
                <h3 className="character-relation-group__title">{messages.characterSystemRefsTitle}</h3>
                {detail.system_refs.length === 0 ? (
                  <p className="character-relation-empty">{messages.characterSystemRefsEmpty}</p>
                ) : (
                  <ul className="character-relation-list">
                    {detail.system_refs.map((ref) => (
                      <li key={ref.category_id} className="character-relation-item">
                        <span className="character-relation-item__label">{ref.category_name}</span>
                        <span className="character-relation-item__value">{nodePath(ref)}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>

              <div className="character-relation-grid">
                <div className="character-relation-group">
                  <h3 className="character-relation-group__title">{messages.characterFactionTitle}</h3>
                  <p className="character-relation-empty">{messages.characterFactionEmpty}</p>
                </div>
                <div className="character-relation-group">
                  <h3 className="character-relation-group__title">{messages.characterLocationTitle}</h3>
                  <p className="character-relation-empty">{messages.characterLocationEmpty}</p>
                </div>
                <div className="character-relation-group">
                  <h3 className="character-relation-group__title">{messages.characterRankingTitle}</h3>
                  <p className="character-relation-empty">{messages.characterRankingEmpty}</p>
                </div>
              </div>
            </section>
          </div>

          <aside className="character-detail__aside" aria-labelledby="character-archive-title">
            <h2 id="character-archive-title" className="character-detail__section-title">
              {messages.characterArchiveTitle}
            </h2>
            <div className="character-archive-block">
              <h3 className="character-relation-group__title">{messages.characterDescriptionLabel}</h3>
              <p className="character-description-body">
                {detail.description?.trim() ? detail.description : messages.characterDescriptionEmpty}
              </p>
            </div>
            <dl className="character-archive-meta">
              <div>
                <dt>{messages.characterCreatedAt}</dt>
                <dd>{formatTime(detail.created_at)}</dd>
              </div>
              <div>
                <dt>{messages.characterUpdatedAt}</dt>
                <dd>{formatTime(detail.updated_at)}</dd>
              </div>
            </dl>
            <Link className="character-back-link" href="/characters">
              {messages.characterBackToList}
            </Link>
          </aside>
        </div>
      )}

      {deleteOpen ? (
        <CharacterDeleteDialog
          character={detail}
          onClose={() => setDeleteOpen(false)}
          onDeleted={(result) => {
            setDeleted(true);
            showToast({
              tone: "undo",
              message: messages.characterMovedToTrash(detail.name),
              actionLabel: messages.undo,
              onAction: () => void handleUndo(result.undo_token, detail.name),
            });
          }}
          onVersionConflict={() => setReloadTick((value) => value + 1)}
        />
      ) : null}

      {confirmOpen ? (
        <Dialog
          open
          onOpenChange={(next) => {
            if (!next) keepEditing();
          }}
          title={messages.unsavedTitle}
          description={messages.unsavedDescription}
          size="sm"
          footer={
            <>
              <Button variant="secondary" onClick={keepEditing}>
                {messages.unsavedKeep}
              </Button>
              <Button variant="danger" onClick={discardAndLeave}>
                {messages.unsavedDiscard}
              </Button>
            </>
          }
        />
      ) : null}
    </div>
  );
}
