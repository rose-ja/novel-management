import { useCallback, useRef, useState } from "react";

export type UnsavedGuard = {
  dirty: boolean;
  markDirty: (dirty: boolean) => void;
  /** 有未保存修改时挂起动作并打开确认框，否则立即执行。 */
  requestLeave: (action: () => void) => void;
  confirmOpen: boolean;
  discardAndLeave: () => void;
  keepEditing: () => void;
  reset: () => void;
};

/** 拦截「关闭/切换」动作：脏状态下先确认是否放弃修改。 */
export function useUnsavedGuard(): UnsavedGuard {
  const [dirty, setDirty] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const pending = useRef<(() => void) | null>(null);

  const requestLeave = useCallback(
    (action: () => void) => {
      if (!dirty) {
        action();
        return;
      }
      pending.current = action;
      setConfirmOpen(true);
    },
    [dirty],
  );

  const discardAndLeave = useCallback(() => {
    const action = pending.current;
    pending.current = null;
    setConfirmOpen(false);
    setDirty(false);
    action?.();
  }, []);

  const keepEditing = useCallback(() => {
    pending.current = null;
    setConfirmOpen(false);
  }, []);

  const reset = useCallback(() => {
    pending.current = null;
    setConfirmOpen(false);
    setDirty(false);
  }, []);

  return {
    dirty,
    markDirty: setDirty,
    requestLeave,
    confirmOpen,
    discardAndLeave,
    keepEditing,
    reset,
  };
}
