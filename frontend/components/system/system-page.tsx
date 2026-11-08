"use client";

import Link from "next/link";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { CategoryDetail } from "@/components/system/category-detail";
import { CategoryList } from "@/components/system/category-list";
import { ReferenceDialog, type ReferenceAction, type ReferenceItem } from "@/components/system/reference-dialog";
import {
  CreateCategoryDialog,
  DeleteCategoryDialog,
  DisableCategoryDialog,
  RenameCategoryDialog,
} from "@/components/system/system-dialogs";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { listCategories, updateCategory, type SystemCategory } from "@/lib/api/system";
import { listWorks } from "@/lib/api/works";
import { messages } from "@/lib/messages";

type ListStatus = "idle" | "loading" | "ready" | "error";

type CategoryDialogState =
  | { mode: "create" }
  | { mode: "rename"; category: SystemCategory }
  | { mode: "disable"; category: SystemCategory }
  | { mode: "delete"; category: SystemCategory };

type ReferenceState = {
  action: ReferenceAction;
  categoryName: string;
  items: ReferenceItem[];
};

function SystemSkeleton() {
  return (
    <div className="system-layout" aria-busy="true">
      <div className="system-category-list">
        <Skeleton variant="lines" lines={6} label={messages.systemLoadingCategories} />
      </div>
      <div className="system-detail">
        <Skeleton variant="lines" lines={8} label={messages.systemLoadingDetail} />
      </div>
    </div>
  );
}

