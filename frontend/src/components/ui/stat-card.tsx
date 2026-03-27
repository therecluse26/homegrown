import { type ReactNode } from "react";

type StatCardProps = {
  /** The metric value to display prominently */
  value: string | number;
  /** Description of what the value represents */
  label: string;
  /** Optional trend indicator (+5%, -2%, etc.) */
  trend?: { direction: "up" | "down" | "neutral"; label: string };
  /** Optional icon slot */
  icon?: ReactNode;
  className?: string;
};

const trendClasses = {
  up: "text-success",
  down: "text-error",
  neutral: "text-on-surface-variant",
} as const;

export function StatCard({
  value,
  label,
  trend,
  icon,
  className = "",
}: StatCardProps) {
  return (
    <div
      className={`bg-surface-container-lowest p-spacing-card-padding parent:rounded-lg student:rounded-xl ${className}`}
    >
      <div className="flex items-start justify-between">
        <div className="flex flex-col gap-1">
          <p className="type-label-md text-on-surface-variant">{label}</p>
          <p className="type-headline-md text-on-surface">{value}</p>
          {trend && (
            <p className={`type-label-sm ${trendClasses[trend.direction]}`}>
              {trend.label}
            </p>
          )}
        </div>
        {icon && (
          <div className="text-primary">{icon}</div>
        )}
      </div>
    </div>
  );
}
