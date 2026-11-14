import { requestJson } from "./client";
import type { CharacterSummary } from "./characters";
import type { ReviewStatus } from "./system";

export type RankingTopCharacter = {
  id: string;
  name: string;
};

/** 列表前三席；空席 character 为 null。 */
export type RankingTopSeat = {
  seat_no: number;
  character: RankingTopCharacter | null;
};

export type RankingSummary = {
  id: string;
  work_id: string;
  name: string;
  seat_count: number;
  occupied_count: number;
  empty_count: number;
  top_seats: RankingTopSeat[];
  review_status: ReviewStatus;
  reviewed_at: string | null;
  updated_at: string;
  version: number;
};

export type SeatView = {
  seat_no: number;
  character: CharacterSummary | null;
  note: string | null;
  version: number;
};

export type SeatPage = {
  items: SeatView[];
  page: number;
  page_size: number;
  total: number;
};

export type RankingDetail = RankingSummary & {
  seats: SeatPage;
};

export type RankingList = {
  items: RankingSummary[];
  total: number;
  page: number;
  page_size: number;
};

export type RankingSort = "updated_at:desc" | "name:asc";

export type ListRankingsOptions = {
  workID: string;
  q?: string;
  reviewStatus?: ReviewStatus;
  sort?: RankingSort;
  page?: number;
  pageSize?: number;
};

export type CreateRankingInput = {
  workID: string;
  name: string;
  seatCount: number;
};

export type UpdateRankingInput = {
  name?: string;
  seatCount?: number;
  version: number;
};

export type DeleteRankingResult = {
  deleted: boolean;
  id: string;
  deleted_at: string;
  undo_token: string;
  expires_at: string;
};

export type ListSeatsOptions = {
  page?: number;
  pageSize?: number;
  occupied?: boolean;
};

export type PutSeatInput = {
  /** 省略保持不变；null 清空并清除备注。 */
  characterID?: string | null;
  /** 省略保持不变；null 或空串清空备注。 */
  note?: string | null;
  version: number;
};

export type MoveSeatInput = {
  targetSeatNo: number;
  mode: "empty" | "swap";
  version: number;
};

export type MoveSeatResult = {
  from: SeatView;
  to: SeatView;
};

export function listRankings(options: ListRankingsOptions): Promise<RankingList> {
  const search = new URLSearchParams({ work_id: options.workID });
  if (options.q) search.set("q", options.q);
  if (options.reviewStatus) search.set("review_status", options.reviewStatus);
  if (options.sort) search.set("sort", options.sort);
  if (options.page) search.set("page", String(options.page));
  if (options.pageSize) search.set("page_size", String(options.pageSize));
  return requestJson<RankingList>(`/api/v1/rankings?${search.toString()}`);
}

export function getRanking(id: string, page = 1, pageSize = 50): Promise<RankingDetail> {
  const search = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  return requestJson<RankingDetail>(`/api/v1/rankings/${id}?${search.toString()}`);
}

export function createRanking(input: CreateRankingInput): Promise<RankingDetail> {
  return requestJson<RankingDetail>("/api/v1/rankings", {
    method: "POST",
    body: {
      work_id: input.workID,
      name: input.name,
      seat_count: input.seatCount,
    },
  });
}

export function updateRanking(id: string, input: UpdateRankingInput): Promise<RankingDetail> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.seatCount !== undefined) body.seat_count = input.seatCount;
  return requestJson<RankingDetail>(`/api/v1/rankings/${id}`, {
    method: "PATCH",
    body,
    ifMatch: input.version,
  });
}

export function deleteRanking(id: string, version: number): Promise<DeleteRankingResult> {
  return requestJson<DeleteRankingResult>(`/api/v1/rankings/${id}`, {
    method: "DELETE",
    ifMatch: version,
  });
}

export function listSeats(id: string, options: ListSeatsOptions = {}): Promise<SeatPage> {
  const search = new URLSearchParams();
  if (options.page) search.set("page", String(options.page));
  if (options.pageSize) search.set("page_size", String(options.pageSize));
  if (typeof options.occupied === "boolean") search.set("occupied", String(options.occupied));
  const query = search.toString();
  return requestJson<SeatPage>(`/api/v1/rankings/${id}/seats${query ? `?${query}` : ""}`);
}

export function putSeat(id: string, seatNo: number, input: PutSeatInput): Promise<SeatView> {
  const body: Record<string, unknown> = {};
  if (input.characterID !== undefined) body.character_id = input.characterID;
  if (input.note !== undefined) body.note = input.note;
  return requestJson<SeatView>(`/api/v1/rankings/${id}/seats/${seatNo}`, {
    method: "PUT",
    body,
    ifMatch: input.version,
  });
}

export function moveSeat(id: string, seatNo: number, input: MoveSeatInput): Promise<MoveSeatResult> {
  return requestJson<MoveSeatResult>(`/api/v1/rankings/${id}/seats/${seatNo}/move`, {
    method: "POST",
    body: {
      target_seat_no: input.targetSeatNo,
      mode: input.mode,
    },
    ifMatch: input.version,
  });
}
