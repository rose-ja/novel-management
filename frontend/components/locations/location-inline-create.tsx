"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { toUserMessage } from "@/lib/api/errors";
import { createLocation, type LocationDetail, type LocationSummary } from "@/lib/api/locations";
import { locationTypeOptions, toLocationOptions } from "@/lib/locations-tree";
import { messages } from "@/lib/messages";

export type LocationInlineCreateProps = {
  workID: string;
  items: LocationSummary[];
  defaultParentID: string | null;
  onCancel: () => void;
  onCreated: (created: LocationDetail) => void;
};

/** 树内联新建：选择上级、填写名称、类型与简介；有输入时取消需确认。 */
export function LocationInlineCreate({
  workID,
  items,
  defaultParentID,
  onCancel,
  onCreated,
}: LocationInlineCreateProps) {
  const [parentID, setParentID] = useState(defaultParentID ?? "");
  const [name, setName] = useState("");
  const [typeText, setTypeText] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const parentOptions = toLocationOptions(items);
  const typeSuggestions = locationTypeOptions(items);
  const dirty =
    name.trim() !== "" ||
    typeText.trim() !== "" ||
    description.trim() !== "" ||
    parentID !== (defaultParentID ?? "");

  function cancel() {
    if (dirty && !window.confirm(messages.locationInlineDiscardConfirm)) return;
    onCancel();
  }

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError(messages.locationNameRequired);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const created = await createLocation({
        workID,
        name: trimmed,
        parentID: parentID || null,
        typeText: typeText.trim() ? typeText : null,
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
    <section className="location-inline" aria-label={messages.locationInlineCreateTitle}>
      <h2 className="location-inline__title">{messages.locationInlineCreateTitle}</h2>
      {error ? <Alert tone="danger" title={error} /> : null}
      <div className="location-inline__fields">
        <FormField id="location-inline-parent" label={messages.locationParentLabel}>
          <Select
            id="location-inline-parent"
            value={parentID}
            placeholder={messages.locationParentPlaceholder}
            options={parentOptions}
            disabled={submitting}
            onChange={(event) => setParentID(event.target.value)}
          />
        </FormField>
        <FormField
          id="location-inline-name"
          label={messages.locationNameLabel}
          required
          hint={messages.locationNameHint}
        >
          <Input
            id="location-inline-name"
            value={name}
            maxLength={200}
            disabled={submitting}
            onChange={(event) => setName(event.target.value)}
          />
        </FormField>
        <FormField
          id="location-inline-type"
          label={messages.locationTypeLabel}
          hint={messages.locationTypeHint}
        >
          <Input
            id="location-inline-type"
            value={typeText}
            maxLength={200}
            list="location-type-suggestions"
            placeholder={messages.locationTypePlaceholder}
            disabled={submitting}
            onChange={(event) => setTypeText(event.target.value)}
          />
          <datalist id="location-type-suggestions">
            {typeSuggestions.map((value) => (
              <option key={value} value={value} />
            ))}
          </datalist>
        </FormField>
      </div>
      <FormField
        id="location-inline-description"
        label={messages.locationDescriptionLabel}
        hint={messages.locationDescriptionHint}
      >
        <textarea
          id="location-inline-description"
          className="ui-textarea"
          rows={3}
          maxLength={20000}
          value={description}
          disabled={submitting}
          onChange={(event) => setDescription(event.target.value)}
        />
      </FormField>
      <div className="location-inline__actions">
        <Button variant="secondary" size="sm" onClick={cancel} disabled={submitting}>
          {messages.cancel}
        </Button>
        <Button size="sm" onClick={submit} loading={submitting} disabled={!name.trim()}>
          {messages.locationCreateSubmit}
        </Button>
      </div>
    </section>
  );
}
