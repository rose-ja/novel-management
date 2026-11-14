"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { messages } from "@/lib/messages";

export type JumpToSeatProps = {
  seatCount: number;
  pageSize: number;
  onJump: (seatNo: number, page: number) => void;
};

/** 跳转到指定席位：按 50 席/页换算目标页并高亮。 */
export function JumpToSeat({ seatCount, pageSize, onJump }: JumpToSeatProps) {
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);

  function submit() {
    const seatNo = Number(value);
    if (!Number.isInteger(seatNo) || seatNo < 1 || seatNo > seatCount) {
      setError(messages.rankingJumpOutOfRange(seatCount));
      return;
    }
    setError(null);
    onJump(seatNo, Math.floor((seatNo - 1) / pageSize) + 1);
  }

  return (
    <div className="ranking-jump">
      <Input
        type="number"
        min={1}
        max={seatCount}
        value={value}
        placeholder={messages.rankingJumpPlaceholder}
        aria-label={messages.rankingJumpLabel}
        onChange={(event) => {
          setValue(event.target.value);
          if (error) setError(null);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") submit();
        }}
      />
      <Button variant="secondary" size="sm" onClick={submit}>
        {messages.rankingJumpSubmit}
      </Button>
      {error ? <span className="ui-field__error">{error}</span> : null}
    </div>
  );
}
