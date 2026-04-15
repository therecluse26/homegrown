import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft, DollarSign } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  StatCard,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useCreatorDashboard,
  usePayoutHistory,
  useRequestPayout,
} from "@/hooks/use-marketplace";

function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export function CreatorEarnings() {
  const intl = useIntl();
  const { data: dashboard, isPending: dashLoading } = useCreatorDashboard();
  const { data: payouts, isPending: payoutsLoading } = usePayoutHistory();
  const requestPayout = useRequestPayout();

  const isPending = dashLoading;

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-3 gap-4">
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
        </div>
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "creator.earnings.title" })}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/creator"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="creator.earnings.backToDashboard" />
        </RouterLink>
      </div>

      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="creator.earnings.title" />
        </h1>
        {dashboard && dashboard.pending_payout_cents > 0 && (
          <Button
            variant="primary"
            size="sm"
            onClick={() => requestPayout.mutate()}
            loading={requestPayout.isPending}
          >
            <Icon icon={DollarSign} size="sm" className="mr-1" />
            <FormattedMessage id="creator.earnings.requestPayout" />
          </Button>
        )}
      </div>

      {/* Summary stats */}
      {dashboard && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <StatCard
            label={intl.formatMessage({
              id: "creator.earnings.totalEarnings",
            })}
            value={formatCents(dashboard.total_earnings_cents)}
          />
          <StatCard
            label={intl.formatMessage({
              id: "creator.earnings.periodEarnings",
            })}
            value={formatCents(dashboard.period_earnings_cents)}
          />
          <StatCard
            label={intl.formatMessage({
              id: "creator.earnings.pendingPayout",
            })}
            value={formatCents(dashboard.pending_payout_cents)}
          />
        </div>
      )}

      {/* Recent sales */}
      {dashboard && dashboard.recent_sales.length > 0 && (
        <Card className="p-card-padding">
          <h2 className="type-title-md text-on-surface mb-4">
            <FormattedMessage id="creator.earnings.recentSales" />
          </h2>
          <div className="space-y-2">
            {dashboard.recent_sales.map((sale) => (
              <div
                key={sale.purchase_id}
                className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0"
              >
                <div>
                  <p className="type-body-sm text-on-surface">
                    {sale.listing_title}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {new Date(sale.purchased_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="text-right">
                  <p className="type-body-sm text-on-surface">
                    {formatCents(sale.amount_cents)}
                  </p>
                  <p className="type-label-sm text-primary">
                    <FormattedMessage id="creator.earnings.yourShare" />{" "}
                    {formatCents(sale.creator_payout_cents)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Payout history */}
      {!payoutsLoading && payouts && payouts.length > 0 && (
        <Card className="p-card-padding">
          <h2 className="type-title-md text-on-surface mb-4">
            <FormattedMessage id="creator.earnings.payoutHistory" />
          </h2>
          <div className="space-y-2">
            {payouts.map((payout) => (
              <div
                key={payout.id}
                className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0"
              >
                <div>
                  <p className="type-body-sm text-on-surface">
                    {formatCents(payout.amount_cents)}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {new Date(payout.created_at).toLocaleDateString()}
                  </p>
                </div>
                <Badge
                  variant={
                    payout.status === "completed" ? "primary" : "secondary"
                  }
                >
                  {payout.status}
                </Badge>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
