"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useToast } from "@/components/ui/toast";
import { referenceConflictItems, type ReferenceItem } from "@/components/system/reference-dialog";
import { ApiError } from "@/lib/api/client";
import { isApiCode, toUserMessage, type ConflictItem } from "@/lib/api/errors";
import {
  createCategory,
  createNode,
  deleteCategory,
  deleteNode,
  updateCategory,
  updateNode,
  type CategoryType,
  type SystemCategory,
  type SystemNode,
  type UpdateNodeInput,
} from "@/lib/api/system";
import { undoTrash } from "@/lib/api/trash";
import { messages } from "@/lib/messages";

/** 仅用于把 422 VALIDATION_ERROR 的 details.field 映射到具体输入框。 */
function validationField(error: unknown): string | null {
  if (!(error instanceof ApiError)) {
    return null;
  }
  const details = error.details;
  if (!details || typeof details !== "object" || Array.isArray(details)) {
    return null;
  }
  const field = (details as { field?: unknown }).field;
  return typeof field === "string" ? field : null;
}

function categoryTypeOptions() {
  return [
    { value: "normal", label: messages.systemCategoryTypeNormalOption },
    { value: "era", label: messages.systemCategoryTypeEraOption },
  ];
}

export type CreateCategoryDialogProps = {
  workID: string;
  onClose: () => void;
  onCreated: (category: SystemCategory) => Promise<void> | void;
};

/** 新增类别：名称 + 普通/纪元；重复名、10 类上限与校验错误都挂在名称输入框。 */
export function CreateCategoryDialog({ workID, onClose, onCreated }: CreateCategoryDialogProps) {
  const { showToast } = useToast();
  const [name, setName] = useState("");
  const [categoryType, setCategoryType] = useState<CategoryType>("normal");
  const [nameError, setNameError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(messages.systemCategoryNameRequired);
      return;
    }
    setSubmitting(true);
    setNameError(null);
    setFormError(null);
    try {
      const created = await createCategory({ workID, name: trimmed, categoryType });
      await onCreated(created);
      onClose();
      showToast({ tone: "success", message: messages.systemCategoryCreated(created.name) });
    } catch (cause) {
      if (isApiCode(cause, "NAME_CONFLICT") || isApiCode(cause, "CATEGORY_LIMIT")) {
        setNameError(toUserMessage(cause, messages.saveFailed));
      } else if (isApiCode(cause, "VALIDATION_ERROR")) {
        if (validationField(cause) === "name") {
          setNameError(toUserMessage(cause, messages.saveFailed));
        } else {
          setFormError(toUserMessage(cause, messages.saveFailed));
        }
      } else {
        setFormError(toUserMessage(cause, messages.saveFailed));
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
      title={messages.systemAddCategoryTitle}
      description={messages.systemAddCategoryDescription}
      size="md"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={() => void submit()} loading={submitting} disabled={!name.trim()}>
            {messages.systemCategoryCreateSubmit}
          </Button>
        </>
      }
    >
      {formError ? <Alert tone="danger" title={formError} /> : null}
      <FormField
        id="system-create-category-name"
        label={messages.systemCategoryNameLabel}
        required
        hint={messages.systemCategoryNameHint}
        error={nameError ?? undefined}
      >
        <Input
          id="system-create-category-name"
          value={name}
          maxLength={200}
          autoFocus
          disabled={submitting}
          onChange={(event) => setName(event.target.value)}
        />
      </FormField>
      <FormField
        id="system-create-category-type"
        label={messages.systemCategoryTypeLabel}
        required
        hint={messages.systemCategoryTypeHint}
      >
        <Select
          id="system-create-category-type"
          options={categoryTypeOptions()}
          value={categoryType}
          disabled={submitting}
          onChange={(event) => setCategoryType(event.target.value as CategoryType)}
        />
      </FormField>
    </Dialog>
  );
}

export type RenameCategoryDialogProps = {
  category: SystemCategory;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
  /** 版本冲突时刷新列表并返回最新的 version，供下次提交使用。 */
  onVersionConflict: () => Promise<number | null>;
};

