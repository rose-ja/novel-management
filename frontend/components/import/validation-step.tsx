"use client";

import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import type { ImportDetail } from "@/lib/api/imports";
import { messages } from "@/lib/messages";

export type ValidationStepProps = {
  detail: ImportDetail | null;
  error: string | null;
  onReupload: () => void;
  onConfirm: () => void;
};

function errorText(line: number, field: string, message: string): string {
  return messages.importErrorLine(line, field, message);
}

export function ValidationStep({ detail, error, onReupload, onConfirm }: ValidationStepProps) {
  if (!detail) {
    return (
      <Card title={messages.importValidating} description={messages.importValidatingHint}>
        <Progress label={messages.importValidating} />
        <Skeleton variant="lines" lines={4} label={messages.importValidating} />
      </Card>
    );
  }

  const hasError = detail.error_count > 0;
  const counts = [
    { label: messages.importCountCharacters, value: detail.counts.characters },
    { label: messages.importCountFactions, value: detail.counts.factions },
    { label: messages.importCountLocations, value: detail.counts.locations },
    { label: messages.importCountRankings, value: detail.counts.rankings },
    { label: messages.importCountCategories, value: detail.counts.system_categories },
    { label: messages.importCountNodes, value: detail.counts.system_nodes },
    { label: messages.importCountChronicles, value: detail.counts.chronicles },
    { label: messages.importCountImages, value: detail.counts.images },
    { label: messages.importCountReviewable, value: detail.counts.reviewable },
  ];

  return (
    <div className="import-grid">
      <div className="import-grid__main">
        {hasError ? (
          <Alert tone="danger" title={messages.importErrorBanner(detail.error_count)} />
        ) : (
          <Alert tone="success" title={messages.importPassBanner} />
        )}
        {error ? <Alert tone="danger" title={error} /> : null}

        <Card title={messages.importCountsTitle}>
          <dl className="import-counts">
            {counts.map((item) => (
              <div className="import-counts__item" key={item.label}>
                <dt>{item.label}</dt>
                <dd>{item.value}</dd>
              </div>
            ))}
          </dl>
        </Card>

        {detail.errors.length > 0 ? (
          <Card title={messages.importErrorList(detail.error_count)}>
            <ol className="import-error-list" aria-live="polite">
              {detail.errors.map((issue, index) => (
                <li key={`${issue.code}-${issue.line}-${index}`}>
                  {errorText(issue.line, issue.field, issue.message)}
                </li>
              ))}
            </ol>
          </Card>
        ) : null}

        <Card title={messages.importImageMatch}>
          <p>
            {messages.importImageMatched(detail.images.matched)} · {messages.importImageMissing(detail.images.missing.length)} ·{" "}
            {messages.importImageExtra(detail.images.extra.length)} ·{" "}
            {messages.importImageUnsupported(detail.images.unsupported.length)} ·{" "}
            {messages.importImageTooLarge(detail.images.too_large?.length ?? 0)}
          </p>
          {detail.images.missing.length +
            detail.images.extra.length +
            detail.images.unsupported.length +
            (detail.images.too_large?.length ?? 0) >
          0 ? (
            <details>
              <summary>
                {messages.importErrorList(
                  detail.images.missing.length +
                    detail.images.extra.length +
                    detail.images.unsupported.length +
                    (detail.images.too_large?.length ?? 0),
                )}
              </summary>
              <ul className="import-image-list">
                {detail.images.missing.map((name) => (
                  <li key={`missing-${name}`}>
                    {messages.importImageMissing(1)} · {name}
                  </li>
                ))}
                {detail.images.extra.map((name) => (
                  <li key={`extra-${name}`}>
                    {messages.importImageExtra(1)} · {name}
                  </li>
                ))}
                {detail.images.unsupported.map((name) => (
                  <li key={`unsupported-${name}`}>
                    {messages.importImageUnsupported(1)} · {name}
                  </li>
                ))}
                {(detail.images.too_large ?? []).map((name) => (
                  <li key={`too-large-${name}`}>
                    {messages.importImageTooLarge(1)} · {name}
                  </li>
                ))}
              </ul>
            </details>
          ) : null}
        </Card>
      </div>

      <aside className="import-grid__aside">
        <Card title={messages.importTargetTitle}>
          <dl className="info-list">
            <div className="info-list__row">
              <dt>{messages.importTargetTitle}</dt>
              <dd>{detail.work_name ? messages.importTargetWork(detail.work_name) : "—"}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importTemplateVersion}</dt>
              <dd>v{detail.template_version ?? "1.0"}</dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importAtomic}</dt>
              <dd>
                <Badge tone="success">是</Badge>
              </dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importRollback}</dt>
              <dd>
                <Badge tone="success">是</Badge>
              </dd>
            </div>
            <div className="info-list__row">
              <dt>{messages.importSnapshot}</dt>
              <dd>
                <Badge tone="success">是</Badge>
              </dd>
            </div>
          </dl>
          <p className="import-muted">{messages.importPostReview}</p>
          <div className="import-actions import-actions--stack">
            <Button variant="secondary" onClick={onReupload}>
              {messages.importStepUpload}
            </Button>
            <Button onClick={onConfirm} disabled={hasError}>
              {messages.importConfirmSubmit}
            </Button>
            {hasError ? <p className="import-error-text">{messages.importConfirmDisabled(detail.error_count)}</p> : null}
          </div>
        </Card>
      </aside>
    </div>
  );
}
