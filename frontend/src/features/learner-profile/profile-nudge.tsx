import { type ReactNode } from "react";
import { BarChart2, X } from "lucide-react";
import { Icon } from "@/components/ui/icon";
import { Button } from "@/components/ui";

type ProfileNudgeProps = {
  studentName: string;
  onStart: () => void;
  onDismiss: () => void;
};

/**
 * Dismissible dashboard nudge for families who skipped the learner-profile
 * quiz during onboarding. Shown at most once per session after dismissal.
 *
 * Confirm with CTO whether this is in v0 scope before wiring to dashboard.
 */
export function ProfileNudge({
  studentName,
  onStart,
  onDismiss,
}: ProfileNudgeProps): ReactNode {
  return (
    <div
      role="status"
      aria-label={`Complete ${studentName}'s learning profile`}
      className="relative rounded-lg bg-tertiary-fixed/30 px-4 py-4 shadow-ghost-border"
    >
      {/* Dismiss */}
      <button
        type="button"
        aria-label="Dismiss"
        onClick={onDismiss}
        className="absolute right-3 top-3 rounded p-1 text-on-surface-variant hover:text-on-surface hover:bg-surface-container-low transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <Icon icon={X} size="xs" aria-hidden />
      </button>

      <div className="flex items-start gap-3 pr-6">
        <span
          aria-hidden
          className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-tertiary/10 text-tertiary"
        >
          <Icon icon={BarChart2} size="sm" />
        </span>

        <div className="flex-1">
          <p className="type-title-sm text-on-surface font-semibold mb-1">
            Personalize {studentName}'s recommendations
          </p>
          <p className="type-body-sm text-on-surface-variant mb-3">
            Answer a few quick questions to help us suggest content that fits
            how {studentName} likes to learn.
          </p>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={onStart}
          >
            Start learning profile →
          </Button>
        </div>
      </div>
    </div>
  );
}
