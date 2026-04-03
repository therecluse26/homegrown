import { FormattedMessage, useIntl } from "react-intl";
import { FileText } from "lucide-react";
import {
  Badge,
  Card,
  EmptyState,
  Icon,
  Input,
  Skeleton,
} from "@/components/ui";
import { useTransactions } from "@/hooks/use-subscription";
import { useState, useEffect, useRef } from "react";

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatCurrency(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

type InvoiceFilterType = "all" | "subscription" | "purchase";

const FILTER_OPTIONS: { value: InvoiceFilterType; labelId: string }[] = [
  { value: "all", labelId: "billing.invoice.filter.all" },
  { value: "subscription", labelId: "billing.invoice.filter.subscription" },
  { value: "purchase", labelId: "billing.invoice.filter.purchase" },
];

function getStatusVariant(status: string | undefined): "primary" | "secondary" | "error" {
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

export function InvoiceHistory() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);

  const [filterType, setFilterType] = useState<InvoiceFilterType>("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  // When "all" is selected, we fetch subscription + purchase by making two
  // separate queries and merging. However, the hook only supports a single
  // type filter. Instead, we use undefined for "all" and filter client-side
  // to only subscription + purchase types (excluding payout and refund).
  const transactions = useTransactions({
    type: filterType === "all" ? undefined : filterType,
    from: dateFrom || undefined,
    to: dateTo || undefined,
  });

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.invoice.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
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
          <FormattedMessage id="billing.invoice.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  // Client-side filter: only show subscription and purchase types (invoices)
  const allTx = transactions.data?.transactions ?? [];
  const invoiceList = allTx.filter(
    (tx) => tx.transaction_type === "subscription" || tx.transaction_type === "purchase",
  );

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="billing.invoice.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="billing.invoice.description" />
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
          <label htmlFor="invoice-date-from" className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="billing.invoice.from" />
          </label>
          <Input
            id="invoice-date-from"
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            className="type-body-sm w-auto px-2 py-1.5"
          />
          <label htmlFor="invoice-date-to" className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="billing.invoice.to" />
          </label>
          <Input
            id="invoice-date-to"
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            className="type-body-sm w-auto px-2 py-1.5"
          />
        </div>
      </div>

      {/* Invoice list */}
      {invoiceList.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "billing.invoice.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {invoiceList.map((tx) => (
            <li key={tx.id}>
              <Card className="flex items-center justify-between">
                <div className="flex items-start gap-3">
                  <Icon
                    icon={FileText}
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
                          id={`billing.invoice.type.${tx.transaction_type ?? "unknown"}`}
                        />
                      </Badge>
                      <Badge variant={getStatusVariant(tx.status)}>
                        <FormattedMessage
                          id={`billing.invoice.status.${tx.status ?? "unknown"}`}
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
                <p className="type-title-sm text-on-surface font-medium shrink-0">
                  {formatCurrency(tx.amount_cents ?? 0, tx.currency ?? "usd")}
                </p>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
