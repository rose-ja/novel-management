"use client";

import { useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/client";
import { listCharacters, type CharacterSummary } from "@/lib/api/characters";
import { toUserMessage } from "@/lib/api/errors";
import { putSeat, type SeatView } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

export type AssignSeatDialogProps = {
  open: boolean;
  rankingID: string;
  workID: string;
  seat: SeatView | null;
  seats: SeatView[];
  onOpenChange: (open: boolean) => void;
  onAssigned: (updated: SeatView) => void;
  onJumpSeat: (seatNo: number) => void;
};

/** 指派人物到空席；同榜重复指派时给出已占席位跳转。 */
export function AssignSeatDialog({
  open,
  rankingID,
  workID,
  seat,
  seats,
  onOpenChange,
  onAssigned,
  onJumpSeat,
}: AssignSeatDialogProps) {
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [items, setItems] = useState<CharacterSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<CharacterSummary | null>(null);
  const [conflictSeat, setConflictSeat] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) {
      setQueryInput("");
      setQuery("");
      setItems([]);
      setSelected(null);
      setConflictSeat(null);
      setError(null);
      setSubmitting(false);
    }
  }, [open]);

  useEffect(() => {
    const handle = window.setTimeout(() => setQuery(queryInput), 300);
    return () => window.clearTimeout(handle);
  }, [queryInput]);

  useEffect(() => {
    if (!open) return;
    let active = true;
    setLoading(true);
    listCharacters({ workID, q: query.trim() || undefined, pageSize: 20 })
      .then((result) => {
        if (active) setItems(result.items);
      })
      .catch(() => {
        if (active) setItems([]);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [open, workID, query]);

  function existingSeatOf(characterID: string): number | null {
    const existing = seats.find((item) => item.character?.id === characterID);
    return existing ? existing.seat_no : null;
  }

  function choose(character: CharacterSummary) {
    const existing = existingSeatOf(character.id);
    setSelected(character);
    setError(null);
    setConflictSeat(existing);
  }

  async function submit() {
    if (!seat || !selected || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const updated = await putSeat(rankingID, seat.seat_no, {
        characterID: selected.id,
        version: seat.version,
      });
      onAssigned(updated);
      onOpenChange(false);
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "SEAT_CONFLICT") {
        const details = cause.details as { existing_seat_no?: number } | null | undefined;
        if (typeof details?.existing_seat_no === "number") {
          setConflictSeat(details.existing_seat_no);
          return;
        }
      }
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={seat ? messages.rankingAssignTitle(seat.seat_no) : messages.rankingAssign}
      description={messages.rankingAssignSearchPlaceholder}
      size="md"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={submit} disabled={!selected || conflictSeat !== null || submitting}>
            {submitting ? messages.saving : messages.rankingAssignSubmit}
          </Button>
        </>
      }
    >
      <div className="form-stack">
        {error ? <Alert tone="danger" title={error} /> : null}
        <Input
          value={queryInput}
          maxLength={100}
          placeholder={messages.rankingAssignSearchPlaceholder}
          aria-label={messages.rankingAssignSearchLabel}
          onChange={(event) => setQueryInput(event.target.value)}
        />
        {conflictSeat !== null ? (
          <Alert
            tone="warning"
            title={messages.rankingAssignInOtherSeat(conflictSeat)}
            action={
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  onJumpSeat(conflictSeat);
                  onOpenChange(false);
                }}
              >
                {messages.rankingAssignGoSeat}
              </Button>
            }
          />
        ) : null}
        <div className="ranking-assign-list" role="listbox" aria-label={messages.rankingAssignSearchLabel}>
          {loading ? <p className="character-muted">{messages.characterLoading}</p> : null}
          {!loading && items.length === 0 ? (
            <p className="character-muted">{messages.rankingAssignEmpty}</p>
          ) : null}
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              role="option"
              aria-selected={selected?.id === item.id}
              className={
                selected?.id === item.id
                  ? "ranking-assign-item ranking-assign-item--selected"
                  : "ranking-assign-item"
              }
              onClick={() => choose(item)}
            >
              <span>{item.name}</span>
              <span className="character-muted">{item.system_summary ?? messages.rankingNoValue}</span>
            </button>
          ))}
        </div>
      </div>
    </Dialog>
  );
}
