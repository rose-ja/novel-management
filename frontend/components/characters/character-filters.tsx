"use client";

import { useEffect, useState } from "react";

import { FilterBar } from "@/components/ui/filter-bar";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import type { CharacterSort } from "@/lib/api/characters";
import { messages } from "@/lib/messages";

export type CharacterFilterValues = {
  q: string;
  systemNodeID: string;
  pendingOnly: boolean;
  sort: CharacterSort;
  pageSize: number;
};

export type CharacterFiltersProps = {
  values: CharacterFilterValues;
  realmOptions: { value: string; label: string }[];
  onChange: (patch: Partial<CharacterFilterValues>) => void;
  onReset: () => void;
  activeCount: number;
};

const sortOptions = [
  { value: "-updated_at", label: messages.characterSortUpdatedDesc },
  { value: "updated_at", label: messages.characterSortUpdatedAsc },
  { value: "name", label: messages.characterSortNameAsc },
  { value: "-name", label: messages.characterSortNameDesc },
];

const pageSizeOptions = [
  { value: "20", label: "20" },
  { value: "50", label: "50" },
  { value: "100", label: "100" },
];

/** 列表筛选：搜索 300ms 防抖，境界、待校对、排序与每页。 */
export function CharacterFilters({
  values,
  realmOptions,
  onChange,
  onReset,
  activeCount,
}: CharacterFiltersProps) {
  const [query, setQuery] = useState(values.q);

  useEffect(() => {
    setQuery(values.q);
  }, [values.q]);

  useEffect(() => {
    if (query === values.q) return;
    const timer = setTimeout(() => onChange({ q: query }), 300);
    return () => clearTimeout(timer);
  }, [query, values.q, onChange]);

  return (
    <FilterBar onReset={onReset} activeCount={activeCount} label={messages.characterTitle}>
      <FormField id="character-search" label={messages.characterSearchLabel}>
        <Input
          id="character-search"
          className="page-toolbar__search"
          value={query}
          maxLength={100}
          placeholder={messages.characterSearchPlaceholder}
          onChange={(event) => setQuery(event.target.value)}
        />
      </FormField>
      <FormField id="character-realm-filter" label={messages.characterFilterRealm}>
        <Select
          id="character-realm-filter"
          className="page-toolbar__select"
          value={values.systemNodeID}
          placeholder={messages.characterFilterRealmAll}
          options={realmOptions}
          onChange={(event) => onChange({ systemNodeID: event.target.value })}
        />
      </FormField>
      <div className="character-filter-toggle">
        <input
          id="character-pending-filter"
          type="checkbox"
          checked={values.pendingOnly}
          onChange={(event) => onChange({ pendingOnly: event.target.checked })}
        />
        <label htmlFor="character-pending-filter">{messages.characterFilterPending}</label>
      </div>
      <FormField id="character-sort" label={messages.characterSortLabel}>
        <Select
          id="character-sort"
          className="page-toolbar__select"
          value={values.sort}
          options={sortOptions}
          onChange={(event) => onChange({ sort: event.target.value as CharacterSort })}
        />
      </FormField>
      <FormField id="character-page-size" label={messages.characterPageSizeLabel}>
        <Select
          id="character-page-size"
          value={String(values.pageSize)}
          options={pageSizeOptions}
          onChange={(event) => onChange({ pageSize: Number(event.target.value) })}
        />
      </FormField>
    </FilterBar>
  );
}
