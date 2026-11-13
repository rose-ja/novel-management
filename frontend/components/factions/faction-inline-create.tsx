"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { createFaction, type FactionDetail, type FactionSummary } from "@/lib/api/factions";
import { toUserMessage } from "@/lib/api/errors";
import { toFactionOptions } from "@/lib/factions-tree";
import { messages } from "@/lib/messages";

export type FactionInlineCreateProps = {
  workID: string;
  items: FactionSummary[];
  defaultParentID: string | null;
  onCancel: () => void;
  onCreated: (created: FactionDetail) => void;
};

/** 树内联新建：选择上级、填写名称与简介；有输入时取消需确认。 */
export function FactionInlineCreate({
  workID,
  items,
  defaultParentID,
  onCancel,
  onCreated,
}: FactionInlineCreateProps) {
  const [parentID, setParentID] = useState(defaultParentID ?? "");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const parentOptions = toFactionOptions(items);
  const dirty = name.trim() !== "" || description.trim() !== "" || parentID !== (defaultParentID ?? "");

  function cancel() {
    if (dirty && !window.confirm(messages.factionInlineDiscardConfirm)) return;
    onCancel();
  }

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError(messages.factionNameRequired);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const created = await createFaction({
        workID,
        name: trimmed,
        parentID: parentID || null,
        description: description.trim() ? description : null,
      });
      onCreated(created);
    } catch (cause) {
      setError(toUserMessage(cause, messages.saveFailed));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="faction-inline" aria-label={messages.factionInlineCreateTitle}>
      <h2 className="faction-inline__title">{messages.factionInlineCreateTitle}</h2>
      {error ? <Alert tone="danger" title={error} /> : null}
      <div className="faction-inline__fields">
        <FormField id="faction-inline-parent" label={messages.factionParentLabel}>
          <Select
            id="faction-inline-parent"
            value={parentID}
            placeholder={messages.factionParentPlaceholder}
            options={parentOptions}
            disabled={submitting}
            onChange={(event) => setParentID(event.target.value)}
          />
        </FormField>
        <FormField
          id="faction-inline-name"
          label={messages.factionNameLabel}
          required
          hint={messages.factionNameHint}
        >
          <Input
            id="faction-inline-name"
            value={name}
            maxLength={200}
            disabled={submitting}
            onChange={(event) => setName(event.target.value)}
          />
        </FormField>
      </div>
      <FormField
        id="faction-inline-description"
        label={messages.factionDescriptionLabel}
        hint={messages.factionDescriptionHint}
      >
        <textarea
          id="faction-inline-description"
          className="ui-textarea"
          rows={3}
          maxLength={20000}
          value={description}
          disabled={submitting}
          onChange={(event) => setDescription(event.target.value)}
        />
      </FormField>
      <div className="faction-inline__actions">
        <Button variant="secondary" size="sm" onClick={cancel} disabled={submitting}>
          {messages.cancel}
        </Button>
        <Button size="sm" onClick={submit} loading={submitting} disabled={!name.trim()}>
          {messages.factionCreateSubmit}
        </Button>
      </div>
    </section>
  );
}
