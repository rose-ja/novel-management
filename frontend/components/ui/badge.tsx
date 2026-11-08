import type { ReactNode } from "react";

export type BadgeTone = "neutral" | "brand" | "success" | "warning" | "danger" | "info";

export type BadgeProps = {
  tone?: BadgeTone;
  children: ReactNode;
  /** 未校对、已停用等状态用虚线边框区分。 */
  outlined?: boolean;
};

export function Badge({ tone = "neutral", outlined = false, children }: BadgeProps) {
  return (
    <span className={`ui-badge ui-badge--${tone}${outlined ? " ui-badge--outlined" : ""}`}>{children}</span>
  );
}
