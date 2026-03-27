import { type ReactNode } from "react";

type EmptyStateProps = {
  /** Main message */
  message: string;
  /** Optional longer description */
  description?: string;
  /** Illustration or icon slot */
  illustration?: ReactNode;
  /** Call-to-action element (typically a Button) */
  action?: ReactNode;
  className?: string;
};

export function EmptyState({
  message,
  description,
  illustration,
  action,
  className = "",
}: EmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center gap-4 py-16 text-center ${className}`}
    >
      {illustration && (
        <div className="text-on-surface-variant">{illustration}</div>
      )}
      <div className="flex flex-col gap-1">
        <p className="type-title-md text-on-surface">{message}</p>
        {description && (
          <p className="type-body-md text-on-surface-variant max-w-md">
            {description}
          </p>
        )}
      </div>
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
