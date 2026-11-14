"use client";

import Link from "next/link";
import { useState } from "react";

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

type RawItem = {
  id?: string;
  entity_id?: string;
  name?: string;
  entity_type?: string;
  type?: string;
  in_trash?: boolean;
};

type BlockItem = {
  id: string;
  name: string;
  href: string;
  inTrash: boolean;
};

type BlockGroup = {
  key: string;
  title: string;
  items: BlockItem[];
};

const GROUP_TITLES: Record<string, string> = {
  location_child: messages.locationChildrenTitle,
  character_location: messages.locationCharactersTitle,
  location_faction: messages.locationFactionsTitle,
};

function hrefFor(type: string, id: string): string {
  if (type === "character_location") return `/characters?id=${encodeURIComponent(id)}`;
  if (type === "location_faction") return `/factions?id=${encodeURIComponent(id)}`;
  if (type === "location_child") return `/locations?id=${encodeURIComponent(id)}`;
  return "/trash";
}

/** 同时兼容 HAS_CHILDREN（{items}）与 REFERENCE_CONFLICT（数组）两类明细。 */
function parseGroups(error: unknown): BlockGroup[] {
  if (!(error instanceof ApiError) || !error.details || typeof error.details !== "object") return [];
  const details = error.details;
  const items = Array.isArray(details)
    ? (details as RawItem[])
    : ((details as { items?: RawItem[] }).items ?? []);
  if (!Array.isArray(items)) return [];
  const groups = new Map<string, BlockGroup>();
  for (const item of items) {
    const type = item.entity_type ?? item.type ?? "";
    const id = item.entity_id ?? item.id ?? "";
    const name = item.name ?? "";
    if (!type || !id || !name) continue;
    const current = groups.get(type) ?? {
      key: type,
      title: GROUP_TITLES[type] ?? messages.locationDeleteReference,
      items: [],
    };
    current.items.push({
      id,
      name,
      href: item.in_trash ? "/trash" : hrefFor(type, id),
      inTrash: item.in_trash === true,
    });
    groups.set(type, current);
  }
  return Array.from(groups.values());
}

export type LocationDeleteDialogProps = {
  location: LocationDetail;
  onClose: () => void;
  onDeleted: (result: DeleteLocationResult) => void;
  onVersionConflict: () => void;
};

/** 删除确认：按下级/人物/势力分组列出三类入向引用，冲突时禁用删除主操作。 */
export function LocationDeleteDialog({
  location,
  onClose,
  onDeleted,
  onVersionConflict,
}: LocationDeleteDialogProps) {
  const [error, setError] = useState<string | null>(null);
  const [groups, setGroups] = useState<BlockGroup[]>([]);
  const [versionConflict, setVersionConflict] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    setGroups([]);
    setVersionConflict(false);
    try {
      const result = await deleteLocation(location.id, location.version);
      onDeleted(result);
      onClose();
    } catch (cause) {
      if (isApiCode(cause, "HAS_CHILDREN")) {
        setGroups(parseGroups(cause));
        setError(messages.locationDeleteHasChildren);
      } else if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        setGroups(parseGroups(cause));
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

  const blocked = groups.length > 0;

  return (
    <Dialog
      open
      onOpenChange={(next) => {
        if (!next && !submitting) onClose();
      }}
      title={messages.locationDeleteTitle(location.name)}
      description={messages.locationDeleteDescription}
      size={blocked ? "md" : "sm"}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} loading={submitting} disabled={blocked}>
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
      {groups.map((group) => (
        <section key={group.key} className="character-relation-group" aria-label={group.title}>
          <h3 className="character-relation-group__title">
            {group.title}（{group.items.length}）
          </h3>
          <ul className="character-reference-list">
            {group.items.map((item) => (
              <li key={`${group.key}:${item.id}`}>
                {item.inTrash ? (
                  <>
                    <span className="character-muted">
                      {item.name}
                      {messages.locationRelationInTrash}
                    </span>
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
        </section>
      ))}
    </Dialog>
  );
}
