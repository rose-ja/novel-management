"use client";

import { useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { UnsavedDialog } from "@/components/ui/unsaved-dialog";
import { ApiError } from "@/lib/api/client";
import { toUserMessage } from "@/lib/api/errors";
import { updateRanking, type RankingDetail } from "@/lib/api/rankings";
import { useUnsavedGuard } from "@/lib/hooks/use-unsaved-guard";
import { messages } from "@/lib/messages";

type ShrinkItem = { seat_no: number; character_id: string; name: string };

export type CapacityDialogProps = {
  open: boolean;
  rankingID: string;
  seatCount: number;
  version: number;
  onOpenChange: (open: boolean) => void;
  onUpdated: (detail: RankingDetail) => void;
  onJumpSeat: (seatNo: number) => void;
};

/** 扩容/缩容；缩容遇到尾部占用时列出人物并可跳转。 */
export function CapacityDialog({
  open,
  rankingID,
  seatCount,
  version,
  onOpenChange,
  onUpdated,
  onJumpSeat,
}: CapacityDialogProps) {
  const [value, setValue] = useState(String(seatCount));
  const [error, setError] = useState<string | null>(null);
  const [occupants, setOccupants] = useState<ShrinkItem[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const { markDirty, requestLeave, reset, confirmOpen, keepEditing, discardAndLeave } = useUnsavedGuard();

  const dirty = value !== String(seatCount);
  useEffect(() => {
    markDirty(dirty);
  }, [dirty, markDirty]);

  useEffect(() => {
    if (open) {
      setValue(String(seatCount));
      setError(null);
      setOccupants([]);
      setSubmitting(false);
      reset();
    }
  }, [open, seatCount, reset]);

  function handleOpenChange(next: boolean) {
    if (!next && submitting) return;
    if (!next) {
      requestLeave(() => onOpenChange(false));
      return;
    }
    onOpenChange(next);
  }

  const parsed = Number(value);
  const valid = Number.isInteger(parsed) && parsed >= 1 && parsed <= 200;

  async function submit() {
    if (!valid || submitting) return;
    setSubmitting(true);
    setError(null);
    setOccupants([]);
    try {
      const detail = await updateRanking(rankingID, { seatCount: parsed, version });
      reset();
      onUpdated(detail);
      onOpenChange(false);
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "SEAT_SHRINK_OCCUPIED") {
        const details = cause.details as { items?: ShrinkItem[] } | null | undefined;
        setOccupants(Array.isArray(details?.items) ? details.items : []);
        return;
      }
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={handleOpenChange}
        title={messages.rankingCapacityTitle}
        description={messages.rankingCapacityDescription}
        size="md"
        footer={
          <>
            <Button variant="secondary" onClick={() => handleOpenChange(false)} disabled={submitting}>
              {messages.cancel}
            </Button>
            <Button onClick={submit} disabled={!valid || submitting}>
              {submitting ? messages.saving : messages.rankingCapacitySubmit}
            </Button>
          </>
        }
      >
      <div className="form-stack">
        {error ? <Alert tone="danger" title={error} /> : null}
        <FormField
          id="ranking-capacity"
          label={messages.rankingCapacityLabel}
          required
          hint={messages.rankingCapacityHint}
          error={valid ? undefined : messages.rankingSeatCountInvalid}
        >
          <Input
            type="number"
            min={1}
            max={200}
            value={value}
            onChange={(event) => {
              setValue(event.target.value);
              if (error) setError(null);
              if (occupants.length > 0) setOccupants([]);
            }}
          />
        </FormField>
        {occupants.length > 0 ? (
          <Alert
            tone="warning"
            title={messages.rankingShrinkBlockedTitle}
            action={
              <div className="form-stack">
                <p>{messages.rankingShrinkBlockedDescription}</p>
                {occupants.map((item) => (
                  <button
                    key={item.seat_no}
                    type="button"
                    className="ui-button ui-button--ghost ui-button--sm"
                    onClick={() => {
                      reset();
                      onJumpSeat(item.seat_no);
                      onOpenChange(false);
                    }}
                  >
                    {messages.rankingShrinkBlockedItem(item.seat_no, item.name)}
                  </button>
                ))}
              </div>
            }
          />
        ) : null}
      </div>
      </Dialog>
      <UnsavedDialog
        open={confirmOpen}
        onKeepEditing={keepEditing}
        onDiscard={discardAndLeave}
      />
    </>
  );
}
