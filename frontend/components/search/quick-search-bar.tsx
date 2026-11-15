"use client";

import { Search, X } from "lucide-react";
import type { RefObject } from "react";

import { messages } from "@/lib/messages";

export type QuickSearchBarProps = {
  value: string;
  disabled?: boolean;
  inputRef?: RefObject<HTMLInputElement | null>;
  onChange: (value: string) => void;
  onEnter: () => void;
  onClear: () => void;
};

/** 检索输入框：自动聚焦、1 字符可搜、IME 组合期间不触发 Enter。 */
export function QuickSearchBar({ value, disabled = false, inputRef, onChange, onEnter, onClear }: QuickSearchBarProps) {
  return (
    <div className="search-bar" role="search">
      <label className="search-bar__label" htmlFor="search-input">
        {messages.searchInputLabel}
      </label>
      <div className="search-bar__control">
        <Search className="search-bar__icon" aria-hidden="true" size={18} />
        <input
          ref={inputRef}
          id="search-input"
          className="search-bar__input"
          type="search"
          value={value}
          disabled={disabled}
          autoFocus
          maxLength={200}
          placeholder={messages.searchPlaceholder}
          aria-describedby="search-hint"
          onChange={(event) => onChange(event.currentTarget.value)}
          onCompositionStart={(event) => {
            event.currentTarget.dataset.composing = "true";
          }}
          onCompositionEnd={(event) => {
            delete event.currentTarget.dataset.composing;
          }}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              onClear();
              return;
            }
            if (event.key !== "Enter") return;
            if (event.currentTarget.dataset.composing === "true" || event.nativeEvent.isComposing) return;
            event.preventDefault();
            onEnter();
          }}
        />
        {value ? (
          <button
            type="button"
            className="ui-icon-button ui-icon-button--sm search-bar__clear"
            aria-label={messages.searchClear}
            disabled={disabled}
            onClick={onClear}
          >
            <X aria-hidden="true" size={16} />
          </button>
        ) : null}
      </div>
      <p id="search-hint" className="search-bar__hint">
        {messages.searchHint}
      </p>
    </div>
  );
}
