"use client";

import { useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { toUserMessage } from "@/lib/api/errors";
import { putSeat, type SeatView } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

export type RemoveSeatDialogProps = {
  open: boolean;
  rankingID: string;
  seat: SeatView | null;
  onOpenChange: (open: boolean) => void;
  onRemoved: (seat: SeatView, updated: SeatView) => void;
};

/** 移出榜单：清空人物与备注，返回新版本供撤销快照使用。 */
export function RemoveSeatDialog({ open, rankingID, seat, onOpenChange, onRemoved }: RemoveSeatDialogProps) {
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setError(null);
      setSubmitting(false);
    }
  }, [open]);

  async function submit() {
    if (!seat || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const updated = await putSeat(rankingID, seat.seat_no, {
        characterID: null,
        version: seat.version,
      });
      onRemoved(seat, updated);
      onOpenChange(false);
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={seat?.character ? messages.rankingRemoveTitle(seat.character.name) : messages.rankingRemove}
      description={messages.rankingRemoveDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} disabled={submitting}>
            {submitting ? messages.saving : messages.rankingRemoveSubmit}
          </Button>
        </>
      }
    >
      {error ? <Alert tone="danger" title={error} /> : null}
    </Dialog>
  );
}
