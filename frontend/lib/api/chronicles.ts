import { requestJson } from "./client";

export type ChronicleOwnerType = "character" | "faction";

export type Chronicle = {
  id: string;
  owner_type: ChronicleOwnerType;
  owner_id: string;
  text: string;
  time_label: string | null;
  sort_order: number;
  version: number;
  created_at: string;
  updated_at: string;
};

export type ChronicleInput = {
  ownerType: ChronicleOwnerType;
  ownerID: string;
  text: string;
  timeLabel?: string | null;
};

export type ChronicleUpdate = {
  text?: string;
  timeLabel?: string | null;
  sortOrder?: number;
  version: number;
};

export type DeleteChronicleResult = {
  id: string;
  deleted_at: string;
  undo_token: string;
  expires_at: string;
};

export async function listChronicles(
  ownerType: ChronicleOwnerType,
  ownerID: string,
): Promise<Chronicle[]> {
  const result = await requestJson<{ items: Chronicle[] }>(
    `/api/v1/chronicles?owner_type=${ownerType}&owner_id=${encodeURIComponent(ownerID)}`,
  );
  return result.items ?? [];
}

export function createChronicle(input: ChronicleInput): Promise<Chronicle> {
  return requestJson<Chronicle>("/api/v1/chronicles", {
    method: "POST",
    body: {
      owner_type: input.ownerType,
      owner_id: input.ownerID,
      text: input.text,
      time_label: input.timeLabel ?? null,
    },
  });
}

export function updateChronicle(id: string, input: ChronicleUpdate): Promise<Chronicle> {
  const body: Record<string, unknown> = {};
  if (input.text !== undefined) body.text = input.text;
  if (input.timeLabel !== undefined) body.time_label = input.timeLabel;
  if (input.sortOrder !== undefined) body.sort_order = input.sortOrder;
  return requestJson<Chronicle>(`/api/v1/chronicles/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteChronicle(id: string): Promise<DeleteChronicleResult> {
  return requestJson<DeleteChronicleResult>(`/api/v1/chronicles/${id}`, { method: "DELETE" });
}
