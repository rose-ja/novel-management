"use client";

import { useSearchParams } from "next/navigation";

import { RankingDetailPage } from "@/components/rankings/ranking-detail-page";
import { RankingListPage } from "@/components/rankings/ranking-list-page";

/** `/rankings` 与 `/rankings?id=` 共用静态路由，详情由查询参数切换。 */
export function RankingRoute() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id");
  if (id) {
    return <RankingDetailPage key={id} rankingID={id} />;
  }
  return <RankingListPage />;
}
