"use client";

import Link from "next/link";

import type { CharacterRankingRef } from "@/lib/api/characters";
import { messages } from "@/lib/messages";
import { rankingRefLabel } from "@/lib/rankings";

export type CharacterRankingLinksProps = {
  rankings: CharacterRankingRef[];
  count: number;
  /** 详情页展示全部名次；列表/成员表仅展示两条。 */
  limit?: number;
  emptyText?: string;
};

/** 人物名次回显：`榜单名·第 N 席`，超出上限用 +N 折叠。 */
export function CharacterRankingLinks({ rankings, count, limit = 2, emptyText }: CharacterRankingLinksProps) {
  if (rankings.length === 0) {
    return <span className="character-muted">{emptyText ?? messages.characterNoValue}</span>;
  }
  const shown = rankings.slice(0, limit);
  const more = Math.max(0, count - shown.length);
  return (
    <span className="character-rankings">
      {shown.map((ref) => (
        <Link
          key={ref.ranking_id}
          className="character-name-link"
          href={`/rankings?id=${encodeURIComponent(ref.ranking_id)}`}
          aria-label={messages.characterRankingLink(ref.name, ref.seat_no)}
        >
          {rankingRefLabel(ref)}
        </Link>
      ))}
      {more > 0 ? <span className="character-muted">{messages.characterRankingMore(more)}</span> : null}
    </span>
  );
}
