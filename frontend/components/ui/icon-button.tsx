"use client";

import { forwardRef, type ButtonHTMLAttributes } from "react";

export type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  /** 图标按钮无可读文本，必须提供无障碍名称。 */
  label: string;
  size?: "sm" | "md";
  tone?: "neutral" | "danger";
};

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { label, size = "md", tone = "neutral", className, children, ...props },
  ref,
) {
  const classes = ["ui-icon-button", `ui-icon-button--${size}`, `ui-icon-button--${tone}`, className]
    .filter(Boolean)
    .join(" ");

  return (
    <button {...props} ref={ref} type={props.type ?? "button"} className={classes} aria-label={label} title={label}>
      {children}
    </button>
  );
});
