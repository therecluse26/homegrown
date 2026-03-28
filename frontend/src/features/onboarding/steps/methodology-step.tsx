import { useState } from "react";
import { useIntl, FormattedMessage } from "react-intl";
import {
  Button,
  FormField,
  Input,
  Card,
  Skeleton,
} from "@/components/ui";
import { Icon } from "@/components/ui";
import { ExternalLink, RefreshCw } from "lucide-react";
import { useSelectMethodology, useImportQuiz } from "@/hooks/use-onboarding";
import { useMethodologyList } from "@/hooks/use-methodologies";
import { MethodologyCard } from "@/components/common/methodology-card";
import type { components } from "@/api/generated/schema";

type MethodologySummary =
  components["schemas"]["method.MethodologySummaryResponse"];
type MethodologyID = components["schemas"]["method.MethodologyID"];
type QuizImportResponse = components["schemas"]["onboard.QuizImportResponse"];

type MethodologyPath = "quiz_informed" | "exploration" | "skip";

type MethodologyStepProps = {
  onNext: () => void;
  onBack: () => void;
};

/**
 * Onboarding Step 3 — Methodology.
 * Three paths:
 * - quiz_informed: import result from the public quiz by share_id
 * - exploration: browse and select from the methodology list
 * - skip: proceed without a methodology (backend assigns default)
 *
 * @see 04-onboard §9.2–§9.3
 * @see ARCHITECTURE §2.4 (quiz lives on Astro public site; result imported via share_id)
 */
