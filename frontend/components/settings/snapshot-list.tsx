"use client";

import { useCallback, useEffect, useState } from "react";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { EmptyState } from "@/components/ui/empty-state";
import { Table } from "@/components/ui/table";
import { useToast } from "@/components/ui/toast";
import { toUserMessage } from "@/lib/api/errors";
import { listSnapshots, type Snapshot } from "@/lib/api/snapshots";
import { copyText } from "@/lib/clipboard";
import { formatBytes, formatDateTime, truncateMiddle } from "@/lib/format";
import { messages } from "@/lib/messages";

type Status = "loading" | "ready" | "error";

const columns = [
  { key: "time", label: messages.snapshotColumnTime, width: "168px" },
  { key: "reason", label: messages.snapshotColumnReason, width: "128px" },
  { key: "size", label: messages.snapshotColumnSize, width: "128px" },
  { key: "status", label: messages.snapshotColumnStatus, width: "96px" },
  { key: "path", label: messages.snapshotColumnPath, width: "280px" },
];

/** 设置页只读快照列表；恢复入口在命令行，不提供页面按钮（D-05）。 */
export function SnapshotList() {
  const { showToast } = useToast();
  const [items, setItems] = useState<Snapshot[]>([]);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<Error | null>(null);

  const copyPath = useCallback(
    async (path: string) => {
      const ok = await copyText(path);
      showToast({
        tone: ok ? "success" : "error",
        message: ok ? messages.settingsSnapshotPathCopied : messages.copyFailed,
      });
    },
    [showToast],
  );

  const load = useCallback(async () => {
    setStatus("loading");
    setError(null);
    try {
      const result = await listSnapshots();
      setItems(result);
      setStatus("ready");
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
      setStatus("error");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <Card title={messages.snapshotCardTitle} description={messages.snapshotCardDescription}>
      {status === "error" ? (
        <Alert
          tone="danger"
          title={toUserMessage(error, messages.loadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={() => void load()}>
              {messages.retry}
            </Button>
          }
        />
      ) : (
        <Table
          caption="快照列表"
          columns={columns}
          loading={status === "loading"}
          rows={items.map((snapshot) => [
            <span key="time" className="snapshot-time">
              {formatDateTime(snapshot.created_at)}
            </span>,
            messages.snapshotReasons[snapshot.reason] ?? snapshot.reason,
            formatBytes(snapshot.db_bytes),
            snapshot.status === "ok" ? (
              <Badge key="status" tone="success">
                {messages.snapshotStatusOK}
              </Badge>
            ) : (
              <Badge key="status" tone="danger">
                {messages.snapshotStatusFailed}
              </Badge>
            ),
            <span key="path" className="settings-path" title={snapshot.path}>
              <code className="settings-path__text">{truncateMiddle(snapshot.path, 42)}</code>
              <Button
                variant="ghost"
                size="sm"
                aria-label={messages.settingsSnapshotPathCopy}
                onClick={() => void copyPath(snapshot.path)}
              >
                {messages.copy}
              </Button>
            </span>,
          ])}
          rowKey={(index) => items[index]?.id ?? String(index)}
          empty={
            <EmptyState
              variant="inline"
              title={messages.snapshotEmpty}
              description={messages.snapshotEmptyDescription}
            />
          }
        />
      )}
    </Card>
  );
}
