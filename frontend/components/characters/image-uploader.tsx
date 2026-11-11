"use client";

import { useEffect, useRef, useState } from "react";

import { CharacterImage } from "@/components/characters/character-image";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { imageAccept, validateImageFile } from "@/lib/api/media";
import { messages } from "@/lib/messages";
import type { CharacterImage as CharacterImageData } from "@/lib/api/characters";

export type ImageUploaderProps = {
  name: string;
  current: CharacterImageData | null;
  selected: File | null;
  removed: boolean;
  uploading: boolean;
  percent: number;
  error: string | null;
  disabled?: boolean;
  onSelect: (file: File) => void;
  onInvalid: (message: string) => void;
  onRemove: () => void;
  onCancelRemove: () => void;
};

/** 编辑态主图：本地预览、替换、移除确认与上传进度（PG-03 §4.1）。 */
export function ImageUploader({
  name,
  current,
  selected,
  removed,
  uploading,
  percent,
  error,
  disabled = false,
  onSelect,
  onInvalid,
  onRemove,
  onCancelRemove,
}: ImageUploaderProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [confirmRemove, setConfirmRemove] = useState(false);

  useEffect(() => {
    if (!selected) {
      setPreview(null);
      return;
    }
    const url = URL.createObjectURL(selected);
    setPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [selected]);

  function handleFile(file: File | undefined) {
    if (!file) return;
    const message = validateImageFile(file);
    if (message) {
      onInvalid(message);
      return;
    }
    onSelect(file);
    if (inputRef.current) inputRef.current.value = "";
  }

  const previewImage: CharacterImageData | null = preview
    ? { asset_id: "preview", url: preview, width: 120, height: 120 }
    : removed
      ? null
      : current;

  return (
    <div className="image-uploader">
      <div className="image-uploader__preview">
        <CharacterImage name={name} image={previewImage} size="lg" />
      </div>
      <div className="image-uploader__body">
        <div className="image-uploader__title">{messages.characterImageTitle}</div>
        <p className="image-uploader__hint">
          {removed
            ? messages.characterImageNotUploaded
            : current
              ? messages.characterImageUploaded
              : messages.characterImageNotUploaded}
          {" · "}JPG/PNG/WebP，≤10MB
        </p>
        {uploading ? (
          <Progress value={percent} label={messages.characterImageUploading} />
        ) : (
          <div className="image-uploader__actions">
            <input
              ref={inputRef}
              type="file"
              accept={imageAccept}
              className="visually-hidden"
              aria-hidden="true"
              tabIndex={-1}
              disabled={disabled}
              onChange={(event) => handleFile(event.target.files?.[0])}
            />
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={disabled || uploading}
              onClick={() => inputRef.current?.click()}
            >
              {current && !removed ? messages.characterImageReplace : messages.characterImageUpload}
            </Button>
            {removed ? (
              <Button type="button" variant="ghost" size="sm" disabled={disabled} onClick={onCancelRemove}>
                {messages.undo}
              </Button>
            ) : current ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={disabled || uploading}
                onClick={() => setConfirmRemove(true)}
              >
                {messages.characterImageRemove}
              </Button>
            ) : null}
          </div>
        )}
        {error ? <Alert tone="danger" title={error} /> : null}
      </div>

      {confirmRemove ? (
        <Dialog
          open
          onOpenChange={(next) => {
            if (!next) setConfirmRemove(false);
          }}
          title={messages.characterImageRemoveTitle}
          description={messages.characterImageRemoveDescription}
          size="sm"
          footer={
            <>
              <Button variant="secondary" onClick={() => setConfirmRemove(false)}>
                {messages.cancel}
              </Button>
              <Button
                variant="danger"
                onClick={() => {
                  setConfirmRemove(false);
                  onRemove();
                }}
              >
                {messages.characterImageRemoveConfirm}
              </Button>
            </>
          }
        />
      ) : null}
    </div>
  );
}