/** 类别改名：使用当前 version 作为 If-Match；冲突时刷新但保留输入。 */
export function RenameCategoryDialog({
  category,
  onClose,
  onChanged,
  onVersionConflict,
}: RenameCategoryDialogProps) {
  const { showToast } = useToast();
  const [name, setName] = useState(category.name);
  const [version, setVersion] = useState(category.version);
  const [nameError, setNameError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(messages.systemCategoryNameRequired);
      return;
    }
    setSubmitting(true);
    setNameError(null);
    setFormError(null);
    try {
      const updated = await updateCategory(category.id, { name: trimmed, version });
      await onChanged();
      onClose();
      showToast({ tone: "success", message: messages.systemCategoryRenamed(updated.name) });
    } catch (cause) {
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        const latest = await onVersionConflict();
        if (latest !== null) setVersion(latest);
        setFormError(messages.systemVersionRefreshed);
        return;
      }
      if (isApiCode(cause, "NAME_CONFLICT")) {
        setNameError(toUserMessage(cause, messages.saveFailed));
      } else if (isApiCode(cause, "VALIDATION_ERROR")) {
        if (validationField(cause) === "name") {
          setNameError(toUserMessage(cause, messages.saveFailed));
        } else {
          setFormError(toUserMessage(cause, messages.saveFailed));
        }
      } else {
        setFormError(toUserMessage(cause, messages.saveFailed));
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
      title={messages.systemRenameCategoryTitle(category.name)}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={() => void submit()} loading={submitting} disabled={!name.trim()}>
            {messages.systemRenameCategorySubmit}
          </Button>
        </>
      }
    >
      {formError ? <Alert tone="warning" title={formError} /> : null}
      <FormField
        id="system-rename-category-name"
        label={messages.systemCategoryNameLabel}
        required
        hint={messages.systemCategoryNameHint}
        error={nameError ?? undefined}
      >
        <Input
          id="system-rename-category-name"
          value={name}
          maxLength={200}
          autoFocus
          disabled={submitting}
          onChange={(event) => setName(event.target.value)}
        />
      </FormField>
    </Dialog>
  );
}

export type DisableCategoryDialogProps = {
  category: SystemCategory;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
  onVersionConflict: () => Promise<number | null>;
  onReferenceConflict: (items: ReferenceItem[]) => void;
};

/** 停用确认：有引用时关闭确认框并移交 ReferenceDialog。 */
export function DisableCategoryDialog({
  category,
  onClose,
  onChanged,
  onVersionConflict,
  onReferenceConflict,
}: DisableCategoryDialogProps) {
  const { showToast } = useToast();
  const [version, setVersion] = useState(category.version);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    setError(null);
    try {
      const updated = await updateCategory(category.id, { isEnabled: false, version });
      await onChanged();
      onClose();
      showToast({ tone: "success", message: messages.systemCategoryDisabledToast(updated.name) });
    } catch (cause) {
      if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        const items = referenceConflictItems(cause);
        onClose();
        onReferenceConflict(items);
        return;
      }
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        const latest = await onVersionConflict();
        if (latest !== null) setVersion(latest);
        setError(messages.systemVersionRefreshed);
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
      title={messages.systemDisableCategoryTitle(category.name)}
      description={messages.systemDisableCategoryDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={() => void submit()} loading={submitting}>
            {messages.systemDisableCategorySubmit}
          </Button>
        </>
      }
    >
      {error ? <Alert tone="warning" title={error} /> : null}
    </Dialog>
  );
}

export type DeleteCategoryDialogProps = {
  category: SystemCategory;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
  onVersionConflict: () => Promise<number | null>;
  onReferenceConflict: (items: ReferenceItem[]) => void;
};

