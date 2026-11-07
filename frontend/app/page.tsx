"use client";

import { useEffect, useState } from "react";

import { describeHealth, fetchHealth, type Health } from "@/lib/api/health";

export default function HomePage() {
  const [health, setHealth] = useState<Health | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    fetchHealth()
      .then((value) => {
        if (active) {
          setHealth(value);
        }
      })
      .catch(() => {
        if (active) {
          setError("无法连接后端服务");
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const message = health
    ? `版本 ${health.version} · ${describeHealth(health)}`
    : error ?? "正在检查服务状态…";

  return (
    <main className="app-shell">
      <section className="status-card">
        <h1 className="status-card__title">小说设定管理系统</h1>
        <p className="status-card__meta">{message}</p>
      </section>
    </main>
  );
}
