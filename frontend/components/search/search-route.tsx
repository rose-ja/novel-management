"use client";

import { useSearchParams } from "next/navigation";

import { parseSearchState } from "@/lib/search";

import { SearchPage } from "./search-page";

export function SearchRoute() {
  const params = useSearchParams();
  return <SearchPage initial={parseSearchState(params)} />;
}
