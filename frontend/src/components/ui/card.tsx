import { forwardRef, type HTMLAttributes, type ReactNode } from "react";

type CardProps = HTMLAttributes<HTMLDivElement> & {
  /** Whether the card responds to hover/click */
  interactive?: boolean;
  children: ReactNode;
};

export const Card = forwardRef<HTMLDivElement, CardProps>(function Card(
  { interactive = false, children, className = "", ...props },
  ref,
) {
  return (
    <div
      ref={ref}
      className={`bg-surface-container-lowest p-card-padding parent:rounded-lg student:rounded-xl transition-all ${
        interactive
          ? "cursor-pointer hover:shadow-ambient-sm active:bg-surface-container-low"
          : ""
      } ${className}`}
      {...props}
    >
      {children}
    </div>
  );
});
