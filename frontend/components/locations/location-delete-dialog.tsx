"use client";

import Link from "next/link";
import { useState } from "react";

import { referenceConflictItems } from "@/components/system/reference-dialog";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { ApiError } from "@/lib/api/client";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import {
  deleteLocation,
  type DeleteLocationResult,
  type LocationDetail,
} from "@/lib/api/locations";
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
      href: item.in_trash ? "/trash" : `/locations?id=${encodeURIComponent(item.id)}`,
      inTrash: item.in_trash === true,
    }));
}

function referenceHref(type: string, entityID: string): string {
  if (type === "character_location") return `/characters?id=${encodeURIComponent(entityID)}`;
  if (type === "location_faction") return `/factions?id=${encodeURIComponent(entityID)}`;
  return "/trash";
}

export type LocationDeleteDialogProps = {
  location: LocationDetail;
  onClose: () => void;
  onDeleted: (result: DeleteLocationResult) => void;
  onVersionConflict: () => void;
};

/** 删除确认：HAS_CHILDREN 列出下级地点，REFERENCE_CONFLICT 列出人物/势力关系并支持跳转。 */
export function LocationDeleteDialog({
  location,
  onClose,
  onDeleted,
  onVersionConflict,
}: LocationDeleteDialogProps) {
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
      const result = await deleteLocation(location.id, location.version);
      onDeleted(result);
      onClose();
    } catch (cause) {
      if (isApiCode(cause, "HAS_CHILDREN")) {
        setBlocks(hasChildrenItems(cause));
        setError(messages.locationDeleteHasChildren);
      } else if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        setBlocks(
          referenceConflictItems(cause).map((item) => ({
            id: `${item.type}:${item.entity_id}`,
            name: item.name ?? messages.systemReferenceUnnamed,
            href: referenceHref(item.type, item.entity_id),
            inTrash: item.in_trash,
          })),
        );
        setError(messages.locationDeleteReference);
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
      title={messages.locationDeleteTitle(location.name)}
      description={messages.locationDeleteDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting}>
            {messages.locationDeleteSubmit}
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
                {messages.locationRefresh}
              </Button>
            ) : undefined
          }
        />
      ) : null}
      {blocks.length > 0 ? (
        <ul className="character-reference-list">
          {blocks.map((item) => (
            <li key={item.id}>
              {item.inTrash ? (
                <>
                  <span className="character-muted">{item.name}{messages.locationRelationInTrash}</span>
                  <Link className="character-name-link" href="/trash">
                    {messages.locationGoTrash}
                  </Link>
                </>
              ) : (
                <Link className="character-name-link" href={item.href}>
                  {item.name}
                </Link>
              )}
            </li>
          ))}
        </ul>
      ) : null}
    </Dialog>
  );
}
