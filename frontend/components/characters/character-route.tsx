"use client";

import { useSearchParams } from "next/navigation";

import { CharacterDetailPage } from "@/components/characters/character-detail-page";
import { CharacterListPage } from "@/components/characters/character-list-page";

/** `/characters` 与 `/characters?id=` 共用静态路由，详情由查询参数切换。 */
export function CharacterRoute() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id");
  if (id) {
    return <CharacterDetailPage key={id} characterID={id} />;
  }
  return <CharacterListPage />;
}
