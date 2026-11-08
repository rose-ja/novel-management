import { requestJson } from "./client";
import type { AppSettings } from "./types";

export function fetchSettings(): Promise<AppSettings> {
  return requestJson<AppSettings>("/api/v1/settings");
}

/** 切换当前作品；传 null 清空选择。 */
export function updateCurrentWork(currentWorkId: string | null): Promise<AppSettings> {
  return requestJson<AppSettings>("/api/v1/settings", {
    method: "PATCH",
    body: { current_work_id: currentWorkId },
  });
}
