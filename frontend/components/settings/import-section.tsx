"use client";

import Link from "next/link";
import { UploadCloud } from "lucide-react";

import { Card } from "@/components/ui/card";
import { messages } from "@/lib/messages";

export function ImportSection() {
  return (
    <Card title={messages.importCardTitle} description={messages.importCardDescription}>
      <p className="import-muted">{messages.importEntryHint}</p>
      <div className="import-actions">
        <Link className="ui-button ui-button--secondary ui-button--md" href="/import">
          <UploadCloud aria-hidden="true" size={16} />
          <span className="ui-button__label">{messages.importEntry}</span>
        </Link>
      </div>
    </Card>
  );
}
