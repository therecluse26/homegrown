import { FormattedMessage, useIntl } from "react-intl";
import { Check, Star } from "lucide-react";
import { useNavigate } from "react-router";
import {
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  Icon,
} from "@/components/ui";
import { useAuth } from "@/hooks/use-auth";
import {
  useChangePlan,
  useSubscription,
} from "@/hooks/use-subscription";
import { useState, useEffect, useRef } from "react";

// ─── Plan configuration ────────────────────────────────────────────────────

type PlanTier = "free" | "plus" | "premium";
type BillingInterval = "monthly" | "annual";

interface Plan {
  tier: PlanTier;
  nameId: string;
  priceMonthlyId: string;
  priceAnnualId: string;
  periodMonthlyId: string;
  periodAnnualId: string;
  features: string[];
  highlighted?: boolean;
}

const PLANS: Plan[] = [
  {
    tier: "free",
    nameId: "billing.pricing.free.name",
    priceMonthlyId: "billing.pricing.free.price",
    priceAnnualId: "billing.pricing.free.price",
    periodMonthlyId: "billing.pricing.free.period",
    periodAnnualId: "billing.pricing.free.period",
    features: [
      "billing.pricing.feature.familyProfile",
      "billing.pricing.feature.basicLearning",
      "billing.pricing.feature.community",
      "billing.pricing.feature.marketplace",
    ],
  },
  {
    tier: "plus",
    nameId: "billing.pricing.plus.name",
    priceMonthlyId: "billing.pricing.plus.priceMonthly",
    priceAnnualId: "billing.pricing.plus.priceAnnual",
    periodMonthlyId: "billing.pricing.plus.period.monthly",
    periodAnnualId: "billing.pricing.plus.period.annual",
    features: [
      "billing.pricing.feature.familyProfile",
      "billing.pricing.feature.basicLearning",
      "billing.pricing.feature.community",
      "billing.pricing.feature.marketplace",
      "billing.pricing.feature.advancedTools",
      "billing.pricing.feature.analytics",
    ],
    highlighted: true,
  },
  {
    tier: "premium",
    nameId: "billing.pricing.premium.name",
    priceMonthlyId: "billing.pricing.premium.priceMonthly",
    priceAnnualId: "billing.pricing.premium.priceAnnual",
    periodMonthlyId: "billing.pricing.premium.period.monthly",
    periodAnnualId: "billing.pricing.premium.period.annual",
    features: [
      "billing.pricing.feature.familyProfile",
      "billing.pricing.feature.basicLearning",
      "billing.pricing.feature.community",
      "billing.pricing.feature.marketplace",
      "billing.pricing.feature.advancedTools",
      "billing.pricing.feature.analytics",
      "billing.pricing.feature.compliance",
      "billing.pricing.feature.portfolios",
      "billing.pricing.feature.recommendations",
      "billing.pricing.feature.prioritySupport",
    ],
  },
];

const TIER_ORDER: Record<PlanTier, number> = { free: 0, plus: 1, premium: 2 };

// ─── Component ─────────────────────────────────────────────────────────────

