"use client";

import { useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { toUserMessage } from "@/lib/api/errors";
import { listSeats, moveSeat, type SeatView } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";
import { seatPage } from "@/lib/rankings";

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
  const [remoteOccupant, setRemoteOccupant] = useState<{ occupied: boolean; name: string | null } | null>(null);

  useEffect(() => {
    if (open) {
      setTarget("");
      setError(null);
      setSubmitting(false);
      setRemoteOccupant(null);
    }
  }, [open]);

  const targetNo = Number(target);
  const validTarget = Number.isInteger(targetNo) && targetNo >= 1 && targetNo <= seatCount;
  const sameSeat = Boolean(seat) && validTarget && targetNo === seat?.seat_no;
  const localSeat = validTarget ? seats.find((item) => item.seat_no === targetNo) : undefined;

  // 目标可能不在当前页，跨页时按需查询占用状态，避免把交换误判为空席。
  useEffect(() => {
    if (!open || !validTarget || sameSeat || localSeat) {
      setRemoteOccupant(null);
      return;
    }
    let active = true;
    const handle = window.setTimeout(() => {
      listSeats(rankingID, { page: seatPage(targetNo, 50), pageSize: 50 })
        .then((page) => {
          if (!active) return;
          const found = page.items.find((item) => item.seat_no === targetNo);
          setRemoteOccupant({ occupied: Boolean(found?.character), name: found?.character?.name ?? null });
        })
        .catch(() => {
          if (active) setRemoteOccupant(null);
        });
    }, 250);
    return () => {
      active = false;
      window.clearTimeout(handle);
    };
  }, [open, rankingID, targetNo, validTarget, sameSeat, localSeat]);

  const occupied = localSeat ? Boolean(localSeat.character) : Boolean(remoteOccupant?.occupied);
  const occupantName = localSeat?.character?.name ?? remoteOccupant?.name ?? null;

  let hint: string | null = null;
  if (validTarget && !sameSeat) {
    hint = occupied
      ? messages.rankingAdjustTargetSwap(targetNo, occupantName ?? messages.rankingNoValue)
      : messages.rankingAdjustTargetEmpty(targetNo);
  }

  async function submit() {
    if (!seat || !validTarget || sameSeat || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      await moveSeat(rankingID, seat.seat_no, {
        targetSeatNo: targetNo,
        mode: occupied ? "swap" : "empty",
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
