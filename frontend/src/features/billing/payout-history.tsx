import { FormattedMessage, useIntl } from "react-intl";
import { Wallet } from "lucide-react";
import {
  Badge,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { usePayouts } from "@/hooks/use-subscription";
import { useEffect, useRef } from "react";

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatCurrency(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

function formatDateRange(start: string, end: string, intl: ReturnType<typeof useIntl>): string {
  const fmt = (d: string) =>
    intl.formatDate(d, { year: "numeric", month: "short", day: "numeric" });
  return `${fmt(start)} – ${fmt(end)}`;
}

function getStatusVariant(status: string | undefined): "primary" | "secondary" | "error" {
  switch (status) {
    case "completed":
      return "primary";
    case "processing":
    case "pending":
      return "secondary";
    case "failed":
      return "error";
    default:
      return "secondary";
  }
}

// ─── Component ─────────────────────────────────────────────────────────────

export function PayoutHistory() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const payouts = usePayouts();

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.payouts.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  if (payouts.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </div>
    );
  }

  if (payouts.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="billing.payouts.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const payoutList = payouts.data?.payouts ?? [];

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="billing.payouts.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="billing.payouts.description" />
      </p>

      {payoutList.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "billing.payouts.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {payoutList.map((payout) => (
            <li key={payout.id}>
              <Card className="flex items-center justify-between gap-4">
                <div className="flex items-start gap-3 min-w-0">
                  <Icon
                    icon={Wallet}
                    size="md"
                    aria-hidden
                    className="text-on-surface-variant mt-0.5 shrink-0"
                  />
                  <div className="min-w-0">
                    <p className="type-title-sm text-on-surface font-medium truncate">
                      {formatDateRange(
                        payout.period_start ?? "",
                        payout.period_end ?? "",
                        intl,
                      )}
                    </p>
                    <div className="flex flex-wrap items-center gap-2 mt-0.5">
                      <Badge variant={getStatusVariant(payout.status)}>
                        <FormattedMessage
                          id={`billing.payouts.status.${payout.status ?? "unknown"}`}
                          defaultMessage={payout.status ?? "Unknown"}
                        />
                      </Badge>
                      {(payout.purchase_count ?? 0) > 0 && (
                        <span className="type-body-sm text-on-surface-variant">
                          <FormattedMessage
                            id="billing.payouts.purchase_count"
                            defaultMessage="{count, plural, one {# sale} other {# sales}}"
                            values={{ count: payout.purchase_count }}
                          />
                        </span>
                      )}
                      {payout.processed_at && (
                        <span className="type-body-sm text-on-surface-variant">
                          {intl.formatDate(payout.processed_at, {
                            year: "numeric",
                            month: "short",
                            day: "numeric",
                          })}
                        </span>
                      )}
                    </div>
                    {(payout.refund_deduction_cents ?? 0) > 0 && (
                      <p className="type-body-sm text-on-surface-variant mt-1">
                        <FormattedMessage
                          id="billing.payouts.refund_deduction"
                          defaultMessage="Refund deduction: {amount}"
                          values={{
                            amount: formatCurrency(
                              payout.refund_deduction_cents ?? 0,
                              payout.currency ?? "usd",
                            ),
                          }}
                        />
                      </p>
                    )}
                  </div>
                </div>
                <p className="type-title-md text-on-surface font-semibold shrink-0">
                  {formatCurrency(payout.amount_cents ?? 0, payout.currency ?? "usd")}
                </p>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
