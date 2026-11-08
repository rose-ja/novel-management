import type { Metadata } from "next";

import { AppShell } from "@/components/shell/app-shell";

import "./globals.css";
import "../styles/components.css";

export const metadata: Metadata = {
  title: "小说设定管理系统",
  description: "本地优先的小说设定资料库：人物、势力、地点、榜单、体系与大事记。",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>
        <AppShell>{children}</AppShell>
      </body>
    </html>
  );
}
