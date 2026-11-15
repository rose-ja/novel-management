import { requestJson } from "./client";

export type ExportRunStatus = "queued" | "running" | "completed" | "failed";

export type ExportRun = {
  id: string;
  status: ExportRunStatus;
  include_trash: boolean;
  file_name: string | null;
  file_path: string | null;
  byte_size: number | null;
  error_code: string | null;
  created_at: string;
  finished_at: string | null;
};

export type ExportStatus = {
  active: ExportRun | null;
  last: ExportRun | null;
};

export type CreateExportResult = {
  export_id: string;
  status: ExportRunStatus;
};

/** 创建后台导出任务；include_trash 默认由页面勾选。 */
export function createExport(includeTrash: boolean): Promise<CreateExportResult> {
  return requestJson<CreateExportResult>("/api/v1/exports", {
    method: "POST",
    body: { include_trash: includeTrash },
  });
}

export function fetchExportStatus(): Promise<ExportStatus> {
  return requestJson<ExportStatus>("/api/v1/export/status");
}

export function fetchExport(id: string): Promise<ExportRun> {
  return requestJson<ExportRun>(`/api/v1/exports/${encodeURIComponent(id)}`);
}

export function listExports(): Promise<ExportRun[]> {
  return requestJson<ExportRun[]>("/api/v1/exports");
}

export function exportDownloadUrl(id: string): string {
  return `/api/v1/exports/${encodeURIComponent(id)}/download`;
}

/** 组合展示用绝对路径；data_dir 可能为空或带尾斜杠。 */
export function resolveExportPath(dataDir: string, filePath: string): string {
  const cleanDir = dataDir.replace(/\/+$/, "");
  const cleanFile = filePath.replace(/^\/+/, "");
  if (!cleanDir) return cleanFile;
  return `${cleanDir}/${cleanFile}`;
}
