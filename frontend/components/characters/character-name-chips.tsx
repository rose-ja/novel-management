"use client";

import type { CharacterName } from "@/lib/api/characters";
import { messages } from "@/lib/messages";

export type CharacterNameChipsProps = {
  names: CharacterName[];
  max?: number;
};

/** 别名/称号标签；超出上限时折叠为 +N。 */
export function CharacterNameChips({ names, max }: CharacterNameChipsProps) {
  if (names.length === 0) {
    return <span className="character-muted">{messages.characterNoValue}</span>;
  }
  const limit = max ?? names.length;
  const visible = names.slice(0, limit);
  const overflow = names.length - visible.length;
  return (
    <span className="character-chips">
      {visible.map((name, index) => (
        <span
          key={`${name.kind}-${name.value}-${index}`}
          className={`character-chip${name.kind === "title" ? " character-chip--title" : ""}`}
        >
          {name.value}
        </span>
      ))}
      {overflow > 0 ? (
        <span className="character-chip character-chip--more">
          {messages.characterNameOverflow(overflow)}
        </span>
      ) : null}
    </span>
  );
}
