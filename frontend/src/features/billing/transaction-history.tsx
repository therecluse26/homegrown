import { FormattedMessage, useIntl } from "react-intl";
import { Receipt } from "lucide-react";
import {
  Badge,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { useTransactions, type Transaction } from "@/hooks/use-subscription";
import { useState, useEffect, useRef } from "react";

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatCurrency(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

type FilterType = "all" | Transaction["type"];

const FILTER_OPTIONS: { value: FilterType; labelId: string }[] = [
  { value: "all", labelId: "billing.transactions.filter.all" },
  { value: "subscription", labelId: "billing.transactions.filter.subscription" },
  { value: "purchase", labelId: "billing.transactions.filter.purchase" },
  { value: "payout", labelId: "billing.transactions.filter.payout" },
];

function getStatusVariant(status: Transaction["status"]): "primary" | "secondary" | "error" {
  switch (status) {
    case "completed":
      return "primary";
    case "pending":
      return "secondary";
    case "failed":
    case "refunded":
      return "error";
    default:
      return "secondary";
  }
}

// ─── Component ─────────────────────────────────────────────────────────────

export function TransactionHistory() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);

  const [filterType, setFilterType] = useState<FilterType>("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const transactions = useTransactions({
    type: filterType === "all" ? undefined : filterType,
    from: dateFrom || undefined,
    to: dateTo || undefined,
  });

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.transactions.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  if (transactions.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-12" className="mb-4" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-16" />
          <Skeleton height="h-16" />
          <Skeleton height="h-16" />
        </div>
      </div>
    );
  }

  if (transactions.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="billing.transactions.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const txList = transactions.data?.transactions ?? [];

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="billing.transactions.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="billing.transactions.description" />
      </p>

      {/* Filters */}
      <div className="flex flex-wrap items-end gap-3 mb-6">
        {/* Type filter tabs */}
        <div className="flex items-center gap-1 bg-surface-container-high rounded-radius-full p-1">
          {FILTER_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              className={`px-3 py-1.5 rounded-radius-full type-label-sm transition-colors ${
                filterType === opt.value
                  ? "bg-primary text-on-primary"
                  : "text-on-surface-variant hover:bg-surface-container-highest"
              }`}
              onClick={() => setFilterType(opt.value)}
            >
              <FormattedMessage id={opt.labelId} />
            </button>
          ))}
        </div>

        {/* Date range */}
        <div className="flex items-center gap-2">
          <label className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="billing.transactions.from" />
          </label>
          <input
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            className="type-body-sm text-on-surface bg-surface-container-highest px-2 py-1.5 rounded-radius-sm"
          />
          <label className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="billing.transactions.to" />
          </label>
          <input
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            className="type-body-sm text-on-surface bg-surface-container-highest px-2 py-1.5 rounded-radius-sm"
          />
        </div>
      </div>

      {/* Transaction list */}
      {txList.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "billing.transactions.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {txList.map((tx) => (
            <li key={tx.id}>
              <Card className="flex items-center justify-between">
                <div className="flex items-start gap-3">
                  <Icon
                    icon={Receipt}
                    size="md"
                    aria-hidden
                    className="text-on-surface-variant mt-0.5 shrink-0"
                  />
                  <div>
                    <p className="type-title-sm text-on-surface font-medium">
                      {tx.description}
                    </p>
                    <div className="flex items-center gap-2 mt-0.5">
                      <Badge variant="secondary">
                        <FormattedMessage
                          id={`billing.transactions.type.${tx.type}`}
                        />
                      </Badge>
                      <Badge variant={getStatusVariant(tx.status)}>
                        <FormattedMessage
                          id={`billing.transactions.status.${tx.status}`}
                        />
                      </Badge>
                      <span className="type-body-sm text-on-surface-variant">
                        {intl.formatDate(tx.created_at, {
                          year: "numeric",
                          month: "short",
                          day: "numeric",
                        })}
                      </span>
                    </div>
                  </div>
                </div>
                <p
                  className={`type-title-sm font-medium shrink-0 ${
                    tx.type === "refund" || tx.type === "payout"
                      ? "text-primary"
                      : "text-on-surface"
                  }`}
                >
                  {tx.type === "refund" ? "-" : ""}
                  {formatCurrency(tx.amount_cents, tx.currency)}
                </p>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
