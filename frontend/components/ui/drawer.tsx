"use client";

import * as RadixDialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ReactNode } from "react";

export type DrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  side?: "right" | "left";
  width?: "default" | "wide";
  children?: ReactNode;
  footer?: ReactNode;
};

export function Drawer({
  open,
  onOpenChange,
  title,
  description,
  side = "right",
  width = "default",
  children,
  footer,
}: DrawerProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="ui-overlay" />
        <RadixDialog.Content
          className={`ui-drawer ui-drawer--${side} ui-drawer--${width}`}
          aria-describedby={undefined}
        >
          <div className="ui-dialog__header">
            <RadixDialog.Title className="ui-dialog__title">{title}</RadixDialog.Title>
            <RadixDialog.Close asChild>
              <button type="button" className="ui-icon-button ui-icon-button--sm" aria-label="关闭">
                <X aria-hidden="true" size={16} />
              </button>
            </RadixDialog.Close>
          </div>
          {description ? <RadixDialog.Description className="ui-dialog__description">{description}</RadixDialog.Description> : null}
          <div className="ui-drawer__body">{children}</div>
          {footer ? <div className="ui-dialog__footer">{footer}</div> : null}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}
