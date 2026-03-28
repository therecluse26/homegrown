import { FormattedMessage } from "react-intl";
import { Badge, Button, Card, Icon } from "@/components/ui";
import { Check } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";

type Tier = {
  id: string;
  nameId: string;
  priceId: string;
  features: string[];
};

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

export function SubscriptionUpgrade() {
  const { tier } = useAuth();

  return (
    <div className="mx-auto max-w-4xl">
      <h1 className="type-headline-md text-on-surface font-semibold mb-2">
        <FormattedMessage id="settings.subscription.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-8">
        <FormattedMessage id="settings.subscription.description" />
      </p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {TIERS.map((t) => {
          const isCurrent = t.id === tier;
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
                variant={isCurrent ? "tertiary" : "primary"}
                disabled
                className="w-full"
              >
                {isCurrent ? (
                  <FormattedMessage id="settings.subscription.currentPlan" />
                ) : (
                  <FormattedMessage id="settings.subscription.upgrade" />
                )}
              </Button>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
