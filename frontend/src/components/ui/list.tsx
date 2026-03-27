import { type ReactNode } from "react";

type ListVariant = "gap" | "striped";

type ListProps = {
  /** "gap" = items separated by spacing-list-gap; "striped" = alternating bg */
  variant?: ListVariant;
  children: ReactNode;
  className?: string;
};

const variantClasses: Record<ListVariant, string> = {
  gap: "flex flex-col gap-[var(--spacing-list-gap)]",
  striped: "flex flex-col [&>*:nth-child(even)]:bg-surface-container-low",
};

export function List({ variant = "gap", children, className = "" }: ListProps) {
  return (
    <div role="list" className={`${variantClasses[variant]} ${className}`}>
      {children}
    </div>
  );
}

type ListItemProps = {
  children: ReactNode;
  className?: string;
};

export function ListItem({ children, className = "" }: ListItemProps) {
  return (
    <div role="listitem" className={`${className}`}>
      {children}
    </div>
  );
}