export function PricingPage() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier: currentTier } = useAuth();
  const subscription = useSubscription();
  const changePlan = useChangePlan();
  const [interval, setBillingInterval] = useState<BillingInterval>("annual");
  const [pendingPlan, setPendingPlan] = useState<Plan | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.pricing.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function getButtonAction(planTier: PlanTier) {
    if (planTier === currentTier) return "current";
    if (TIER_ORDER[planTier] > TIER_ORDER[(currentTier ?? "free") as PlanTier]) return "upgrade";
    return "downgrade";
  }

  // ─── Plan-change handlers ─────────────────────────────────────────────────
  // Backend currently supports billing-interval changes only (not tier
  // migration). For cross-tier transitions we route users through the correct
  // next step (payment-method setup, contact support) rather than silently
  // failing. [S§10.2, 10-billing §5]

  const intervalLabelId =
    interval === "annual"
      ? "billing.pricing.annual"
      : "billing.pricing.monthly";

  function bodyMessageIdFor(plan: Plan): string {
    const action = getButtonAction(plan.tier);
    const tier = (currentTier ?? "free") as PlanTier;
    if (tier === "free") return "billing.pricing.confirm.freeBody";
    if (plan.tier === tier) return "billing.pricing.confirm.intervalBody";
    if (action === "upgrade") return "billing.pricing.confirm.upgradeBody";
    return "billing.pricing.confirm.downgradeBody";
  }

  function handleConfirm() {
    if (!pendingPlan) return;
    setErrorMessage(null);
    const tier = (currentTier ?? "free") as PlanTier;

    // Free → paid: requires a payment method + subscription creation flow that
    // does not exist on the backend yet. Direct users to add a payment method
    // and surface a clear next step.
    if (tier === "free") {
      setPendingPlan(null);
      navigate("/billing/payment-methods");
      return;
    }

    // Same tier, different interval: backend supports this via PATCH.
    if (pendingPlan.tier === tier) {
      changePlan.mutate(
        { billing_interval: interval },
        {
          onSuccess: () => setPendingPlan(null),
          onError: () =>
            setErrorMessage(intl.formatMessage({ id: "error.generic" })),
        },
      );
      return;
    }

    // Cross-tier change: not supported by backend yet.
    setErrorMessage(
      intl.formatMessage({ id: "billing.pricing.tierChangeUnavailable" }),
    );
  }

  const isBusy = changePlan.isPending;
  const onFreeTier = (currentTier ?? "free") === "free";
  // Free-tier users are routed to payment methods regardless of whether they
  // already have one saved — the subscription creation flow requires
  // explicit payment method selection today.
  const confirmLabelId = onFreeTier
    ? "billing.pricing.confirm.addPaymentMethod"
    : "billing.pricing.confirm.confirm";

  function getButtonLabelId(action: string) {
    switch (action) {
      case "current":
        return "billing.pricing.currentPlan";
      case "upgrade":
        return "billing.pricing.upgrade";
      case "downgrade":
        return "billing.pricing.downgrade";
      default:
        return "billing.pricing.subscribe";
    }
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="text-center mb-8">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-lg text-on-surface font-bold outline-none mb-2"
        >
          <FormattedMessage id="billing.pricing.title" />
        </h1>
        <p className="type-body-lg text-on-surface-variant mb-6">
          <FormattedMessage id="billing.pricing.description" />
        </p>

        {/* Monthly/Annual toggle */}
        <div className="inline-flex items-center gap-1 bg-surface-container-high rounded-radius-full p-1">
          <button
            type="button"
            className={`px-4 py-2 rounded-radius-full type-label-md transition-colors ${
              interval === "monthly"
                ? "bg-primary text-on-primary"
                : "text-on-surface-variant hover:bg-surface-container-highest"
            }`}
            onClick={() => setBillingInterval("monthly")}
          >
            <FormattedMessage id="billing.pricing.monthly" />
          </button>
          <button
            type="button"
            className={`px-4 py-2 rounded-radius-full type-label-md transition-colors flex items-center gap-1.5 ${
              interval === "annual"
                ? "bg-primary text-on-primary"
                : "text-on-surface-variant hover:bg-surface-container-highest"
            }`}
            onClick={() => setBillingInterval("annual")}
          >
            <FormattedMessage id="billing.pricing.annual" />
            <Badge variant="secondary" className="text-xs">
              <FormattedMessage
                id="billing.pricing.annualSave"
                values={{ percent: 17 }}
              />
            </Badge>
          </button>
        </div>
      </div>

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {PLANS.map((plan) => {
          const action = getButtonAction(plan.tier);
          const isCurrent = action === "current";
          return (
            <Card
              key={plan.tier}
              className={`flex flex-col relative ${
                plan.highlighted ? "ring-2 ring-primary" : ""
              }`}
            >
              {plan.highlighted && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                  <Badge variant="primary" className="flex items-center gap-1">
                    <Icon icon={Star} size="xs" aria-hidden />
                    <FormattedMessage id="billing.pricing.popular" />
                  </Badge>
                </div>
              )}
              <h2 className="type-title-lg text-on-surface font-semibold mb-1">
                <FormattedMessage id={plan.nameId} />
              </h2>
              <div className="flex items-baseline gap-1 mb-4">
                <span className="type-headline-md text-on-surface font-bold">
                  <FormattedMessage
                    id={
                      interval === "annual"
                        ? plan.priceAnnualId
                        : plan.priceMonthlyId
                    }
                  />
                </span>
                <span className="type-body-sm text-on-surface-variant">
                  <FormattedMessage
                    id={
                      interval === "annual"
                        ? plan.periodAnnualId
                        : plan.periodMonthlyId
                    }
                  />
                </span>
              </div>
              <ul className="flex flex-col gap-2.5 mb-6 flex-1">
                {plan.features.map((f) => (
                  <li
                    key={f}
                    className="flex items-start gap-2 type-body-sm text-on-surface-variant"
                  >
                    <Icon
                      icon={Check}
                      size="xs"
                      aria-hidden
                      className="mt-0.5 text-primary shrink-0"
                    />
                    <FormattedMessage id={f} />
                  </li>
                ))}
              </ul>
              <Button
                variant={isCurrent ? "tertiary" : plan.highlighted ? "primary" : "secondary"}
                disabled={isCurrent || isBusy}
                className="w-full"
                onClick={() => {
                  if (isCurrent) return;
                  setErrorMessage(null);
                  setPendingPlan(plan);
                }}
              >
                <FormattedMessage id={getButtonLabelId(action)} />
              </Button>
            </Card>
          );
        })}
      </div>

      {/* Plan change confirmation */}
      <ConfirmationDialog
        open={pendingPlan !== null}
        onClose={() => {
          setPendingPlan(null);
          setErrorMessage(null);
        }}
        onConfirm={handleConfirm}
        title={intl.formatMessage({ id: "billing.pricing.confirm.title" })}
        confirmLabel={intl.formatMessage({ id: confirmLabelId })}
        loading={isBusy || subscription.isPending}
      >
        {pendingPlan && (
          <div className="flex flex-col gap-3">
            <FormattedMessage
              id={bodyMessageIdFor(pendingPlan)}
              values={{
                planName: intl.formatMessage({ id: pendingPlan.nameId }),
                interval: intl.formatMessage({ id: intervalLabelId }),
              }}
            />
            {errorMessage && (
              <p className="type-body-sm text-error" role="alert">
                {errorMessage}
              </p>
            )}
          </div>
        )}
      </ConfirmationDialog>
    </div>
  );
}
