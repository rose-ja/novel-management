"use client";

import { forwardRef, type InputHTMLAttributes } from "react";

export type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  size?: "sm" | "md";
  invalid?: boolean;
};

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { size = "md", invalid = false, className, ...props },
  ref,
) {
  const classes = ["ui-input", `ui-input--${size}`, className].filter(Boolean).join(" ");

  return (
    <input
      {...props}
      ref={ref}
      className={classes}
      aria-invalid={invalid || props["aria-invalid"] || undefined}
    />
  );
});
