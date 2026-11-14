import type { Metadata } from "next";
import { Suspense } from "react";

import { RankingRoute } from "@/components/rankings/ranking-route";

export const metadata: Metadata = {
  title: "榜单",
};

export default function RankingsPage() {
  return (
    <Suspense fallback={null}>
      <RankingRoute />
    </Suspense>
  );
}
