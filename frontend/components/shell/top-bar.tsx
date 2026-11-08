"use client";

import { BookOpen } from "lucide-react";

import { useSettings } from "@/components/providers/settings-provider";
import { messages } from "@/lib/messages";

/**
 * 顶栏：品牌、作品切换器占位、本机状态胶囊（PG-00 §2.1）。
 * CHG-001 尚无作品能力，作品位只做只读展示，不渲染不可用按钮。
 */
export function TopBar() {
  const { settings, status } = useSettings();
  const hasWork = Boolean(settings.current_work_id);

  return (
    <header className="shell__topbar">
      <div className="shell__topbar-left">
        <span className="shell__brand">
          <span className="shell__brand-mark" aria-hidden="true">
            设
          </span>
          <span className="shell__brand-name">小说设定管理系统</span>
        </span>
        <span className="shell__work" aria-live="polite">
          <BookOpen aria-hidden="true" size={14} />
          <span className="shell__work-text">
            {status === "loading" ? "正在读取作品…" : hasWork ? "当前作品" : messages.noWorkBadge}
          </span>
        </span>
      </div>
      <div className="shell__topbar-right">
        <span className="shell__status-pill">
          <span className="shell__status-dot" aria-hidden="true" />
          本机 · 127.0.0.1
        </span>
      </div>
    </header>
  );
}
