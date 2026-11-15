"use client";

import { useSearchParams } from "next/navigation";

import { SystemPage } from "./system-page";

/** 供检索结果深链：/system?category=<id>&node=<id>。 */
export function SystemRoute() {
  const params = useSearchParams();
  return <SystemPage initialCategoryID={params.get("category")} initialNodeID={params.get("node")} />;
}
