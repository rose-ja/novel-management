import type { Metadata } from "next";
import { Suspense } from "react";

import { LocationRoute } from "@/components/locations/location-route";

export const metadata: Metadata = {
  title: "地点",
};

export default function LocationsPage() {
  return (
    <Suspense fallback={null}>
      <LocationRoute />
    </Suspense>
  );
}
