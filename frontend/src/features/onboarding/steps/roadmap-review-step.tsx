import { useIntl, FormattedMessage } from "react-intl";
import { useNavigate } from "react-router";
import {
  Button,
  Card,
  Skeleton,
} from "@/components/ui";
import { Icon } from "@/components/ui";
import { CheckCircle, Circle, BookOpen, Users, Star } from "lucide-react";
import {
  useOnboardingRoadmap,
  useOnboardingRecommendations,
  useOnboardingCommunity,
  useCompleteOnboarding,
  useCompleteRoadmapItem,
} from "@/hooks/use-onboarding";
import type { components } from "@/api/generated/schema";

type RoadmapItem = components["schemas"]["onboard.RoadmapItemResponse"];
type RecommendationItem =
  components["schemas"]["onboard.RecommendationItemResponse"];
type CommunitySuggestion =
  components["schemas"]["onboard.CommunitySuggestionResponse"];

const AGE_GROUP_LABELS: Record<string, string> = {
  "0-4": "Ages 0–4",
  "5-7": "Ages 5–7",
  "8-10": "Ages 8–10",
  "11-13": "Ages 11–13",
  "14-18": "Ages 14–18",
};

type RoadmapReviewStepProps = {
  onBack: () => void;
};

/**
 * Onboarding Step 4 — Roadmap Review.
 * Displays personalized roadmap items, starter curriculum recommendations
 * (grouped by age bracket), and community suggestions.
 * Completing this step finishes onboarding.
 *
 * @see 04-onboard §9.4, §10.3
 */
export function RoadmapReviewStep({ onBack }: RoadmapReviewStepProps) {
  const navigate = useNavigate();
  const complete = useCompleteOnboarding();
  const completeItem = useCompleteRoadmapItem();

  const roadmap = useOnboardingRoadmap();
  const recs = useOnboardingRecommendations();
  const community = useOnboardingCommunity();

  const isDataLoading =
    roadmap.isLoading || recs.isLoading || community.isLoading;

  async function handleComplete() {
    await complete.mutateAsync();
    void navigate("/", { replace: true });
  }

  return (
    <div>
      <h2 className="type-headline-sm text-on-surface font-semibold mb-2">
        <FormattedMessage id="onboarding.roadmap.title" />
      </h2>
      <p className="type-body-md text-on-surface-variant mb-8">
        <FormattedMessage id="onboarding.roadmap.subtitle" />
      </p>

      {isDataLoading ? (
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-20 rounded-xl" />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-8">
          {/* Roadmap items */}
          {(roadmap.data?.groups?.length ?? 0) > 0 && (
            <section aria-labelledby="roadmap-heading">
              <h3
                id="roadmap-heading"
                className="type-title-md text-on-surface font-semibold mb-4"
              >
                <FormattedMessage id="onboarding.roadmap.section.roadmap" />
              </h3>
              <div className="flex flex-col gap-3">
                {roadmap.data?.groups?.flatMap((group) =>
                  (group.items ?? []).map((item: RoadmapItem) => (
                    <RoadmapItemRow
                      key={item.id}
                      item={item}
                      onComplete={async (id) => {
                        await completeItem.mutateAsync(id);
                      }}
                    />
                  )),
                )}
              </div>
            </section>
          )}

          {/* Starter recommendations */}
          {(recs.data?.groups?.length ?? 0) > 0 && (
            <section aria-labelledby="recs-heading">
              <h3
                id="recs-heading"
                className="type-title-md text-on-surface font-semibold mb-1"
              >
                <FormattedMessage id="onboarding.roadmap.section.recommendations" />
              </h3>
              <p className="type-body-sm text-on-surface-variant mb-4">
                <FormattedMessage id="onboarding.roadmap.section.recommendations.desc" />
              </p>
              {recs.data?.groups?.map((group) => (
                <div key={group.age_group ?? "all"} className="mb-6">
                  {group.age_group && (
                    <p className="type-label-md text-on-surface-variant mb-3 uppercase tracking-wide">
                      {AGE_GROUP_LABELS[group.age_group] ?? group.age_group}
                    </p>
                  )}
                  <div className="flex flex-col gap-3">
                    {(group.items ?? []).map((item: RecommendationItem) => (
                      <RecommendationRow key={item.id} item={item} />
                    ))}
                  </div>
                </div>
              ))}
            </section>
          )}

          {/* Community suggestions */}
          {(community.data?.items?.length ?? 0) > 0 && (
            <section aria-labelledby="community-heading">
              <h3
                id="community-heading"
                className="type-title-md text-on-surface font-semibold mb-4"
              >
                <FormattedMessage id="onboarding.roadmap.section.community" />
              </h3>
              <div className="flex flex-col gap-3">
                {(community.data?.items ?? []).map(
                  (item: CommunitySuggestion) => (
                    <CommunitySuggestionRow key={item.id} item={item} />
                  ),
                )}
              </div>
            </section>
          )}
        </div>
      )}

      {complete.error && (
        <div
          role="alert"
          className="mt-6 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
        >
          <FormattedMessage id="error.generic" />
        </div>
      )}

      <div className="flex gap-3 mt-8">
        <Button type="button" variant="tertiary" onClick={onBack}>
          <FormattedMessage id="common.back" />
        </Button>
        <Button
          type="button"
          variant="gradient"
          onClick={handleComplete}
          loading={complete.isPending}
          disabled={complete.isPending || isDataLoading}
          className="flex-1"
        >
          <FormattedMessage id="onboarding.roadmap.complete" />
        </Button>
      </div>
    </div>
  );
}

