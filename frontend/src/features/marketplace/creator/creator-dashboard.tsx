import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { DollarSign, Package, Star, TrendingUp, Plus } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useCreatorDashboard,
  useCreatorListings,
} from "@/hooks/use-marketplace";

function StatCard({
  icon,
  label,
  value,
  subLabel,
}: {
  icon: typeof DollarSign;
  label: string;
  value: string;
  subLabel?: string;
}) {
  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-radius-md bg-primary-container flex items-center justify-center shrink-0">
          <Icon icon={icon} size="md" className="text-on-primary-container" />
        </div>
        <div>
          <p className="type-label-sm text-on-surface-variant">{label}</p>
          <p className="type-headline-sm text-on-surface">{value}</p>
          {subLabel && (
            <p className="type-label-sm text-on-surface-variant">{subLabel}</p>
          )}
        </div>
      </div>
    </Card>
  );
}

export function CreatorDashboard() {
  const intl = useIntl();
  const [period, setPeriod] = useState("last_30_days");
  const { data: dashboard, isPending } = useCreatorDashboard(period);
  const { data: listingsResp } = useCreatorListings();
  const listings = listingsResp?.data;

  if (isPending) {
    return (
      <div className="max-w-content-wide mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((n) => (
            <Skeleton key={n} className="h-24 rounded-radius-md" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-content-wide mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.creator.dashboard" })}
      />

      <div className="flex items-center justify-between mb-6">
        <select
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          className="bg-surface-container-highest rounded-radius-sm px-3 py-2 text-on-surface type-body-sm"
        >
          <option value="last_7_days">Last 7 days</option>
          <option value="last_30_days">Last 30 days</option>
          <option value="last_90_days">Last 90 days</option>
          <option value="all_time">All time</option>
        </select>
        <RouterLink to="/creator/listings/new">
          <Button variant="primary" size="sm">
            <Icon icon={Plus} size="sm" className="mr-1" />
            <FormattedMessage id="marketplace.creator.newListing" />
          </Button>
        </RouterLink>
      </div>

      {dashboard && (
        <>
          {/* Stats grid */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
            <StatCard
              icon={DollarSign}
              label={intl.formatMessage({
                id: "marketplace.creator.earnings",
              })}
              value={`$${(dashboard.period_earnings_cents / 100).toFixed(2)}`}
              subLabel={`$${(dashboard.total_earnings_cents / 100).toFixed(2)} total`}
            />
            <StatCard
              icon={Package}
              label={intl.formatMessage({ id: "marketplace.creator.sales" })}
              value={String(dashboard.period_sales_count)}
              subLabel={`${dashboard.total_sales_count} total`}
            />
            <StatCard
              icon={Star}
              label={intl.formatMessage({ id: "marketplace.creator.rating" })}
              value={
                dashboard.total_reviews > 0
                  ? dashboard.average_rating.toFixed(1)
                  : "—"
              }
              subLabel={`${dashboard.total_reviews} reviews`}
            />
            <StatCard
              icon={TrendingUp}
              label={intl.formatMessage({ id: "marketplace.creator.payout" })}
              value={`$${(dashboard.pending_payout_cents / 100).toFixed(2)}`}
              subLabel="pending"
            />
          </div>

          {/* Recent sales */}
          {dashboard.recent_sales.length > 0 && (
            <Card className="p-card-padding mb-8">
              <h2 className="type-title-md text-on-surface mb-4">
                <FormattedMessage id="marketplace.creator.recentSales" />
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
                    <span className="type-title-sm text-primary">
                      +${(sale.creator_payout_cents / 100).toFixed(2)}
                    </span>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </>
      )}

      {/* Listings */}
      {listings && listings.length > 0 && (
        <Card className="p-card-padding">
          <h2 className="type-title-md text-on-surface mb-4">
            <FormattedMessage id="marketplace.creator.myListings" />
          </h2>
          <div className="space-y-2">
            {listings.map((listing) => (
              <RouterLink
                key={listing.id}
                to={`/creator/listings/${listing.id}/edit`}
                className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0 hover:bg-surface-container-low px-2 rounded-radius-sm transition-colors"
              >
                <div>
                  <p className="type-body-sm text-on-surface">{listing.title}</p>
                  <p className="type-label-sm text-on-surface-variant">
                    ${(listing.price_cents / 100).toFixed(2)} · v{listing.version}
                  </p>
                </div>
                <Badge
                  variant={listing.status === "published" ? "primary" : "secondary"}
                >
                  {listing.status}
                </Badge>
              </RouterLink>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