/** 删除整组：说明节点一并进入回收站；成功后接入 8 秒撤销 Toast。 */
export function DeleteCategoryDialog({
  category,
  onClose,
  onChanged,
  onVersionConflict,
  onReferenceConflict,
}: DeleteCategoryDialogProps) {
  const { showToast } = useToast();
  const [version, setVersion] = useState(category.version);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleUndo(undoToken: string) {
    try {
      await undoTrash(undoToken);
      await onChanged();
      showToast({ tone: "success", message: messages.systemCategoryUndone(category.name) });
    } catch (cause) {
      showToast({ tone: "error", message: toUserMessage(cause, messages.loadFailed) });
    }
  }

  async function submit() {
    setSubmitting(true);
    setError(null);
    try {
      const result = await deleteCategory(category.id, version);
      await onChanged();
      onClose();
      if (result?.undo_token) {
        showToast({
          tone: "undo",
          message: messages.systemCategoryDeleted(category.name),
          actionLabel: messages.undo,
          onAction: () => void handleUndo(result.undo_token),
        });
      } else {
        showToast({ tone: "success", message: messages.systemCategoryDeleted(category.name) });
      }
    } catch (cause) {
      if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        const items = referenceConflictItems(cause);
        onClose();
        onReferenceConflict(items);
        return;
      }
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        const latest = await onVersionConflict();
        if (latest !== null) setVersion(latest);
        setError(messages.systemVersionRefreshed);
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
      title={messages.systemDeleteCategoryTitle(category.name)}
      description={messages.systemDeleteCategoryDescription(category.name, category.node_count)}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={() => void submit()} loading={submitting}>
            {messages.systemDeleteCategorySubmit}
          </Button>
        </>
      }
    >
      {error ? <Alert tone="danger" title={error} /> : null}
    </Dialog>
  );
}

/** HAS_CHILDREN 的 details 兼容 `ConflictItem[]` 与 `{ items: [...] }` 两种形状。 */
function hasChildrenItems(error: unknown): ConflictItem[] {
  if (!(error instanceof ApiError)) {
    return [];
  }
  const details = error.details;
  const raw: unknown = Array.isArray(details)
    ? details
    : details && typeof details === "object" && Array.isArray((details as { items?: unknown }).items)
      ? (details as { items: unknown[] }).items
      : [];
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw.filter((item): item is ConflictItem => {
    if (typeof item !== "object" || item === null) return false;
    const candidate = item as Partial<ConflictItem>;
    return typeof candidate.id === "string" && typeof candidate.name === "string";
  });
}

/** 纪元文本：创建时可空省略；改名时清空需显式传 null。 */
function trimmedEraValue(value: string): string | null {
  const trimmed = value.trim();
  return trimmed ? trimmed : null;
}

function applyNodeFieldError(
  cause: unknown,
  setters: {
    name: (message: string) => void;
    eraStart: (message: string) => void;
    eraEnd: (message: string) => void;
    form: (message: string) => void;
  },
) {
  if (isApiCode(cause, "NAME_CONFLICT")) {
    setters.name(toUserMessage(cause, messages.saveFailed));
    return;
  }
  if (isApiCode(cause, "VALIDATION_ERROR")) {
    const field = validationField(cause);
    if (field === "name") {
      setters.name(toUserMessage(cause, messages.saveFailed));
      return;
    }
    if (field === "era_start_text") {
      setters.eraStart(toUserMessage(cause, messages.saveFailed));
      return;
    }
    if (field === "era_end_text") {
      setters.eraEnd(toUserMessage(cause, messages.saveFailed));
      return;
    }
  }
  setters.form(toUserMessage(cause, messages.saveFailed));
}

export type CreateNodeDialogProps = {
  category: SystemCategory;
  /** null 表示新增主层级；否则为新增子层级。 */
  parent: SystemNode | null;
  onClose: () => void;
  onCreated: () => Promise<void> | void;
};

