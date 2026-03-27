import {
  createContext,
  useContext,
  useState,
  useCallback,
  useRef,
  useEffect,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { Icon } from "./icon";

type ToastVariant = "success" | "error" | "warning" | "info";

type Toast = {
  id: string;
  message: string;
  variant: ToastVariant;
};

type ToastContextValue = {
  toast: (message: string, variant?: ToastVariant) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx;
}

const variantClasses: Record<ToastVariant, string> = {
  success: "bg-success-container text-on-success-container",
  error: "bg-error-container text-on-error-container",
  warning: "bg-warning-container text-on-warning-container",
  info: "bg-surface-container-lowest text-on-surface shadow-ambient-md",
};

const AUTO_DISMISS_MS = 5000;

function ToastItem({
  toast: t,
  onDismiss,
}: {
  toast: Toast;
  onDismiss: (id: string) => void;
}) {
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    timerRef.current = setTimeout(() => onDismiss(t.id), AUTO_DISMISS_MS);
    return () => clearTimeout(timerRef.current);
  }, [t.id, onDismiss]);

  return (
    <div
      className={`flex items-center gap-3 rounded-lg px-4 py-3 type-body-md shadow-ambient-sm animate-in slide-in-from-right ${variantClasses[t.variant]}`}
      role="status"
    >
      <span className="flex-1">{t.message}</span>
      <button
        onClick={() => onDismiss(t.id)}
        className="shrink-0 rounded-sm p-1 hover:bg-on-surface/[var(--opacity-hover)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        aria-label="Dismiss"
      >
        <Icon icon={X} size="sm" />
      </button>
    </div>
  );
}

type ToastProviderProps = {
  children: ReactNode;
};

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const counterRef = useRef(0);

  const addToast = useCallback((message: string, variant: ToastVariant = "info") => {
    counterRef.current += 1;
    const id = `toast-${String(counterRef.current)}`;
    setToasts((prev) => [...prev, { id, message, variant }]);
  }, []);

  const dismissToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext value={{ toast: addToast }}>
      {children}
      {createPortal(
        <div
          className="fixed right-4 top-4 z-notification flex flex-col gap-2 w-full max-w-sm"
          aria-live="polite"
          aria-label="Notifications"
        >
          {toasts.map((t) => (
            <ToastItem key={t.id} toast={t} onDismiss={dismissToast} />
          ))}
        </div>,
        document.body,
      )}
    </ToastContext>
  );
}
