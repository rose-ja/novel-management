import { ApiError, NetworkError, requestJson } from "./client";

export type ImportStatus = "uploaded" | "validating" | "validated" | "running" | "completed" | "failed";

export type ImportCounts = {
  characters: number;
  factions: number;
  locations: number;
  rankings: number;
  system_categories: number;
  system_nodes: number;
  chronicles: number;
  images: number;
  reviewable: number;
};

export type ImportIssue = {
  line: number;
  field: string;
  code: string;
  message: string;
};

export type ImportImageSummary = {
  matched: number;
  missing: string[];
  extra: string[];
  unsupported: string[];
  too_large: string[];
};

export type ImportDetail = {
  import_id: string;
  status: ImportStatus;
  stage: string | null;
  file_name: string;
  archive_name: string | null;
  template_version: string | null;
  work_name: string | null;
  work_id: string | null;
  snapshot_id: string | null;
  counts: ImportCounts;
  images: ImportImageSummary;
  errors: ImportIssue[];
  warnings: ImportIssue[];
  error_count: number;
  error_code: string | null;
  created_at: string;
  finished_at: string | null;
};

export type ImportReport = {
  import_id: string;
  status: ImportStatus;
  file_name: string;
  archive_name: string | null;
  template_version: string;
  work_id: string | null;
  work_name: string | null;
  snapshot_id: string | null;
  counts: ImportCounts;
  images: ImportImageSummary;
  errors: ImportIssue[];
  warnings: ImportIssue[];
  error_count: number;
  elapsed_ms: number;
  created_at: string;
  finished_at: string | null;
};

export type CreateImportResult = {
  import_id: string;
  status: ImportStatus;
};

export type CommitImportResult = {
  status: "completed";
  import_id: string;
  work_id: string | null;
  report: ImportReport;
  review_url: string;
};

export const importTemplateUrl = "/api/v1/imports/template";
export const importExampleUrl = "/api/v1/imports/template/example";

/** 上传 Markdown 与图片 zip；用 XHR 提供上传进度，并支持取消。 */
export function uploadImport(
  markdown: File,
  archive: File | null,
  options: { onProgress?: (percent: number) => void; signal?: AbortSignal } = {},
): Promise<CreateImportResult> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const form = new FormData();
    form.append("markdown", markdown);
    if (archive) form.append("archive", archive);

    const abort = () => xhr.abort();
    options.signal?.addEventListener("abort", abort);

    xhr.open("POST", "/api/v1/imports");
    xhr.setRequestHeader("Accept", "application/json");
    xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) return;
      options.onProgress?.(Math.round((event.loaded / event.total) * 100));
    };
    xhr.onerror = () => reject(new NetworkError());
    xhr.onabort = () => reject(new NetworkError());
    xhr.onload = () => {
      options.signal?.removeEventListener("abort", abort);
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText) as CreateImportResult);
        } catch {
          reject(new ApiError({ code: "INVALID_RESPONSE", message: "服务器响应无法解析", status: xhr.status }));
        }
        return;
      }
      reject(parseXHRError(xhr));
    };
    xhr.send(form);
  });
}

function parseXHRError(xhr: XMLHttpRequest): ApiError {
  try {
    const body = JSON.parse(xhr.responseText) as { error?: { code?: string; message?: string; request_id?: string } };
    return new ApiError({
      code: body.error?.code ?? `HTTP_${xhr.status}`,
      message: body.error?.message ?? "上传失败，请稍后重试",
      status: xhr.status,
      requestId: body.error?.request_id,
    });
  } catch {
    return new ApiError({ code: `HTTP_${xhr.status}`, message: "上传失败，请稍后重试", status: xhr.status });
  }
}

export function validateImport(id: string): Promise<{ import_id: string; status: ImportStatus }> {
  return requestJson(`/api/v1/imports/${encodeURIComponent(id)}/validate`, { method: "POST" });
}

export function fetchImport(id: string): Promise<ImportDetail> {
  return requestJson<ImportDetail>(`/api/v1/imports/${encodeURIComponent(id)}`);
}

export function fetchImportReport(id: string): Promise<ImportReport> {
  return requestJson<ImportReport>(`/api/v1/imports/${encodeURIComponent(id)}/report`);
}

export function commitImport(id: string, workName?: string): Promise<CommitImportResult> {
  return requestJson<CommitImportResult>(`/api/v1/imports/${encodeURIComponent(id)}/commit`, {
    method: "POST",
    body: workName ? { work_name: workName } : {},
  });
}
