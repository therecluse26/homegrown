import { useEffect, useRef } from "react";

type PageTitleProps = {
  /** The page title — set as document.title and rendered as h1 */
  title: string;
  /** Optional subtitle below the h1 */
  subtitle?: string;
  className?: string;
};

export function PageTitle({ title, subtitle, className = "" }: PageTitleProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    document.title = `${title} — Homegrown Academy`;
  }, [title]);

  // Focus heading on mount for screen reader announcement on route transitions
  useEffect(() => {
    headingRef.current?.focus();
  }, [title]);

  return (
    <div className={className}>
      <h1
        ref={headingRef}
        className="type-headline-lg text-on-surface outline-none"
        tabIndex={-1}
      >
        {title}
      </h1>
      {subtitle && (
        <p className="mt-1 type-body-lg text-on-surface-variant">{subtitle}</p>
      )}
    </div>
  );
}
