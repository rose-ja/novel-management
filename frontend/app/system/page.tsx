import { Suspense } from "react";

import { SystemRoute } from "@/components/system/system-route";
import { Skeleton } from "@/components/ui/skeleton";
import { messages } from "@/lib/messages";

export default function SystemPageRoute() {
  return (
    <Suspense fallback={<Skeleton variant="lines" lines={8} label={messages.systemLoadingCategories} />}>
      <SystemRoute />
    </Suspense>
  );
}
