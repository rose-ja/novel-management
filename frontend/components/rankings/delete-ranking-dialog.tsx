"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { toUserMessage } from "@/lib/api/errors";
import { deleteRanking, type DeleteRankingResult } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

export type DeleteRankingDialogProps = {
  ranking: { id: string; name: string; version: number } | null;
  onOpenChange: (open: boolean) => void;
  onDeleted: (result: DeleteRankingResult, name: string) => void;
};

/** 删除榜单：整榜软删除并返回 8 秒撤销令牌。 */
export function DeleteRankingDialog({ ranking, onOpenChange, onDeleted }: DeleteRankingDialogProps) {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  async function submit() {
    if (!ranking || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const result = await deleteRanking(ranking.id, ranking.version);
      onDeleted(result, ranking.name);
      onOpenChange(false);
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error(messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog
      open={Boolean(ranking)}
      onOpenChange={(open) => {
        if (!open) setError(null);
        onOpenChange(open);
      }}
      title={ranking ? messages.rankingDeleteTitle(ranking.name) : messages.rankingDelete}
      description={messages.rankingDeleteDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={submitting}>
            {messages.cancel}
          </Button>
          <Button variant="danger" onClick={submit} disabled={submitting}>
            {submitting ? messages.saving : messages.rankingDeleteSubmit}
          </Button>
        </>
      }
    >
      {error ? <Alert tone="danger" title={toUserMessage(error, messages.saveFailed)} /> : null}
    </Dialog>
  );
}
