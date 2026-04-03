import { useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  BookOpen,
  Lightbulb,
  Package,
  Sparkles,
  X,
  Ban,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Tabs,
} from "@/components/ui";
import {
  useRecommendations,
  useDismissRecommendation,
  useBlockCategory,
  type Recommendation,
} from "@/hooks/use-recommendations";

// ─── Type badge config ──────────────────────────────────────────────────────

const TYPE_CONFIG: Record<
  string,
  { icon: typeof BookOpen; badgeVariant: "primary" | "secondary" | "success"; labelId: string }
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

function RecommendationCard({
  recommendation,
}: {
  recommendation: Recommendation;
}) {
  const intl = useIntl();
  const dismiss = useDismissRecommendation();
  const blockCategory = useBlockCategory();

  const recType = recommendation.recommendation_type ?? "marketplace_content";
  const config = TYPE_CONFIG[recType] ?? DEFAULT_CONFIG;

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
          <Badge variant="default">
            {recommendation.source_signal}
          </Badge>
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
        <p className="type-body-sm text-on-surface-variant mb-2">
          {recommendation.source_label}
        </p>
      </div>

      <div className="flex items-center gap-2 pt-1">
        <Button
          variant="tertiary"
          size="sm"
          leadingIcon={<Icon icon={X} size="xs" aria-hidden />}
          onClick={() => dismiss.mutate(recommendation.id ?? "")}
          loading={dismiss.isPending}
          aria-label={intl.formatMessage(
            { id: "recommendations.dismiss.label" },
            { title: recommendation.target_entity_label },
          )}
        >
          <FormattedMessage id="recommendations.dismiss" />
        </Button>
        <Button
          variant="tertiary"
          size="sm"
          leadingIcon={<Icon icon={Ban} size="xs" aria-hidden />}
          onClick={() => blockCategory.mutate(recType)}
          loading={blockCategory.isPending}
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
    <ul className="flex flex-col gap-3" role="list">
      {recommendations.map((rec) => (
        <li key={rec.id}>
          <RecommendationCard recommendation={rec} />
        </li>
      ))}
    </ul>
  );
}

// ─── Page component ─────────────────────────────────────────────────────────

export function RecommendationsPage() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data, isPending, error } = useRecommendations();

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "recommendations.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const recommendations = data ?? [];

  // ─── Loading state ──────────────────────────────────────────────────────

  if (isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <div className="flex items-center justify-between mb-6">
          <Skeleton height="h-8" width="w-48" />
        </div>
        <Skeleton height="h-10" className="mb-4" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-32" />
          <Skeleton height="h-32" />
          <Skeleton height="h-32" />
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
          className="type-headline-md text-on-surface font-semibold mb-6 outline-none"
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

  // ─── Tabs definition ───────────────────────────────────────────────────

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
          recommendations={recommendations.filter((r) => r.recommendation_type === "marketplace_content")}
        />
      ),
    },
    {
      id: "activity_idea",
      label: intl.formatMessage({ id: "recommendations.filter.activity" }),
      content: (
        <RecommendationList
          recommendations={recommendations.filter((r) => r.recommendation_type === "activity_idea")}
        />
      ),
    },
    {
      id: "reading_suggestion",
      label: intl.formatMessage({ id: "recommendations.filter.resource" }),
      content: (
        <RecommendationList
          recommendations={recommendations.filter((r) => r.recommendation_type === "reading_suggestion")}
        />
      ),
    },
  ];

  // ─── Main render ────────────────────────────────────────────────────────

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
