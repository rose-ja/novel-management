"use client";

import Link from "next/link";
import { ChevronDown, ChevronRight, Plus } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import type { FactionSummary } from "@/lib/api/factions";
import { messages } from "@/lib/messages";
import type { FactionTreeIndex } from "@/lib/factions-tree";

export type FactionTreeProps = {
  index: FactionTreeIndex;
  expanded: Set<string>;
  query: string;
  selectedID: string | null;
  onToggle: (id: string) => void;
  onAddChild: (id: string) => void;
};

function highlighted(name: string, query: string) {
  const keyword = query.trim();
  if (!keyword) return name;
  const at = name.toLowerCase().indexOf(keyword.toLowerCase());
  if (at < 0) return name;
  return (
    <>
      {name.slice(0, at)}
      <mark className="faction-tree__mark">{name.slice(at, at + keyword.length)}</mark>
      {name.slice(at + keyword.length)}
    </>
  );
}

/** 势力树：逐层渲染可见节点；展开状态由页面统一维护。 */
export function FactionTree({ index, expanded, query, selectedID, onToggle, onAddChild }: FactionTreeProps) {
  function renderNodes(nodes: FactionSummary[], depth: number) {
    return (
      <ul className="faction-tree__list" role={depth === 0 ? "tree" : "group"} aria-label={depth === 0 ? messages.factionTreeAria : undefined}>
        {nodes.map((node) => {
          const children = (index.children.get(node.id) ?? []).filter((child) => index.visible.has(child.id));
          const open = expanded.has(node.id);
          return (
            <li
              key={node.id}
              className="faction-tree__item"
              role="treeitem"
              aria-selected={node.id === selectedID}
              aria-expanded={children.length > 0 ? open : undefined}
            >
              <div
                className={
                  node.id === selectedID
                    ? "faction-tree__row faction-tree__row--selected"
                    : "faction-tree__row"
                }
                style={{ ["--faction-depth" as string]: String(depth) }}
                data-faction-id={node.id}
                data-match={index.matches.has(node.id) ? "true" : undefined}
              >
                {children.length > 0 ? (
                  <button
                    type="button"
                    className="faction-tree__toggle"
                    aria-label={
                      open
                        ? messages.factionCollapseNode(node.name)
                        : messages.factionExpandNode(node.name)
                    }
                    aria-expanded={open}
                    onClick={() => onToggle(node.id)}
                  >
                    {open ? <ChevronDown aria-hidden="true" size={16} /> : <ChevronRight aria-hidden="true" size={16} />}
                  </button>
                ) : (
                  <span className="faction-tree__toggle faction-tree__toggle--empty" aria-hidden="true" />
                )}
                <Link className="faction-tree__name" href={`/factions?id=${encodeURIComponent(node.id)}`}>
                  {highlighted(node.name, query)}
                </Link>
                <span className="faction-tree__meta">
                  {messages.factionLeaderLabel}：{node.leader?.name || messages.factionNoValue}
                </span>
                <span className="faction-tree__count">
                  {messages.factionMembersLabel(node.member_count)}
                </span>
                <span className="faction-tree__count">
                  {messages.factionChildrenLabel(node.children_count)}
                </span>
                {node.review_status === "pending" ? (
                  <Badge tone="warning" outlined>
                    {messages.characterReviewPending}
                  </Badge>
                ) : null}
                <button
                  type="button"
                  className="ui-button ui-button--ghost ui-button--sm faction-tree__add"
                  aria-label={messages.factionAddChild(node.name)}
                  onClick={() => onAddChild(node.id)}
                >
                  <Plus aria-hidden="true" size={14} />
                  <span>下级</span>
                </button>
              </div>
              {children.length > 0 && open ? renderNodes(children, depth + 1) : null}
            </li>
          );
        })}
      </ul>
    );
  }

  return <div className="faction-tree">{renderNodes(index.roots, 0)}</div>;
}
