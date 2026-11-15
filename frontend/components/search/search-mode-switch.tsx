"use client";

import { messages } from "@/lib/messages";
import type { SearchMode } from "@/lib/search";

export type SearchModeSwitchProps = {
  mode: SearchMode;
  onChange: (mode: SearchMode) => void;
  disabled?: boolean;
};

export function SearchModeSwitch({ mode, onChange, disabled = false }: SearchModeSwitchProps) {
  return (
    <div className="search-mode" role="group" aria-label={messages.searchModeLabel}>
      <button
        type="button"
        className={`search-mode__button${mode === "quick" ? " search-mode__button--active" : ""}`}
        aria-pressed={mode === "quick"}
        disabled={disabled}
        onClick={() => onChange("quick")}
      >
        {messages.searchModeQuick}
      </button>
      <button
        type="button"
        className={`search-mode__button${mode === "conditional" ? " search-mode__button--active" : ""}`}
        aria-pressed={mode === "conditional"}
        disabled={disabled}
        onClick={() => onChange("conditional")}
      >
        {messages.searchModeConditional}
      </button>
    </div>
  );
}
