"use client";

import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { messages } from "@/lib/messages";

export type UnsavedDialogProps = {
  open: boolean;
  onKeepEditing: () => void;
  onDiscard: () => void;
};

/** 统一的「放弃未保存修改」确认框，供表单/对话框复用。 */
export function UnsavedDialog({ open, onKeepEditing, onDiscard }: UnsavedDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onKeepEditing();
      }}
      title={messages.unsavedTitle}
      description={messages.unsavedDescription}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onKeepEditing}>
            {messages.unsavedKeep}
          </Button>
          <Button variant="danger" onClick={onDiscard}>
            {messages.unsavedDiscard}
          </Button>
        </>
      }
    />
  );
}
