import { useEffect, useRef, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  BookOpen,
  Lightbulb,
  Package,
  Sparkles,
  X,
  Ban,
  RotateCcw,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  EmptyState,
  FitBadge,
  Icon,
  Radio,
  Skeleton,
  Tabs,
} from "@/components/ui";
import { TierGate } from "@/components/common/tier-gate";
import { useAuth } from "@/hooks/use-auth";
import {
  useRecommendations,
  useDismissRecommendation,
  useBlockRecommendation,
  useUndoFeedback,
  useRecommendationPreferences,
  useUpdatePreferences,
  type Recommendation,
} from "@/hooks/use-recommendations";
import { useStudents } from "@/hooks/use-family";

// ─── Constants ──────────────────────────────────────────────────────────────

const REC_TYPES = [
  "marketplace_content",
  "activity_idea",
  "reading_suggestion",
  "community_group",
] as const;

type RecType = (typeof REC_TYPES)[number];

const EXPLORATION_FREQUENCIES = ["off", "occasional", "frequent"] as const;

const TYPE_CONFIG: Record<
  RecType,
  {
    icon: typeof BookOpen;
    badgeVariant: "primary" | "secondary" | "success";
    labelId: string;
  }
> = {
  marketplace_content: {
    icon: BookOpen,
    badgeVariant: "primary",
    labelId: "recommendations.type.content",
  },
  activity_idea: {
    icon: Lightbulb,
    badgeVariant: "secondary",
    labelId: "recommendations.type.activity",
  },
  reading_suggestion: {
    icon: Package,
    badgeVariant: "success",
    labelId: "recommendations.type.resource",
  },
  community_group: {
    icon: Sparkles,
    badgeVariant: "primary",
    labelId: "recommendations.type.community",
  },
};

const DEFAULT_CONFIG = {
  icon: BookOpen,
  badgeVariant: "primary" as const,
  labelId: "recommendations.type.content",
};

// ─── Recommendation card ────────────────────────────────────────────────────

const FIT_BADGE_GATE = 0.60;

function RecommendationCard({
  recommendation,
  studentName,
}: {
  recommendation: Recommendation;
  studentName?: string;
}) {
  const intl = useIntl();
  const [dismissed, setDismissed] = useState(false);
  const dismiss = useDismissRecommendation();
  const block = useBlockRecommendation();
  const undo = useUndoFeedback();

  const recType = (recommendation.recommendation_type ?? "marketplace_content") as RecType;
  const config = TYPE_CONFIG[recType] ?? DEFAULT_CONFIG;
  const id = recommendation.id ?? "";

  // Inline undo prompt shown immediately after dismissal
  if (dismissed) {
    return (
      <Card className="flex items-center justify-between gap-3 bg-surface-container-low">
        <p className="type-body-sm text-on-surface-variant">
          <FormattedMessage id="recommendations.card.dismissed" />
        </p>
        <Button
          variant="tertiary"
          size="sm"
          leadingIcon={<Icon icon={RotateCcw} size="xs" aria-hidden />}
          loading={undo.isPending}
          onClick={() => {
            undo.mutate(id, { onSuccess: () => setDismissed(false) });
          }}
        >
          <FormattedMessage id="recommendations.card.undo" />
        </Button>
      </Card>
    );
  }

  return (
    <Card className="flex flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2 flex-wrap">
          <Badge variant={config.badgeVariant}>
            <span className="flex items-center gap-1">
              <Icon icon={config.icon} size="xs" aria-hidden />
              <FormattedMessage id={config.labelId} />
            </span>
          </Badge>
          {recommendation.source_signal && (
            <Badge variant="default">{recommendation.source_signal}</Badge>
          )}
          {recommendation.is_suggestion && (
            <Badge variant="warning">
              <span className="flex items-center gap-1">
                <Icon icon={Sparkles} size="xs" aria-hidden />
                <FormattedMessage id="recommendations.badge.ai" />
              </span>
            </Badge>
          )}
        </div>
      </div>

      <div className="flex-1 min-w-0">
        <h3 className="type-title-sm text-on-surface font-medium mb-1">
          {recommendation.target_entity_label}
        </h3>
        {recommendation.source_label && (
          <p className="type-body-sm text-on-surface-variant mb-2">
            {recommendation.source_label}
          </p>
        )}
        {recommendation.fit_score !== undefined &&
          recommendation.fit_score >= FIT_BADGE_GATE && (
            <div className="mt-1">
              <FitBadge
                studentName={studentName}
                whyText={recommendation.fit_why}
              />
            </div>
          )}
      </div>

      <div className="flex items-center gap-2 pt-1">
        <Button
          variant="tertiary"
          size="sm"
          leadingIcon={<Icon icon={X} size="xs" aria-hidden />}
          loading={dismiss.isPending}
          onClick={() => {
            dismiss.mutate(id, { onSuccess: () => setDismissed(true) });
          }}
          aria-label={intl.formatMessage(
            { id: "recommendations.card.dismiss.label" },
            { title: recommendation.target_entity_label },
          )}
        >
          <FormattedMessage id="recommendations.dismiss" />
        </Button>
        <Button
          variant="tertiary"
          size="sm"
          leadingIcon={<Icon icon={Ban} size="xs" aria-hidden />}
          loading={block.isPending}
          onClick={() => block.mutate(id)}
          aria-label={intl.formatMessage(
            { id: "recommendations.blockCategory.label" },
            { category: recType },
          )}
        >
          <FormattedMessage id="recommendations.blockCategory" />
        </Button>
      </div>
    </Card>
  );
}

