"use client";

import { useEffect, useMemo, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select, type SelectOption } from "@/components/ui/select";
import { listCharacters } from "@/lib/api/characters";
import { toUserMessage, isApiCode } from "@/lib/api/errors";
import { updateFaction, type FactionDetail } from "@/lib/api/factions";
import { messages } from "@/lib/messages";

export type FactionEditFormProps = {
  faction: FactionDetail;
  parentOptions: SelectOption[];
  leaderOptions: SelectOption[];
  onCancel: () => void;
  onSaved: (updated: FactionDetail) => void;
  onRefresh: () => void;
};

/** 势力内联编辑：名称、上级（禁用自身与后代）、首领与简介。 */
export function FactionEditForm({
  faction,
  parentOptions,
  leaderOptions,
  onCancel,
  onSaved,
  onRefresh,
}: FactionEditFormProps) {
  const [name, setName] = useState(faction.name);
  const [parentID, setParentID] = useState(faction.parent?.id ?? "");
  const [leaderID, setLeaderID] = useState(faction.leader?.id ?? "");
  const [description, setDescription] = useState(faction.description ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState(false);
  const [leaderQuery, setLeaderQuery] = useState("");
  const [searchOptions, setSearchOptions] = useState<SelectOption[] | null>(null);
  const [searching, setSearching] = useState(false);

  const leaderUnavailable = Boolean(faction.leader?.in_trash);
  const leaderChanged = leaderID !== (faction.leader?.id ?? "");

  // 首领按名称服务端搜索（PG-05 §6.2.3），300ms 防抖避免逐键请求。
  useEffect(() => {
    const keyword = leaderQuery.trim();
    if (!keyword) {
      setSearchOptions(null);
      setSearching(false);
      return;
    }
    let active = true;
    setSearching(true);
    const handle = window.setTimeout(() => {
      listCharacters({ workID: faction.work_id, q: keyword, pageSize: 20, sort: "name" })
        .then((page) => {
          if (active) setSearchOptions(page.items.map((item) => ({ value: item.id, label: item.name })));
        })
        .catch(() => {
          if (active) setSearchOptions([]);
        })
        .finally(() => {
          if (active) setSearching(false);
        });
    }, 300);
    return () => {
      active = false;
      window.clearTimeout(handle);
    };
  }, [leaderQuery, faction.work_id]);

  // 当前首领不在搜索结果时补齐选项；回收站首领保留展示但禁用选择。
  const leaderSelectOptions = useMemo(() => {
    const base = searchOptions ?? leaderOptions;
    const current = faction.leader;
    if (!leaderID) return base;
    const suffix = leaderUnavailable ? messages.factionLeaderTrashSuffix : "";
    return base.map((option) =>
      option.value === leaderID
        ? { ...option, label: `${option.label}${suffix}`, disabled: leaderUnavailable }
        : option,
    ).concat(
      current && current.id === leaderID && !base.some((option) => option.value === leaderID)
        ? [{ value: leaderID, label: `${current.name}${suffix}`, disabled: leaderUnavailable }]
        : [],
    );
  }, [searchOptions, leaderOptions, leaderID, faction.leader, leaderUnavailable]);

  const dirty =
    name.trim() !== faction.name ||
    parentID !== (faction.parent?.id ?? "") ||
    leaderChanged ||
    description.trim() !== (faction.description ?? "");

  function cancel() {
    if (dirty && !window.confirm(messages.unsavedDescription)) return;
    onCancel();
  }

  async function save() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError(messages.factionNameRequired);
      return;
    }
    setSaving(true);
    setError(null);
    setConflict(false);
    try {
      const updated = await updateFaction(faction.id, {
        name: trimmed,
        parentID: parentID || null,
        // 首领未变更时省略字段，避免把回收站中的首领再次提交而 409。
        ...(leaderChanged ? { leaderCharacterID: leaderID || null } : {}),
        description: description.trim() ? description : null,
        version: faction.version,
      });
      onSaved(updated);
    } catch (cause) {
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        setConflict(true);
        setError(messages.factionVersionConflict);
      } else if (isApiCode(cause, "CYCLE_DETECTED")) {
        setError(messages.factionCycleDetected);
      } else {
        setError(toUserMessage(cause, messages.saveFailed));
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="character-detail__card" aria-label={messages.factionEditTitle}>
      {error ? (
        <Alert
          tone="danger"
          title={error}
          action={
            conflict ? (
              <Button variant="secondary" size="sm" onClick={onRefresh}>
                {messages.factionRefresh}
              </Button>
            ) : undefined
          }
        />
      ) : null}
      <FormField id="faction-name" label={messages.factionNameLabel} required hint={messages.factionNameHint}>
        <Input
          id="faction-name"
          value={name}
          maxLength={200}
          disabled={saving}
          onChange={(event) => setName(event.target.value)}
        />
      </FormField>
      <FormField id="faction-parent" label={messages.factionParentLabel}>
        <Select
          id="faction-parent"
          value={parentID}
          placeholder={messages.factionParentPlaceholder}
          options={parentOptions}
          disabled={saving}
          onChange={(event) => setParentID(event.target.value)}
        />
      </FormField>
      <FormField id="faction-leader-search" label={messages.factionLeaderSearchLabel}>
        <Input
          id="faction-leader-search"
          value={leaderQuery}
          placeholder={messages.factionLeaderSearchPlaceholder}
          disabled={saving}
          onChange={(event) => setLeaderQuery(event.target.value)}
        />
      </FormField>
      <FormField
        id="faction-leader"
        label={messages.factionLeaderLabel}
        hint={leaderUnavailable ? messages.factionLeaderUnavailable : undefined}
      >
        <Select
          id="faction-leader"
          value={leaderID}
          placeholder={
            searching
              ? messages.characterLoading
              : searchOptions?.length === 0
                ? messages.factionLeaderSearchEmpty
                : messages.characterFactionPlaceholder
          }
          options={leaderSelectOptions}
          disabled={saving}
          onChange={(event) => setLeaderID(event.target.value)}
        />
      </FormField>
      <FormField
        id="faction-description"
        label={messages.factionDescriptionLabel}
        hint={messages.factionDescriptionHint}
      >
        <textarea
          id="faction-description"
          className="ui-textarea"
          rows={5}
          maxLength={20000}
          value={description}
          disabled={saving}
          onChange={(event) => setDescription(event.target.value)}
        />
      </FormField>
      <div className="character-edit-actions">
        <Button variant="secondary" onClick={cancel} disabled={saving}>
          {messages.cancel}
        </Button>
        <Button onClick={save} loading={saving} disabled={!dirty || !name.trim()}>
          {messages.factionSaveSubmit}
        </Button>
      </div>
    </section>
  );
}
