"use client";

import { useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { conflictItems, isApiCode, toUserMessage } from "@/lib/api/errors";
import { purgeAllTrash, purgeTrashItem, restoreTrash, type TrashItem } from "@/lib/api/trash";
import { messages } from "@/lib/messages";

type Callbacks = {
  onClose: () => void;
  onChanged: () => void;
};

/** 恢复：无冲突直接确认；遇到重名时切换为改名输入，原记录不被覆盖。 */
export function RestoreTrashDialog({ item, onClose, onChanged }: Callbacks & { item: TrashItem }) {
  const { reload } = useSettings();
  const { showToast } = useToast();
  const [conflict, setConflict] = useState(false);
  const [newName, setNewName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    try {
      const result = await restoreTrash(item.root_type, item.root_id, {
        newName: conflict ? newName.trim() : null,
      });
      reload();
      onChanged();
      onClose();
      showToast({ tone: "success", message: messages.trashRestored(item.root_type, result.name) });
    } catch (cause) {
      if (isApiCode(cause, "RESTORE_CONFLICT")) {
        const conflicts = conflictItems(cause);
        setConflict(true);
        setNewName((current) => current || `${item.name}（恢复）`);
        setError(
          conflicts.length > 0
            ? `${conflicts.map((conflictItem) => `「${conflictItem.name}」`).join("、")}已存在，请修改恢复后名称`
            : toUserMessage(cause, messages.saveFailed),
        );
      } else if (isApiCode(cause, "PARENT_DELETED") && item.root_type === "faction") {
        setError(messages.trashFactionParentDeleted);
      } else {
        setError(toUserMessage(cause, messages.saveFailed));
      }
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
      title={
        conflict ? messages.trashRestoreConflictTitle : messages.trashRestoreTitle(item.root_type, item.name)
      }
      description={
        conflict
          ? messages.trashRestoreConflictDescription
          : messages.trashRestoreDescription(item.root_type)
      }
      size={conflict ? "lg" : "sm"}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={submit} loading={submitting} disabled={conflict && !newName.trim()}>
            {messages.trashRestoreSubmit}
          </Button>
        </>
      }
    >
      {error ? <Alert tone="danger" title={error} /> : null}
      {conflict ? (
        <FormField id="trash-restore-name" label={messages.trashRestoreNewNameLabel} required>
          <Input
            id="trash-restore-name"
            value={newName}
            maxLength={200}
            autoFocus
            disabled={submitting}
            onChange={(event) => setNewName(event.target.value)}
          />
        </FormField>
      ) : null}
    </Dialog>
  );
}

/** 永久删除单条：二次确认 + 自动快照；快照失败时阻止并可重试。 */
export function PurgeTrashDialog({ item, onClose, onChanged }: Callbacks & { item: TrashItem }) {
  const { showToast } = useToast();
  const [error, setError] = useState<string | null>(null);
  const [snapshotFailure, setSnapshotFailure] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    setSnapshotFailure(false);
    try {
      await purgeTrashItem(item.root_type, item.root_id);
      onChanged();
      onClose();
      showToast({ tone: "success", message: messages.trashPurged(item.root_type, item.name) });
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
      title={messages.trashPurgeTitle(item.root_type, item.name)}
      description={messages.trashPurgeDescription(item.root_type)}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting}>
            {messages.trashPurgeSubmit}
          </Button>
        </>
      }
    >
      {error ? (
        <Alert
          tone="danger"
          title={snapshotFailure ? messages.trashSnapshotFailed : error}
          action={
            snapshotFailure ? (
              <Button variant="secondary" size="sm" onClick={submit}>
                {messages.retry}
              </Button>
            ) : undefined
          }
        />
      ) : null}
    </Dialog>
  );
}

/** 清空回收站：显示总数量、不可撤销与自动快照。 */
export function ClearTrashDialog({ count, onClose, onChanged }: Callbacks & { count: number }) {
  const { showToast } = useToast();
  const [error, setError] = useState<string | null>(null);
  const [snapshotFailure, setSnapshotFailure] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    setSnapshotFailure(false);
    try {
      await purgeAllTrash();
      onChanged();
      onClose();
      showToast({ tone: "success", message: messages.trashCleared });
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
      title={messages.trashClearTitle}
      description={messages.trashClearDescription(count)}
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting} disabled={count === 0}>
            {messages.trashClear}
          </Button>
        </>
      }
    >
      {error ? (
        <Alert
          tone="danger"
          title={snapshotFailure ? messages.trashSnapshotFailed : error}
          action={
            snapshotFailure ? (
              <Button variant="secondary" size="sm" onClick={submit}>
                {messages.retry}
              </Button>
            ) : undefined
          }
        />
      ) : null}
    </Dialog>
  );
}
