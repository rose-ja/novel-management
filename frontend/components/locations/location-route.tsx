"use client";

import { useSearchParams } from "next/navigation";

import { LocationDetailPage } from "@/components/locations/location-detail-page";
import { LocationListPage } from "@/components/locations/location-list-page";

/** `/locations` 与 `/locations?id=` 共用静态路由，详情由查询参数切换。 */
export function LocationRoute() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id");
  if (id) {
    return <LocationDetailPage key={id} locationID={id} />;
  }
  return <LocationListPage />;
}
