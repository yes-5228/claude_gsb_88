// 轻量提示：右上角浮层，数秒后自动消失。
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';

type ToastKind = 'success' | 'error' | 'info';

interface ToastItem {
  id: number;
  kind: ToastKind;
  text: string;
}

interface ToastValue {
  show: (kind: ToastKind, text: string) => void;
  success: (text: string) => void;
  error: (text: string) => void;
}

const ToastContext = createContext<ToastValue | null>(null);

let toastSeq = 0;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);

  const show = useCallback((kind: ToastKind, text: string) => {
    toastSeq += 1;
    const id = toastSeq;
    setItems((prev) => [...prev, { id, kind, text }]);
    window.setTimeout(() => {
      setItems((prev) => prev.filter((item) => item.id !== id));
    }, 4000);
  }, []);

  const value = useMemo<ToastValue>(
    () => ({
      show,
      success: (text) => show('success', text),
      error: (text) => show('error', text)
    }),
    [show]
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-stack">
        {items.map((item) => (
          <div key={item.id} className={`toast toast-${item.kind}`}>
            {item.text}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastValue {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error('useToast 必须在 ToastProvider 内部使用');
  }
  return value;
}
