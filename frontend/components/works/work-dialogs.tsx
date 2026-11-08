"use client";

import { useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { undoTrash } from "@/lib/api/trash";
import { createWork, deleteWork, updateWork, type Work } from "@/lib/api/works";
import { useUnsavedGuard } from "@/lib/hooks/use-unsaved-guard";
import { messages } from "@/lib/messages";

type DialogCallbacks = {
  onClose: () => void;
  onChanged: () => void;
};

/** 未保存修改的离开确认；被作品表单的关闭路径复用。 */
function DiscardDialog({ open, onKeep, onDiscard }: { open: boolean; onKeep: () => void; onDiscard: () => void }) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onKeep();
      }}
      title={messages.unsavedTitle}
      description={messages.unsavedDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onKeep}>
            {messages.unsavedKeep}
          </Button>
          <Button variant="danger" onClick={onDiscard}>
            {messages.unsavedDiscard}
          </Button>
        </>
      }
    />
  );
}

export function CreateWorkDialog({ onClose, onChanged }: DialogCallbacks) {
  const { reload } = useSettings();
  const { showToast } = useToast();
  const guard = useUnsavedGuard();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError("请输入作品名称");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const created = await createWork({ name: trimmed, description: description.trim() ? description : null });
      guard.reset();
      reload();
      onChanged();
      onClose();
      showToast({ tone: "success", message: messages.workCreated(created.name) });
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Dialog
        open
        onOpenChange={(next) => {
          if (!next) guard.requestLeave(onClose);
        }}
        title={messages.workCreateTitle}
        description={messages.workCreateDescription}
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => guard.requestLeave(onClose)} disabled={submitting}>
              {messages.cancel}
            </Button>
            <Button onClick={submit} loading={submitting} disabled={!name.trim()}>
              {messages.workCreateSubmit}
            </Button>
          </>
        }
      >
        {error ? <Alert tone="danger" title={error} /> : null}
        <FormField id="work-create-name" label={messages.workNameLabel} required hint={messages.workNameHint}>
          <Input
            id="work-create-name"
            value={name}
            maxLength={200}
            autoFocus
            disabled={submitting}
            onChange={(event) => {
              setName(event.target.value);
              guard.markDirty(true);
            }}
          />
        </FormField>
        <FormField id="work-create-description" label={messages.workDescriptionLabel} hint={messages.workDescriptionHint}>
          <textarea
            className="ui-textarea"
            value={description}
            maxLength={20000}
            rows={4}
            disabled={submitting}
            onChange={(event) => {
              setDescription(event.target.value);
              guard.markDirty(true);
            }}
          />
        </FormField>
      </Dialog>
      <DiscardDialog open={guard.confirmOpen} onKeep={guard.keepEditing} onDiscard={guard.discardAndLeave} />
    </>
  );
}

export function EditWorkDialog({ work, onClose, onChanged }: DialogCallbacks & { work: Work }) {
  const { reload } = useSettings();
  const { showToast } = useToast();
  const guard = useUnsavedGuard();
  const [name, setName] = useState(work.name);
  const [description, setDescription] = useState(work.description ?? "");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError("请输入作品名称");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const updated = await updateWork(work.id, {
        name: trimmed,
        description: description.trim() ? description : null,
        version: work.version,
      });
      guard.reset();
      reload();
      onChanged();
      onClose();
      showToast({ tone: "success", message: messages.workUpdated(updated.name) });
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Dialog
        open
        onOpenChange={(next) => {
          if (!next) guard.requestLeave(onClose);
        }}
        title={messages.workEditTitle(work.name)}
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => guard.requestLeave(onClose)} disabled={submitting}>
              {messages.cancel}
            </Button>
            <Button onClick={submit} loading={submitting} disabled={!name.trim()}>
              {messages.workSaveSubmit}
            </Button>
          </>
        }
      >
        {error ? <Alert tone="danger" title={error} /> : null}
        <FormField id="work-edit-name" label={messages.workNameLabel} required hint={messages.workNameHint}>
          <Input
            id="work-edit-name"
            value={name}
            maxLength={200}
            autoFocus
            disabled={submitting}
            onChange={(event) => {
              setName(event.target.value);
              guard.markDirty(true);
            }}
          />
        </FormField>
        <FormField id="work-edit-description" label={messages.workDescriptionLabel} hint={messages.workDescriptionHint}>
          <textarea
            className="ui-textarea"
            value={description}
            maxLength={20000}
            rows={4}
            disabled={submitting}
            onChange={(event) => {
              setDescription(event.target.value);
              guard.markDirty(true);
            }}
          />
        </FormField>
      </Dialog>
      <DiscardDialog open={guard.confirmOpen} onKeep={guard.keepEditing} onDiscard={guard.discardAndLeave} />
    </>
  );
}

export function DeleteWorkDialog({ work, onClose, onChanged }: DialogCallbacks & { work: Work }) {
  const { reload } = useSettings();
  const { showToast } = useToast();
  const [confirmName, setConfirmName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [snapshotFailure, setSnapshotFailure] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const matched = confirmName === work.name;

  async function handleUndo(undoToken: string) {
    try {
      await undoTrash(undoToken);
      reload();
      onChanged();
      showToast({ tone: "success", message: messages.workUndone(work.name) });
    } catch (cause) {
      showToast({ tone: "error", message: toUserMessage(cause, messages.loadFailed) });
    }
  }

  async function submit() {
    if (!matched) return;
    setSubmitting(true);
    setError(null);
    setSnapshotFailure(false);
    try {
      const result = await deleteWork(work.id, work.name, work.version);
      reload();
      onChanged();
      onClose();
      showToast({
        tone: "undo",
        message: messages.workMovedToTrash(work.name),
        actionLabel: messages.undo,
        onAction: () => void handleUndo(result.undo_token),
      });
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
      setSnapshotFailure(isApiCode(cause, "SNAPSHOT_FAILED"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(next) => {
        if (!next && !submitting) onClose();
      }}
      title={messages.workDeleteTitle(work.name)}
      description={messages.workDeleteDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting} disabled={!matched}>
            {messages.workDeleteSubmit}
          </Button>
        </>
      }
    >
      {error ? (
        <Alert
          tone="danger"
          title={error}
          action={
            snapshotFailure ? (
              <Button variant="secondary" size="sm" onClick={submit}>
                {messages.retry}
              </Button>
            ) : undefined
          }
        />
      ) : null}
      <FormField id="work-delete-confirm" label={messages.workDeleteConfirmLabel} required hint={messages.workDeleteConfirmHint}>
        <Input
          id="work-delete-confirm"
          value={confirmName}
          autoFocus
          disabled={submitting}
          onChange={(event) => setConfirmName(event.target.value)}
        />
      </FormField>
    </Dialog>
  );
}
