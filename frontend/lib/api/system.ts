import { requestJson, requestVoid } from "./client";

export type CategoryType = "normal" | "era";

/** API 层校对状态：数据库 reviewed 映射为 confirmed。 */
export type ReviewStatus = "confirmed" | "pending";

export type SystemCategory = {
  id: string;
  work_id: string;
  name: string;
  category_type: CategoryType;
  is_builtin: boolean;
  is_enabled: boolean;
  sort_order: number;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
  node_count: number;
  reference_count: number;
};

export type SystemNode = {
  id: string;
  work_id: string;
  category_id: string;
  parent_id?: string | null;
  name: string;
  sort_order: number;
  era_start_text?: string | null;
  era_end_text?: string | null;
  review_status: ReviewStatus;
  reviewed_at?: string | null;
  reviewed_by?: string | null;
  version: number;
  reference_count: number;
  /** 默认（树形）响应中存在的子节点；平铺响应无此字段。 */
  children?: SystemNode[];
};

/** 类别删除返回三件套；affected_nodes 表示被级联移入回收站的节点数。 */
export type DeleteCategoryResult = {
  deleted: boolean;
  undo_token: string;
  affected_nodes: number;
  expires_at: string;
};

/** 节点删除返回同一三件套，后端不返回 affected_nodes。 */
export type DeleteNodeResult = {
  deleted: boolean;
  undo_token: string;
  expires_at: string;
};

export type CreateCategoryInput = {
  workID: string;
  name: string;
  categoryType: CategoryType;
  isEnabled?: boolean;
};

export type UpdateCategoryInput = {
  name?: string;
  isEnabled?: boolean;
  version: number;
};

export type ListNodesOptions = {
  flat?: boolean;
  reviewStatus?: "pending";
};

export type CreateNodeInput = {
  categoryID: string;
  name: string;
  parentID?: string;
  eraStartText?: string | null;
  eraEndText?: string | null;
};

export type UpdateNodeInput = {
  name?: string;
  sortOrder?: number;
  /** 传 null 表示清空纪元文本；省略表示保持不变。 */
  eraStartText?: string | null;
  /** 传 null 表示清空纪元文本；省略表示保持不变。 */
  eraEndText?: string | null;
  version: number;
};

export type ReorderNodesInput = {
  categoryID: string;
  parentID?: string | null;
  orderedIDs: string[];
};

export function listCategories(workID: string, includeDisabled = true): Promise<SystemCategory[]> {
  const search = new URLSearchParams();
  search.set("work_id", workID);
  search.set("include_disabled", includeDisabled ? "true" : "false");
  return requestJson<{ items: SystemCategory[] }>(`/api/v1/system-categories?${search.toString()}`).then(
    (response) => response.items,
  );
}

export function createCategory(input: CreateCategoryInput): Promise<SystemCategory> {
  const body: Record<string, unknown> = {
    work_id: input.workID,
    name: input.name,
    category_type: input.categoryType,
  };
  if (input.isEnabled !== undefined) body.is_enabled = input.isEnabled;
  return requestJson<SystemCategory>("/api/v1/system-categories", {
    method: "POST",
    body,
  });
}

export function updateCategory(id: string, input: UpdateCategoryInput): Promise<SystemCategory> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.isEnabled !== undefined) body.is_enabled = input.isEnabled;
  return requestJson<SystemCategory>(`/api/v1/system-categories/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteCategory(id: string, version: number): Promise<DeleteCategoryResult> {
  return requestJson<DeleteCategoryResult>(`/api/v1/system-categories/${id}`, {
    method: "DELETE",
    ifMatch: version,
  });
}

/** 默认返回树形；flat=true 时返回平铺列表，可按校对状态筛选。 */
export function listNodes(categoryID: string, options: ListNodesOptions = {}): Promise<SystemNode[]> {
  const search = new URLSearchParams();
  if (options.flat) search.set("flat", "true");
  if (options.reviewStatus) search.set("review_status", options.reviewStatus);
  const query = search.toString();
  return requestJson<{ items: SystemNode[] }>(
    `/api/v1/system-categories/${categoryID}/nodes${query ? `?${query}` : ""}`,
  ).then((response) => response.items);
}

export function createNode(input: CreateNodeInput): Promise<SystemNode> {
  const body: Record<string, unknown> = {
    category_id: input.categoryID,
    name: input.name,
  };
  if (input.parentID !== undefined) body.parent_id = input.parentID;
  if (input.eraStartText !== undefined) body.era_start_text = input.eraStartText;
  if (input.eraEndText !== undefined) body.era_end_text = input.eraEndText;
  return requestJson<SystemNode>("/api/v1/system-nodes", {
    method: "POST",
    body,
  });
}

export function updateNode(id: string, input: UpdateNodeInput): Promise<SystemNode> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.sortOrder !== undefined) body.sort_order = input.sortOrder;
  // 显式 null 必须原样发送，用于清空纪元文本。
  if (input.eraStartText !== undefined) body.era_start_text = input.eraStartText;
  if (input.eraEndText !== undefined) body.era_end_text = input.eraEndText;
  return requestJson<SystemNode>(`/api/v1/system-nodes/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteNode(id: string, version: number): Promise<DeleteNodeResult> {
  return requestJson<DeleteNodeResult>(`/api/v1/system-nodes/${id}`, {
    method: "DELETE",
    ifMatch: version,
  });
}

export function reorderNodes(input: ReorderNodesInput): Promise<void> {
  const body: Record<string, unknown> = {
    category_id: input.categoryID,
    ordered_ids: input.orderedIDs,
  };
  if (input.parentID !== undefined) body.parent_id = input.parentID;
  return requestVoid("/api/v1/system-nodes/reorder", {
    method: "POST",
    body,
  });
}