// ─── Sub-components ──────────────────────────────────────────────────────────

function RoadmapItemRow({
  item,
  onComplete,
}: {
  item: RoadmapItem;
  onComplete: (id: string) => Promise<void>;
}) {
  const intl = useIntl();
  const isCompleted = item.is_completed ?? false;

  return (
    <Card className="flex items-start gap-3">
      <button
        type="button"
        onClick={() => !isCompleted && void onComplete(item.id ?? "")}
        disabled={isCompleted}
        aria-label={
          isCompleted
            ? intl.formatMessage(
                { id: "onboarding.roadmap.item.completed" },
                { title: item.title },
              )
            : intl.formatMessage(
                { id: "onboarding.roadmap.item.markDone" },
                { title: item.title },
              )
        }
        className="mt-0.5 shrink-0 text-primary disabled:text-on-surface-variant transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <Icon
          icon={isCompleted ? CheckCircle : Circle}
          size="sm"
          aria-hidden
        />
      </button>
      <div>
        <p
          className={`type-body-md font-medium ${isCompleted ? "line-through text-on-surface-variant" : "text-on-surface"}`}
        >
          {item.title}
        </p>
        {item.description && (
          <p className="type-body-sm text-on-surface-variant mt-0.5">
            {item.description}
          </p>
        )}
        {item.link_url && !isCompleted && (
          <a
            href={item.link_url}
            target="_blank"
            rel="noopener noreferrer"
            className="type-label-md text-primary hover:underline mt-1 inline-block"
          >
            <FormattedMessage id="onboarding.roadmap.item.learnMore" />
          </a>
        )}
      </div>
    </Card>
  );
}

function RecommendationRow({ item }: { item: RecommendationItem }) {
  return (
    <Card className="flex items-start gap-3">
      <div className="mt-0.5 shrink-0 flex h-8 w-8 items-center justify-center rounded-lg bg-tertiary-fixed text-on-surface">
        <Icon icon={BookOpen} size="xs" aria-hidden />
      </div>
      <div>
        <p className="type-body-md font-medium text-on-surface">{item.title}</p>
        {item.description && (
          <p className="type-body-sm text-on-surface-variant mt-0.5">
            {item.description}
          </p>
        )}
      </div>
    </Card>
  );
}

function CommunitySuggestionRow({ item }: { item: CommunitySuggestion }) {
  const icon =
    item.suggestion_type === "group"
      ? Users
      : item.suggestion_type === "mentor"
        ? Star
        : Users;

  return (
    <Card className="flex items-start gap-3">
      <div className="mt-0.5 shrink-0 flex h-8 w-8 items-center justify-center rounded-lg bg-secondary-container text-on-secondary-container">
        <Icon icon={icon} size="xs" aria-hidden />
      </div>
      <div>
        <p className="type-body-md font-medium text-on-surface">{item.title}</p>
        {item.description && (
          <p className="type-body-sm text-on-surface-variant mt-0.5">
            {item.description}
          </p>
        )}
      </div>
    </Card>
  );
}
