"use client";

import { WorkSwitcher } from "@/components/shell/work-switcher";

/**
 * 顶栏：品牌、作品切换器、本机状态胶囊（PG-00 §2.1）。
 * 作品切换器是唯一作品入口，包含新建/编辑/删除与回收站入口。
 */
export function TopBar() {
  return (
    <header className="shell__topbar">
      <div className="shell__topbar-left">
        <span className="shell__brand">
          <span className="shell__brand-mark" aria-hidden="true">
            设
          </span>
          <span className="shell__brand-name">小说设定管理系统</span>
        </span>
        <WorkSwitcher />
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
