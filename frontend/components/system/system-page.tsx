"use client";

import Link from "next/link";
import { BookOpen } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { CategoryDetail } from "@/components/system/category-detail";
import { CategoryList } from "@/components/system/category-list";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { toUserMessage } from "@/lib/api/errors";
import { listCategories, type SystemCategory } from "@/lib/api/system";
import { listWorks } from "@/lib/api/works";
import { messages } from "@/lib/messages";

type ListStatus = "idle" | "loading" | "ready" | "error";

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
  const workID = settings.current_work_id;
  const [workName, setWorkName] = useState<string | null>(null);
  const [categories, setCategories] = useState<SystemCategory[]>([]);
  const [listStatus, setListStatus] = useState<ListStatus>("idle");
  const [listError, setListError] = useState<Error | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);

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

  const load = useCallback(async () => {
    if (!workID) return;
    setListStatus("loading");
    setListError(null);
    try {
      const items = await listCategories(workID);
      setCategories(items);
      setSelectedID((current) =>
        current && items.some((item) => item.id === current) ? current : (items[0]?.id ?? null),
      );
      setListStatus("ready");
    } catch (cause) {
      setListError(cause instanceof Error ? cause : new Error(messages.loadFailed));
      setListStatus("error");
    }
  }, [workID]);

  useEffect(() => {
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
              <CategoryList categories={[]} selectedID={null} loading onSelect={() => undefined} />
              <CategoryDetail category={null} loading />
            </div>
          ) : null}

          {listStatus === "ready" ? (
            <div className="system-layout">
              <CategoryList categories={categories} selectedID={selectedID} onSelect={setSelectedID} />
              <CategoryDetail category={selectedCategory} />
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  );
}
