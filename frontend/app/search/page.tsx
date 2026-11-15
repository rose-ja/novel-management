import { Suspense } from "react";

import { SearchRoute } from "@/components/search/search-route";
import { Skeleton } from "@/components/ui/skeleton";
import { messages } from "@/lib/messages";

export default function SearchPageRoute() {
  return (
    <Suspense fallback={<Skeleton variant="lines" lines={6} label={messages.loadFailed} />}>
      <SearchRoute />
    </Suspense>
  );
}
