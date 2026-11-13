"use client";

import { useEffect, useState } from "react";

import {
  CharacterFormFields,
  createCharacterFormValues,
  toCharacterFactionID,
  toCharacterNameInputs,
  toCharacterSystemRefs,
  useCharacterSystemFields,
} from "@/components/characters/character-form-fields";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Drawer } from "@/components/ui/drawer";
import { useToast } from "@/components/ui/toast";
import { createCharacter } from "@/lib/api/characters";
import { toUserMessage } from "@/lib/api/errors";
import { listFactions } from "@/lib/api/factions";
import { toFactionOptions } from "@/lib/factions-tree";
import { useUnsavedGuard } from "@/lib/hooks/use-unsaved-guard";
import { messages } from "@/lib/messages";

export type CharacterCreateDrawerProps = {
  open: boolean;
  workID: string;
  onClose: () => void;
  onCreated: (character: { id: string; name: string }) => void;
};

/** 新建人物 Drawer：名称、别名/称号、简介与体系动态字段（PG-11 §6.1）。 */
export function CharacterCreateDrawer({ open, workID, onClose, onCreated }: CharacterCreateDrawerProps) {
  const { showToast } = useToast();
  const guard = useUnsavedGuard();
  const { categories, status: systemStatus } = useCharacterSystemFields(workID, open);
  const [values, setValues] = useState(createCharacterFormValues);
  const [factionOptions, setFactionOptions] = useState<{ value: string; label: string }[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const resetGuard = guard.reset;

  useEffect(() => {
    if (open) return;
    setValues(createCharacterFormValues());
    setError(null);
    resetGuard();
  }, [open, resetGuard]);

  useEffect(() => {
    if (!open) return;
    let active = true;
    listFactions({ workID })
      .then((result) => {
        if (active) setFactionOptions(toFactionOptions(result.items));
      })
      .catch(() => {
        if (active) setFactionOptions([]);
      });
    return () => {
      active = false;
    };
  }, [open, workID]);

  async function submit() {
    const name = values.name.trim();
    if (!name) {
      setError(messages.characterNameRequired);
      return;
    }
    if (values.names.some((item) => !item.value.trim())) {
      setError(messages.characterNameValueRequired);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const created = await createCharacter({
        workID,
        name,
        names: toCharacterNameInputs(values),
        description: values.description.trim() ? values.description : null,
        factionID: toCharacterFactionID(values),
        systemRefs: toCharacterSystemRefs(values),
      });
      guard.reset();
      onCreated({ id: created.id, name: created.name });
      onClose();
      showToast({ tone: "success", message: messages.characterCreated(created.name) });
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Drawer
        open={open}
        onOpenChange={(next) => {
          if (!next && !submitting) guard.requestLeave(onClose);
        }}
        title={messages.characterCreateTitle}
        description={messages.characterCreateDescription}
        width="wide"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => guard.requestLeave(onClose)}
              disabled={submitting}
            >
              {messages.cancel}
            </Button>
            <Button onClick={submit} loading={submitting} disabled={!values.name.trim()}>
              {messages.characterCreateSubmit}
            </Button>
          </>
        }
      >
        {error ? <Alert tone="danger" title={error} /> : null}
        <CharacterFormFields
          values={values}
          onChange={(next) => {
            setValues(next);
            guard.markDirty(true);
          }}
          disabled={submitting}
          categories={categories}
          systemStatus={systemStatus}
          factionOptions={factionOptions}
        />
      </Drawer>
      <Dialog
        open={guard.confirmOpen}
        onOpenChange={(next) => {
          if (!next) guard.keepEditing();
        }}
        title={messages.unsavedTitle}
        description={messages.unsavedDescription}
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={guard.keepEditing}>
              {messages.unsavedKeep}
            </Button>
            <Button variant="danger" onClick={guard.discardAndLeave}>
              {messages.unsavedDiscard}
            </Button>
          </>
        }
      />
    </>
  );
}
