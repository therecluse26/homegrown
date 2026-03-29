import { FormattedMessage, useIntl } from "react-intl";
import { Check, Star } from "lucide-react";
import { Badge, Button, Card, Icon } from "@/components/ui";
import { useAuth } from "@/hooks/use-auth";
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
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier: currentTier } = useAuth();
  const [interval, setInterval] = useState<BillingInterval>("annual");

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.pricing.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function getButtonAction(planTier: PlanTier) {
    if (planTier === currentTier) return "current";
    if (TIER_ORDER[planTier] > TIER_ORDER[(currentTier ?? "free") as PlanTier]) return "upgrade";
    return "downgrade";
  }

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
            onClick={() => setInterval("monthly")}
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
            onClick={() => setInterval("annual")}
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
                disabled={isCurrent}
                className="w-full"
              >
                <FormattedMessage id={getButtonLabelId(action)} />
              </Button>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
