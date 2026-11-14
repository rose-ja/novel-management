"use client";

import Link from "next/link";
import { Plus, X } from "lucide-react";
import { useMemo, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useToast } from "@/components/ui/toast";
import { getCharacter, updateCharacter } from "@/lib/api/characters";
import { toUserMessage } from "@/lib/api/errors";
import {
  addLocationFaction,
  removeLocationFaction,
  updateLocationFactionNote,
  type LocationCharacter,
  type LocationFaction,
} from "@/lib/api/locations";
import { messages } from "@/lib/messages";

type CharacterRow = { key: string; characterID: string; note: string };
type FactionRow = { key: string; factionID: string; note: string };

export type LocationRelationsProps = {
  locationID: string;
  characters: LocationCharacter[];
  factions: LocationFaction[];
  characterOptions: { value: string; label: string }[];
  factionOptions: { value: string; label: string }[];
  onChanged: () => void;
};

function nextKey(prefix: string) {
  return `${prefix}-${Math.random().toString(36).slice(2, 8)}`;
}

/** 地点详情关系区：人物关系通过人物 PATCH 全量替换，势力关系按单条增删改。 */
export function LocationRelations({
  locationID,
  characters,
  factions,
  characterOptions,
  factionOptions,
  onChanged,
}: LocationRelationsProps) {
  const { showToast } = useToast();
  const [editing, setEditing] = useState<"characters" | "factions" | null>(null);
  const [characterRows, setCharacterRows] = useState<CharacterRow[]>([]);
  const [factionRows, setFactionRows] = useState<FactionRow[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const usedCharacterIDs = useMemo(
    () => new Set(characterRows.map((row) => row.characterID).filter(Boolean)),
    [characterRows],
  );
  const usedFactionIDs = useMemo(
    () => new Set(factionRows.map((row) => row.factionID).filter(Boolean)),
    [factionRows],
  );

  function startCharacters() {
    setError(null);
    setEditing("characters");
    setCharacterRows(
      characters.map((item) => ({
        key: `character-${item.character_id}`,
        characterID: item.character_id,
        note: item.note ?? "",
      })),
    );
  }

  function startFactions() {
    setError(null);
    setEditing("factions");
    setFactionRows(
      factions.map((item) => ({
        key: `faction-${item.faction_id}`,
        factionID: item.faction_id,
        note: item.note ?? "",
      })),
    );
  }

  async function saveCharacters() {
    const original = new Map(characters.map((item) => [item.character_id, item.note ?? ""]));
    const next = new Map(
      characterRows
        .filter((row) => row.characterID !== "")
        .map((row) => [row.characterID, row.note.trim()]),
    );
    const affected = new Set([...original.keys(), ...next.keys()]);
    const changed = [...affected].filter((id) => (original.get(id) ?? null) !== (next.get(id) ?? null));

    setSubmitting(true);
    setError(null);
    try {
      for (const characterID of changed) {
        const detail = await getCharacter(characterID);
        const links = detail.location_links
          .filter((link) => link.location_id !== locationID)
          .map((link) => ({ location_id: link.location_id, note: link.note }));
        if (next.has(characterID)) {
          const note = next.get(characterID) ?? "";
          links.push({ location_id: locationID, note: note === "" ? null : note });
        }
        await updateCharacter(characterID, { locationLinks: links, version: detail.version });
      }
      setEditing(null);
      onChanged();
      showToast({ tone: "success", message: messages.locationRelationSaved });
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  async function saveFactions() {
    const original = new Map(factions.map((item) => [item.faction_id, item.note ?? ""]));
    const next = new Map(
      factionRows
        .filter((row) => row.factionID !== "")
        .map((row) => [row.factionID, row.note.trim()]),
    );
    const removed = [...original.keys()].filter((id) => !next.has(id));
    const added = [...next.keys()].filter((id) => !original.has(id));
    const changed = [...next.keys()].filter(
      (id) => original.has(id) && original.get(id) !== next.get(id),
    );

    setSubmitting(true);
    setError(null);
    try {
      for (const factionID of removed) {
        await removeLocationFaction(locationID, factionID);
      }
      for (const factionID of added) {
        const note = next.get(factionID) ?? "";
        await addLocationFaction(locationID, {
          factionID,
          note: note === "" ? null : note,
        });
      }
      for (const factionID of changed) {
        const note = next.get(factionID) ?? "";
        await updateLocationFactionNote(locationID, factionID, note === "" ? null : note);
      }
      setEditing(null);
      onChanged();
      showToast({ tone: "success", message: messages.locationRelationSaved });
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="character-detail__card" aria-labelledby="location-relations-title">
      <div className="location-relations__header">
        <h2 id="location-relations-title" className="character-detail__section-title">
          {messages.locationRelationsTitle}
        </h2>
        {editing === null ? (
          <div className="location-relations__header-actions">
            <Button variant="secondary" size="sm" onClick={startCharacters}>
              {messages.locationCharactersEdit}
            </Button>
            <Button variant="secondary" size="sm" onClick={startFactions}>
              {messages.locationFactionsEdit}
            </Button>
          </div>
        ) : null}
      </div>

      {editing === "characters" ? (
        <div className="location-relation-editor">
          <p className="character-form-note">{messages.locationCharactersHint}</p>
          {error ? <Alert tone="danger" title={error} /> : null}
          {characterRows.map((row, index) => (
            <div className="location-relation-row" key={row.key}>
              <Select
                aria-label={`${messages.locationCharactersTitle} ${index + 1}`}
                value={row.characterID}
                disabled={submitting || row.characterID !== ""}
                placeholder={messages.locationCharactersTitle}
                options={characterOptions.filter(
                  (option) => option.value === row.characterID || !usedCharacterIDs.has(option.value),
                )}
                onChange={(event) =>
                  setCharacterRows((current) =>
                    current.map((item, position) =>
                      position === index ? { ...item, characterID: event.target.value } : item,
                    ),
                  )
                }
              />
              <Input
                aria-label={`${messages.locationRelationNoteLabel} ${index + 1}`}
                value={row.note}
                maxLength={500}
                placeholder={messages.locationRelationNotePlaceholder}
                disabled={submitting}
                onChange={(event) =>
                  setCharacterRows((current) =>
                    current.map((item, position) =>
                      position === index ? { ...item, note: event.target.value } : item,
                    ),
                  )
                }
              />
              <button
                type="button"
                className="ui-icon-button ui-icon-button--sm"
                aria-label={messages.locationRelationRemove(row.characterID || "—")}
                disabled={submitting}
                onClick={() =>
                  setCharacterRows((current) => current.filter((_, position) => position !== index))
                }
              >
                <X aria-hidden="true" size={16} />
              </button>
            </div>
          ))}
          <Button
            variant="secondary"
            size="sm"
            disabled={submitting}
            onClick={() =>
              setCharacterRows((current) => [
                ...current,
                { key: nextKey("character"), characterID: "", note: "" },
              ])
            }
          >
            <Plus aria-hidden="true" size={14} />
            <span>{messages.locationRelationAddCharacter}</span>
          </Button>
          <div className="location-relation-actions">
            <Button variant="secondary" size="sm" onClick={() => setEditing(null)} disabled={submitting}>
              {messages.cancel}
            </Button>
            <Button size="sm" onClick={saveCharacters} loading={submitting}>
              {messages.locationRelationSave}
            </Button>
          </div>
        </div>
      ) : (
        <div className="character-relation-group">
          <h3 className="character-relation-group__title">{messages.locationCharactersTitle}</h3>
          {characters.length === 0 ? (
            <p className="character-relation-empty">{messages.locationCharactersEmpty}</p>
          ) : (
            <ul className="character-relation-list">
              {characters.map((item) => (
                <li className="character-relation-item" key={item.character_id}>
                  {item.in_trash ? (
                    <span className="character-muted">
                      {item.name}
                      {messages.locationRelationInTrash}
                    </span>
                  ) : (
                    <Link
                      className="character-name-link"
                      href={`/characters?id=${encodeURIComponent(item.character_id)}`}
                      aria-label={messages.locationCharacterLink(item.name)}
                    >
                      {item.name}
                    </Link>
                  )}
                  {item.note ? <span className="character-relation-note">{item.note}</span> : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {editing === "factions" ? (
        <div className="location-relation-editor">
          <p className="character-form-note">{messages.locationFactionsHint}</p>
          {error ? <Alert tone="danger" title={error} /> : null}
          {factionRows.map((row, index) => (
            <div className="location-relation-row" key={row.key}>
              <Select
                aria-label={`${messages.locationFactionsTitle} ${index + 1}`}
                value={row.factionID}
                disabled={submitting || row.factionID !== ""}
                placeholder={messages.locationFactionsTitle}
                options={factionOptions.filter(
                  (option) => option.value === row.factionID || !usedFactionIDs.has(option.value),
                )}
                onChange={(event) =>
                  setFactionRows((current) =>
                    current.map((item, position) =>
                      position === index ? { ...item, factionID: event.target.value } : item,
                    ),
                  )
                }
              />
              <Input
                aria-label={`${messages.locationRelationNoteLabel} ${index + 1}`}
                value={row.note}
                maxLength={500}
                placeholder={messages.locationRelationNotePlaceholder}
                disabled={submitting}
                onChange={(event) =>
                  setFactionRows((current) =>
                    current.map((item, position) =>
                      position === index ? { ...item, note: event.target.value } : item,
                    ),
                  )
                }
              />
              <button
                type="button"
                className="ui-icon-button ui-icon-button--sm"
                aria-label={messages.locationRelationRemove(row.factionID || "—")}
                disabled={submitting}
                onClick={() =>
                  setFactionRows((current) => current.filter((_, position) => position !== index))
                }
              >
                <X aria-hidden="true" size={16} />
              </button>
            </div>
          ))}
          <Button
            variant="secondary"
            size="sm"
            disabled={submitting}
            onClick={() =>
              setFactionRows((current) => [
                ...current,
                { key: nextKey("faction"), factionID: "", note: "" },
              ])
            }
          >
            <Plus aria-hidden="true" size={14} />
            <span>{messages.locationRelationAddFaction}</span>
          </Button>
          <div className="location-relation-actions">
            <Button variant="secondary" size="sm" onClick={() => setEditing(null)} disabled={submitting}>
              {messages.cancel}
            </Button>
            <Button size="sm" onClick={saveFactions} loading={submitting}>
              {messages.locationRelationSave}
            </Button>
          </div>
        </div>
      ) : (
        <div className="character-relation-group">
          <h3 className="character-relation-group__title">{messages.locationFactionsTitle}</h3>
          {factions.length === 0 ? (
            <p className="character-relation-empty">{messages.locationFactionsEmpty}</p>
          ) : (
            <ul className="character-relation-list">
              {factions.map((item) => (
                <li className="character-relation-item" key={item.faction_id}>
                  {item.in_trash ? (
                    <span className="character-muted">
                      {item.name}
                      {messages.locationRelationInTrash}
                    </span>
                  ) : (
                    <Link
                      className="character-name-link"
                      href={`/factions?id=${encodeURIComponent(item.faction_id)}`}
                      aria-label={messages.locationFactionLink(item.name)}
                    >
                      {item.name}
                    </Link>
                  )}
                  {item.note ? <span className="character-relation-note">{item.note}</span> : null}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}
