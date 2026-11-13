"use client";

import Link from "next/link";
import { useState } from "react";

import { referenceConflictItems } from "@/components/system/reference-dialog";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { ApiError } from "@/lib/api/client";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { deleteFaction, type DeleteFactionResult, type FactionDetail } from "@/lib/api/factions";
import { messages } from "@/lib/messages";

type BlockItem = {
  id: string;
  name: string;
  href: string;
  inTrash?: boolean;
};

type HasChildrenDetails = {
  items?: { id?: string; name?: string; in_trash?: boolean }[];
};

function hasChildrenItems(error: unknown): BlockItem[] {
  if (!(error instanceof ApiError) || !error.details || typeof error.details !== "object") return [];
  const details = error.details as HasChildrenDetails;
  if (!Array.isArray(details.items)) return [];
  return details.items
    .filter((item): item is { id: string; name: string; in_trash?: boolean } =>
      Boolean(item.id && item.name),
    )
    .map((item) => ({
      id: item.id,
      name: item.name,
      // 软删下级已不在活跃树，改为回收站定位，避免详情 404。
      href: item.in_trash ? "/trash" : `/factions?id=${encodeURIComponent(item.id)}`,
      inTrash: item.in_trash === true,
    }));
}

export type FactionDeleteDialogProps = {
  faction: FactionDetail;
  onClose: () => void;
  onDeleted: (result: DeleteFactionResult) => void;
  onVersionConflict: () => void;
};

/** 删除确认：HAS_CHILDREN 列出下级势力，REFERENCE_CONFLICT 列出成员人物并支持跳转。 */
export function FactionDeleteDialog({
  faction,
  onClose,
  onDeleted,
  onVersionConflict,
}: FactionDeleteDialogProps) {
  const [error, setError] = useState<string | null>(null);
  const [blocks, setBlocks] = useState<BlockItem[]>([]);
  const [versionConflict, setVersionConflict] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    setBlocks([]);
    setVersionConflict(false);
    try {
      const result = await deleteFaction(faction.id, faction.version);
      onDeleted(result);
      onClose();
    } catch (cause) {
      if (isApiCode(cause, "HAS_CHILDREN")) {
        setBlocks(hasChildrenItems(cause));
        setError(messages.factionDeleteHasChildren);
      } else if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        setBlocks(
          referenceConflictItems(cause).map((item) => ({
            id: item.entity_id,
            name: item.name ?? messages.systemReferenceUnnamed,
            href: `/characters?id=${encodeURIComponent(item.entity_id)}`,
            inTrash: item.in_trash,
          })),
        );
        setError(messages.factionDeleteReference);
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
      title={messages.factionDeleteTitle(faction.name)}
      description={messages.factionDeleteDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting}>
            {messages.factionDeleteSubmit}
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
                {messages.factionRefresh}
              </Button>
            ) : undefined
          }
        />
      ) : null}
      {blocks.length > 0 ? (
        <ul className="character-reference-list">
          {blocks.map((item) => (
            <li key={item.id}>
              <Link className="character-name-link" href={item.href}>
                {item.name}
              </Link>
              {item.inTrash ? <span className="character-muted">（回收站）</span> : null}
            </li>
          ))}
        </ul>
      ) : null}
    </Dialog>
  );
}
