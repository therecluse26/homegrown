import { FormattedMessage, useIntl } from "react-intl";
import { CreditCard, Calendar, AlertTriangle } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  ConfirmationDialog,
  Icon,
  Skeleton,
} from "@/components/ui";
import {
  useSubscription,
  useCancelSubscription,
  useReactivateSubscription,
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

function formatDate(dateStr: string, intl: ReturnType<typeof useIntl>): string {
  return intl.formatDate(dateStr, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

// ─── Downgrade consequences ─────────────────────────────────────────────────

const DOWNGRADE_CONSEQUENCES = [
  "subscription.manager.downgrade.consequence.data",
  "subscription.manager.downgrade.consequence.tools",
  "subscription.manager.downgrade.consequence.compliance",
  "subscription.manager.downgrade.consequence.recommendations",
] as const;

// ─── Component ─────────────────────────────────────────────────────────────

export function SubscriptionManager() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const subscription = useSubscription();
  const cancelSub = useCancelSubscription();
  const reactivateSub = useReactivateSubscription();

  const [showCancelDialog, setShowCancelDialog] = useState(false);
  const [showDowngradeDialog, setShowDowngradeDialog] = useState(false);
  const [downgradeConfirmed, setDowngradeConfirmed] = useState(false);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "subscription.manager.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // ─── Loading ──────────────────────────────────────────────────────────

  if (subscription.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-64" className="mb-6" />
        <Skeleton height="h-48" />
      </div>
    );
  }

  // ─── Error ────────────────────────────────────────────────────────────

  if (subscription.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="subscription.manager.title" />
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
  const isCancelled = sub?.cancel_at_period_end ?? false;
  const isFree = sub?.tier === "free";

  return (
    <div className="mx-auto max-w-2xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="subscription.manager.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="subscription.manager.description" />
      </p>

      {/* Current plan card */}
      <Card className="mb-4">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <Icon icon={CreditCard} size="md" className="text-primary" aria-hidden />
              <h2 className="type-title-md text-on-surface font-semibold">
                <FormattedMessage id="subscription.manager.currentPlan" />
              </h2>
            </div>
            {sub && (
              <div className="ml-8">
                <div className="flex items-center gap-2 mb-2">
                  <p className="type-headline-sm text-on-surface font-semibold">
                    {sub.plan_name}
                  </p>
                  {isCancelled && (
                    <Badge variant="error">
                      <FormattedMessage id="subscription.manager.cancelled" />
                    </Badge>
                  )}
                  {sub.status === "active" && !isCancelled && (
                    <Badge variant="primary">{sub.tier}</Badge>
                  )}
                </div>
                {!isFree && (
                  <>
                    <p className="type-body-sm text-on-surface-variant">
                      {formatCurrency(sub.amount_cents, sub.currency)} / {sub.interval === "annual" ? intl.formatMessage({ id: "billing.pricing.annual" }).toLowerCase() : intl.formatMessage({ id: "billing.pricing.monthly" }).toLowerCase()}
                    </p>
                    <div className="flex items-center gap-1 mt-2">
                      <Icon icon={Calendar} size="xs" className="text-on-surface-variant" aria-hidden />
                      <p className="type-body-sm text-on-surface-variant">
                        <FormattedMessage id="subscription.manager.nextBilling" />:{" "}
                        {formatDate(sub.current_period_end, intl)}
                      </p>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Cancelled notice */}
        {isCancelled && sub && (
          <div className="mt-4 bg-warning-container/30 rounded-radius-md p-3">
            <p className="type-body-sm text-on-surface">
              <FormattedMessage
                id="subscription.manager.cancelledDescription"
                values={{ date: formatDate(sub.current_period_end, intl) }}
              />
            </p>
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-2 mt-4 ml-8">
          {!isFree && (
            <>
              <Link to="/billing">
                <Button variant="secondary" size="sm">
                  <FormattedMessage id="subscription.manager.changePlan" />
                </Button>
              </Link>
              {isCancelled ? (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => {
                    void reactivateSub.mutateAsync();
                  }}
                  disabled={reactivateSub.isPending}
                >
                  <FormattedMessage id="subscription.manager.reactivate" />
                </Button>
              ) : (
                <Button
                  variant="tertiary"
                  size="sm"
                  onClick={() => setShowCancelDialog(true)}
                  className="text-error"
                >
                  <FormattedMessage id="subscription.manager.cancel" />
                </Button>
              )}
            </>
          )}
          {isFree && (
            <Link to="/billing">
              <Button variant="primary" size="sm">
                <FormattedMessage id="billing.pricing.upgrade" />
              </Button>
            </Link>
          )}
        </div>
      </Card>

      {/* Quick links */}
      <div className="flex gap-3">
        <Link
          to="/billing/payment-methods"
          className="type-label-md text-primary hover:underline"
        >
          <FormattedMessage id="billing.paymentMethods.title" />
        </Link>
        <Link
          to="/billing/transactions"
          className="type-label-md text-primary hover:underline"
        >
          <FormattedMessage id="billing.transactions.title" />
        </Link>
      </div>

      {/* Cancel dialog */}
      <ConfirmationDialog
        open={showCancelDialog}
        onClose={() => setShowCancelDialog(false)}
        onConfirm={() => {
          void cancelSub.mutateAsync().then(() => {
            setShowCancelDialog(false);
          });
        }}
        title={intl.formatMessage({ id: "subscription.manager.cancel.title" })}
        confirmLabel={intl.formatMessage({
          id: "subscription.manager.cancel.confirm",
        })}
        destructive
        loading={cancelSub.isPending}
      >
        <FormattedMessage id="subscription.manager.cancel.description" />
      </ConfirmationDialog>

      {/* Downgrade dialog (used when changing to a lower tier) */}
      <ConfirmationDialog
        open={showDowngradeDialog}
        onClose={() => {
          setShowDowngradeDialog(false);
          setDowngradeConfirmed(false);
        }}
        onConfirm={() => {
          if (downgradeConfirmed) {
            setShowDowngradeDialog(false);
            setDowngradeConfirmed(false);
          }
        }}
        title={intl.formatMessage({
          id: "subscription.manager.downgrade.title",
        })}
        confirmLabel={intl.formatMessage({
          id: "subscription.manager.downgrade.confirm",
        })}
        destructive
      >
        <div className="flex flex-col gap-3">
          <p className="type-body-md text-on-surface">
            <FormattedMessage id="subscription.manager.downgrade.description" />
          </p>
          <ul className="flex flex-col gap-2">
            {DOWNGRADE_CONSEQUENCES.map((key) => (
              <li
                key={key}
                className="flex items-start gap-2 type-body-sm text-on-surface-variant"
              >
                <Icon
                  icon={AlertTriangle}
                  size="xs"
                  className="text-warning mt-0.5 shrink-0"
                  aria-hidden
                />
                <FormattedMessage id={key} />
              </li>
            ))}
          </ul>
          <Checkbox
            checked={downgradeConfirmed}
            onChange={(e) => setDowngradeConfirmed(e.target.checked)}
            label={intl.formatMessage({
              id: "subscription.manager.downgrade.checkbox",
            })}
          />
        </div>
      </ConfirmationDialog>
    </div>
  );
}
