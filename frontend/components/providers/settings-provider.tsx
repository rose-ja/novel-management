"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

import { fetchJson } from "@/lib/api/client";
import { EMPTY_SETTINGS, type AppSettings } from "@/lib/api/types";

type SettingsStatus = "loading" | "ready" | "error";

type SettingsContextValue = {
  settings: AppSettings;
  status: SettingsStatus;
  error: Error | null;
  reload: () => void;
};

const SettingsContext = createContext<SettingsContextValue | null>(null);

/** 全局只读设置：顶栏与页面共用同一次请求，避免重复拉取与状态不一致。 */
export function SettingsProvider({ children }: { children: ReactNode }) {
  const [settings, setSettings] = useState<AppSettings>(EMPTY_SETTINGS);
  const [status, setStatus] = useState<SettingsStatus>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [tick, setTick] = useState(0);

  const reload = useCallback(() => setTick((value) => value + 1), []);

  useEffect(() => {
    let active = true;
    setStatus("loading");
    setError(null);
    fetchJson<AppSettings>("/api/v1/settings")
      .then((value) => {
        if (!active) return;
        setSettings(value);
        setStatus("ready");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setError(cause instanceof Error ? cause : new Error("加载失败，请重试"));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [tick]);

  const value = useMemo(() => ({ settings, status, error, reload }), [settings, status, error, reload]);

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}

export function useSettings(): SettingsContextValue {
  const context = useContext(SettingsContext);
  if (!context) {
    throw new Error("useSettings 必须在 SettingsProvider 内使用");
  }
  return context;
}