/** 体系页编排：读取当前作品、类别列表与选中类别，统一处理加载/空/错误状态（PG-10）。 */
export function SystemPage() {
  const { settings, status: settingsStatus, error: settingsError, reload } = useSettings();
  const { showToast } = useToast();
  const workID = settings.current_work_id;
  const [workName, setWorkName] = useState<string | null>(null);
  const [categories, setCategories] = useState<SystemCategory[]>([]);
  const [listStatus, setListStatus] = useState<ListStatus>("idle");
  const [listError, setListError] = useState<Error | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [dialog, setDialog] = useState<CategoryDialogState | null>(null);
  const [reference, setReference] = useState<ReferenceState | null>(null);

  useEffect(() => {
    if (!workID) {
      setWorkName(null);
      return;
    }
    let active = true;
    listWorks({ pageSize: 100 })
      .then((page) => {
        if (!active) return;
        setWorkName(page.items.find((work) => work.id === workID)?.name ?? null);
      })
      .catch(() => {
        if (active) setWorkName(null);
      });
    return () => {
      active = false;
    };
  }, [workID]);

  const applyCategories = useCallback((items: SystemCategory[]) => {
    setCategories(items);
    setSelectedID((current) =>
      current && items.some((item) => item.id === current) ? current : (items[0]?.id ?? null),
    );
  }, []);

  const load = useCallback(async () => {
    if (!workID) return;
    setListStatus("loading");
    setListError(null);
    try {
      const items = await listCategories(workID);
      applyCategories(items);
      setListStatus("ready");
    } catch (cause) {
      setListError(cause instanceof Error ? cause : new Error(messages.loadFailed));
      setListStatus("error");
    }
  }, [workID, applyCategories]);

  /** 写操作成功后轻量刷新列表与详情，避免整页回到 Skeleton。 */
  const refreshCategories = useCallback(async (): Promise<SystemCategory[] | null> => {
    if (!workID) return null;
    try {
      const items = await listCategories(workID);
      applyCategories(items);
      setListStatus("ready");
      setListError(null);
      return items;
    } catch (cause) {
      setListError(cause instanceof Error ? cause : new Error(messages.loadFailed));
      setListStatus("error");
      return null;
    }
  }, [workID, applyCategories]);

  /** 版本冲突后重新拉取并返回最新 version，保留弹窗内输入。 */
  const refreshCategoryVersion = useCallback(
    async (id: string): Promise<number | null> => {
      const items = await refreshCategories();
      return items?.find((category) => category.id === id)?.version ?? null;
    },
    [refreshCategories],
  );

  async function handleCreated(category: SystemCategory) {
    setSelectedID(category.id);
    await refreshCategories();
  }

  async function handleEnable(category: SystemCategory) {
    try {
      const updated = await updateCategory(category.id, { isEnabled: true, version: category.version });
      await refreshCategories();
      showToast({ tone: "success", message: messages.systemCategoryEnabled(updated.name) });
    } catch (cause) {
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        await refreshCategories();
        showToast({ tone: "info", message: messages.systemVersionRefreshed });
        return;
      }
      showToast({ tone: "error", message: toUserMessage(cause, messages.saveFailed) });
    }
  }

  function openReference(items: ReferenceItem[], action: ReferenceAction, categoryName: string) {
    setReference({ items, action, categoryName });
  }

  useEffect(() => {
    setDialog(null);
    setReference(null);
    if (!workID) {
      setCategories([]);
      setSelectedID(null);
      setListStatus("idle");
      setListError(null);
      return;
    }
    void load();
  }, [workID, load]);

  const selectedCategory = useMemo(
    () => categories.find((category) => category.id === selectedID) ?? null,
    [categories, selectedID],
  );

  const description =
    settingsStatus === "ready" && workID
      ? messages.systemDescription(workName ?? messages.systemWorkFallback)
      : undefined;

  return (
    <div className="page">
      <PageHeader title={messages.systemTitle} description={description} />

      {settingsStatus === "loading" ? <SystemSkeleton /> : null}

      {settingsStatus === "error" ? (
        <Alert
          tone="danger"
          title={toUserMessage(settingsError, messages.loadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={reload}>
              {messages.retry}
            </Button>
          }
        />
      ) : null}

      {settingsStatus === "ready" && !workID ? (
        <EmptyState
          variant="page"
          icon={<BookOpen aria-hidden="true" size={22} />}
          title={messages.noWorkTitle}
          description={messages.noWorkDescription}
          action={
            <Link className="ui-button ui-button--primary ui-button--md" href="/">
              <span className="ui-button__label">{messages.trashEmptyAction}</span>
            </Link>
          }
        />
      ) : null}

      {settingsStatus === "ready" && workID ? (
        <>
          {listStatus === "error" ? (
            <Alert
              tone="danger"
              title={toUserMessage(listError, messages.loadFailed)}
              action={
                <Button variant="secondary" size="sm" onClick={() => void load()}>
                  {messages.retry}
                </Button>
              }
            />
          ) : null}

          {listStatus === "loading" ? (
            <div className="system-layout" aria-busy="true">
              <CategoryList
                categories={[]}
                selectedID={null}
                loading
                onSelect={() => undefined}
                onCreate={() => undefined}
              />
              <CategoryDetail
                category={null}
                loading
                onRename={() => undefined}
                onDisable={() => undefined}
                onEnable={() => undefined}
                onDelete={() => undefined}
              />
            </div>
          ) : null}

          {listStatus === "ready" ? (
            <div className="system-layout">
              <CategoryList
                categories={categories}
                selectedID={selectedID}
                onSelect={setSelectedID}
                onCreate={() => setDialog({ mode: "create" })}
              />
              <CategoryDetail
                category={selectedCategory}
                onRename={(category) => setDialog({ mode: "rename", category })}
                onDisable={(category) => setDialog({ mode: "disable", category })}
                onEnable={(category) => void handleEnable(category)}
                onDelete={(category) => setDialog({ mode: "delete", category })}
              />
            </div>
          ) : null}

          {dialog?.mode === "create" ? (
            <CreateCategoryDialog
              workID={workID}
              onClose={() => setDialog(null)}
              onCreated={handleCreated}
            />
          ) : null}
          {dialog?.mode === "rename" ? (
            <RenameCategoryDialog
              category={dialog.category}
              onClose={() => setDialog(null)}
              onChanged={async () => {
                await refreshCategories();
              }}
              onVersionConflict={() => refreshCategoryVersion(dialog.category.id)}
            />
          ) : null}
          {dialog?.mode === "disable" ? (
            <DisableCategoryDialog
              category={dialog.category}
              onClose={() => setDialog(null)}
              onChanged={async () => {
                await refreshCategories();
              }}
              onVersionConflict={() => refreshCategoryVersion(dialog.category.id)}
              onReferenceConflict={(items) => openReference(items, "disable", dialog.category.name)}
            />
          ) : null}
          {dialog?.mode === "delete" ? (
            <DeleteCategoryDialog
              category={dialog.category}
              onClose={() => setDialog(null)}
              onChanged={async () => {
                await refreshCategories();
              }}
              onVersionConflict={() => refreshCategoryVersion(dialog.category.id)}
              onReferenceConflict={(items) => openReference(items, "delete", dialog.category.name)}
            />
          ) : null}

          {reference ? (
            <ReferenceDialog
              action={reference.action}
              categoryName={reference.categoryName}
              items={reference.items}
              onClose={() => setReference(null)}
            />
          ) : null}
        </>
      ) : null}
    </div>
  );
}