// ─── Recommendation list ────────────────────────────────────────────────────

function RecommendationList({
  recommendations,
}: {
  recommendations: Recommendation[];
}) {
  const intl = useIntl();
  const { data: students } = useStudents();
  const studentsById = new Map(students?.map((s) => [s.id ?? "", s]) ?? []);

  if (recommendations.length === 0) {
    return (
      <EmptyState
        message={intl.formatMessage({ id: "recommendations.empty" })}
        description={intl.formatMessage({
          id: "recommendations.empty.description",
        })}
      />
    );
  }

  return (
    <ul
      className="flex flex-col gap-3"
      role="list"
      aria-label={intl.formatMessage({ id: "recommendations.section.list.label" })}
    >
      {recommendations.map((rec) => (
        <li key={rec.id}>
          <RecommendationCard
            recommendation={rec}
            studentName={studentsById.get(rec.student_id ?? "")?.display_name}
          />
        </li>
      ))}
    </ul>
  );
}

// ─── Preferences panel ──────────────────────────────────────────────────────

function PreferencesPanel() {
  const intl = useIntl();
  const { data: prefs, isPending } = useRecommendationPreferences();
  const update = useUpdatePreferences();

  const enabledTypes: string[] = prefs?.enabled_types ?? [...REC_TYPES];
  const explorationFrequency = prefs?.exploration_frequency ?? "occasional";

  function toggleType(type: string) {
    const next = enabledTypes.includes(type)
      ? enabledTypes.filter((t) => t !== type)
      : [...enabledTypes, type];
    if (next.length === 0) return; // Always keep at least one
    update.mutate({ enabled_types: next, exploration_frequency: explorationFrequency });
  }

  function setFrequency(freq: string) {
    update.mutate({ enabled_types: enabledTypes, exploration_frequency: freq });
  }

  if (isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton height="h-6" width="w-48" />
        <Skeleton height="h-24" />
        <Skeleton height="h-6" width="w-48" />
        <Skeleton height="h-24" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <Card className="flex flex-col gap-4">
        <div>
          <h2 className="type-title-sm text-on-surface font-medium mb-1">
            <FormattedMessage id="recommendations.preferences.types.title" />
          </h2>
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="recommendations.preferences.types.description" />
          </p>
        </div>
        <div className="flex flex-col gap-3">
          {REC_TYPES.map((type) => {
            const cfg = TYPE_CONFIG[type];
            return (
              <Checkbox
                key={type}
                label={intl.formatMessage({ id: cfg.labelId })}
                checked={enabledTypes.includes(type)}
                onChange={() => toggleType(type)}
                disabled={update.isPending}
              />
            );
          })}
        </div>
      </Card>

      <Card className="flex flex-col gap-4">
        <div>
          <h2 className="type-title-sm text-on-surface font-medium mb-1">
            <FormattedMessage id="recommendations.preferences.exploration.title" />
          </h2>
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="recommendations.preferences.exploration.description" />
          </p>
        </div>
        <div className="flex flex-col gap-4">
          {EXPLORATION_FREQUENCIES.map((freq) => (
            <div key={freq} className="flex flex-col gap-1">
              <Radio
                label={intl.formatMessage({
                  id: `recommendations.preferences.exploration.${freq}`,
                })}
                name="exploration_frequency"
                value={freq}
                checked={explorationFrequency === freq}
                onChange={() => setFrequency(freq)}
                disabled={update.isPending}
              />
              <p className="type-body-sm text-on-surface-variant ml-8">
                <FormattedMessage
                  id={`recommendations.preferences.exploration.${freq}.description`}
                />
              </p>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}

// ─── Page component ─────────────────────────────────────────────────────────

export function RecommendationsPage() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier } = useAuth();
  const { data, isPending, error } = useRecommendations();

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "recommendations.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // ─── Premium gate ───────────────────────────────────────────────────────

  if (tier === "free") {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="recommendations.title" />
        </h1>
        <TierGate featureName="Recommendations" requiredTier="premium" />
      </div>
    );
  }

  // ─── Loading state ──────────────────────────────────────────────────────

  if (isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <div className="mb-6">
          <Skeleton height="h-8" width="w-48" className="mb-2" />
          <Skeleton height="h-5" width="w-80" />
        </div>
        <div className="flex flex-col gap-3">
          <Skeleton height="h-28" />
          <Skeleton height="h-28" />
          <Skeleton height="h-28" />
        </div>
      </div>
    );
  }

  // ─── Error state ────────────────────────────────────────────────────────

  if (error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="recommendations.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const recommendations = data ?? [];

  const filterTabs = [
    {
      id: "all",
      label: intl.formatMessage({ id: "recommendations.filter.all" }),
      content: <RecommendationList recommendations={recommendations} />,
    },
    {
      id: "marketplace_content",
      label: intl.formatMessage({ id: "recommendations.filter.content" }),
      content: (
        <RecommendationList
          recommendations={recommendations.filter(
            (r) => r.recommendation_type === "marketplace_content",
          )}
        />
      ),
    },
    {
      id: "activity_idea",
      label: intl.formatMessage({ id: "recommendations.filter.activity" }),
      content: (
        <RecommendationList
          recommendations={recommendations.filter(
            (r) => r.recommendation_type === "activity_idea",
          )}
        />
      ),
    },
    {
      id: "reading_suggestion",
      label: intl.formatMessage({ id: "recommendations.filter.resource" }),
      content: (
        <RecommendationList
          recommendations={recommendations.filter(
            (r) => r.recommendation_type === "reading_suggestion",
          )}
        />
      ),
    },
    {
      id: "preferences",
      label: intl.formatMessage({ id: "recommendations.preferences.tab" }),
      content: <PreferencesPanel />,
    },
  ];

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-6">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-2"
        >
          <FormattedMessage id="recommendations.title" />
        </h1>
        <p className="type-body-md text-on-surface-variant">
          <FormattedMessage id="recommendations.description" />
        </p>
      </div>

      <Tabs tabs={filterTabs} defaultTab="all" />
    </div>
  );
}
