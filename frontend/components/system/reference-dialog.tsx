"use client";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { ApiError } from "@/lib/api/client";
import { messages } from "@/lib/messages";

/** 后端 REFERENCE_CONFLICT 的 details 条目；name 在 CHG-003 阶段可能为 null。 */
export type ReferenceItem = {
  type: string;
  entity_id: string;
  name: string | null;
  field: string;
  value: string;
  in_trash: boolean;
};

export type ReferenceAction = "disable" | "delete";

export type ReferenceDialogProps = {
  action: ReferenceAction;
  categoryName: string;
  items: ReferenceItem[];
  onClose: () => void;
};

/** 只接受结构完整的引用明细，避免把后端内部字段带到界面。 */
export function referenceConflictItems(error: unknown): ReferenceItem[] {
  if (!(error instanceof ApiError)) {
    return [];
  }
  const details = error.details;
  if (!Array.isArray(details)) {
    return [];
  }
  return details.filter((item): item is ReferenceItem => {
    if (typeof item !== "object" || item === null) return false;
    const candidate = item as Partial<ReferenceItem>;
    return (
      typeof candidate.type === "string" &&
      typeof candidate.entity_id === "string" &&
      typeof candidate.field === "string" &&
      typeof candidate.value === "string" &&
      typeof candidate.in_trash === "boolean" &&
      (candidate.name === null || typeof candidate.name === "string")
    );
  });
}

type ReferenceGroup = {
  value: string;
  items: ReferenceItem[];
};

/** 按节点名（details.value）分组，保持后端返回顺序。 */
function groupByNode(items: ReferenceItem[]): ReferenceGroup[] {
  const groups = new Map<string, ReferenceItem[]>();
  for (const item of items) {
    const key = item.value.trim() || messages.systemReferenceUnnamedNode;
    const current = groups.get(key);
    if (current) {
      current.push(item);
    } else {
      groups.set(key, [item]);
    }
  }
  return Array.from(groups, ([value, groupItems]) => ({ value, items: groupItems }));
}

/** 引用阻断弹窗：按节点名分组列出引用人物，不展示后端英文堆栈。 */
export function ReferenceDialog({ action, categoryName, items, onClose }: ReferenceDialogProps) {
  const groups = groupByNode(items);
  const title =
    action === "disable"
      ? messages.systemReferenceDisableTitle(categoryName, items.length)
      : messages.systemReferenceDeleteTitle(categoryName, items.length);

  return (
    <Dialog
      open
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={title}
      description={messages.systemReferenceDescription}
      size="md"
      footer={<Button onClick={onClose}>{messages.systemReferenceClose}</Button>}
    >
      {items.length === 0 ? (
        <Alert tone="warning" title={messages.errorByCode.REFERENCE_CONFLICT} />
      ) : (
        <div className="reference-dialog">
          {groups.map((group) => (
            <section key={group.value} className="reference-dialog__group" aria-label={group.value}>
              <h3 className="reference-dialog__group-title">
                <span className="reference-dialog__node">{group.value}</span>
                <span className="reference-dialog__count">{messages.systemReferencePeople(group.items.length)}</span>
              </h3>
              <ul className="reference-dialog__list">
                {group.items.map((item) => (
                  <li key={item.entity_id} className="reference-dialog__item">
                    <span className="reference-dialog__person">
                      {item.name ?? messages.systemReferenceUnnamed}
                    </span>
                    {item.in_trash ? <Badge tone="warning">{messages.systemReferenceInTrash}</Badge> : null}
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}
    </Dialog>
  );
}
