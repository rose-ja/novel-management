"use client";

import { ArrowRight, Building2, GitBranch, ListOrdered, MapPin } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";

import { CharacterImage } from "@/components/characters/character-image";
import { Badge } from "@/components/ui/badge";
import type { SearchItem } from "@/lib/api/search";
import { messages } from "@/lib/messages";

import { MatchedSnippet } from "./matched-snippet";

const typeIcons: Record<Exclude<SearchItem["entity_type"], "character">, ReactNode> = {
  faction: <Building2 aria-hidden="true" size={20} />,
  location: <MapPin aria-hidden="true" size={20} />,
  ranking: <ListOrdered aria-hidden="true" size={20} />,
  system_node: <GitBranch aria-hidden="true" size={20} />,
};

export function SearchResultRow({ item }: { item: SearchItem }) {
  const thumb =
    item.entity_type === "character" ? (
      <CharacterImage
        name={item.title}
        size="sm"
        thumb
        image={
          item.image_thumb_url
            ? { asset_id: item.entity_id, url: item.image_thumb_url, width: 40, height: 40 }
            : null
        }
      />
    ) : (
      <span className="search-result__icon">{typeIcons[item.entity_type]}</span>
    );

  return (
    <Link className="search-result" href={item.url}>
      <span className="search-result__thumb">{thumb}</span>
      <span className="search-result__body">
        <span className="search-result__title-row">
          <span className="search-result__title">{item.title}</span>
          {item.review_status === "pending" ? (
            <Badge tone="warning" outlined>
              {messages.searchPendingReview}
            </Badge>
          ) : null}
        </span>
        {item.subtitle ? <span className="search-result__meta">{item.subtitle}</span> : null}
        <MatchedSnippet fields={item.matched_fields} />
      </span>
      <ArrowRight className="search-result__arrow" aria-hidden="true" size={16} />
    </Link>
  );
}
