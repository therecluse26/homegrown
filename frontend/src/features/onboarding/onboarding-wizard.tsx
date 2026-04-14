import { useState, useCallback } from "react";
import { useNavigate } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Spinner, Button } from "@/components/ui";
import { Icon } from "@/components/ui";
import { Check } from "lucide-react";
import { PageTitle } from "@/components/common";
import { useOnboardingProgress, useSkipOnboarding } from "@/hooks/use-onboarding";
import { FamilyProfileStep } from "./steps/family-profile-step";
import { ChildrenStep } from "./steps/children-step";
import { MethodologyStep } from "./steps/methodology-step";
import { RoadmapReviewStep } from "./steps/roadmap-review-step";
import type { components } from "@/api/generated/schema";

type WizardStep = components["schemas"]["onboard.WizardStep"];

const STEPS: WizardStep[] = [
  "family_profile",
  "children",
  "methodology",
  "roadmap_review",
];

const STEP_LABELS: Record<WizardStep, string> = {
  family_profile: "onboarding.step.familyProfile",
  children: "onboarding.step.children",
  methodology: "onboarding.step.methodology",
  roadmap_review: "onboarding.step.roadmap",
};

/**
 * Derives the step to display from completed_steps. This is intentionally
 * independent of the server's current_step because:
 * - The children step can be skipped without an explicit API call, causing
 *   current_step to remain "children" even after methodology is selected.
 * - We want forward-progress semantics: show the furthest reachable step.
 *
 * @see specs/TODO-frontend.md Phase 6
 */
function deriveDisplayStep(completedSteps: WizardStep[] | undefined): WizardStep {
  const done = new Set(completedSteps ?? []);
  if (done.has("methodology")) return "roadmap_review";
  if (done.has("family_profile")) return "children";
  return "family_profile";
}

/**
 * Onboarding Wizard — 4-step guided flow for new families.
 * Steps: Family Profile → Children → Methodology → Roadmap Review
 *
 * Navigation is tracked locally. The backend's current_step is only used
 * for initial hydration. This allows the children step to be optional
 * without requiring a dedicated "skip children" API endpoint.
 *
 * @see specs/TODO-frontend.md Phase 6
 * @see 04-onboard §9
 */
export function OnboardingWizard() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: progress, isLoading } = useOnboardingProgress();
  const skipOnboarding = useSkipOnboarding();

  const [activeStep, setActiveStep] = useState<WizardStep | null>(null);

  // Initialize activeStep once from server data
  const resolvedStep: WizardStep =
    activeStep ??
    (progress ? deriveDisplayStep(progress.completed_steps) : "family_profile");

  const stepIndex = STEPS.indexOf(resolvedStep);
  const progressPercent = Math.round(((stepIndex) / STEPS.length) * 100);

  const completedSteps = new Set(progress?.completed_steps ?? []);

  const goToStep = useCallback((step: WizardStep) => {
    setActiveStep(step);
  }, []);

  const handleNext = useCallback(() => {
    const nextIndex = stepIndex + 1;
    if (nextIndex < STEPS.length) {
      setActiveStep(STEPS[nextIndex] as WizardStep);
    }
  }, [stepIndex]);

  const handleBack = useCallback(() => {
    const prevIndex = stepIndex - 1;
    if (prevIndex >= 0) {
      setActiveStep(STEPS[prevIndex] as WizardStep);
    }
  }, [stepIndex]);

  async function handleSkipAll() {
    try {
      await skipOnboarding.mutateAsync();
    } catch {
      // 409 = already completed — treat as success
    }
    void navigate("/", { replace: true });
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div data-context="parent">
      <PageTitle
        title={intl.formatMessage({ id: "onboarding.title" })}
      />

      {/* Step indicator */}
      <nav
        aria-label={intl.formatMessage({ id: "onboarding.steps.nav.label" })}
        className="mb-8"
      >
        <ol className="flex items-center gap-2" role="list">
          {STEPS.map((step, i) => {
            const isDone = completedSteps.has(step);
            const isActive = step === resolvedStep;
            const isReachable =
              isDone ||
              isActive ||
              (i > 0 && completedSteps.has(STEPS[i - 1] as WizardStep));

            return (
              <li key={step} className="flex items-center gap-2 flex-1">
                {/* Step node */}
                <button
                  type="button"
                  onClick={() => isReachable && goToStep(step)}
                  disabled={!isReachable}
                  aria-current={isActive ? "step" : undefined}
                  aria-label={`${intl.formatMessage({ id: STEP_LABELS[step] })}${isDone ? " — completed" : isActive ? " — current" : ""}`}
                  className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full type-label-md transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${
                    isDone
                      ? "bg-primary text-on-primary"
                      : isActive
                        ? "bg-primary text-on-primary ring-4 ring-primary/20"
                        : "bg-surface-container-high text-on-surface-variant"
                  } ${isReachable ? "cursor-pointer" : "cursor-default"}`}
                >
                  {isDone ? (
                    <Icon icon={Check} size="xs" aria-hidden />
                  ) : (
                    <span aria-hidden>{i + 1}</span>
                  )}
                </button>

                {/* Step label (hidden on mobile except active) */}
                <span
                  className={`type-label-sm hidden sm:block truncate ${
                    isActive
                      ? "text-on-surface font-semibold"
                      : "text-on-surface-variant"
                  }`}
                >
                  {intl.formatMessage({ id: STEP_LABELS[step] })}
                </span>

                {/* Connector */}
                {i < STEPS.length - 1 && (
                  <div
                    aria-hidden
                    className={`h-0.5 flex-1 rounded-full transition-colors ${
                      completedSteps.has(step)
                        ? "bg-primary"
                        : "bg-surface-container-high"
                    }`}
                  />
                )}
              </li>
            );
          })}
        </ol>

        {/* Progress bar for mobile */}
        <div
          aria-hidden
          className="mt-3 h-1 rounded-full bg-surface-container-high sm:hidden"
        >
          <div
            className="h-full rounded-full bg-primary transition-all duration-300"
            style={{ width: `${progressPercent}%` }}
          />
        </div>
      </nav>

      {/* Skip all button */}
      <div className="flex justify-end mb-6">
        <Button
          variant="tertiary"
          size="sm"
          onClick={handleSkipAll}
          loading={skipOnboarding.isPending}
          disabled={skipOnboarding.isPending}
        >
          <FormattedMessage id="onboarding.skipAll" />
        </Button>
      </div>

      {/* Active step content */}
      <div role="main" aria-live="polite" aria-atomic="false">
        {resolvedStep === "family_profile" && (
          <FamilyProfileStep onNext={handleNext} />
        )}
        {resolvedStep === "children" && (
          <ChildrenStep onNext={handleNext} onBack={handleBack} />
        )}
        {resolvedStep === "methodology" && (
          <MethodologyStep onNext={handleNext} onBack={handleBack} />
        )}
        {resolvedStep === "roadmap_review" && (
          <RoadmapReviewStep onBack={handleBack} />
        )}
      </div>
    </div>
  );
}
