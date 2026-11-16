"use client";

import { useCallback, useEffect, useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { ReportStep } from "@/components/import/report-step";
import { ReviewStep } from "@/components/import/review-step";
import { ImportStepper, type ImportStep } from "@/components/import/stepper";
import { UploadStep } from "@/components/import/upload-step";
import { ValidationStep } from "@/components/import/validation-step";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog } from "@/components/ui/dialog";
import { PageHeader } from "@/components/ui/page-header";
import { Progress } from "@/components/ui/progress";
import { useToast } from "@/components/ui/toast";
import { ApiError, NetworkError } from "@/lib/api/client";
import {
  commitImport,
  fetchImport,
  uploadImport,
  type ImportDetail,
  type ImportReport,
} from "@/lib/api/imports";
import { messages } from "@/lib/messages";

function errorText(error: unknown): string {
  if (error instanceof ApiError || error instanceof NetworkError) return error.message;
  return messages.loadFailed;
}

export function ImportWizard() {
  const { reload } = useSettings();
  const { showToast } = useToast();
  const [step, setStep] = useState<ImportStep>(1);
  const [markdown, setMarkdown] = useState<File | null>(null);
  const [archive, setArchive] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [importID, setImportID] = useState<string | null>(null);
  const [detail, setDetail] = useState<ImportDetail | null>(null);
  const [report, setReport] = useState<ImportReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [committing, setCommitting] = useState(false);

  useEffect(() => {
    if (!importID || step !== 2) return;
    let active = true;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const load = async () => {
      try {
        const next = await fetchImport(importID);
        if (!active) return;
        setDetail(next);
        if (next.status === "uploaded" || next.status === "validating") {
          timer = setTimeout(() => void load(), 1000);
        }
      } catch (pollError) {
        if (!active) return;
        setError(errorText(pollError));
      }
    };
    void load();
    return () => {
      active = false;
      if (timer) clearTimeout(timer);
    };
  }, [importID, step]);

  const startUpload = useCallback(async () => {
    if (!markdown) return;
    setError(null);
    setUploading(true);
    setProgress(0);
    try {
      const result = await uploadImport(markdown, archive, { onProgress: setProgress });
      setImportID(result.import_id);
      setDetail(null);
      setStep(2);
    } catch (uploadError) {
      setError(errorText(uploadError));
    } finally {
      setUploading(false);
    }
  }, [archive, markdown]);

  const reset = useCallback(() => {
    setStep(1);
    setMarkdown(null);
    setArchive(null);
    setImportID(null);
    setDetail(null);
    setReport(null);
    setError(null);
    setProgress(0);
  }, []);

  const submitCommit = useCallback(async () => {
    if (!importID) return;
    setConfirmOpen(false);
    setCommitting(true);
    setError(null);
    setStep(3);
    try {
      const result = await commitImport(importID, detail?.work_name ?? undefined);
      setReport(result.report);
      setStep(4);
      showToast({ tone: "success", message: messages.importSuccessTitle });
      void reload();
    } catch (commitError) {
      setError(errorText(commitError));
      setStep(2);
    } finally {
      setCommitting(false);
    }
  }, [detail?.work_name, importID, reload, showToast]);

  return (
    <div className="page">
      <PageHeader title={messages.importTitle} description={messages.importDescription} />
      <ImportStepper step={step} hasError={Boolean(error) || (detail?.error_count ?? 0) > 0} />

      {step === 1 ? (
        <>
          <UploadStep
            markdown={markdown}
            archive={archive}
            uploading={uploading}
            progress={progress}
            error={error}
            onMarkdown={setMarkdown}
            onArchive={setArchive}
            onStart={() => void startUpload()}
          />
          <p className="import-muted">{messages.importNoWorkDescription}</p>
        </>
      ) : null}

      {step === 2 ? <ValidationStep detail={detail} error={error} onReupload={reset} onConfirm={() => setConfirmOpen(true)} /> : null}

      {step === 3 ? (
        <Card title={messages.importRunning} description={messages.importRunningHint}>
          <Progress label={messages.importRunning} />
          <ol className="import-stages">
            <li>{messages.importStageParse}</li>
            <li>{messages.importStageValidate}</li>
            <li>{messages.importStageImages}</li>
            <li>{messages.importStageWrite}</li>
          </ol>
          {error ? <Alert tone="danger" title={error} /> : null}
        </Card>
      ) : null}

      {step === 4 && report ? <ReportStep report={report} onReview={() => setStep(5)} /> : null}

      {step === 5 && report?.work_id ? <ReviewStep workID={report.work_id} /> : null}

      <Dialog
        open={confirmOpen}
        onOpenChange={(open) => {
          if (!committing) setConfirmOpen(open);
        }}
        title={messages.importConfirmTitle(detail?.work_name ?? "新作品")}
        description={messages.importConfirmDescription}
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmOpen(false)} disabled={committing}>
              {messages.importBackCheck}
            </Button>
            <Button onClick={() => void submitCommit()} loading={committing}>
              {messages.importConfirmSubmit}
            </Button>
          </>
        }
      />
    </div>
  );
}
