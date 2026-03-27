type ProgressBarProps = {
  /** Progress value between 0 and 100 */
  value: number;
  /** Accessible label */
  label?: string;
  className?: string;
};

export function ProgressBar({
  value,
  label = "Progress",
  className = "",
}: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value));

  return (
    <div className={`w-full ${className}`}>
      <div
        role="progressbar"
        aria-valuenow={clamped}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={label}
        className="h-2 w-full overflow-hidden rounded-full bg-tertiary-fixed"
      >
        <div
          className="h-full rounded-full bg-primary transition-all duration-[var(--duration-normal)] ease-[var(--ease-decelerate)]"
          style={{ width: `${String(clamped)}%` }}
        />
      </div>
    </div>
  );
}
