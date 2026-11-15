"use client";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useToast } from "@/components/ui/toast";
import { copyText } from "@/lib/clipboard";
import { messages } from "@/lib/messages";

function CommandBlock({ label, command, copyLabel }: { label: string; command: string; copyLabel: string }) {
  const { showToast } = useToast();

  async function handleCopy() {
    const ok = await copyText(command);
    showToast({ tone: ok ? "success" : "error", message: ok ? messages.copySuccess : messages.copyFailed });
  }

  return (
    <div className="command-block">
      <div className="command-block__header">
        <span className="command-block__label">{label}</span>
        <Button variant="secondary" size="sm" aria-label={copyLabel} onClick={() => void handleCopy()}>
          {messages.copy}
        </Button>
      </div>
      <code className="command-block__code">{command}</code>
    </div>
  );
}

/** PG-13 CLI 恢复区块：只提供命令与注意事项，页面内不执行恢复。 */
export function CliRestoreSection() {
  return (
    <Card title={messages.settingsCommandTitle} description={messages.settingsCommandDescription}>
      <div className="command-list">
        <CommandBlock
          label={messages.settingsCommandData}
          command={messages.settingsCommandDataValue}
          copyLabel={messages.settingsCommandCopy}
        />
        <CommandBlock
          label={messages.settingsCommandSnapshot}
          command={messages.settingsCommandSnapshotValue}
          copyLabel={messages.settingsCommandCopySnapshot}
        />
      </div>
      <p className="settings-hint">{messages.settingsCommandStopHint}</p>
      <Alert tone="warning" title={messages.settingsCommandWarning} />
    </Card>
  );
}