/** 新增节点：名称必填；纪元类别额外显示可空的起止文本。 */
export function CreateNodeDialog({ category, parent, onClose, onCreated }: CreateNodeDialogProps) {
  const { showToast } = useToast();
  const isEra = category.category_type === "era";
  const [name, setName] = useState("");
  const [eraStart, setEraStart] = useState("");
  const [eraEnd, setEraEnd] = useState("");
  const [nameError, setNameError] = useState<string | null>(null);
  const [eraStartError, setEraStartError] = useState<string | null>(null);
  const [eraEndError, setEraEndError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(messages.systemNodeNameRequired);
      return;
    }
    setSubmitting(true);
    setNameError(null);
    setEraStartError(null);
    setEraEndError(null);
    setFormError(null);
    try {
      const created = await createNode({
        categoryID: category.id,
        name: trimmed,
        parentID: parent?.id,
        eraStartText: isEra ? (trimmedEraValue(eraStart) ?? undefined) : undefined,
        eraEndText: isEra ? (trimmedEraValue(eraEnd) ?? undefined) : undefined,
      });
      await onCreated();
      onClose();
      showToast({ tone: "success", message: messages.systemNodeCreated(created.name) });
    } catch (cause) {
      applyNodeFieldError(cause, {
        name: setNameError,
        eraStart: setEraStartError,
        eraEnd: setEraEndError,
        form: setFormError,
      });
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
      title={parent ? messages.systemAddChildNodeTitle(parent.name) : messages.systemAddNodeTitle}
      description={
        parent ? messages.systemAddChildNodeDescription(parent.name) : messages.systemAddNodeDescription
      }
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={() => void submit()} loading={submitting} disabled={!name.trim()}>
            {messages.systemAddNodeSubmit}
          </Button>
        </>
      }
    >
      {formError ? <Alert tone="danger" title={formError} /> : null}
      <FormField
        id="system-create-node-name"
        label={messages.systemNodeNameLabel}
        required
        hint={messages.systemNodeNameHint}
        error={nameError ?? undefined}
      >
        <Input
          id="system-create-node-name"
          value={name}
          maxLength={200}
          autoFocus
          disabled={submitting}
          onChange={(event) => setName(event.target.value)}
        />
      </FormField>
      {isEra ? (
        <>
          <FormField
            id="system-create-node-era-start"
            label={messages.systemEraStartLabel}
            hint={messages.systemEraTextHint}
            error={eraStartError ?? undefined}
          >
            <Input
              id="system-create-node-era-start"
              value={eraStart}
              disabled={submitting}
              onChange={(event) => setEraStart(event.target.value)}
            />
          </FormField>
          <FormField
            id="system-create-node-era-end"
            label={messages.systemEraEndLabel}
            hint={messages.systemEraTextHint}
            error={eraEndError ?? undefined}
          >
            <Input
              id="system-create-node-era-end"
              value={eraEnd}
              disabled={submitting}
              onChange={(event) => setEraEnd(event.target.value)}
            />
          </FormField>
        </>
      ) : null}
    </Dialog>
  );
}

export type RenameNodeDialogProps = {
  category: SystemCategory;
  node: SystemNode;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
  /** 版本冲突时刷新列表并返回最新的 version，供下次提交使用。 */
  onVersionConflict: () => Promise<number | null>;
};

/** 节点改名：使用当前 version 作为 If-Match；纪元文本清空时显式发送 null。 */
export function RenameNodeDialog({
  category,
  node,
  onClose,
  onChanged,
  onVersionConflict,
}: RenameNodeDialogProps) {
  const { showToast } = useToast();
  const isEra = category.category_type === "era";
  const [name, setName] = useState(node.name);
  const [eraStart, setEraStart] = useState(node.era_start_text ?? "");
  const [eraEnd, setEraEnd] = useState(node.era_end_text ?? "");
  const [version, setVersion] = useState(node.version);
  const [nameError, setNameError] = useState<string | null>(null);
  const [eraStartError, setEraStartError] = useState<string | null>(null);
  const [eraEndError, setEraEndError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setNameError(messages.systemNodeNameRequired);
      return;
    }
    setSubmitting(true);
    setNameError(null);
    setEraStartError(null);
    setEraEndError(null);
    setFormError(null);
    try {
      const input: UpdateNodeInput = { name: trimmed, version };
      if (isEra) {
        input.eraStartText = trimmedEraValue(eraStart);
        input.eraEndText = trimmedEraValue(eraEnd);
      }
      const updated = await updateNode(node.id, input);
      await onChanged();
      onClose();
      showToast({ tone: "success", message: messages.systemNodeRenamed(updated.name) });
    } catch (cause) {
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        const latest = await onVersionConflict();
        if (latest !== null) setVersion(latest);
        setFormError(messages.systemVersionRefreshed);
        return;
      }
      applyNodeFieldError(cause, {
        name: setNameError,
        eraStart: setEraStartError,
        eraEnd: setEraEndError,
        form: setFormError,
      });
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
      title={messages.systemRenameNodeTitle(node.name)}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={() => void submit()} loading={submitting} disabled={!name.trim()}>
            {messages.systemRenameNodeSubmit}
          </Button>
        </>
      }
    >
      {formError ? <Alert tone="warning" title={formError} /> : null}
      <FormField
        id="system-rename-node-name"
        label={messages.systemNodeNameLabel}
        required
        hint={messages.systemNodeNameHint}
        error={nameError ?? undefined}
      >
        <Input
          id="system-rename-node-name"
          value={name}
          maxLength={200}
          autoFocus
          disabled={submitting}
          onChange={(event) => setName(event.target.value)}
        />
      </FormField>
      {isEra ? (
        <>
          <FormField
            id="system-rename-node-era-start"
            label={messages.systemEraStartLabel}
            hint={messages.systemEraTextHint}
            error={eraStartError ?? undefined}
          >
            <Input
              id="system-rename-node-era-start"
              value={eraStart}
              disabled={submitting}
              onChange={(event) => setEraStart(event.target.value)}
            />
          </FormField>
          <FormField
            id="system-rename-node-era-end"
            label={messages.systemEraEndLabel}
            hint={messages.systemEraTextHint}
            error={eraEndError ?? undefined}
          >
            <Input
              id="system-rename-node-era-end"
              value={eraEnd}
              disabled={submitting}
              onChange={(event) => setEraEnd(event.target.value)}
            />
          </FormField>
        </>
      ) : null}
    </Dialog>
  );
}

