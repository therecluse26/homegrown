import { type ReactNode } from "react";
import { Button, Card } from "@/components/ui";
import { Lock, RefreshCw, Pencil, Sparkles, BookOpen } from "lucide-react";
import { Icon } from "@/components/ui";
import { Link } from "react-router";

// ─── Dimension display config ────────────────────────────────────────────────

type DimensionCfg = {
  label: string;
  lowLabel: string;
  highLabel: string;
  describe: (v: number) => string;
};

const DIMENSIONS: Record<string, DimensionCfg> = {
  activity_format: {
    label: "Learning Style",
    lowLabel: "Listens & Reads",
    highLabel: "Hands-On",
    describe: (v) =>
      v < 0.4
        ? "Prefers reading, audio, and video over hands-on activities."
        : v < 0.65
          ? "Comfortable with both text-based and hands-on learning."
          : "Learns best through hands-on projects and doing.",
  },
  session_length: {
    label: "Focus Stamina",
    lowLabel: "Short Bursts",
    highLabel: "Deep Dives",
    describe: (v) =>
      v < 0.4
        ? "Thrives with short, frequent pockets of focused work."
        : v < 0.65
          ? "Adapts well to both shorter and longer work sessions."
          : "Prefers long, sustained sessions to go deep on one topic.",
  },
  motivation: {
    label: "What Drives Learning",
    lowLabel: "Mastery",
    highLabel: "Discovery",
    describe: (v) =>
      v < 0.4
        ? "Motivated by mastering a skill and seeing measurable progress."
        : v < 0.65
          ? "Energized by both building skills and exploring new ideas."
          : "Driven by curiosity and the joy of discovering new things.",
  },
  solo_collaborative: {
    label: "Learning Together",
    lowLabel: "Works Alone",
    highLabel: "With Others",
    describe: (v) =>
      v < 0.4
        ? "Concentrates and learns best working independently."
        : v < 0.65
          ? "Comfortable learning both alone and alongside others."
          : "Thrives when learning with peers, siblings, or family.",
  },
  structure: {
    label: "Guidance Preference",
    lowLabel: "Step-by-Step",
    highLabel: "Open-Ended",
    describe: (v) =>
      v < 0.4
        ? "Learns best with clear instructions and well-defined steps."
        : v < 0.65
          ? "Can work with both structured guidance and open exploration."
          : "Flourishes with open-ended questions and self-directed exploration.",
  },
  outdoor_kinesthetic: {
    label: "Movement & Space",
    lowLabel: "Desk-Based",
    highLabel: "On the Move",
    describe: (v) =>
      v < 0.4
        ? "Indoor, desk-based learning environments work well."
        : v < 0.65
          ? "Sometimes benefits from movement and outdoor activities."
          : "Thinks and learns best with physical movement and outdoor time.",
  },
};

const DIMENSION_ORDER = [
  "activity_format",
  "session_length",
  "motivation",
  "solo_collaborative",
  "structure",
  "outdoor_kinesthetic",
] as const;

// ─── Helpers ─────────────────────────────────────────────────────────────────

