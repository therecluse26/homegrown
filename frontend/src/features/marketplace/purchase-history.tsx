import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { Package, Download, ArrowLeft } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { usePurchases } from "@/hooks/use-marketplace";
import type { PurchaseResponse } from "@/hooks/use-marketplace";

function PurchaseCard({ purchase }: { purchase: PurchaseResponse }) {
  return (
    <Card className="p-card-padding flex items-center gap-4">
      <div className="w-12 h-12 rounded-radius-sm bg-primary-container flex items-center justify-center shrink-0">
        <Icon icon={Package} size="md" className="text-on-primary-container" />
      </div>
      <div className="flex-1 min-w-0">
        <RouterLink
          to={`/marketplace/listings/${purchase.listing_id}`}
          className="type-title-sm text-on-surface hover:text-primary transition-colors"
        >
          {purchase.listing_title}
        </RouterLink>
        <div className="flex items-center gap-3 mt-1">
          <span className="type-label-sm text-on-surface-variant">
            ${(purchase.amount_cents / 100).toFixed(2)}
          </span>
          <span className="type-label-sm text-on-surface-variant">
            {new Date(purchase.created_at).toLocaleDateString()}
          </span>
          {purchase.refunded && (
            <Badge variant="secondary">
              <FormattedMessage id="marketplace.purchase.refunded" />
            </Badge>
          )}
        </div>
      </div>
      <div className="flex gap-2 shrink-0">
        <Button variant="secondary" size="sm">
          <Icon icon={Download} size="sm" className="mr-1" />
          <FormattedMessage id="marketplace.purchase.download" />
        </Button>
        {!purchase.refunded && (
          <RouterLink
            to={`/marketplace/purchases/${purchase.id}/refund`}
          >
            <Button variant="tertiary" size="sm">
              <FormattedMessage id="marketplace.purchase.refund" />
            </Button>
          </RouterLink>
        )}
      </div>
    </Card>
  );
}

export function PurchaseHistory() {
  const intl = useIntl();
  const { data: purchasesResp, isPending } = usePurchases();
  const purchases = purchasesResp?.data;

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.purchases.title" })}
      />

      <RouterLink
        to="/marketplace"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.title" />
      </RouterLink>

      {isPending && (
        <div className="space-y-3">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {purchases && purchases.length === 0 && (
        <EmptyState
          illustration={<Icon icon={Package} size="xl" />}
          message={intl.formatMessage({
            id: "marketplace.purchases.empty.title",
          })}
          description={intl.formatMessage({
            id: "marketplace.purchases.empty.description",
          })}
          action={
            <RouterLink to="/marketplace">
              <Button variant="primary">
                <FormattedMessage id="marketplace.browseCTA" />
              </Button>
            </RouterLink>
          }
        />
      )}

      <div className="space-y-3">
        {purchases?.map((purchase) => (
          <PurchaseCard key={purchase.id} purchase={purchase} />
        ))}
      </div>
    </div>
  );
}
