"use client";

import { forwardRef, type SelectHTMLAttributes } from "react";

export type SelectOption = {
  value: string;
  label: string;
  disabled?: boolean;
};

export type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  options: SelectOption[];
  placeholder?: string;
  invalid?: boolean;
};

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { options, placeholder, invalid = false, className, ...props },
  ref,
) {
  const classes = ["ui-select", className].filter(Boolean).join(" ");

  return (
    <select
      {...props}
      ref={ref}
      className={classes}
      aria-invalid={invalid || props["aria-invalid"] || undefined}
    >
      {placeholder ? <option value="">{placeholder}</option> : null}
      {options.map((option) => (
        <option key={option.value} value={option.value} disabled={option.disabled}>
          {option.label}
        </option>
      ))}
    </select>
  );
});
