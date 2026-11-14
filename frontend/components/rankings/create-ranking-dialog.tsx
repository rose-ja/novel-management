"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { toUserMessage } from "@/lib/api/errors";
import { createRanking, type RankingDetail } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

export type CreateRankingDialogProps = {
  open: boolean;
  workID: string;
  onOpenChange: (open: boolean) => void;
  onCreated: (created: RankingDetail) => void;
};

/** 新建榜单：名称与席位数一次提交，创建后自动生成固定席位。 */
export function CreateRankingDialog({ open, workID, onOpenChange, onCreated }: CreateRankingDialogProps) {
  const [name, setName] = useState("");
  const [seatCount, setSeatCount] = useState("100");
  const [nameError, setNameError] = useState<string | null>(null);
  const [seatError, setSeatError] = useState<string | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function reset() {
    setName("");
    setSeatCount("100");
    setNameError(null);
    setSeatError(null);
    setError(null);
    setSubmitting(false);
  }

  function handleOpenChange(next: boolean) {
    if (!next && submitting) return;
    if (!next) reset();
    onOpenChange(next);
  }

  function validate(): number | null {
    const trimmed = name.trim();
    let valid = true;
    if (!trimmed) {
      setNameError(messages.rankingNameRequired);
      valid = false;
    } else if (trimmed.length > 200) {
      setNameError(messages.rankingNameHint);
      valid = false;
    } else {
      setNameError(null);
    }
    const parsed = Number(seatCount);
    if (!Number.isInteger(parsed) || parsed < 1 || parsed > 200) {
      setSeatError(messages.rankingSeatCountInvalid);
      valid = false;
    } else {
      setSeatError(null);
    }
    return valid ? parsed : null;
  }

  async function submit() {
    const count = validate();
    if (count === null || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const created = await createRanking({ workID, name: name.trim(), seatCount: count });
      onCreated(created);
      reset();
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error(messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      title={messages.rankingCreateTitle}
      description={messages.rankingCreateDescription}
      size="md"
      footer={
        <>
          <Button variant="secondary" onClick={() => handleOpenChange(false)} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button onClick={submit} disabled={submitting}>
            {submitting ? messages.saving : messages.rankingCreateSubmit}
          </Button>
        </>
      }
    >
      <div className="form-stack">
        {error ? <Alert tone="danger" title={toUserMessage(error, messages.saveFailed)} /> : null}
        <FormField id="ranking-name" label={messages.rankingNameLabel} required hint={messages.rankingNameHint} error={nameError ?? undefined}>
          <Input
            value={name}
            maxLength={200}
            placeholder="如封神榜"
            onChange={(event) => {
              setName(event.target.value);
              if (nameError) setNameError(null);
            }}
          />
        </FormField>
        <FormField id="ranking-seat-count" label={messages.rankingSeatCountLabel} required hint={messages.rankingSeatCountHint} error={seatError ?? undefined}>
          <Input
            type="number"
            min={1}
            max={200}
            value={seatCount}
            onChange={(event) => {
              setSeatCount(event.target.value);
              if (seatError) setSeatError(null);
            }}
          />
        </FormField>
      </div>
    </Dialog>
  );
}
