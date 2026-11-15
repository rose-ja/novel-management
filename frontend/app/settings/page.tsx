"use client";

import { useEffect, useState } from "react";

import { useSettings } from "@/components/providers/settings-provider";
import { CliRestoreSection } from "@/components/settings/cli-restore-section";
import { ExportSection } from "@/components/settings/export-section";
import { SnapshotList } from "@/components/settings/snapshot-list";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { NetworkError } from "@/lib/api/client";
import { listWorks } from "@/lib/api/works";
import { messages } from "@/lib/messages";

function formatTime(value: string | null): string {
  if (!value) return messages.settingsNotYet;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? messages.settingsNotYet : date.toLocaleString("zh-CN", { hour12: false });
}

export default function SettingsPage() {
  const { settings, status, error, reload } = useSettings();
  const errorTitle = error instanceof NetworkError ? messages.networkError : messages.loadFailed;
  const [workName, setWorkName] = useState<string | null>(null);

  useEffect(() => {
    if (!settings.current_work_id) {
      setWorkName(null);
      return;
    }
    let active = true;
    listWorks({ pageSize: 100 })
      .then((page) => {
        if (!active) return;
        const found = page.items.find((work) => work.id === settings.current_work_id);
        setWorkName(found?.name ?? null);
      })
      .catch(() => {
        if (active) setWorkName(null);
      });
    return () => {
      active = false;
    };
  }, [settings.current_work_id]);

  return (
    <div className="page">
      <PageHeader title={messages.settingsTitle} description={messages.settingsDescription} />

      <ExportSection />
      <SnapshotList />
      <CliRestoreSection />

      {status === "loading" ? (
        <Card title={messages.settingsInfoTitle}>
          <Skeleton variant="lines" lines={6} label="正在加载设置" />
        </Card>
      ) : null}

      {status === "error" ? (
        <Alert
          tone="danger"
          title={errorTitle}
          action={
            <Button variant="secondary" size="sm" onClick={reload}>
              {messages.retry}
            </Button>
          }
        />
      ) : null}

      {status === "ready" ? (
        <Card title={messages.settingsInfoTitle}>
          <dl className="info-list">
            <div className="info-list__row">
              <dt>{messages.settingsVersionLabel}</dt>
              <dd>v{settings.app_version}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.settingsDataDirLabel}</dt>
              <dd className="info-list__mono">{settings.data_dir}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.settingsSnapshotKeepLabel}</dt>
              <dd>{settings.snapshot_keep} 份</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.settingsCurrentWorkLabel}</dt>
              <dd>{settings.current_work_id ? `《${workName ?? settings.current_work_id}》` : messages.noWorkBadge}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.settingsAddressLabel}</dt>
              <dd className="info-list__mono">{messages.settingsAddress}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.settingsLastExportLabel}</dt>
              <dd>{formatTime(settings.last_export_at)}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.settingsLastRestoreLabel}</dt>
              <dd>{formatTime(settings.last_restore_at)}</dd>
            </div>
          </dl>
        </Card>
      ) : null}
    </div>
  );
}
