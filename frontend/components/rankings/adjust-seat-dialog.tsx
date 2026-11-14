"use client";

import { useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { toUserMessage } from "@/lib/api/errors";
import { moveSeat, type SeatView } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

export type AdjustSeatDialogProps = {
  open: boolean;
  rankingID: string;
  seat: SeatView | null;
  seats: SeatView[];
  seatCount: number;
  onOpenChange: (open: boolean) => void;
  onAdjusted: (targetSeatNo: number) => void;
};

/** 调整席位：目标为空席则移动，已占则交换。 */
export function AdjustSeatDialog({
  open,
  rankingID,
  seat,
  seats,
  seatCount,
  onOpenChange,
  onAdjusted,
}: AdjustSeatDialogProps) {
  const [target, setTarget] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setTarget("");
      setError(null);
      setSubmitting(false);
    }
  }, [open]);

  const targetNo = Number(target);
  const validTarget = Number.isInteger(targetNo) && targetNo >= 1 && targetNo <= seatCount;
  const sameSeat = Boolean(seat) && validTarget && targetNo === seat?.seat_no;
  const occupied = validTarget ? seats.find((item) => item.seat_no === targetNo) : undefined;

  let hint: string | null = null;
  if (validTarget && !sameSeat) {
    hint = occupied?.character
      ? messages.rankingAdjustTargetSwap(targetNo, occupied.character.name)
      : messages.rankingAdjustTargetEmpty(targetNo);
  }

  async function submit() {
    if (!seat || !validTarget || sameSeat || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      await moveSeat(rankingID, seat.seat_no, {
        targetSeatNo: targetNo,
        mode: occupied?.character ? "swap" : "empty",
        version: seat.version,
      });
      onAdjusted(targetNo);
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
      title={seat ? messages.rankingAdjustTitle(seat.seat_no) : messages.rankingAdjust}
      description={messages.rankingAdjustTargetHint}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={submit} disabled={!validTarget || sameSeat || submitting}>
            {submitting ? messages.saving : messages.rankingAdjustSubmit}
          </Button>
        </>
      }
    >
      <div className="form-stack">
        {error ? <Alert tone="danger" title={error} /> : null}
        <FormField
          id="ranking-adjust-target"
          label={messages.rankingAdjustTargetLabel}
          required
          hint={hint ?? messages.rankingAdjustTargetHint}
          error={sameSeat ? messages.rankingAdjustCurrent : undefined}
        >
          <Input
            type="number"
            min={1}
            max={seatCount}
            value={target}
            onChange={(event) => {
              setTarget(event.target.value);
              if (error) setError(null);
            }}
          />
        </FormField>
      </div>
    </Dialog>
  );
}
