import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft, TrendingUp, Star, ShoppingCart, Users } from "lucide-react";
import {
  Card,
  Icon,
  Skeleton,
  Select,
  StatCard,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useCreatorDashboard } from "@/hooks/use-marketplace";

function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

const PERIODS = [
  { value: "last_7_days", labelId: "creator.analytics.last7Days" },
  { value: "last_30_days", labelId: "creator.analytics.last30Days" },
  { value: "last_90_days", labelId: "creator.analytics.last90Days" },
  { value: "all_time", labelId: "creator.analytics.allTime" },
];

export function CreatorAnalytics() {
  const intl = useIntl();
  const [period, setPeriod] = useState("last_30_days");
  const { data: dashboard, isPending } = useCreatorDashboard(period);

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "creator.analytics.title" })}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/creator"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="creator.analytics.backToDashboard" />
        </RouterLink>
      </div>

      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="creator.analytics.title" />
        </h1>
        <div className="w-48">
          <Select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
          >
            {PERIODS.map((p) => (
              <option key={p.value} value={p.value}>
                {intl.formatMessage({ id: p.labelId })}
              </option>
            ))}
          </Select>
        </div>
      </div>

      {dashboard && (
        <>
          {/* Key metrics */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <StatCard
              label={intl.formatMessage({
                id: "creator.analytics.totalSales",
              })}
              value={dashboard.total_sales_count}
              icon={<Icon icon={ShoppingCart} size="sm" />}
            />
            <StatCard
              label={intl.formatMessage({
                id: "creator.analytics.periodSales",
              })}
              value={dashboard.period_sales_count}
              icon={<Icon icon={TrendingUp} size="sm" />}
            />
            <StatCard
              label={intl.formatMessage({
                id: "creator.analytics.avgRating",
              })}
              value={
                dashboard.average_rating > 0
                  ? dashboard.average_rating.toFixed(1)
                  : "—"
              }
              icon={<Icon icon={Star} size="sm" />}
            />
            <StatCard
              label={intl.formatMessage({
                id: "creator.analytics.totalReviews",
              })}
              value={dashboard.total_reviews}
              icon={<Icon icon={Users} size="sm" />}
            />
          </div>

          {/* Earnings breakdown */}
          <Card className="p-card-padding">
            <h2 className="type-title-md text-on-surface mb-4">
              <FormattedMessage id="creator.analytics.earningsSummary" />
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <p className="type-label-sm text-on-surface-variant mb-1">
                  <FormattedMessage id="creator.analytics.totalEarnings" />
                </p>
                <p className="type-headline-sm text-on-surface">
                  {formatCents(dashboard.total_earnings_cents)}
                </p>
              </div>
              <div>
                <p className="type-label-sm text-on-surface-variant mb-1">
                  <FormattedMessage id="creator.analytics.periodEarnings" />
                </p>
                <p className="type-headline-sm text-primary">
                  {formatCents(dashboard.period_earnings_cents)}
                </p>
              </div>
              <div>
                <p className="type-label-sm text-on-surface-variant mb-1">
                  <FormattedMessage id="creator.analytics.pendingPayout" />
                </p>
                <p className="type-headline-sm text-on-surface">
                  {formatCents(dashboard.pending_payout_cents)}
                </p>
              </div>
            </div>
          </Card>

          {/* Recent sales */}
          {dashboard.recent_sales.length > 0 && (
            <Card className="p-card-padding">
              <h2 className="type-title-md text-on-surface mb-4">
                <FormattedMessage id="creator.analytics.recentSales" />
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
                    <p className="type-body-sm text-primary">
                      {formatCents(sale.creator_payout_cents)}
                    </p>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
