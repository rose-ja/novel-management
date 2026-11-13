"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { LayoutDashboard, Layers, Network, Settings, Trash2, Users } from "lucide-react";

import { useSettings } from "@/components/providers/settings-provider";

/**
 * 导航只呈现已交付路由（F1 裁决 A2）；PG-00 的分组结构保留，未实现项在对应变更中追加。
 */
const navGroups = [
  {
    title: "资料",
    items: [
      { href: "/", label: "概览", icon: LayoutDashboard },
      { href: "/system", label: "体系", icon: Layers },
      { href: "/characters", label: "人物", icon: Users },
      { href: "/factions", label: "势力", icon: Network },
    ],
  },
  {
    title: "工具",
    items: [
      { href: "/trash", label: "回收站", icon: Trash2 },
      { href: "/settings", label: "设置", icon: Settings },
    ],
  },
] as const;

export function GlobalNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const { settings, status } = useSettings();

  return (
    <nav className="shell__nav" aria-label="主导航">
      {navGroups.map((group) => (
        <div className="shell__nav-group" key={group.title}>
          <p className="shell__nav-group-title">{group.title}</p>
          <ul className="shell__nav-list">
            {group.items.map((item) => {
              const Icon = item.icon;
              const active = pathname === item.href;
              return (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    className={active ? "shell__nav-link shell__nav-link--active" : "shell__nav-link"}
                    aria-current={active ? "page" : undefined}
                    onClick={onNavigate}
                  >
                    <Icon aria-hidden="true" size={18} />
                    <span>{item.label}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
      <div className="shell__nav-footer">
        <span>{status === "ready" ? `v${settings.app_version}` : "v—"}</span>
        <span>数据仅存本机</span>
      </div>
    </nav>
  );
}
