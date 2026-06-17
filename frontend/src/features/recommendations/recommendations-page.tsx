import { useEffect, useRef, useState } from "react";
import { Link as RouterLink, useSearchParams } from "react-router";
import { FormattedMessage, useIntl } from "react-intl";
import {
  BookOpen,
  MoreHorizontal,
  RotateCcw,
  Settings,
  X,
  Ban,
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
import { ViewingAsSelector } from "@/components/ui/viewing-as-selector";
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
import { useProfile } from "@/features/learner-profile/use-learner-profile";
import {
  REC_TYPES,
  TYPE_CONFIG,
  DEFAULT_REC_CONFIG,
  type RecType,
} from "./rec-type-config";

// ─── Constants ──────────────────────────────────────────────────────────────

const EXPLORATION_FREQUENCIES = ["off", "occasional", "frequent"] as const;

// ─── URL routing helpers ────────────────────────────────────────────────────

function getTargetUrl(type: string, entityId?: string): string | null {
  if (!entityId) return null;
  switch (type) {
    case "marketplace_content":
    case "activity_idea":
    case "reading_suggestion":
      return `/marketplace/listings/${entityId}`;
    case "community_group":
      return `/groups/${entityId}`;
    default:
      return null;
  }
}

// ─── Student profile nudge (used in empty state) ────────────────────────────

function StudentProfileNudge({ studentId, studentName }: { studentId: string; studentName: string }) {
  const profileQuery = useProfile(studentId);
  if (profileQuery.isPending || profileQuery.data) return null;

  return (
    <Card className="flex flex-col gap-3 bg-surface-container-low">
      <p className="type-body-sm text-on-surface-variant">
        <FormattedMessage
          id="recommendations.empty.missingProfile"
          values={{ name: studentName }}
        />
      </p>
      <RouterLink
        to={`/students/${studentId}/learner-profile`}
        className="self-start rounded-button bg-primary px-4 py-2 type-label-md font-medium text-on-primary hover:bg-primary/90 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <FormattedMessage id="recommendations.empty.buildProfile" values={{ name: studentName }} />
      </RouterLink>
    </Card>
  );
}

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
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const dismiss = useDismissRecommendation();
  const block = useBlockRecommendation();
  const undo = useUndoFeedback();

  const recType = (recommendation.recommendation_type ?? "marketplace_content") as RecType;
  const config = TYPE_CONFIG[recType] ?? DEFAULT_REC_CONFIG;
  const id = recommendation.id ?? "";
  const targetUrl = getTargetUrl(recType, recommendation.target_entity_id);

  const showFitBadge =
    recommendation.fit_score !== undefined &&
    recommendation.fit_score >= FIT_BADGE_GATE;

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
      {/* Badge row — 1.5: source_signal removed */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2 flex-wrap">
          <Badge variant={config.badgeVariant}>
            <span className="flex items-center gap-1">
              <Icon icon={config.icon} size="xs" aria-hidden />
              <FormattedMessage id={config.labelId} />
            </span>
          </Badge>
          {recommendation.is_suggestion && (
            <Badge variant="warning">
              <span className="flex items-center gap-1">
                <Icon icon={BookOpen} size="xs" aria-hidden />
                <FormattedMessage id="recommendations.badge.ai" />
              </span>
            </Badge>
          )}
        </div>

        {/* 1.3: Block Category in overflow menu; Dismiss stays as primary card action */}
        <div className="flex items-center gap-1 shrink-0">
          <div className="relative" ref={menuRef}>
            <button
              type="button"
              onClick={() => setMenuOpen((v) => !v)}
              aria-label={intl.formatMessage({ id: "recommendations.card.menu.label" })}
              aria-expanded={menuOpen}
              aria-haspopup="menu"
              className="p-1 rounded text-on-surface-variant hover:bg-surface-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            >
              <Icon icon={MoreHorizontal} size="sm" aria-hidden />
            </button>
            {menuOpen && (
              <div
                role="menu"
                aria-label={intl.formatMessage({ id: "recommendations.card.menu.label" })}
                className="absolute right-0 mt-1 w-48 rounded-lg bg-surface-container shadow-ghost-border shadow-ambient-md z-popover"
              >
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setMenuOpen(false);
                    block.mutate(id);
                  }}
                  disabled={block.isPending}
                  className="w-full text-left px-3 py-2 type-body-sm text-on-surface hover:bg-surface-container-high rounded-lg focus:outline-none focus:bg-surface-container-high"
                  aria-label={intl.formatMessage(
                    { id: "recommendations.blockCategory.label" },
                    { category: recType },
                  )}
                >
                  <span className="flex items-center gap-2">
                    <Icon icon={Ban} size="xs" aria-hidden />
                    <FormattedMessage id="recommendations.blockCategory" />
                  </span>
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 1.4: FitBadge inline with title (Gestalt proximity) */}
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1 mb-1">
          <h3 className="type-title-sm text-on-surface font-medium">
            {targetUrl ? (
              <RouterLink
                to={targetUrl}
                className="hover:text-primary hover:underline focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                aria-label={intl.formatMessage(
                  { id: "recommendations.card.view.label" },
                  { title: recommendation.target_entity_label },
                )}
              >
                {recommendation.target_entity_label}
              </RouterLink>
            ) : (
              recommendation.target_entity_label
            )}
          </h3>
          {showFitBadge && (
            <FitBadge studentName={studentName} whyText={recommendation.fit_why} />
          )}
        </div>
        {recommendation.source_label && (
          <p className="type-body-sm text-on-surface-variant mt-1">
            {recommendation.source_label}
          </p>
        )}
      </div>

      {/* Action row: Dismiss only (Block Category moved to overflow menu) */}
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
      </div>
    </Card>
  );
}

