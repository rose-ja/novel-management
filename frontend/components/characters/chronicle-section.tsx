"use client";

import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { toUserMessage } from "@/lib/api/errors";
import {
  createChronicle,
  deleteChronicle,
  listChronicles,
  updateChronicle,
  type Chronicle,
} from "@/lib/api/chronicles";
import { undoTrash } from "@/lib/api/trash";
import { messages } from "@/lib/messages";

const maxLength = 2000;

type ChronicleSectionProps = {
  characterID: string;
};

type Status = "loading" | "ready" | "error";

/** 人物大事记：时间线、行内增改、删除轻确认与 8 秒撤销（PG-03 §4.3）。 */
export function ChronicleSection({ characterID }: ChronicleSectionProps) {
  const { showToast } = useToast();
  const [status, setStatus] = useState<Status>("loading");
  const [items, setItems] = useState<Chronicle[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [draftText, setDraftText] = useState("");
  const [draftTime, setDraftTime] = useState("");
  const [draftError, setDraftError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [editingID, setEditingID] = useState<string | null>(null);
  const [editText, setEditText] = useState("");
  const [editTime, setEditTime] = useState("");
  const [editError, setEditError] = useState<string | null>(null);
  const [confirmID, setConfirmID] = useState<string | null>(null);
  const [busyID, setBusyID] = useState<string | null>(null);

  const load = useCallback(async () => {
    setStatus("loading");
    setLoadError(null);
    try {
      setItems(await listChronicles("character", characterID));
      setStatus("ready");
    } catch (cause) {
      setLoadError(toUserMessage(cause, messages.chronicleLoadFailed));
      setStatus("error");
    }
  }, [characterID]);

  useEffect(() => {
    void load();
  }, [load]);

  function validate(text: string): string | null {
    if (!text.trim()) return messages.chronicleTextRequired;
    if (text.trim().length > maxLength) return messages.chronicleTextTooLong;
    return null;
  }

  async function submitNew() {
    const message = validate(draftText);
    if (message) {
      setDraftError(message);
      return;
    }
    setSaving(true);
    setDraftError(null);
    try {
      await createChronicle({
        ownerType: "character",
        ownerID: characterID,
        text: draftText.trim(),
        timeLabel: draftTime.trim() || null,
      });
      setDraftText("");
      setDraftTime("");
      setAdding(false);
      await load();
    } catch (cause) {
      setDraftError(toUserMessage(cause, messages.chronicleSaveFailed));
    } finally {
      setSaving(false);
    }
  }

  function startEdit(item: Chronicle) {
    setEditingID(item.id);
    setEditText(item.text);
    setEditTime(item.time_label ?? "");
    setEditError(null);
    setConfirmID(null);
  }

  async function submitEdit(item: Chronicle) {
    const message = validate(editText);
    if (message) {
      setEditError(message);
      return;
    }
    setSaving(true);
    setEditError(null);
    try {
      await updateChronicle(item.id, {
        text: editText.trim(),
        timeLabel: editTime.trim() || null,
        version: item.version,
      });
      setEditingID(null);
      await load();
    } catch (cause) {
      setEditError(toUserMessage(cause, messages.chronicleSaveFailed));
    } finally {
      setSaving(false);
    }
  }

  async function remove(item: Chronicle) {
    setBusyID(item.id);
    setConfirmID(null);
    try {
      const result = await deleteChronicle(item.id);
      setItems((current) => current.filter((entry) => entry.id !== item.id));
      showToast({
        tone: "undo",
        message: messages.chronicleDeleted,
        actionLabel: messages.undo,
        onAction: async () => {
          try {
            await undoTrash(result.undo_token);
            await load();
            showToast({ tone: "success", message: messages.chronicleUndone });
          } catch (cause) {
            showToast({ tone: "error", message: toUserMessage(cause, messages.chronicleDeleteFailed) });
          }
        },
      });
    } catch (cause) {
      showToast({ tone: "error", message: toUserMessage(cause, messages.chronicleDeleteFailed) });
      await load();
    } finally {
      setBusyID(null);
    }
  }

  return (
    <section className="character-detail__card chronicle" aria-labelledby="chronicle-title">
      <div className="chronicle__header">
        <h2 id="chronicle-title" className="character-detail__section-title">
          {messages.chronicleTitle}
        </h2>
        {!adding && status === "ready" && items.length > 0 ? (
          <Button variant="secondary" size="sm" onClick={() => setAdding(true)}>
            {messages.chronicleAdd}
          </Button>
        ) : null}
      </div>

      {status === "loading" ? <Skeleton variant="lines" lines={3} label={messages.chronicleTitle} /> : null}

      {status === "error" ? (
        <Alert
          tone="danger"
          title={loadError ?? messages.chronicleLoadFailed}
          action={
            <Button variant="secondary" size="sm" onClick={() => void load()}>
              {messages.retry}
            </Button>
          }
        />
      ) : null}

      {status === "ready" && items.length === 0 && !adding ? (
        <div className="chronicle__empty">
          <h3>{messages.chronicleEmptyTitle}</h3>
          <p>{messages.chronicleEmptyDescription}</p>
          <Button variant="secondary" size="sm" onClick={() => setAdding(true)}>
            {messages.chronicleAddFirst}
          </Button>
        </div>
      ) : null}

      {items.length > 0 ? (
        <ol className="chronicle__list">
          {items.map((item, index) => (
            <li key={item.id} className="chronicle__item">
              <span className="chronicle__index" aria-hidden="true">
                {String(index + 1).padStart(2, "0")}
              </span>
              {editingID === item.id ? (
                <div className="chronicle__form">
                  <label className="chronicle__field">
                    <span>{messages.chronicleTimeLabel}</span>
                    <input
                      className="ui-input"
                      value={editTime}
                      maxLength={200}
                      placeholder={messages.chronicleTimePlaceholder}
                      onChange={(event) => setEditTime(event.target.value)}
                      disabled={saving}
                    />
                  </label>
                  <label className="chronicle__field">
                    <span>{messages.chronicleTextLabel}</span>
                    <textarea
                      className="ui-input chronicle__textarea"
                      value={editText}
                      onChange={(event) => setEditText(event.target.value)}
                      disabled={saving}
                      aria-invalid={Boolean(editError)}
                    />
                  </label>
                  <div className="chronicle__meta">
                    <span>{messages.chronicleCount(editText.trim().length, maxLength)}</span>
                  </div>
                  {editError ? <Alert tone="danger" title={editError} /> : null}
                  <div className="chronicle__actions">
                    <Button variant="secondary" size="sm" onClick={() => setEditingID(null)} disabled={saving}>
                      {messages.cancel}
                    </Button>
                    <Button size="sm" onClick={() => void submitEdit(item)} loading={saving}>
                      {messages.chronicleSave}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="chronicle__content">
                  <div className="chronicle__meta">
                    {item.time_label ? <span className="chronicle__time">{item.time_label}</span> : null}
                  </div>
                  <p className="chronicle__text">{item.text}</p>
                  {confirmID === item.id ? (
                    <div className="chronicle__confirm" role="group" aria-label={messages.chronicleDeleteTitle}>
                      <span>{messages.chronicleDeleteTitle}</span>
                      <Button variant="ghost" size="sm" onClick={() => setConfirmID(null)}>
                        {messages.cancel}
                      </Button>
                      <Button
                        variant="danger"
                        size="sm"
                        loading={busyID === item.id}
                        onClick={() => void remove(item)}
                      >
                        {messages.chronicleDelete}
                      </Button>
                    </div>
                  ) : (
                    <div className="chronicle__actions">
                      <Button variant="ghost" size="sm" onClick={() => startEdit(item)} disabled={busyID === item.id}>
                        {messages.chronicleEdit}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setConfirmID(item.id)}
                        disabled={busyID === item.id}
                      >
                        {messages.chronicleDelete}
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </li>
          ))}
        </ol>
      ) : null}

      {adding ? (
        <div className="chronicle__form chronicle__form--new">
          <label className="chronicle__field">
            <span>{messages.chronicleTimeLabel}</span>
            <input
              className="ui-input"
              value={draftTime}
              maxLength={200}
              placeholder={messages.chronicleTimePlaceholder}
              onChange={(event) => setDraftTime(event.target.value)}
              disabled={saving}
            />
          </label>
          <label className="chronicle__field">
            <span>{messages.chronicleTextLabel}</span>
            <textarea
              className="ui-input chronicle__textarea"
              value={draftText}
              placeholder={messages.chronicleTextPlaceholder}
              onChange={(event) => setDraftText(event.target.value)}
              disabled={saving}
              aria-invalid={Boolean(draftError)}
            />
          </label>
          <div className="chronicle__meta">
            <span>{messages.chronicleCount(draftText.trim().length, maxLength)}</span>
          </div>
          {draftError ? <Alert tone="danger" title={draftError} /> : null}
          <div className="chronicle__actions">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setAdding(false);
                setDraftError(null);
              }}
              disabled={saving}
            >
              {messages.cancel}
            </Button>
            <Button size="sm" onClick={() => void submitNew()} loading={saving}>
              {messages.chronicleSave}
            </Button>
          </div>
        </div>
      ) : null}
    </section>
  );
}
