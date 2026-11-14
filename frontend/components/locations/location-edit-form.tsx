"use client";

import { useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { isApiCode, toUserMessage } from "@/lib/api/errors";
import { updateLocation, type LocationDetail } from "@/lib/api/locations";
import { messages } from "@/lib/messages";

export type LocationEditFormProps = {
  location: LocationDetail;
  parentOptions: { value: string; label: string; disabled?: boolean }[];
  typeSuggestions: string[];
  onCancel: () => void;
  onSaved: (updated: LocationDetail) => void;
  onRefresh: () => void;
};

/** 地点内联编辑：名称、上级、类型与简介，保存使用 If-Match。 */
export function LocationEditForm({
  location,
  parentOptions,
  typeSuggestions,
  onCancel,
  onSaved,
  onRefresh,
}: LocationEditFormProps) {
  const [name, setName] = useState(location.name);
  const [parentID, setParentID] = useState(location.parent?.id ?? "");
  const [typeText, setTypeText] = useState(location.type_text ?? "");
  const [description, setDescription] = useState(location.description ?? "");
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError(messages.locationNameRequired);
      return;
    }
    setSubmitting(true);
    setError(null);
    setConflict(false);
    try {
      const updated = await updateLocation(location.id, {
        name: trimmed,
        parentID: parentID || null,
        typeText: typeText.trim() ? typeText : null,
        description: description.trim() ? description : null,
        version: location.version,
      });
      onSaved(updated);
    } catch (cause) {
      if (isApiCode(cause, "VERSION_CONFLICT")) {
        setConflict(true);
        setError(messages.locationVersionConflict);
      } else if (isApiCode(cause, "CYCLE_DETECTED")) {
        setError(messages.locationCycleDetected);
      } else {
        setError(toUserMessage(cause, messages.saveFailed));
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="location-edit" aria-label={messages.locationEditTitle}>
      <h2 className="location-edit__title">{messages.locationEditTitle}</h2>
      {error ? (
        <Alert
          tone="danger"
          title={error}
          action={
            conflict ? (
              <Button variant="secondary" size="sm" onClick={onRefresh}>
                {messages.locationRefresh}
              </Button>
            ) : undefined
          }
        />
      ) : null}
      <div className="location-edit__fields">
        <FormField id="location-edit-name" label={messages.locationNameLabel} required hint={messages.locationNameHint}>
          <Input
            id="location-edit-name"
            value={name}
            maxLength={200}
            disabled={submitting}
            onChange={(event) => setName(event.target.value)}
          />
        </FormField>
        <FormField id="location-edit-parent" label={messages.locationParentLabel}>
          <Select
            id="location-edit-parent"
            value={parentID}
            placeholder={messages.locationParentPlaceholder}
            options={parentOptions}
            disabled={submitting}
            onChange={(event) => setParentID(event.target.value)}
          />
        </FormField>
        <FormField id="location-edit-type" label={messages.locationTypeLabel} hint={messages.locationTypeHint}>
          <Input
            id="location-edit-type"
            value={typeText}
            maxLength={200}
            list="location-edit-type-suggestions"
            placeholder={messages.locationTypePlaceholder}
            disabled={submitting}
            onChange={(event) => setTypeText(event.target.value)}
          />
          <datalist id="location-edit-type-suggestions">
            {typeSuggestions.map((value) => (
              <option key={value} value={value} />
            ))}
          </datalist>
        </FormField>
      </div>
      <FormField
        id="location-edit-description"
        label={messages.locationDescriptionLabel}
        hint={messages.locationDescriptionHint}
      >
        <textarea
          id="location-edit-description"
          className="ui-textarea"
          rows={5}
          maxLength={20000}
          value={description}
          disabled={submitting}
          onChange={(event) => setDescription(event.target.value)}
        />
      </FormField>
      <div className="location-edit__actions">
        <Button variant="secondary" onClick={onCancel} disabled={submitting}>
          {messages.cancel}
        </Button>
        <Button onClick={submit} loading={submitting} disabled={!name.trim()}>
          {messages.locationSaveSubmit}
        </Button>
      </div>
    </section>
  );
}
