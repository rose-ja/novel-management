import { requestJson } from "./client";

export type Work = {
  id: string;
  name: string;
  description: string | null;
  version: number;
  created_at: string;
  updated_at: string;
};

export type WorkPage = {
  items: Work[];
  page: number;
  page_size: number;
  total: number;
};

export type DeleteWorkResult = {
  deleted: boolean;
  undo_token: string;
  snapshot_id: string;
  expires_at: string;
};

export type CreateWorkInput = {
  name: string;
  description?: string | null;
};

export type UpdateWorkInput = {
  name?: string;
  /** 传 null 表示清空简介；省略表示保持不变。 */
  description?: string | null;
  version: number;
};

function pageQuery(params: { q?: string; page?: number; pageSize?: number }): string {
  const search = new URLSearchParams();
  if (params.q) search.set("q", params.q);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("page_size", String(params.pageSize));
  const value = search.toString();
  return value ? `?${value}` : "";
}

export function listWorks(params: { q?: string; page?: number; pageSize?: number } = {}): Promise<WorkPage> {
  return requestJson<WorkPage>(`/api/v1/works${pageQuery(params)}`);
}

export function getWork(id: string): Promise<Work> {
  return requestJson<Work>(`/api/v1/works/${id}`);
}

export function createWork(input: CreateWorkInput): Promise<Work> {
  return requestJson<Work>("/api/v1/works", {
    method: "POST",
    body: { name: input.name, description: input.description ?? null },
  });
}

export function updateWork(id: string, input: UpdateWorkInput): Promise<Work> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.description !== undefined) body.description = input.description;
  return requestJson<Work>(`/api/v1/works/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteWork(id: string, confirmName: string, version: number): Promise<DeleteWorkResult> {
  return requestJson<DeleteWorkResult>(`/api/v1/works/${id}`, {
    method: "DELETE",
    body: { confirm_name: confirmName },
    ifMatch: version,
  });
}
