"use client";

import Image from "next/image";
import { useState } from "react";

import { messages } from "@/lib/messages";
import type { CharacterImage as CharacterImageData } from "@/lib/api/characters";

export type CharacterImageProps = {
  name: string;
  image: CharacterImageData | null;
  size?: "lg" | "sm";
  /** 优先使用缩略图（列表）。 */
  thumb?: boolean;
};

function initial(name: string): string {
  return name.trim().slice(0, 1) || "?";
}

/** 人物主图；缺失或加载失败时显示姓名首字占位（PG-03 §4.1）。 */
export function CharacterImage({ name, image, size = "lg", thumb = false }: CharacterImageProps) {
  const [failed, setFailed] = useState(false);
  const label = `人物主图：${name}`;
  const src = thumb ? image?.thumb_url ?? image?.url : image?.url;

  if (!image || !src || failed) {
    return (
      <div
        className={`character-image character-image--${size} character-image--placeholder`}
        role="img"
        aria-label={label}
      >
        <span aria-hidden="true">{initial(name)}</span>
        {size === "lg" ? <small>{messages.characterImagePlaceholder}</small> : null}
      </div>
    );
  }

  return (
    <Image
      className={`character-image character-image--${size}`}
      src={src}
      alt={label}
      width={size === "lg" ? 120 : 40}
      height={size === "lg" ? 120 : 40}
      loading="lazy"
      unoptimized
      onError={() => setFailed(true)}
    />
  );
}
