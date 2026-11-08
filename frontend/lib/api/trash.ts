import { requestJson, requestVoid } from "./client";

export type TrashRootType =
  | "work"
  | "character"
  | "faction"
  | "location"
  | "ranking"
  | "system_category"
  | "system_node"
  | "chronicle";

export type TrashItem = {
  root_type: TrashRootType;
  root_id: string;
  name: string;
  parent_name: string | null;
  deleted_at: string | null;
  affected_count: number;
};

export type TrashPage = {
  items: TrashItem[];
  page: number;
  page_size: number;
  total: number;
};

export type RenameItem = {
  entity_type: string;
  entity_id: string;
  new_name: string;
};

export type RestoreInput = {
  newName?: string | null;
  renames?: RenameItem[];
};

export type RestoreResult = {
  root_type: TrashRootType;
  root_id: string;
  name: string;
  restored_count: number;
};

export type UndoResult = {
  restored: boolean;
  root_type: TrashRootType;
  root_id: string;
};

export type PurgeAllResult = {
  purged: number;
  snapshot_id: string;
};

export function listTrash(
  params: { rootType?: TrashRootType; page?: number; pageSize?: number } = {},
): Promise<TrashPage> {
  const search = new URLSearchParams();
  if (params.rootType) search.set("root_type", params.rootType);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("page_size", String(params.pageSize));
  const query = search.toString();
  return requestJson<TrashPage>(`/api/v1/trash${query ? `?${query}` : ""}`);
}

export function restoreTrash(rootType: TrashRootType, rootId: string, input: RestoreInput = {}): Promise<RestoreResult> {
  return requestJson<RestoreResult>(`/api/v1/trash/${rootType}/${rootId}/restore`, {
    method: "POST",
    body: {
      new_name: input.newName ?? null,
      renames: input.renames ?? [],
    },
  });
}

export function purgeTrashItem(rootType: TrashRootType, rootId: string): Promise<void> {
  return requestVoid(`/api/v1/trash/${rootType}/${rootId}`, {
    method: "DELETE",
    body: { confirm: true },
  });
}

export function purgeAllTrash(): Promise<PurgeAllResult> {
  return requestJson<PurgeAllResult>("/api/v1/trash", {
    method: "DELETE",
    body: { confirm: "PURGE_ALL", work_id: null },
  });
}

export function undoTrash(undoToken: string): Promise<UndoResult> {
  return requestJson<UndoResult>("/api/v1/trash/undo", {
    method: "POST",
    body: { undo_token: undoToken },
  });
}
