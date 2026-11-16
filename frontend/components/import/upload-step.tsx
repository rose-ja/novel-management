"use client";

import { useRef, useState, type ChangeEvent, type DragEvent, type KeyboardEvent } from "react";
import { FileText, Package, UploadCloud, X } from "lucide-react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { importExampleUrl, importTemplateUrl } from "@/lib/api/imports";
import { messages } from "@/lib/messages";

const MAX_MARKDOWN = 20 * 1024 * 1024;
const MAX_ARCHIVE = 6 * 1024 * 1024 * 1024;

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function validateFile(file: File, kind: "markdown" | "archive"): string | null {
  if (kind === "markdown") {
    if (!/\.md$/i.test(file.name)) return "请选择 .md 模板文件";
    if (file.size > MAX_MARKDOWN) return "Markdown 不能超过 20MB";
    return null;
  }
  if (!/\.zip$/i.test(file.name)) return "请选择 .zip 图片包";
  if (file.size > MAX_ARCHIVE) return "图片 zip 不能超过 6GB";
  return null;
}

export type UploadStepProps = {
  markdown: File | null;
  archive: File | null;
  uploading: boolean;
  progress: number;
  error: string | null;
  onMarkdown: (file: File | null) => void;
  onArchive: (file: File | null) => void;
  onStart: () => void;
};

export function UploadStep({ markdown, archive, uploading, progress, error, onMarkdown, onArchive, onStart }: UploadStepProps) {
  const markdownInput = useRef<HTMLInputElement>(null);
  const archiveInput = useRef<HTMLInputElement>(null);
  const [localError, setLocalError] = useState<string | null>(null);

  const choose = (file: File | null, kind: "markdown" | "archive") => {
    if (!file) return;
    const message = validateFile(file, kind);
    if (message) {
      setLocalError(message);
      return;
    }
    setLocalError(null);
    if (kind === "markdown") onMarkdown(file);
    else onArchive(file);
  };

  const onDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    for (const file of Array.from(event.dataTransfer.files)) {
      choose(file, /\.md$/i.test(file.name) ? "markdown" : "archive");
    }
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    markdownInput.current?.click();
  };

  const onInput = (kind: "markdown" | "archive") => (event: ChangeEvent<HTMLInputElement>) => {
    choose(event.target.files?.[0] ?? null, kind);
    event.target.value = "";
  };

  return (
    <Card title={messages.importUploadTitle} description={messages.importUploadDescription}>
      <div
        className="import-dropzone"
        role="button"
        tabIndex={0}
        aria-label={messages.importDropHint}
        onClick={() => markdownInput.current?.click()}
        onKeyDown={onKeyDown}
        onDragOver={(event) => event.preventDefault()}
        onDrop={onDrop}
      >
        <UploadCloud aria-hidden="true" size={28} />
        <p>{messages.importDropHint}</p>
        <p className="import-muted">
          {messages.importMarkdownHint} · {messages.importArchiveHint}
        </p>
      </div>

      <input ref={markdownInput} type="file" accept=".md,text/markdown" className="ui-visually-hidden" onChange={onInput("markdown")} />
      <input ref={archiveInput} type="file" accept=".zip,application/zip" className="ui-visually-hidden" onChange={onInput("archive")} />

      <dl className="import-files">
        <div className="import-files__row">
          <dt>
            <FileText aria-hidden="true" size={16} /> {messages.importMarkdownLabel}
          </dt>
          <dd>
            {markdown ? (
              <>
                <span>{markdown.name}</span>
                <span className="import-muted">{formatSize(markdown.size)}</span>
                <button type="button" className="import-link" onClick={() => onMarkdown(null)}>
                  <X aria-hidden="true" size={12} /> {messages.importFileRemove}
                </button>
              </>
            ) : (
              <button type="button" className="import-link" onClick={() => markdownInput.current?.click()}>
                {messages.importChooseMarkdown}
              </button>
            )}
          </dd>
        </div>
        <div className="import-files__row">
          <dt>
            <Package aria-hidden="true" size={16} /> {messages.importArchiveLabel}
          </dt>
          <dd>
            {archive ? (
              <>
                <span>{archive.name}</span>
                <span className="import-muted">{formatSize(archive.size)}</span>
                <button type="button" className="import-link" onClick={() => onArchive(null)}>
                  <X aria-hidden="true" size={12} /> {messages.importFileRemove}
                </button>
              </>
            ) : (
              <button type="button" className="import-link" onClick={() => archiveInput.current?.click()}>
                {messages.importChooseArchive}
              </button>
            )}
          </dd>
        </div>
      </dl>

      {localError || error ? <Alert tone="danger" title={localError ?? error ?? ""} /> : null}

      {uploading ? <Progress value={progress} label={messages.importUploading} /> : null}

      <div className="import-actions">
        <div className="import-actions__links">
          <a className="import-link" href={importTemplateUrl} download>
            {messages.importTemplateDownload}
          </a>
          <a className="import-link" href={importExampleUrl} download>
            {messages.importExampleDownload}
          </a>
        </div>
        <Button onClick={onStart} loading={uploading} disabled={!markdown}>
          {messages.importStartValidate}
        </Button>
      </div>
    </Card>
  );
}
