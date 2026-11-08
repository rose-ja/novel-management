"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { AlertTriangle, CheckCircle2, Info, Undo2, X } from "lucide-react";

export type ToastTone = "success" | "error" | "info" | "undo";

export type ToastItem = {
  id: string;
  tone: ToastTone;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
};

type ToastContextValue = {
  showToast: (toast: Omit<ToastItem, "id">) => string;
  dismissToast: (id: string) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

const toneIcon: Record<ToastTone, ReactNode> = {
  success: <CheckCircle2 aria-hidden="true" size={18} />,
  error: <AlertTriangle aria-hidden="true" size={18} />,
  info: <Info aria-hidden="true" size={18} />,
  undo: <Undo2 aria-hidden="true" size={18} />,
};

/** 非阻塞提示：成功/错误 4 秒消失，撤销 8 秒；均可用关闭按钮立即移除。 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());

  const dismissToast = useCallback((id: string) => {
    const timer = timers.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timers.current.delete(id);
    }
    setItems((current) => current.filter((item) => item.id !== id));
  }, []);

  const showToast = useCallback(
    (toast: Omit<ToastItem, "id">) => {
      const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      setItems((current) => [...current, { ...toast, id }]);
      const timer = setTimeout(() => dismissToast(id), toast.tone === "undo" ? 8000 : 4000);
      timers.current.set(id, timer);
      return id;
    },
    [dismissToast],
  );

  useEffect(() => {
    const pending = timers.current;
    return () => {
      pending.forEach((timer) => clearTimeout(timer));
      pending.clear();
    };
  }, []);

  const value = useMemo(() => ({ showToast, dismissToast }), [showToast, dismissToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="ui-toast-viewport" role="status" aria-live="polite">
        {items.map((item) => (
          <div key={item.id} className={`ui-toast ui-toast--${item.tone}`}>
            <span className="ui-toast__icon">{toneIcon[item.tone]}</span>
            <span className="ui-toast__message">{item.message}</span>
            {item.actionLabel && item.onAction ? (
              <button
                type="button"
                className="ui-toast__action"
                onClick={() => {
                  item.onAction?.();
                  dismissToast(item.id);
                }}
              >
                {item.actionLabel}
              </button>
            ) : null}
            <button type="button" className="ui-toast__close" aria-label="关闭提示" onClick={() => dismissToast(item.id)}>
              <X aria-hidden="true" size={14} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast 必须在 ToastProvider 内使用");
  }
  return context;
}