export function MethodologyStep({ onNext, onBack }: MethodologyStepProps) {
  const intl = useIntl();
  const selectMethodology = useSelectMethodology();
  const importQuiz = useImportQuiz();
  const { data: methodologies, isLoading: methodsLoading } =
    useMethodologyList();

  const [path, setPath] = useState<MethodologyPath | null>(null);

  // Quiz path state
  const [shareId, setShareId] = useState("");
  const [quizResult, setQuizResult] = useState<QuizImportResponse | null>(null);
  const [quizError, setQuizError] = useState("");

  // Exploration path state
  const [selectedSlug, setSelectedSlug] = useState<MethodologyID | string>("");

  async function handleQuizImport(e: React.FormEvent) {
    e.preventDefault();
    setQuizError("");
    const raw = shareId.trim();
    if (!raw) return;

    // Accept either a plain share_id or a full URL containing one
    const id = extractShareId(raw);
    try {
      const result = await importQuiz.mutateAsync({ share_id: id });
      setQuizResult(result);
      if (result.suggested_primary_slug) {
        setSelectedSlug(result.suggested_primary_slug);
      }
    } catch {
      setQuizError(intl.formatMessage({ id: "onboarding.methodology.quiz.error" }));
    }
  }

  async function handleSubmitQuizInformed() {
    if (!selectedSlug) return;
    await selectMethodology.mutateAsync({
      methodology_path: "quiz_informed",
      primary_methodology_slug: selectedSlug,
    });
    onNext();
  }

  async function handleSubmitExploration() {
    if (!selectedSlug) return;
    await selectMethodology.mutateAsync({
      methodology_path: "exploration",
      primary_methodology_slug: selectedSlug,
    });
    onNext();
  }

  async function handleSkip() {
    await selectMethodology.mutateAsync({
      methodology_path: "skip",
      // Backend ignores/overrides slug for skip path [04-onboard §9.3]
      primary_methodology_slug: "traditional",
    });
    onNext();
  }

  // Path selector
  if (!path) {
    return (
      <div>
        <h2 className="type-headline-sm text-on-surface font-semibold mb-2">
          <FormattedMessage id="onboarding.methodology.title" />
        </h2>
        <p className="type-body-md text-on-surface-variant mb-8">
          <FormattedMessage id="onboarding.methodology.subtitle" />
        </p>

        <div className="flex flex-col gap-4 mb-8">
          <Card
            interactive
            onClick={() => setPath("quiz_informed")}
            className="cursor-pointer"
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                setPath("quiz_informed");
              }
            }}
          >
            <h3 className="type-title-sm text-on-surface font-semibold mb-1">
              <FormattedMessage id="onboarding.methodology.path.quiz.title" />
            </h3>
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="onboarding.methodology.path.quiz.desc" />
            </p>
          </Card>

          <Card
            interactive
            onClick={() => setPath("exploration")}
            className="cursor-pointer"
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                setPath("exploration");
              }
            }}
          >
            <h3 className="type-title-sm text-on-surface font-semibold mb-1">
              <FormattedMessage id="onboarding.methodology.path.explore.title" />
            </h3>
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="onboarding.methodology.path.explore.desc" />
            </p>
          </Card>
        </div>

        <div className="flex gap-3">
          <Button type="button" variant="tertiary" onClick={onBack}>
            <FormattedMessage id="common.back" />
          </Button>
          <Button
            type="button"
            variant="secondary"
            onClick={handleSkip}
            loading={selectMethodology.isPending}
            disabled={selectMethodology.isPending}
            className="flex-1"
          >
            <FormattedMessage id="onboarding.methodology.skip" />
          </Button>
        </div>
      </div>
    );
  }

  // Quiz path
  if (path === "quiz_informed") {
    return (
      <div>
        <button
          type="button"
          onClick={() => {
            setPath(null);
            setShareId("");
            setQuizResult(null);
            setQuizError("");
          }}
          className="mb-6 flex items-center gap-2 type-label-md text-on-surface-variant hover:text-on-surface transition-colors"
        >
          ← <FormattedMessage id="common.back" />
        </button>

        <h2 className="type-headline-sm text-on-surface font-semibold mb-2">
          <FormattedMessage id="onboarding.methodology.quiz.title" />
        </h2>
        <p className="type-body-md text-on-surface-variant mb-6">
          <FormattedMessage id="onboarding.methodology.quiz.subtitle" />
        </p>

        {/* Link to take the quiz */}
        <Card className="mb-6 bg-surface-container-low flex items-start gap-3">
          <Icon icon={ExternalLink} size="sm" className="mt-0.5 text-primary shrink-0" aria-hidden />
          <div>
            <p className="type-body-sm text-on-surface">
              <FormattedMessage
                id="onboarding.methodology.quiz.takeFirst"
                values={{
                  link: (
                    <a
                      href="https://homegrownacademy.com/quiz"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline font-medium"
                    >
                      <FormattedMessage id="onboarding.methodology.quiz.takeLink" />
                    </a>
                  ),
                }}
              />
            </p>
          </div>
        </Card>

        {!quizResult ? (
          <form onSubmit={handleQuizImport} noValidate className="flex flex-col gap-4">
            <FormField
              label={intl.formatMessage({ id: "onboarding.methodology.quiz.shareId" })}
              hint={intl.formatMessage({ id: "onboarding.methodology.quiz.shareId.hint" })}
              error={quizError}
            >
              {({ id, errorId, hintId }) => (
                <Input
                  id={id}
                  value={shareId}
                  onChange={(e) => {
                    setShareId(e.target.value);
                    setQuizError("");
                  }}
                  placeholder="abc123"
                  aria-describedby={errorId ?? hintId}
                  error={!!quizError}
                  autoFocus
                />
              )}
            </FormField>

            <Button
              type="submit"
              variant="primary"
              loading={importQuiz.isPending}
              disabled={!shareId.trim() || importQuiz.isPending}
              className="w-full"
            >
              <FormattedMessage id="onboarding.methodology.quiz.import" />
            </Button>
          </form>
        ) : (
          <div className="flex flex-col gap-4">
            {/* Quiz result */}
            <div
              role="status"
              className="rounded-xl bg-secondary-container p-4"
            >
              <p className="type-label-md text-on-secondary-container mb-1">
                <FormattedMessage id="onboarding.methodology.quiz.result.match" />
              </p>
              <p className="type-title-lg text-on-surface font-semibold">
                {quizResult.methodology_recommendations?.[0]?.methodology_name ??
                  quizResult.suggested_primary_slug}
              </p>
              {quizResult.methodology_recommendations?.[0]?.score_percentage !==
                undefined && (
                <p className="type-body-sm text-on-surface-variant mt-1">
                  <FormattedMessage
                    id="onboarding.methodology.quiz.result.confidence"
                    values={{
                      score: Math.round(
                        (quizResult.methodology_recommendations[0]
                          .score_percentage ?? 0) * 100,
                      ),
                    }}
                  />
                </p>
              )}
            </div>

            {/* Other recommendations */}
            {(quizResult.methodology_recommendations?.length ?? 0) > 1 && (
              <div>
                <p className="type-label-md text-on-surface-variant mb-3">
                  <FormattedMessage id="onboarding.methodology.quiz.result.others" />
                </p>
                <div className="flex flex-col gap-2">
                  {quizResult.methodology_recommendations
                    ?.slice(0, 3)
                    .map((rec) => {
                      const isSelected =
                        selectedSlug === rec.methodology_slug;
                      return (
                        <button
                          key={rec.methodology_slug}
                          type="button"
                          onClick={() =>
                            setSelectedSlug(rec.methodology_slug ?? "")
                          }
                          className={`flex items-center justify-between rounded-button px-4 py-2.5 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${
                            isSelected
                              ? "bg-primary text-on-primary"
                              : "bg-surface-container hover:bg-surface-container-high text-on-surface"
                          }`}
                          aria-pressed={isSelected}
                        >
                          <span className="type-body-md">
                            {rec.methodology_name}
                          </span>
                          {rec.score_percentage !== undefined && (
                            <span className="type-label-sm opacity-70">
                              {Math.round(rec.score_percentage * 100)}%
                            </span>
                          )}
                        </button>
                      );
                    })}
                </div>
              </div>
            )}

            <button
              type="button"
              onClick={() => {
                setQuizResult(null);
                setShareId("");
              }}
              className="flex items-center gap-2 type-label-md text-on-surface-variant hover:text-on-surface transition-colors self-start"
            >
              <Icon icon={RefreshCw} size="xs" aria-hidden />
              <FormattedMessage id="onboarding.methodology.quiz.tryAnother" />
            </button>

            {selectMethodology.error && (
              <div
                role="alert"
                className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
              >
                <FormattedMessage id="error.generic" />
              </div>
            )}

            <div className="flex gap-3 pt-2">
              <Button
                type="button"
                variant="tertiary"
                onClick={() => setPath(null)}
              >
                <FormattedMessage id="common.back" />
              </Button>
              <Button
                type="button"
                variant="primary"
                onClick={handleSubmitQuizInformed}
                loading={selectMethodology.isPending}
                disabled={!selectedSlug || selectMethodology.isPending}
                className="flex-1"
              >
                <FormattedMessage id="onboarding.methodology.confirmSelection" />
              </Button>
            </div>
          </div>
        )}
      </div>
    );
  }

  // Exploration path
  return (
    <div>
      <button
        type="button"
        onClick={() => {
          setPath(null);
          setSelectedSlug("");
        }}
        className="mb-6 flex items-center gap-2 type-label-md text-on-surface-variant hover:text-on-surface transition-colors"
      >
        ← <FormattedMessage id="common.back" />
      </button>

      <h2 className="type-headline-sm text-on-surface font-semibold mb-2">
        <FormattedMessage id="onboarding.methodology.explore.title" />
      </h2>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="onboarding.methodology.explore.subtitle" />
      </p>

      {methodsLoading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-32 rounded-xl" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 mb-6">
          {(methodologies ?? []).map((m: MethodologySummary) => (
            <MethodologyCard
              key={m.slug}
              methodology={m}
              selected={selectedSlug === m.slug}
              onClick={() => setSelectedSlug(m.slug ?? "")}
            />
          ))}
        </div>
      )}

      {selectMethodology.error && (
        <div
          role="alert"
          className="mb-4 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
        >
          <FormattedMessage id="error.generic" />
        </div>
      )}

      <div className="flex gap-3">
        <Button type="button" variant="tertiary" onClick={() => setPath(null)}>
          <FormattedMessage id="common.back" />
        </Button>
        <Button
          type="button"
          variant="primary"
          onClick={handleSubmitExploration}
          loading={selectMethodology.isPending}
          disabled={!selectedSlug || selectMethodology.isPending}
          className="flex-1"
        >
          <FormattedMessage id="onboarding.methodology.confirmSelection" />
        </Button>
      </div>
    </div>
  );
}

/** Extract a bare share_id from either a plain ID or a URL containing one. */
function extractShareId(input: string): string {
  try {
    const url = new URL(input);
    const parts = url.pathname.split("/").filter(Boolean);
    return parts[parts.length - 1] ?? input;
  } catch {
    return input;
  }
}
