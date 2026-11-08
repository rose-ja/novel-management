"use client";

import * as RadixDialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ReactNode } from "react";

export type DialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  size?: "sm" | "md" | "lg";
  children?: ReactNode;
  footer?: ReactNode;
};

/** 模态对话框：Esc 关闭、焦点锁定与关闭后焦点返回由 Radix 负责。 */
export function Dialog({ open, onOpenChange, title, description, size = "md", children, footer }: DialogProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="ui-overlay" />
        <RadixDialog.Content className={`ui-dialog ui-dialog--${size}`}>
          <div className="ui-dialog__header">
            <RadixDialog.Title className="ui-dialog__title">{title}</RadixDialog.Title>
            <RadixDialog.Close asChild>
              <button type="button" className="ui-icon-button ui-icon-button--sm" aria-label="关闭">
                <X aria-hidden="true" size={16} />
              </button>
            </RadixDialog.Close>
          </div>
          {description ? (
            <RadixDialog.Description className="ui-dialog__description">{description}</RadixDialog.Description>
          ) : null}
          <div className="ui-dialog__body">{children}</div>
          {footer ? <div className="ui-dialog__footer">{footer}</div> : null}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}
