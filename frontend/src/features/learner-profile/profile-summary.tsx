import { type ReactNode } from "react";
import { Card, Button } from "@/components/ui";
import { Lock, RefreshCw, Pencil } from "lucide-react";
import { Icon } from "@/components/ui";

type ProfileSummaryProps = {
  studentName: string;
  /** Pre-composed plain-English description of top 3 preferences */
  summaryText: string;
  interests: string[];
  onRetake: () => void;
  onEditInterests: () => void;
};

export function ProfileSummary({
  studentName,
  summaryText,
  interests,
  onRetake,
  onEditInterests,
}: ProfileSummaryProps): ReactNode {
  return (
    <div data-context="parent">
      {/* Heading */}
      <div className="mb-6">
        <h2 className="type-headline-sm text-on-surface font-semibold mb-1">
          {studentName}'s Learning Profile
        </h2>
        <p className="type-body-sm text-on-surface-variant">
          Based on {studentName}'s answers
        </p>
      </div>

      {/* Summary card */}
      <Card className="mb-4">
        <p className="type-body-lg text-on-surface leading-relaxed">
          {summaryText}
        </p>
      </Card>

      {/* Interests */}
      {interests.length > 0 && (
        <div className="mb-6">
          <h3 className="type-title-sm text-on-surface font-semibold mb-3">
            Interests
          </h3>
          <div className="flex flex-wrap gap-2" role="list" aria-label="Interest tags">
            {interests.map((interest) => (
              <span
                key={interest}
                role="listitem"
                className="inline-flex items-center rounded-full bg-secondary-container text-on-secondary-container px-3 py-1 type-label-md"
              >
                {interest}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Privacy note */}
      <div className="mb-8 flex items-start gap-3 rounded-lg bg-surface-container-low px-4 py-3 shadow-ghost-border">
        <Icon
          icon={Lock}
          size="sm"
          className="mt-0.5 shrink-0 text-on-surface-variant"
          aria-hidden
        />
        <p className="type-body-sm text-on-surface-variant">
          This profile is only used to suggest content — never shared or used
          for ads.
        </p>
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-3">
        <Button
          type="button"
          variant="secondary"
          onClick={onRetake}
        >
          <Icon icon={RefreshCw} size="sm" aria-hidden className="mr-1.5" />
          Retake quiz
        </Button>
        <Button
          type="button"
          variant="tertiary"
          onClick={onEditInterests}
        >
          <Icon icon={Pencil} size="sm" aria-hidden className="mr-1.5" />
          Edit interests
        </Button>
      </div>
    </div>
  );
}
