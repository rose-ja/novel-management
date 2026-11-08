"use client";

import Link from "next/link";
import { BookOpen, Check, ChevronDown, Pencil, Plus, Trash2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { useToast } from "@/components/ui/toast";
import { CreateWorkDialog, DeleteWorkDialog, EditWorkDialog } from "@/components/works/work-dialogs";
import { toUserMessage } from "@/lib/api/errors";
import { updateCurrentWork } from "@/lib/api/settings";
import { listWorks, type Work } from "@/lib/api/works";
import { messages } from "@/lib/messages";

type ListStatus = "loading" | "ready" | "error";

type DialogState = { mode: "create" } | { mode: "edit"; work: Work } | { mode: "delete"; work: Work };

/** 顶栏作品切换器：唯一作品入口，包含新建/编辑/删除与回收站入口（PG-00 §4.1）。 */
export function WorkSwitcher() {
  const { settings, status, reload } = useSettings();
  const { showToast } = useToast();
  const [works, setWorks] = useState<Work[]>([]);
  const [listStatus, setListStatus] = useState<ListStatus>("loading");
  const [open, setOpen] = useState(false);
  const [dialog, setDialog] = useState<DialogState | null>(null);
  const [switching, setSwitching] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    setListStatus("loading");
    try {
      const page = await listWorks({ pageSize: 100 });
      setWorks(page.items);
      setListStatus("ready");
    } catch {
      setListStatus("error");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  // 打开后焦点进入当前项（无当前项时进入首个可操作项）。
  useEffect(() => {
    if (!open) return;
    const frame = requestAnimationFrame(() => {
      const container = menuRef.current;
      if (!container) return;
      const target =
        container.querySelector<HTMLElement>(".work-menu__item--active .work-menu__select") ??
        container.querySelector<HTMLElement>(".work-menu__select") ??
        container.querySelector<HTMLElement>(".work-menu__footer button");
      target?.focus();
    });
    return () => cancelAnimationFrame(frame);
  }, [open]);

  function handleMenuKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    const container = menuRef.current;
    if (!container) return;
    const focusables = Array.from(
      container.querySelectorAll<HTMLElement>(".work-menu__select, .work-menu__actions button, .work-menu__footer button"),
    );
    if (focusables.length === 0) return;
    event.preventDefault();
    const currentIndex = focusables.indexOf(document.activeElement as HTMLElement);
    const nextIndex =
      event.key === "ArrowDown"
        ? (currentIndex + 1 + focusables.length) % focusables.length
        : (currentIndex - 1 + focusables.length) % focusables.length;
    focusables[nextIndex]?.focus();
  }

  const currentWork = works.find((work) => work.id === settings.current_work_id) ?? null;
  const loading = status === "loading" || listStatus === "loading";
  const label = loading ? messages.workLoading : currentWork?.name ?? messages.noWorkBadge;

  const handleChanged = useCallback(() => {
    void load();
  }, [load]);

  async function select(work: Work) {
    if (work.id === settings.current_work_id) {
      setOpen(false);
      return;
    }
    setSwitching(true);
    try {
      await updateCurrentWork(work.id);
      reload();
      showToast({ tone: "success", message: messages.workSwitched(work.name) });
    } catch (cause) {
      showToast({ tone: "error", message: toUserMessage(cause, messages.saveFailed) });
    } finally {
      setSwitching(false);
      setOpen(false);
    }
  }

  return (
    <div className="work-switcher" ref={containerRef}>
      <button
        type="button"
        ref={triggerRef}
        className={[
          "shell__work",
          "shell__work--trigger",
          currentWork ? "" : "shell__work--empty",
        ]
          .filter(Boolean)
          .join(" ")}
        aria-haspopup="true"
        aria-expanded={open}
        title={currentWork?.name}
        onClick={() => setOpen((value) => !value)}
      >
        <BookOpen aria-hidden="true" size={14} />
        <span className="shell__work-text">{label}</span>
        <span className="work-switcher__chevron">
          <ChevronDown aria-hidden="true" size={14} />
        </span>
      </button>

      {open ? (
        <div className="work-menu" ref={menuRef} aria-label={messages.workMenuTitle} onKeyDown={handleMenuKeyDown}>
          <div className="work-menu__header">
            <span>{messages.workMenuTitle}</span>
            <span>{works.length > 0 ? works.length : null}</span>
          </div>

          {listStatus === "loading" ? (
            <div className="work-menu__empty">
              <p className="work-menu__message">{messages.workLoading}</p>
            </div>
          ) : null}

          {listStatus === "error" ? (
            <div className="work-menu__empty">
              <p className="work-menu__message">{messages.workListFailed}</p>
              <Button variant="secondary" size="sm" onClick={() => void load()}>
                {messages.workListRetry}
              </Button>
            </div>
          ) : null}

          {listStatus === "ready" && works.length === 0 ? (
            <div className="work-menu__empty">
              <p className="work-menu__message">{messages.workEmptyMenu}</p>
            </div>
          ) : null}

          {listStatus === "ready" && works.length > 0 ? (
            <ul className="work-menu__list">
              {works.map((work) => {
                const active = work.id === settings.current_work_id;
                return (
                  <li key={work.id} className={active ? "work-menu__item work-menu__item--active" : "work-menu__item"}>
                    <button
                      type="button"
                      className="work-menu__select"
                      aria-current={active ? "true" : undefined}
                      disabled={switching}
                      onClick={() => void select(work)}
                    >
                      <span className="work-menu__check" aria-hidden="true">
                        {active ? <Check size={14} /> : null}
                      </span>
                      <span className="work-menu__name">{work.name}</span>
                    </button>
                    <div className="work-menu__actions">
                      <IconButton
                        size="sm"
                        label={`${messages.workEdit}《${work.name}》`}
                        onClick={() => {
                          setOpen(false);
                          setDialog({ mode: "edit", work });
                        }}
                      >
                        <Pencil aria-hidden="true" size={14} />
                      </IconButton>
                      <IconButton
                        size="sm"
                        tone="danger"
                        label={`${messages.workDelete}《${work.name}》`}
                        onClick={() => {
                          setOpen(false);
                          setDialog({ mode: "delete", work });
                        }}
                      >
                        <Trash2 aria-hidden="true" size={14} />
                      </IconButton>
                    </div>
                  </li>
                );
              })}
            </ul>
          ) : null}

          <div className="work-menu__divider" />
          <div className="work-menu__footer">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setOpen(false);
                setDialog({ mode: "create" });
              }}
            >
              <Plus aria-hidden="true" size={14} />
              {messages.workCreate}
            </Button>
            <Link href="/trash" className="ui-button ui-button--ghost ui-button--sm" onClick={() => setOpen(false)}>
              <Trash2 aria-hidden="true" size={14} />
              <span className="ui-button__label">{messages.trashNav}</span>
            </Link>
          </div>
        </div>
      ) : null}

      {dialog?.mode === "create" ? (
        <CreateWorkDialog onClose={() => setDialog(null)} onChanged={handleChanged} />
      ) : null}
      {dialog?.mode === "edit" ? (
        <EditWorkDialog work={dialog.work} onClose={() => setDialog(null)} onChanged={handleChanged} />
      ) : null}
      {dialog?.mode === "delete" ? (
        <DeleteWorkDialog work={dialog.work} onClose={() => setDialog(null)} onChanged={handleChanged} />
      ) : null}
    </div>
  );
}
