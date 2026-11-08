"use client";

import { BookOpen } from "lucide-react";

import { useSettings } from "@/components/providers/settings-provider";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Skeleton } from "@/components/ui/skeleton";
import { NetworkError } from "@/lib/api/client";
import { messages } from "@/lib/messages";

export default function OverviewPage() {
  const { settings, status, error, reload } = useSettings();

  const errorTitle = error instanceof NetworkError ? messages.networkError : messages.loadFailed;

  return (
    <div className="page">
      <PageHeader
        title="概览"
        description="本地优先的小说设定资料库：输入名称即可查到完整设定，改一处全局同步。"
      />

      {status === "loading" ? (
        <Card title="系统信息">
          <Skeleton variant="lines" lines={3} label="正在加载系统信息" />
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

      {status === "ready" && !settings.current_work_id ? (
        <EmptyState
          variant="page"
          icon={<BookOpen aria-hidden="true" size={22} />}
          title={messages.noWorkTitle}
          description={messages.noWorkDescription}
        />
      ) : null}

      {status === "ready" ? (
        <Card title="系统信息" description="以下信息由本机服务实时读取。">
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
          </dl>
        </Card>
      ) : null}
    </div>
  );
}
