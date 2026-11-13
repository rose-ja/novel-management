import { Suspense } from "react";

import { FactionRoute } from "@/components/factions/faction-route";
import { Skeleton } from "@/components/ui/skeleton";
import { messages } from "@/lib/messages";

export default function FactionsRoute() {
  return (
    <Suspense
      fallback={
        <div className="page">
          <Skeleton variant="lines" lines={6} label={messages.factionTreeAria} />
        </div>
      }
    >
      <FactionRoute />
    </Suspense>
  );
}
