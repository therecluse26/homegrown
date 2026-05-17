import { FormattedMessage, useIntl } from "react-intl";
import { Wallet } from "lucide-react";
import {
  Badge,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useAdminPayoutReport } from "@/hooks/use-admin";
import { useState } from "react";

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatCurrency(cents: number, currency: string): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

function getStatusVariant(
  status: string | undefined,
): "primary" | "secondary" | "error" {
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

const STATUS_OPTIONS = [
  { value: "", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "processing", label: "Processing" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
] as const;

// ─── Component ─────────────────────────────────────────────────────────────

export function PayoutReport() {
  const intl = useIntl();
  const [statusFilter, setStatusFilter] = useState("");

  const { data, isPending, error } = useAdminPayoutReport(
    statusFilter ? { status: statusFilter } : undefined,
  );

  if (isPending) {
    return (
      <div className="space-y-4">
        <Skeleton height="h-8" width="w-48" />
        <Skeleton height="h-16" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-4">
        <PageTitle title={intl.formatMessage({ id: "admin.payouts.title", defaultMessage: "Creator Payouts" })} />
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const items = data?.items ?? [];
  const totalCount = data?.total_count ?? 0;
  const totalAmountCents = data?.total_amount_cents ?? 0;

  return (
    <div className="space-y-6">
      <PageTitle
        title={intl.formatMessage({
          id: "admin.payouts.title",
          defaultMessage: "Creator Payouts",
        })}
      />

      {/* Summary cards */}
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <p className="type-label-sm text-on-surface-variant mb-1">
            <FormattedMessage
              id="admin.payouts.total_count"
              defaultMessage="Total Payouts"
            />
          </p>
          <p className="type-headline-sm text-on-surface font-semibold">
            {totalCount.toLocaleString()}
          </p>
        </Card>
        <Card>
          <p className="type-label-sm text-on-surface-variant mb-1">
            <FormattedMessage
              id="admin.payouts.total_amount"
              defaultMessage="Total Amount"
            />
          </p>
          <p className="type-headline-sm text-on-surface font-semibold">
            {formatCurrency(totalAmountCents, "usd")}
          </p>
        </Card>
      </div>

      {/* Status filter */}
      <div className="flex items-center gap-1 bg-surface-container-high rounded-radius-full p-1 w-fit">
        {STATUS_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            type="button"
            className={`px-3 py-1.5 rounded-radius-full type-label-sm transition-colors ${
              statusFilter === opt.value
                ? "bg-primary text-on-primary"
                : "text-on-surface-variant hover:bg-surface-container-highest"
            }`}
            onClick={() => setStatusFilter(opt.value)}
          >
            {opt.label}
          </button>
        ))}
      </div>

      {/* Payout list */}
      {items.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({
            id: "admin.payouts.empty",
            defaultMessage: "No payouts found.",
          })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {items.map((item) => (
            <li key={item.id}>
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
                      {item.store_name}
                    </p>
                    <p className="type-body-sm text-on-surface-variant truncate">
                      {intl.formatDate(item.period_start ?? "", {
                        month: "short",
                        year: "numeric",
                      })}{" "}
                      –{" "}
                      {intl.formatDate(item.period_end ?? "", {
                        month: "short",
                        year: "numeric",
                      })}
                    </p>
                    <div className="flex flex-wrap items-center gap-2 mt-1">
                      <Badge variant={getStatusVariant(item.status)}>
                        {item.status}
                      </Badge>
                      {(item.purchase_count ?? 0) > 0 && (
                        <span className="type-body-sm text-on-surface-variant">
                          {item.purchase_count} sale
                          {(item.purchase_count ?? 0) !== 1 ? "s" : ""}
                        </span>
                      )}
                      {(item.refund_deduction_cents ?? 0) > 0 && (
                        <span className="type-body-sm text-error">
                          −{formatCurrency(item.refund_deduction_cents ?? 0, item.currency ?? "usd")} refunds
                        </span>
                      )}
                      {item.processed_at && (
                        <span className="type-body-sm text-on-surface-variant">
                          Paid{" "}
                          {intl.formatDate(item.processed_at, {
                            year: "numeric",
                            month: "short",
                            day: "numeric",
                          })}
                        </span>
                      )}
                    </div>
                    {item.hyperswitch_payout_id && (
                      <p className="type-body-xs text-on-surface-variant mt-0.5 font-mono truncate">
                        {item.hyperswitch_payout_id}
                      </p>
                    )}
                  </div>
                </div>
                <p className="type-title-md text-on-surface font-semibold shrink-0">
                  {formatCurrency(item.amount_cents ?? 0, item.currency ?? "usd")}
                </p>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
