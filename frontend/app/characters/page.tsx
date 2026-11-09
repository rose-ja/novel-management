import { Suspense } from "react";

import { CharacterRoute } from "@/components/characters/character-route";
import { Skeleton } from "@/components/ui/skeleton";
import { messages } from "@/lib/messages";

export default function CharactersRoute() {
  return (
    <Suspense
      fallback={
        <div className="page">
          <Skeleton variant="lines" lines={6} label={messages.characterLoading} />
        </div>
      }
    >
      <CharacterRoute />
    </Suspense>
  );
}
