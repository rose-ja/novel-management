import { messages } from "../messages";

import { ApiError, NetworkError } from "./client";

export type ConflictItem = {
  id: string;
  name: string;
};

/** 把未知错误收敛为用户文案：网络错误优先，其次稳定错误码，最后回退。 */
export function toUserMessage(error: unknown, fallback: string = messages.saveFailed): string {
  if (error instanceof NetworkError) {
    return messages.networkError;
  }
  if (error instanceof ApiError) {
    return messages.errorByCode[error.code] ?? error.message ?? fallback;
  }
  return fallback;
}

export function isApiCode(error: unknown, code: string): boolean {
  return error instanceof ApiError && error.code === code;
}

/** RESTORE_CONFLICT 的 details.items；结构不符时返回空数组。 */
export function conflictItems(error: unknown): ConflictItem[] {
  if (!(error instanceof ApiError)) {
    return [];
  }
  const details = error.details as { items?: unknown } | null | undefined;
  if (!details || !Array.isArray(details.items)) {
    return [];
  }
  return details.items.filter((item): item is ConflictItem => {
    if (typeof item !== "object" || item === null) return false;
    const candidate = item as Partial<ConflictItem>;
    return typeof candidate.id === "string" && typeof candidate.name === "string";
  });
}
