"use client";

import { useCallback, useEffect, useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog } from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { useToast } from "@/components/ui/toast";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import {
  createExport,
  exportDownloadUrl,
  fetchExport,
  fetchExportStatus,
  resolveExportPath,
  type ExportRun,
} from "@/lib/api/exports";
import { listWorks } from "@/lib/api/works";
import { copyText } from "@/lib/clipboard";
import { formatBytes, formatDateTime, truncateMiddle } from "@/lib/format";
import { messages } from "@/lib/messages";

type Phase = "idle" | "running" | "completed" | "failed";

function exportErrorText(code: string | null): string {
  if (!code) return messages.errorByCode.EXPORT_FAILED;
  return messages.errorByCode[code] ?? messages.errorByCode.EXPORT_FAILED;
}

/** PG-13 导出区块：Dialog 选择回收站、后台轮询、成功下载、失败重试。 */
export function ExportSection() {
  const { settings, reload } = useSettings();
  const { showToast } = useToast();
  const [hasWorks, setHasWorks] = useState<boolean | null>(null);
  const [phase, setPhase] = useState<Phase>("idle");
  const [open, setOpen] = useState(false);
  const [includeTrash, setIncludeTrash] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [run, setRun] = useState<ExportRun | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    listWorks({ pageSize: 1 })
      .then((page) => {
        if (active) setHasWorks(page.total > 0);
      })
      .catch(() => {
        if (active) setHasWorks(null);
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    let active = true;
    fetchExportStatus()
      .then((status) => {
        if (!active) return;
        if (status.active) {
          setRun(status.active);
          setPhase("running");
          return;
        }
        if (status.last?.status === "completed") {
          setRun(status.last);
          setPhase("completed");
        } else if (status.last?.status === "failed") {
          setRun(status.last);
          setPhase("failed");
          setError(exportErrorText(status.last.error_code));
        }
      })
      .catch(() => {
        // 状态读取失败不阻塞页面，用户仍可点击导出重新创建任务。
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (phase !== "running" || !run) return;
    const id = run.id;
    let active = true;
    const timer = setInterval(() => {
      fetchExport(id)
        .then((latest) => {
          if (!active) return;
          setRun(latest);
          if (latest.status === "completed") {
            setPhase("completed");
            setError(null);
            showToast({ tone: "success", message: messages.exportSuccessToast });
            reload();
          } else if (latest.status === "failed") {
            setPhase("failed");
            setError(exportErrorText(latest.error_code));
          }
        })
        .catch((cause: unknown) => {
          if (!active) return;
          setPhase("failed");
          setError(toUserMessage(cause, messages.loadFailed));
        });
    }, 1000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [phase, run, reload, showToast]);

  const copy = useCallback(
    async (value: string, successMessage: string = messages.copySuccess) => {
      const ok = await copyText(value);
      showToast({ tone: ok ? "success" : "error", message: ok ? successMessage : messages.copyFailed });
    },
    [showToast],
  );

  async function submit() {
    setSubmitting(true);
    setError(null);
    try {
      const result = await createExport(includeTrash);
      const record = await fetchExport(result.export_id);
      setRun(record);
      setPhase("running");
    } catch (cause) {
      if (isApiCode(cause, "EXPORT_IN_PROGRESS")) {
        try {
          const status = await fetchExportStatus();
          if (status.active) {
            setRun(status.active);
            setPhase("running");
            return;
          }
        } catch {
          // 继续走统一错误展示。
        }
      }
      setPhase("failed");
      setError(toUserMessage(cause, messages.errorByCode.EXPORT_FAILED));
    } finally {
      setSubmitting(false);
    }
  }

  function openDialog() {
    setError(null);
    if (phase === "completed" || phase === "failed") {
      setRun(null);
      setPhase("idle");
    }
    setOpen(true);
  }

  function handleOpenChange(next: boolean) {
    if (!next && phase === "running") return;
    setOpen(next);
  }

  const path = run?.file_path ? resolveExportPath(settings.data_dir, run.file_path) : "";
  const exportDisabled = hasWorks === false || phase === "running";

  return (
    <Card
      title={messages.exportCardTitle}
      description={messages.exportCardDescription}
      extra={
        <Button onClick={openDialog} disabled={exportDisabled} loading={submitting}>
          {messages.exportButton}
        </Button>
      }
    >
      {hasWorks === false ? <p className="settings-hint">{messages.exportNoWork}</p> : null}
      {phase === "running" && run ? (
        <div aria-live="polite">
          <Progress label={messages.exportRunning} />
          <p className="settings-hint">{messages.exportRunningHint}</p>
        </div>
      ) : null}

      <Dialog
        open={open}
        onOpenChange={handleOpenChange}
        title={messages.exportDialogTitle}
        description={messages.exportDialogDescription}
        footer={
          phase === "running" ? (
            <Button variant="secondary" disabled>
              {messages.cancel}
            </Button>
          ) : phase === "completed" && run ? (
            <>
              <Button variant="secondary" onClick={() => handleOpenChange(false)}>
                {messages.exportClose}
              </Button>
              <Button variant="secondary" onClick={() => void copy(path)}>
                {messages.exportCopyPath}
              </Button>
              <a className="ui-button ui-button--primary ui-button--md" href={exportDownloadUrl(run.id)} download>
                <span className="ui-button__label">{messages.exportDownload}</span>
              </a>
            </>
          ) : phase === "failed" ? (
            <>
              <Button variant="secondary" onClick={() => handleOpenChange(false)}>
                {messages.exportClose}
              </Button>
              <Button onClick={() => void submit()} loading={submitting}>
                {messages.exportRetry}
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={() => handleOpenChange(false)}>
                {messages.cancel}
              </Button>
              <Button onClick={() => void submit()} loading={submitting}>
                {messages.exportSubmit}
              </Button>
            </>
          )
        }
      >
        {phase === "idle" ? (
          <label className="export-checkbox">
            <input
              type="checkbox"
              checked={includeTrash}
              onChange={(event) => setIncludeTrash(event.target.checked)}
            />
            <span>
              <span className="export-checkbox__label">{messages.exportIncludeTrash}</span>
              <span className="export-checkbox__hint">{messages.exportIncludeTrashHint}</span>
            </span>
          </label>
        ) : null}

        {phase === "running" ? (
          <div aria-live="polite">
            <Progress label={messages.exportRunning} />
            <p className="settings-hint">{messages.exportRunningHint}</p>
          </div>
        ) : null}

        {phase === "completed" && run ? (
          <div className="export-result">
            <p className="export-result__title">{messages.exportResultTitle}</p>
            <dl className="export-result__list">
              <div className="export-result__row">
                <dt>{messages.exportFileLabel}</dt>
                <dd>{run.file_name ?? "—"}</dd>
              </div>
              <div className="export-result__row">
                <dt>{messages.exportSizeLabel}</dt>
                <dd>{formatBytes(run.byte_size)}</dd>
              </div>
              <div className="export-result__row">
                <dt>{messages.exportCreatedLabel}</dt>
                <dd>{formatDateTime(run.finished_at ?? run.created_at)}</dd>
              </div>
              <div className="export-result__row">
                <dt>{messages.settingsDataDirLabel}</dt>
                <dd>
                  <span className="export-path" title={path}>
                    {truncateMiddle(path)}
                  </span>
                </dd>
              </div>
            </dl>
          </div>
        ) : null}

        {phase === "failed" ? (
          <Alert tone="danger" title={messages.exportFailedTitle}>
            {error ?? messages.errorByCode.EXPORT_FAILED}
          </Alert>
        ) : null}
      </Dialog>
    </Card>
  );
}
