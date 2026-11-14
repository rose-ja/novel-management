"use client";

import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { toUserMessage } from "@/lib/api/errors";
import { putSeat, type SeatView } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

export function SeatNumberCell({ seatNo, highlighted }: { seatNo: number; highlighted?: boolean }) {
  return (
    <span className="ranking-seat-no" data-seat-no={seatNo} data-highlighted={highlighted ? "true" : undefined}>
      {seatNo}
    </span>
  );
}

export function SeatCharacterCell({ seat }: { seat: SeatView }) {
  if (!seat.character) {
    return <span className="character-muted">{messages.rankingSeatEmptyLabel}</span>;
  }
  return (
    <div className="ranking-seat-character">
      <Link className="character-name-link" href={`/characters?id=${encodeURIComponent(seat.character.id)}`}>
        {seat.character.name}
      </Link>
      {seat.character.system_summary ? (
        <span className="character-muted">{seat.character.system_summary}</span>
      ) : null}
    </div>
  );
}

export function SeatNoteCell({
  rankingID,
  seat,
  onSaved,
}: {
  rankingID: string;
  seat: SeatView;
  onSaved: (updated: SeatView) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(seat.note ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!editing) {
    return (
      <div className="ranking-seat-note">
        <span>{seat.note ?? messages.rankingSeatNoteEmpty}</span>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setValue(seat.note ?? "");
            setError(null);
            setEditing(true);
          }}
        >
          {seat.note ? messages.rankingSeatNoteSave : "备注"}
        </Button>
      </div>
    );
  }

  async function save() {
    setSaving(true);
    setError(null);
    try {
      const updated = await putSeat(rankingID, seat.seat_no, {
        note: value.trim() ? value.trim() : null,
        version: seat.version,
      });
      onSaved(updated);
      setEditing(false);
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="ranking-seat-note ranking-seat-note--editing">
      <Input
        value={value}
        maxLength={500}
        placeholder={messages.rankingSeatNotePlaceholder}
        aria-label={`${messages.rankingSeatNote}（第 ${seat.seat_no} 席）`}
        onChange={(event) => setValue(event.target.value)}
      />
      <Button size="sm" onClick={save} disabled={saving}>
        {saving ? messages.saving : messages.rankingSeatNoteSave}
      </Button>
      <Button variant="ghost" size="sm" onClick={() => setEditing(false)} disabled={saving}>
        {messages.cancel}
      </Button>
      {error ? <span className="ui-field__error">{error}</span> : null}
    </div>
  );
}

export function SeatActionsCell({
  seat,
  onAssign,
  onAdjust,
  onRemove,
}: {
  seat: SeatView;
  onAssign: (seat: SeatView) => void;
  onAdjust: (seat: SeatView) => void;
  onRemove: (seat: SeatView) => void;
}) {
  if (!seat.character) {
    return (
      <Button size="sm" onClick={() => onAssign(seat)}>
        {messages.rankingAssign}
      </Button>
    );
  }
  return (
    <div className="ranking-seat-actions">
      <Button variant="secondary" size="sm" onClick={() => onAdjust(seat)}>
        {messages.rankingAdjust}
      </Button>
      <Button variant="ghost" size="sm" onClick={() => onRemove(seat)}>
        {messages.rankingRemove}
      </Button>
    </div>
  );
}
