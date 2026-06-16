import { FormattedMessage, useIntl } from "react-intl";
import { Badge, Button, Card, Icon } from "@/components/ui";
import { Check, CreditCard, Receipt, XCircle } from "lucide-react";
import { Link } from "react-router";
import { useAuth } from "@/hooks/use-auth";
import { PageTitle } from "@/components/common/page-title";

type Tier = {
  id: string;
  nameId: string;
  priceId: string;
  features: string[];
};

const TIER_ORDER: Record<string, number> = { free: 0, plus: 1, premium: 2 };

const TIERS: Tier[] = [
  {
    id: "free",
    nameId: "settings.subscription.tier.free",
    priceId: "settings.subscription.tier.free.price",
    features: [
      "settings.subscription.feature.familyProfile",
      "settings.subscription.feature.basicLearning",
      "settings.subscription.feature.community",
    ],
  },
  {
    id: "plus",
    nameId: "settings.subscription.tier.plus",
    priceId: "settings.subscription.tier.plus.price",
    features: [
      "settings.subscription.feature.familyProfile",
      "settings.subscription.feature.basicLearning",
      "settings.subscription.feature.community",
      "settings.subscription.feature.advancedTools",
      "settings.subscription.feature.marketplace",
    ],
  },
  {
    id: "premium",
    nameId: "settings.subscription.tier.premium",
    priceId: "settings.subscription.tier.premium.price",
    features: [
      "settings.subscription.feature.familyProfile",
      "settings.subscription.feature.basicLearning",
      "settings.subscription.feature.community",
      "settings.subscription.feature.advancedTools",
      "settings.subscription.feature.marketplace",
      "settings.subscription.feature.compliance",
      "settings.subscription.feature.prioritySupport",
    ],
  },
];

const BILLING_LINKS = [
  {
    to: "/settings/subscription/manage",
    icon: XCircle,
    labelId: "subscription.manager.title",
    descId: "subscription.manager.description",
  },
  {
    to: "/billing/payment-methods",
    icon: CreditCard,
    labelId: "billing.paymentMethods.title",
    descId: "billing.paymentMethods.description",
  },
  {
    to: "/billing/invoices",
    icon: Receipt,
    labelId: "billing.invoice.title",
    descId: "billing.invoice.description",
  },
] as const;

export function SubscriptionUpgrade() {
  const intl = useIntl();
  const { tier } = useAuth();
  const currentOrder = TIER_ORDER[tier ?? "free"] ?? 0;

  return (
    <div className="mx-auto max-w-4xl">
      <PageTitle
        title={intl.formatMessage({ id: "settings.subscription.title" })}
        subtitle={intl.formatMessage({ id: "settings.subscription.description" })}
        className="mb-8"
      />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        {TIERS.map((t) => {
          const isCurrent = t.id === tier;
          const targetOrder = TIER_ORDER[t.id] ?? 0;
          const isDowngrade = !isCurrent && targetOrder < currentOrder;
          return (
            <Card
              key={t.id}
              className={`flex flex-col ${isCurrent ? "ring-2 ring-primary" : ""}`}
            >
              <div className="flex items-center gap-2 mb-2">
                <h2 className="type-title-md text-on-surface font-semibold">
                  <FormattedMessage id={t.nameId} />
                </h2>
                {isCurrent && (
                  <Badge variant="primary">
                    <FormattedMessage id="settings.subscription.current" />
                  </Badge>
                )}
              </div>
              <p className="type-headline-sm text-on-surface font-semibold mb-4">
                <FormattedMessage id={t.priceId} />
              </p>
              <ul className="flex flex-col gap-2 mb-6 flex-1">
                {t.features.map((f) => (
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
                variant={isCurrent ? "tertiary" : isDowngrade ? "secondary" : "primary"}
                disabled
                className="w-full"
              >
                {isCurrent ? (
                  <FormattedMessage id="settings.subscription.currentPlan" />
                ) : isDowngrade ? (
                  <FormattedMessage id="settings.subscription.downgrade" />
                ) : (
                  <FormattedMessage id="settings.subscription.upgrade" />
                )}
              </Button>
            </Card>
          );
        })}
      </div>

      {/* Billing management links */}
      <h2 className="type-title-md text-on-surface font-semibold mb-3">
        <FormattedMessage id="settings.subscription.manageBilling" />
      </h2>
      <div className="flex flex-col gap-2">
        {BILLING_LINKS.map((link) => (
          <Link key={link.to} to={link.to} className="block no-underline">
            <Card interactive className="flex items-center gap-3">
              <Icon
                icon={link.icon}
                size="sm"
                aria-hidden
                className="text-on-surface-variant shrink-0"
              />
              <div>
                <p className="type-title-sm text-on-surface font-medium">
                  {intl.formatMessage({ id: link.labelId })}
                </p>
                <p className="type-body-sm text-on-surface-variant">
                  {intl.formatMessage({ id: link.descId })}
                </p>
              </div>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