// ─── Recommendation list ────────────────────────────────────────────────────

function RecommendationList({
  recommendations,
  onOpenPreferences,
}: {
  recommendations: Recommendation[];
  onOpenPreferences: () => void;
}) {
  const intl = useIntl();
  const { data: students } = useStudents();
  const studentsById = new Map(students?.map((s) => [s.id ?? "", s]) ?? []);

  // 1.7: Empty state — show profile nudge for students missing profiles
  if (recommendations.length === 0) {
    if (students && students.length > 0) {
      return (
        <div className="flex flex-col gap-3">
          {students.map((s) => (
            <StudentProfileNudge
              key={s.id}
              studentId={s.id ?? ""}
              studentName={s.display_name ?? "your child"}
            />
          ))}
          <EmptyState
            message={intl.formatMessage({ id: "recommendations.empty" })}
            description={intl.formatMessage({ id: "recommendations.empty.description" })}
            action={
              <Button variant="tertiary" size="sm" onClick={onOpenPreferences}>
                <FormattedMessage id="recommendations.empty.adjustPreferences" />
              </Button>
            }
          />
        </div>
      );
    }
    return (
      <EmptyState
        message={intl.formatMessage({ id: "recommendations.empty" })}
        description={intl.formatMessage({ id: "recommendations.empty.description" })}
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
    if (next.length === 0) return;
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
  const [searchParams, setSearchParams] = useSearchParams();
  const [prefsOpen, setPrefsOpen] = useState(false);

  // 1.1: Persist selected student in URL param ?for=<studentId>
  const forStudentId = searchParams.get("for") ?? undefined;

  const { data, isPending, error } = useRecommendations({ forStudentId });

  function handleStudentChange(studentId: string | undefined) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (studentId) {
        next.set("for", studentId);
      } else {
        next.delete("for");
      }
      return next;
    });
  }

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

  // ─── Loading state — 1.6: structured skeleton matching card anatomy ──────

  if (isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <div className="mb-6">
          <Skeleton height="h-8" width="w-48" className="mb-2" />
          <Skeleton height="h-5" width="w-80" />
        </div>
        <div className="flex flex-col gap-3">
          {[1, 2, 3].map((n) => (
            <Card key={n} className="flex flex-col gap-2">
              <Skeleton height="h-4" width="w-24" />
              <Skeleton height="h-5" width="w-3/4" />
              <Skeleton height="h-4" width="w-1/2" />
              <Skeleton height="h-8" width="w-28" className="mt-1" />
            </Card>
          ))}
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

  // 1.2: Tab bar is four homogeneous filters; Preferences is now a collapsible section
  const filterTabs = [
    {
      id: "all",
      label: intl.formatMessage({ id: "recommendations.filter.all" }),
      content: (
        <RecommendationList
          recommendations={recommendations}
          onOpenPreferences={() => setPrefsOpen(true)}
        />
      ),
    },
    {
      id: "marketplace_content",
      label: intl.formatMessage({ id: "recommendations.filter.content" }),
      content: (
        <RecommendationList
          recommendations={recommendations.filter(
            (r) => r.recommendation_type === "marketplace_content",
          )}
          onOpenPreferences={() => setPrefsOpen(true)}
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
          onOpenPreferences={() => setPrefsOpen(true)}
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
          onOpenPreferences={() => setPrefsOpen(true)}
        />
      ),
    },
  ];

  return (
    <div className="mx-auto max-w-2xl">
      {/* Header row with title + Settings trigger */}
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex-1 min-w-0">
          <h1
            ref={headingRef}
            tabIndex={-1}
            className="type-headline-md text-on-surface font-semibold outline-none"
          >
            <FormattedMessage id="recommendations.title" />
          </h1>
          <p className="type-body-md text-on-surface-variant mt-1">
            <FormattedMessage id="recommendations.description" />
          </p>
        </div>
        {/* 1.2: Settings button — right-aligned, opens preferences panel */}
        <Button
          variant="tertiary"
          size="sm"
          leadingIcon={<Icon icon={Settings} size="sm" aria-hidden />}
          aria-expanded={prefsOpen}
          aria-controls="preferences-panel"
          onClick={() => setPrefsOpen((v) => !v)}
        >
          <FormattedMessage id="recommendations.preferences.button" />
        </Button>
      </div>

      {/* 1.2: Collapsible preferences panel below header */}
      {prefsOpen && (
        <section id="preferences-panel" aria-label={intl.formatMessage({ id: "recommendations.preferences.tab" })}>
          <Card className="mb-4">
            <PreferencesPanel />
          </Card>
        </section>
      )}

      {/* 1.1: ViewingAsSelector — persisted in URL ?for=<studentId> */}
      <div className="flex items-center gap-2 mb-4">
        <ViewingAsSelector value={forStudentId} onChange={handleStudentChange} />
      </div>

      <Tabs tabs={filterTabs} defaultTab="all" />
    </div>
  );
}
