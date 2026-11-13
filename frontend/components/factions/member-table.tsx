"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { CharacterImage } from "@/components/characters/character-image";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { Table, type TableColumn } from "@/components/ui/table";
import { toUserMessage } from "@/lib/api/errors";
import {
  listFactionMembers,
  type FactionMemberPage,
  type FactionMemberScope,
} from "@/lib/api/factions";
import { messages } from "@/lib/messages";

const PAGE_SIZE = 20;

export type MemberTableProps = {
  factionID: string;
  hasChildren: boolean;
  refreshKey?: number;
};

type Status = "loading" | "ready" | "error";

/** 势力成员：直属/含下级切换与分页；成员由人物 faction_id 自动推导。 */
export function MemberTable({ factionID, hasChildren, refreshKey = 0 }: MemberTableProps) {
  const [scope, setScope] = useState<FactionMemberScope>("direct");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<FactionMemberPage | null>(null);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);

  useEffect(() => {
    setScope("direct");
    setPage(1);
  }, [factionID]);

  const load = useCallback(async () => {
    setStatus("loading");
    setError(null);
    try {
      const result = await listFactionMembers(factionID, { scope, page, pageSize: PAGE_SIZE });
      setData(result);
      setStatus("ready");
    } catch (cause) {
      setData(null);
      setError(cause instanceof Error ? cause : new Error(messages.factionMembersLoadFailed));
      setStatus("error");
    }
  }, [factionID, scope, page]);

  useEffect(() => {
    void load();
  }, [load, refreshKey, reloadTick]);

  const columns: TableColumn[] = [
    { key: "image", label: messages.characterColumnImage, width: "56px" },
    { key: "name", label: messages.characterColumnName },
    { key: "realm", label: messages.characterColumnRealm },
    ...(scope === "subtree"
      ? [{ key: "faction", label: messages.characterColumnFaction } as TableColumn]
      : []),
    { key: "ranking", label: messages.characterColumnRanking, width: "96px" },
  ];

  const items = data?.items ?? [];
  const empty =
    scope === "subtree" ? messages.factionMembersEmptySubtree : messages.factionMembersEmptyDirect;

  const rows = items.map((item) => {
    const cells = [
      <CharacterImage
        key="image"
        name={item.name}
        image={
          item.image_thumb_url
            ? { asset_id: item.id, url: item.image_thumb_url, width: 40, height: 40 }
            : null
        }
        size="sm"
        thumb
      />,
      <Link key="name" className="character-name-link" href={`/characters?id=${encodeURIComponent(item.id)}`}>
        {item.name}
      </Link>,
      <span key="realm">{item.system_summary ?? messages.characterNoValue}</span>,
    ];
    if (scope === "subtree") {
      cells.push(
        item.faction ? (
          <Link
            key="faction"
            className="character-name-link"
            href={`/factions?id=${encodeURIComponent(item.faction.id)}`}
          >
            {item.faction.name}
          </Link>
        ) : (
          <span key="faction">{messages.characterNoValue}</span>
        ),
      );
    }
    cells.push(
      <span key="ranking" className="character-muted">
        {messages.characterNoValue}
      </span>,
    );
    return cells;
  });

  return (
    <section className="character-detail__card" aria-labelledby="faction-members-title">
      <div className="faction-members__header">
        <h2 id="faction-members-title" className="character-detail__section-title">
          {messages.factionMembersTitle}
        </h2>
        <div className="faction-members__scope" role="group" aria-label={messages.factionMembersTitle}>
          <Button
            variant={scope === "direct" ? "primary" : "secondary"}
            size="sm"
            aria-pressed={scope === "direct"}
            onClick={() => {
              setScope("direct");
              setPage(1);
            }}
          >
            {messages.factionMembersDirect}
          </Button>
          <Button
            variant={scope === "subtree" ? "primary" : "secondary"}
            size="sm"
            aria-pressed={scope === "subtree"}
            disabled={!hasChildren}
            title={hasChildren ? undefined : messages.factionMembersEmptySubtree}
            onClick={() => {
              setScope("subtree");
              setPage(1);
            }}
          >
            {messages.factionMembersSubtree}
          </Button>
        </div>
      </div>

      {status === "error" ? (
        <Alert
          tone="danger"
          title={toUserMessage(error, messages.factionMembersLoadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={() => setReloadTick((value) => value + 1)}>
              {messages.retry}
            </Button>
          }
        />
      ) : (
        <>
          <Table
            caption={messages.factionMembersTitle}
            columns={columns}
            rows={rows}
            loading={status === "loading"}
            skeletonRows={5}
            empty={<EmptyState variant="inline" title={empty} description={messages.factionMembersEmptyHint} />}
            rowKey={(index) => items[index]?.id ?? String(index)}
          />
          {data && data.total > PAGE_SIZE ? (
            <Pagination page={data.page} pageSize={data.page_size} total={data.total} onChange={setPage} />
          ) : null}
        </>
      )}
    </section>
  );
}
