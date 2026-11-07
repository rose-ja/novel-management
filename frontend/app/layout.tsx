import type { Metadata } from "next";

import "./globals.css";

export const metadata: Metadata = {
  title: "小说设定管理系统",
  description: "本地小说设定管理：人物、势力、地点、榜单、体系与大事记。",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}
