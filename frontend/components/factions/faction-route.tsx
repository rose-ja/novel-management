"use client";

import { useSearchParams } from "next/navigation";

import { FactionDetailPage } from "@/components/factions/faction-detail-page";
import { FactionListPage } from "@/components/factions/faction-list-page";

/** `/factions` 与 `/factions?id=` 共用静态路由，详情由查询参数切换。 */
export function FactionRoute() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id");
  if (id) {
    return <FactionDetailPage key={id} factionID={id} />;
  }
  return <FactionListPage />;
}
