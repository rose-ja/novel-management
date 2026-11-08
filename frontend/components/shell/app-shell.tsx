"use client";

import type { ReactNode } from "react";

import { SettingsProvider } from "@/components/providers/settings-provider";
import { GlobalNav } from "@/components/shell/global-nav";
import { TopBar } from "@/components/shell/top-bar";
import { ToastProvider } from "@/components/ui/toast";

/** 应用壳层：跳转链接、顶栏、固定侧栏、内容区与桌面宽度提示（PG-00）。 */
export function AppShell({ children }: { children: ReactNode }) {
  return (
    <SettingsProvider>
      <ToastProvider>
        <div className="shell">
          <a className="shell__skip-link" href="#main-content">
            跳到主要内容
          </a>
          <div className="shell__desktop-notice" role="note">
            本系统按桌面端设计，建议使用 1024px 以上窗口
          </div>
          <TopBar />
          <div className="shell__body">
            <aside className="shell__sidebar">
              <GlobalNav />
            </aside>
            <main id="main-content" className="shell__main" tabIndex={-1}>
              {children}
            </main>
          </div>
        </div>
      </ToastProvider>
    </SettingsProvider>
  );
}
