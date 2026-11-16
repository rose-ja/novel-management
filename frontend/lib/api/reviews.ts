import { requestJson } from "./client";

export type ReviewEntityType = "characters" | "factions" | "locations" | "rankings" | "system-nodes";

export type ReviewResult = {
  entity_type: ReviewEntityType;
  id: string;
  review_status: "confirmed";
  reviewed_at: string | null;
  reviewed_by: string | null;
  version: number;
};

export type ReviewBatchInput = {
  entity_type: ReviewEntityType;
  ids?: string[];
  work_id?: string;
  category_id?: string;
};

export type ReviewBatchResult = {
  updated: number;
  skipped: number;
  items: ReviewResult[];
};

export function confirmReview(type: ReviewEntityType, id: string): Promise<ReviewResult> {
  return requestJson<ReviewResult>(`/api/v1/${type}/${encodeURIComponent(id)}/review`, { method: "POST" });
}

export function batchReview(input: ReviewBatchInput): Promise<ReviewBatchResult> {
  return requestJson<ReviewBatchResult>("/api/v1/reviews/batch", { method: "POST", body: input });
}
