import { useState, useRef, useEffect, type ReactNode } from "react";
import { Sparkles, X } from "lucide-react";
import { Icon } from "@/components/ui/icon";

type FitBadgeProps = {
  /** If provided, shows "Great for {name}". Otherwise shows "Great match". */
  studentName?: string;
  /** 2-sentence max explanation of why this content fits. Shown on tap. */
  whyText?: string;
  className?: string;
};

/**
 * Positive-only fit indicator shown on content cards when fit score ≥ 0.65.
 * Tapping opens an inline popover with the "why" explanation.
 * Never shown to de-rank content — only additive.
 */
export function FitBadge({
  studentName,
  whyText,
  className = "",
}: FitBadgeProps): ReactNode {
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const label = studentName ? `Great for ${studentName}` : "Great match";

  // Click-outside close for desktop (5.2)
  useEffect(() => {
    if (!open) return;
    function handleMouseDown(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleMouseDown);
    return () => document.removeEventListener("mousedown", handleMouseDown);
  }, [open]);

  const badgeClasses = [
    "inline-flex items-center gap-1 rounded-full px-2.5 py-1 type-label-sm font-medium",
    "bg-tertiary-container text-on-tertiary-container",
  ].join(" ");

  // 5.1: Render non-interactive span when no whyText
  if (!whyText) {
    return (
      <span className={[badgeClasses, className].join(" ")}>
        <Icon icon={Sparkles} size="xs" aria-hidden />
        <span>{label}</span>
      </span>
    );
  }

  return (
    <div ref={wrapperRef} className={`relative inline-flex ${className}`}>
      <button
        type="button"
        aria-expanded={open}
        aria-label={`${label} — tap for details`}
        onClick={() => setOpen((v) => !v)}
        className={[
          badgeClasses,
          "transition-colors hover:bg-tertiary hover:text-on-tertiary",
          "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring",
          "cursor-pointer",
        ].join(" ")}
      >
        <Icon icon={Sparkles} size="xs" aria-hidden />
        <span>{label}</span>
      </button>

      {/* Why popover */}
      {open && (
        <>
          {/* Backdrop (mobile only) */}
          <div
            className="fixed inset-0 z-overlay sm:hidden"
            aria-hidden
            onClick={() => setOpen(false)}
          />

          {/* Popover card — 5.3 role=dialog, 5.4 warm surface tone */}
          <div
            role="dialog"
            aria-label={`Why ${studentName ?? "this"} fits`}
            className={[
              "absolute z-popover mt-1 w-64 rounded-lg px-4 py-3",
              "bg-surface-container-highest text-on-surface shadow-ambient-lg shadow-ghost-border",
              "top-full sm:top-auto sm:bottom-full sm:mb-2 left-0",
            ].join(" ")}
          >
            <div className="flex items-start justify-between gap-3">
              <p className="type-body-sm text-on-surface leading-snug">
                {whyText}
              </p>
              <button
                type="button"
                aria-label="Close"
                onClick={() => setOpen(false)}
                className="mt-0.5 shrink-0 rounded text-on-surface-variant opacity-70 hover:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
              >
                <Icon icon={X} size="xs" aria-hidden />
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
