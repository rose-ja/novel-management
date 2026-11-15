"use client";

import { X } from "lucide-react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FilterBar } from "@/components/ui/filter-bar";
import { Select, type SelectOption } from "@/components/ui/select";
import { messages } from "@/lib/messages";
import { SEARCH_ENTITY_LABELS, type SearchState } from "@/lib/search";

export type SearchFilterOptions = {
  factions: SelectOption[];
  realms: SelectOption[];
  locations: SelectOption[];
  rankings: SelectOption[];
};

export type ConditionalFilterPanelProps = {
  state: SearchState;
  options: SearchFilterOptions;
  optionsError: boolean;
  submitting: boolean;
  submitDisabled: boolean;
  onChange: (patch: Partial<SearchState>) => void;
  onReset: () => void;
  onSubmit: () => void;
};

const entityTypeOptions: SelectOption[] = [
  { value: "all", label: "全部" },
  ...Object.entries(SEARCH_ENTITY_LABELS).map(([value, label]) => ({ value, label })),
];

export function ConditionalFilterPanel({
  state,
  options,
  optionsError,
  submitting,
  submitDisabled,
  onChange,
  onReset,
  onSubmit,
}: ConditionalFilterPanelProps) {
  const showCharacterFilters = state.entityType === "character";
  const chips = showCharacterFilters
    ? activeChips(state, options)
    : [];

  return (
    <div className="search-panel">
      <FilterBar onReset={onReset} activeCount={chips.length}>
        <label className="search-field">
          <span className="search-field__label">实体类型</span>
          <Select
            aria-label="实体类型"
            value={state.entityType}
            options={entityTypeOptions}
            onChange={(event) => onChange({ entityType: event.currentTarget.value as SearchState["entityType"] })}
          />
        </label>
        {showCharacterFilters ? (
          <>
            <label className="search-field">
              <span className="search-field__label">势力/分支</span>
              <Select
                aria-label="势力/分支"
                value={state.factionID}
                placeholder="全部势力"
                options={options.factions}
                onChange={(event) => onChange({ factionID: event.currentTarget.value })}
              />
            </label>
            <label className="search-field">
              <span className="search-field__label">势力范围</span>
              <Select
                aria-label="势力范围"
                value={state.factionScope}
                disabled={!state.factionID}
                options={[
                  { value: "direct", label: "直属成员" },
                  { value: "subtree", label: "含下级" },
                ]}
                onChange={(event) =>
                  onChange({ factionScope: event.currentTarget.value === "subtree" ? "subtree" : "direct" })
                }
              />
            </label>
            <label className="search-field">
              <span className="search-field__label">境界</span>
              <Select
                aria-label="境界"
                value={state.systemNodeID}
                placeholder="全部境界"
                options={options.realms}
                onChange={(event) => onChange({ systemNodeID: event.currentTarget.value })}
              />
            </label>
            <label className="search-field">
              <span className="search-field__label">地点</span>
              <Select
                aria-label="地点"
                value={state.locationID}
                placeholder="全部地点"
                options={options.locations}
                onChange={(event) => onChange({ locationID: event.currentTarget.value })}
              />
            </label>
            <label className="search-field">
              <span className="search-field__label">榜单</span>
              <Select
                aria-label="榜单"
                value={state.rankingID}
                placeholder="全部榜单"
                options={options.rankings}
                onChange={(event) => onChange({ rankingID: event.currentTarget.value })}
              />
            </label>
          </>
        ) : null}
      </FilterBar>

      {optionsError ? (
        <Alert tone="warning" title={messages.searchFilterOptionsError}>
          {messages.searchFilterOptionsErrorHint}
        </Alert>
      ) : null}

      {chips.length > 0 ? (
        <div className="search-chips" aria-label={messages.searchActiveFilters}>
          {chips.map((chip) => (
            <span key={chip.field} className="search-chip">
              <span>{chip.label}</span>
              <button
                type="button"
                className="search-chip__remove"
                aria-label={messages.searchRemoveFilter.replace("{name}", chip.label)}
                onClick={() => onChange(chip.clear)}
              >
                <X aria-hidden="true" size={14} />
              </button>
            </span>
          ))}
        </div>
      ) : null}

      <div className="search-panel__actions">
        <Button onClick={onSubmit} loading={submitting} disabled={submitDisabled}>
          {messages.searchSubmit}
        </Button>
      </div>
    </div>
  );
}

type Chip = {
  field: string;
  label: string;
  clear: Partial<SearchState>;
};

function activeChips(state: SearchState, options: SearchFilterOptions): Chip[] {
  const chips: Chip[] = [];
  if (state.factionID) {
    const label = options.factions.find((option) => option.value === state.factionID)?.label ?? state.factionID;
    chips.push({
      field: "faction",
      label: `${messages.searchFilterFaction} ${label}${state.factionScope === "subtree" ? "（含下级）" : ""}`,
      clear: { factionID: "" },
    });
  }
  if (state.systemNodeID) {
    const label = options.realms.find((option) => option.value === state.systemNodeID)?.label ?? state.systemNodeID;
    chips.push({ field: "realm", label: `${messages.searchFilterRealm} ${label}`, clear: { systemNodeID: "" } });
  }
  if (state.locationID) {
    const label =
      options.locations.find((option) => option.value === state.locationID)?.label ?? state.locationID;
    chips.push({ field: "location", label: `${messages.searchFilterLocation} ${label}`, clear: { locationID: "" } });
  }
  if (state.rankingID) {
    const label = options.rankings.find((option) => option.value === state.rankingID)?.label ?? state.rankingID;
    chips.push({ field: "ranking", label: `${messages.searchFilterRanking} ${label}`, clear: { rankingID: "" } });
  }
  return chips;
}
