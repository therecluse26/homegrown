import { FormattedMessage, useIntl } from "react-intl";
import {
  CreditCard,
  Receipt,
  Crown,
  CalendarDays,
  AlertTriangle,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import {
  useSubscription,
  useCancelSubscription,
  useReactivateSubscription,
  type Subscription,
} from "@/hooks/use-subscription";
import { useState, useEffect, useRef } from "react";
import { Link } from "react-router";

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatCurrency(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

function getTierVariant(tier: Subscription["tier"]): "primary" | "secondary" | "error" {
  switch (tier) {
    case "premium":
      return "primary";
    case "plus":
      return "secondary";
    case "free":
    default:
      return "secondary";
  }
}

function getStatusVariant(status: Subscription["status"]): "primary" | "secondary" | "error" {
  switch (status) {
    case "active":
    case "trialing":
      return "primary";
    case "cancelled":
      return "secondary";
    case "past_due":
      return "error";
    default:
      return "secondary";
  }
}

// ─── Component ─────────────────────────────────────────────────────────────

export function SubscriptionManagement() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);

  const subscription = useSubscription();
  const cancelMutation = useCancelSubscription();
  const reactivateMutation = useReactivateSubscription();

  const [showCancelConfirm, setShowCancelConfirm] = useState(false);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.subscription.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  if (subscription.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-32" className="mb-4" />
        <Skeleton height="h-12" className="mb-4" />
        <div className="flex gap-3">
          <Skeleton height="h-10" width="w-40" />
          <Skeleton height="h-10" width="w-40" />
        </div>
      </div>
    );
  }

  if (subscription.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="billing.subscription.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const sub = subscription.data;

  if (!sub) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="billing.subscription.title" />
        </h1>
        <EmptyState
          message={intl.formatMessage({ id: "billing.subscription.empty" })}
        />
      </div>
    );
  }

  const nextBillingDate = intl.formatDate(sub.current_period_end, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  const billingAmount = formatCurrency(sub.amount_cents, sub.currency);

  function handleCancel() {
    cancelMutation.mutate(undefined, {
      onSuccess: () => setShowCancelConfirm(false),
    });
  }

  function handleReactivate() {
    reactivateMutation.mutate();
  }

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="billing.subscription.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="billing.subscription.description" />
      </p>

      {/* Current plan card */}
      <Card className="mb-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <Icon
              icon={Crown}
              size="md"
              aria-hidden
              className="text-primary shrink-0"
            />
            <div>
              <p className="type-title-md text-on-surface font-semibold">
                {sub.plan_name}
              </p>
              <div className="flex items-center gap-2 mt-1">
                <Badge variant={getTierVariant(sub.tier)}>
                  <FormattedMessage
                    id={`billing.subscription.tier.${sub.tier}`}
                  />
                </Badge>
                <Badge variant={getStatusVariant(sub.status)}>
                  <FormattedMessage
                    id={`billing.subscription.status.${sub.status}`}
                  />
                </Badge>
              </div>
            </div>
          </div>
          <p className="type-headline-sm text-on-surface font-semibold shrink-0">
            {billingAmount}
            <span className="type-body-sm text-on-surface-variant font-normal">
              {" / "}
              <FormattedMessage
                id={`billing.subscription.interval.${sub.interval}`}
              />
            </span>
          </p>
        </div>

        {/* Billing period info */}
        <div className="flex items-center gap-2 type-body-sm text-on-surface-variant">
          <Icon
            icon={CalendarDays}
            size="sm"
            aria-hidden
            className="shrink-0"
          />
          {sub.cancel_at_period_end ? (
            <p>
              <FormattedMessage
                id="billing.subscription.cancelsOn"
                values={{ date: nextBillingDate }}
              />
            </p>
          ) : (
            <p>
              <FormattedMessage
                id="billing.subscription.nextBilling"
                values={{ date: nextBillingDate }}
              />
            </p>
          )}
        </div>

        {/* Past-due warning */}
        {sub.status === "past_due" && (
          <div className="flex items-center gap-2 mt-3 p-3 bg-error-container rounded-radius-sm">
            <Icon
              icon={AlertTriangle}
              size="sm"
              aria-hidden
              className="text-on-error-container shrink-0"
            />
            <p className="type-body-sm text-on-error-container">
              <FormattedMessage id="billing.subscription.pastDueWarning" />
            </p>
          </div>
        )}
      </Card>

      {/* Action buttons */}
      <div className="flex flex-wrap gap-3 mb-8">
        {sub.cancel_at_period_end ? (
          <Button
            variant="primary"
            size="sm"
            onClick={handleReactivate}
            loading={reactivateMutation.isPending}
          >
            <FormattedMessage id="billing.subscription.reactivate" />
          </Button>
        ) : sub.tier !== "free" ? (
          <>
            {!showCancelConfirm ? (
              <Button
                variant="secondary"
                size="sm"
                className="bg-error-container text-on-error-container"
                onClick={() => setShowCancelConfirm(true)}
              >
                <FormattedMessage id="billing.subscription.cancel" />
              </Button>
            ) : (
              <div className="flex items-center gap-3">
                <p className="type-body-sm text-on-surface-variant">
                  <FormattedMessage id="billing.subscription.cancelConfirm" />
                </p>
                <Button
                  variant="primary"
                  size="sm"
                  className="bg-error text-on-error"
                  onClick={handleCancel}
                  loading={cancelMutation.isPending}
                >
                  <FormattedMessage id="billing.subscription.confirmCancel" />
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setShowCancelConfirm(false)}
                >
                  <FormattedMessage id="billing.subscription.keepPlan" />
                </Button>
              </div>
            )}
          </>
        ) : null}
      </div>

      {/* Quick links */}
      <div className="flex flex-col gap-3">
        <Link
          to="/billing/payment-methods"
          className="block"
        >
          <Card className="flex items-center gap-3 transition-colors hover:bg-surface-container-high">
            <Icon
              icon={CreditCard}
              size="md"
              aria-hidden
              className="text-on-surface-variant shrink-0"
            />
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id="billing.subscription.linkPaymentMethods" />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="billing.subscription.linkPaymentMethodsDesc" />
              </p>
            </div>
          </Card>
        </Link>
        <Link
          to="/billing/transactions"
          className="block"
        >
          <Card className="flex items-center gap-3 transition-colors hover:bg-surface-container-high">
            <Icon
              icon={Receipt}
              size="md"
              aria-hidden
              className="text-on-surface-variant shrink-0"
            />
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id="billing.subscription.linkTransactions" />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="billing.subscription.linkTransactionsDesc" />
              </p>
            </div>
          </Card>
        </Link>
      </div>
    </div>
  );
}