function formatInterest(slug: string): string {
  return slug
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

// ─── Sub-components ───────────────────────────────────────────────────────────

type DimensionRowProps = {
  cfg: DimensionCfg;
  value: number;
};

function DimensionRow({ cfg, value }: DimensionRowProps): ReactNode {
  const pct = Math.round(value * 100);
  return (
    <div className="flex flex-col gap-1">
      {/* Label */}
      <span className="type-label-md text-on-surface font-medium">{cfg.label}</span>
      {/* Describe text stacked below label, full width (4.1) */}
      <p className="type-body-sm text-on-surface-variant mb-1">{cfg.describe(value)}</p>
      {/* Track + fill */}
      <div className="relative h-2 w-full overflow-hidden rounded-full bg-surface-container-high" role="presentation">
        <div
          className="absolute inset-y-0 left-0 rounded-full bg-primary transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      {/* Pole labels — bumped to type-label-md (4.2) */}
      <div className="flex justify-between gap-2">
        <span className="type-label-md text-on-surface-variant">{cfg.lowLabel}</span>
        <span className="type-label-md text-on-surface-variant">{cfg.highLabel}</span>
      </div>
    </div>
  );
}

// ─── Props ────────────────────────────────────────────────────────────────────

type ProfileSummaryProps = {
  studentName: string;
  /** Pre-composed plain-English description of top 3 preferences */
  summaryText: string;
  interests: string[];
  /** Raw 0.0–1.0 dimension scores from the API response */
  dimensions?: Partial<Record<string, number>>;
  onRetake: () => void;
  onEditInterests: () => void;
};

// ─── Component ────────────────────────────────────────────────────────────────

export function ProfileSummary({
  studentName,
  summaryText,
  interests,
  dimensions,
  onRetake,
  onEditInterests,
}: ProfileSummaryProps): ReactNode {
  const hasDimensions =
    dimensions != null &&
    DIMENSION_ORDER.some((key) => dimensions[key] != null);

  return (
    <div data-context="parent">
      {/* Subordinate caption — the page h1 already provides the heading (3.1) */}
      <p className="type-body-sm text-on-surface-variant mb-6">
        Based on {studentName}'s quiz answers
      </p>

      {/* Summary text */}
      {summaryText && (
        <div className="mb-6 rounded-lg bg-surface-container-low px-4 py-3 shadow-ghost-border">
          <p className="type-body-lg text-on-surface leading-relaxed">{summaryText}</p>
        </div>
      )}

      {/* Learning Dimensions */}
      {hasDimensions && (
        <section aria-labelledby="dimensions-heading" className="mb-6">
          <div className="flex items-center gap-2 mb-4">
            <Icon icon={BookOpen} size="sm" className="text-primary" aria-hidden />
            <h3
              id="dimensions-heading"
              className="type-title-sm text-on-surface font-semibold"
            >
              Learning Dimensions
            </h3>
          </div>
          <div className="flex flex-col gap-5">
            {DIMENSION_ORDER.map((key) => {
              const value = dimensions![key];
              const cfg = DIMENSIONS[key];
              if (value == null || cfg == null) return null;
              return <DimensionRow key={key} cfg={cfg} value={value} />;
            })}
          </div>
        </section>
      )}

      {/* Interests */}
      {interests.length > 0 && (
        <section aria-labelledby="interests-heading" className="mb-6">
          <h3
            id="interests-heading"
            className="type-title-sm text-on-surface font-semibold mb-3"
          >
            Interests
          </h3>
          <div className="flex flex-wrap gap-2" role="list" aria-label="Interest tags">
            {interests.map((interest) => (
              <span
                key={interest}
                role="listitem"
                className="inline-flex items-center rounded-full bg-secondary-container text-on-secondary-container px-3 py-1 type-label-md"
              >
                {formatInterest(interest)}
              </span>
            ))}
          </div>
        </section>
      )}

      {/* How this shapes recommendations — Card for consistent tonal layering (4.3) */}
      <Card
        aria-labelledby="recs-heading"
        role="region"
        className="mb-6 bg-surface-container-low"
      >
        <div className="flex items-center gap-2 mb-2">
          <Icon icon={Sparkles} size="sm" className="text-primary" aria-hidden />
          <h3
            id="recs-heading"
            className="type-title-sm text-on-surface font-semibold"
          >
            How this shapes recommendations
          </h3>
        </div>
        <p className="type-body-sm text-on-surface-variant mb-3 leading-relaxed">
          {studentName}'s profile is used to match content from the Marketplace to{" "}
          {studentName}'s natural learning style. Here's what it does:
        </p>
        <ul className="flex flex-col gap-1.5 list-none">
          <li className="flex items-start gap-2 type-body-sm text-on-surface-variant">
            <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-primary" aria-hidden />
            Content that matches {studentName}'s dimensions earns a{" "}
            <span className="font-medium text-on-surface">Great for {studentName}</span> badge.
          </li>
          <li className="flex items-start gap-2 type-body-sm text-on-surface-variant">
            <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-primary" aria-hidden />
            Recommendations are ranked higher when they align with these preferences and interests.
          </li>
          <li className="flex items-start gap-2 type-body-sm text-on-surface-variant">
            <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-primary" aria-hidden />
            Retaking the quiz updates the profile immediately — recommendations refresh on next visit.
          </li>
        </ul>
        <div className="mt-3">
          <Link
            to="/recommendations"
            className="type-label-md text-primary underline hover:text-primary/80 transition-colors"
          >
            View recommendations →
          </Link>
        </div>
      </Card>

      {/* Privacy note */}
      <div className="mb-8 flex items-start gap-3 rounded-lg bg-surface-container-low px-4 py-3 shadow-ghost-border">
        <Icon
          icon={Lock}
          size="sm"
          className="mt-0.5 shrink-0 text-on-surface-variant"
          aria-hidden
        />
        <p className="type-body-sm text-on-surface-variant">
          This profile is private to your family — never shared or used for ads.
        </p>
      </div>

      {/* Actions — use leadingIcon prop for consistent icon+gap handling (4.4) */}
      <div className="flex flex-wrap gap-3">
        <Button
          type="button"
          variant="secondary"
          leadingIcon={<Icon icon={RefreshCw} size="sm" aria-hidden />}
          onClick={onRetake}
        >
          Retake quiz
        </Button>
        <Button
          type="button"
          variant="tertiary"
          leadingIcon={<Icon icon={Pencil} size="sm" aria-hidden />}
          onClick={onEditInterests}
        >
          Edit interests
        </Button>
      </div>
    </div>
  );
}
