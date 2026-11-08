import { requestJson } from "./client";

export type Snapshot = {
  id: string;
  reason: string;
  path: string;
  db_bytes: number;
  image_count: number;
  status: string;
  created_at: string;
};

/** 最近 10 条快照记录；只读，不提供页面内恢复。 */
export function listSnapshots(): Promise<Snapshot[]> {
  return requestJson<Snapshot[]>("/api/v1/snapshots");
}
