"use client";

import { useState } from "react";

import { referenceConflictItems } from "@/components/system/reference-dialog";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { deleteCharacter, type CharacterDetail, type DeleteCharacterResult } from "@/lib/api/characters";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { messages } from "@/lib/messages";

export type CharacterDeleteDialogProps = {
  character: CharacterDetail;
  onClose: () => void;
  onDeleted: (result: DeleteCharacterResult) => void;
  onVersionConflict: () => void;
};

/** 删除确认：成功回调撤销令牌；409 引用清单与版本冲突就地提示。 */
export function CharacterDeleteDialog({
  character,
  onClose,
  onDeleted,
  onVersionConflict,
}: CharacterDeleteDialogProps) {
  const [error, setError] = useState<string | null>(null);
  const [references, setReferences] = useState<{ id: string; name: string; inTrash: boolean }[]>([]);
  const [versionConflict, setVersionConflict] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    setReferences([]);
    setVersionConflict(false);
    try {
      const result = await deleteCharacter(character.id, character.version);
      onDeleted(result);
      onClose();
    } catch (cause) {
      if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        setReferences(
          referenceConflictItems(cause).map((item) => ({
            id: item.entity_id,
            name: item.name ?? messages.systemReferenceUnnamed,
            inTrash: item.in_trash,
          })),
        );
        setError(messages.characterDeleteReferenceTitle);
      } else if (isApiCode(cause, "VERSION_CONFLICT")) {
        setVersionConflict(true);
        setError(toUserMessage(cause, messages.saveFailed));
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
      title={messages.characterDeleteTitle(character.name)}
      description={messages.characterDeleteDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting}>
            {messages.characterDeleteSubmit}
          </Button>
        </>
      }
    >
      {error ? (
        <Alert
          tone="danger"
          title={error}
          action={
            versionConflict ? (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  onVersionConflict();
                  onClose();
                }}
              >
                {messages.characterRefresh}
              </Button>
            ) : undefined
          }
        />
      ) : null}
      {references.length > 0 ? (
        <ul className="character-reference-list">
          {references.map((item) => (
            <li key={item.id}>
              <span>{item.name}</span>
              {item.inTrash ? <span className="character-muted">（回收站）</span> : null}
            </li>
          ))}
        </ul>
      ) : null}
    </Dialog>
  );
}