export type DeleteNodeDialogProps = {
  node: SystemNode;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
  onVersionConflict: () => Promise<number | null>;
  onReferenceConflict: (items: ReferenceItem[]) => void;
};

/** 删除节点：HAS_CHILDREN 时在弹窗内列出子节点；成功后接入 8 秒撤销 Toast。 */
export function DeleteNodeDialog({
  node,
  onClose,
  onChanged,
  onVersionConflict,
  onReferenceConflict,
}: DeleteNodeDialogProps) {
  const { showToast } = useToast();
  const [version, setVersion] = useState(node.version);
  const [error, setError] = useState<string | null>(null);
  const [childItems, setChildItems] = useState<ConflictItem[]>([]);
  const [submitting, setSubmitting] = useState(false);

  async function handleUndo(undoToken: string) {
    try {
      await undoTrash(undoToken);
      await onChanged();
      showToast({ tone: "success", message: messages.systemNodeUndone(node.name) });
    } catch (cause) {
      showToast({ tone: "error", message: toUserMessage(cause, messages.loadFailed) });
    }
  }

  async function submit() {
    setSubmitting(true);
    setError(null);
    setChildItems([]);
    try {
      const result = await deleteNode(node.id, version);
      await onChanged();
      onClose();
      if (result?.undo_token) {
        showToast({
          tone: "undo",
          message: messages.systemNodeDeleted(node.name),
          actionLabel: messages.undo,
          onAction: () => void handleUndo(result.undo_token),
        });
      } else {
        showToast({ tone: "success", message: messages.systemNodeDeleted(node.name) });
      }
    } catch (cause) {
      if (isApiCode(cause, "REFERENCE_CONFLICT")) {
        const items = referenceConflictItems(cause);
        onClose();
        onReferenceConflict(items);
        return;
      }
      if (isApiCode(cause, "HAS_CHILDREN")) {
        const parsed = hasChildrenItems(cause);
        const fallback = (node.children ?? []).map((child) => ({ id: child.id, name: child.name }));
        setChildItems(parsed.length > 0 ? parsed : fallback);
        setError(messages.systemNodeHasChildrenTitle);
        return;
      }
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        const latest = await onVersionConflict();
        if (latest !== null) setVersion(latest);
        setError(messages.systemVersionRefreshed);
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
      title={messages.systemDeleteNodeTitle(node.name)}
      description={messages.systemDeleteNodeDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={() => void submit()} loading={submitting}>
            {messages.systemDeleteNodeSubmit}
          </Button>
        </>
      }
    >
      {error ? <Alert tone="danger" title={error} /> : null}
      {childItems.length > 0 ? (
        <div className="node-conflict-list">
          <p>{messages.systemNodeHasChildrenDescription}</p>
          <ul>
            {childItems.map((child) => (
              <li key={child.id}>{child.name}</li>
            ))}
          </ul>
        </div>
      ) : error === messages.systemNodeHasChildrenTitle ? (
        <p className="node-conflict-list__fallback">{messages.systemNodeHasChildrenFallback}</p>
      ) : null}
    </Dialog>
  );
}
