"use client";

import Link from "next/link";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { ImportReport } from "@/lib/api/imports";
import { messages } from "@/lib/messages";

function formatTime(value: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString("zh-CN", { hour12: false });
}

export function ReportStep({ report, onReview }: { report: ImportReport; onReview: () => void }) {
  const counts = [
    { label: messages.importCountCharacters, value: report.counts.characters },
    { label: messages.importCountFactions, value: report.counts.factions },
    { label: messages.importCountLocations, value: report.counts.locations },
    { label: messages.importCountRankings, value: report.counts.rankings },
    { label: messages.importCountCategories, value: report.counts.system_categories },
    { label: messages.importCountNodes, value: report.counts.system_nodes },
    { label: messages.importCountChronicles, value: report.counts.chronicles },
    { label: messages.importCountImages, value: report.counts.images },
    { label: messages.importCountReviewable, value: report.counts.reviewable },
  ];

  return (
    <div className="import-grid">
      <div className="import-grid__main">
        <Alert tone="success" title={messages.importSuccessTitle} />
        <Card title={messages.importReportTitle}>
          <dl className="import-counts">
            {counts.map((item) => (
              <div className="import-counts__item" key={item.label}>
                <dt>{item.label}</dt>
                <dd>{item.value}</dd>
              </div>
            ))}
          </dl>
          <dl className="info-list">
            <div className="info-list__row">
              <dt>{messages.importReportFiles}</dt>
              <dd>
                {report.file_name}
                {report.archive_name ? ` + ${report.archive_name}` : ""}
              </dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importReportElapsed}</dt>
              <dd>{(report.elapsed_ms / 1000).toFixed(1)}s</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importReportTime}</dt>
              <dd>{formatTime(report.finished_at)}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importSnapshot}</dt>
              <dd className="info-list__mono">{report.snapshot_id ?? "—"}</dd>
            </div>
          </dl>
          <div className="import-actions">
            <Button onClick={onReview}>{messages.importEnterReview}</Button>
            <Link className="ui-button ui-button--secondary ui-button--md" href="/search">
              {messages.importBackSearch}
            </Link>
          </div>
        </Card>
      </div>
      <aside className="import-grid__aside">
        <Card title={messages.importTargetTitle}>
          <p>{report.work_name ? messages.importTargetWork(report.work_name) : "—"}</p>
          <p className="import-muted">{messages.importPostReview}</p>
        </Card>
      </aside>
    </div>
  );
}
