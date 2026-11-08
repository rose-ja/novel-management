"use client";

import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { NetworkError } from "@/lib/api/client";
import { messages } from "@/lib/messages";

function formatTime(value: string | null): string {
  if (!value) return "尚未执行";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "尚未执行" : date.toLocaleString("zh-CN", { hour12: false });
}

export default function SettingsPage() {
  const { settings, status, error, reload } = useSettings();
  const errorTitle = error instanceof NetworkError ? messages.networkError : messages.loadFailed;

  return (
    <div className="page">
      <PageHeader title="设置" description="当前系统的只读配置与版本信息。" />

      {status === "loading" ? (
        <Card title="应用信息">
          <Skeleton variant="lines" lines={5} label="正在加载设置" />
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
        <Card title="应用信息">
          <dl className="info-list">
            <div className="info-list__row">
              <dt>应用版本</dt>
              <dd>v{settings.app_version}</dd>
            </div>
            <div className="info-list__row">
              <dt>数据目录</dt>
              <dd className="info-list__mono">{settings.data_dir}</dd>
            </div>
            <div className="info-list__row">
              <dt>自动快照保留</dt>
              <dd>{settings.snapshot_keep} 份</dd>
            </div>
            <div className="info-list__row">
              <dt>当前作品</dt>
              <dd>{settings.current_work_id ?? messages.noWorkBadge}</dd>
            </div>
            <div className="info-list__row">
              <dt>最近导出</dt>
              <dd>{formatTime(settings.last_export_at)}</dd>
            </div>
            <div className="info-list__row">
              <dt>最近恢复</dt>
              <dd>{formatTime(settings.last_restore_at)}</dd>
            </div>
          </dl>
        </Card>
      ) : null}
    </div>
  );
}
