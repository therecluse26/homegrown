import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft, Download, Package } from "lucide-react";
import {
  Card,
  Icon,
  Skeleton,
  Badge,
  Button,
  EmptyState,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { usePurchases } from "@/hooks/use-marketplace";

export function MyLibrary() {
  const intl = useIntl();
  const { data, isPending } = usePurchases();

  const purchases = data?.data ?? [];

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.library.title" })}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/marketplace"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="marketplace.library.backToMarketplace" />
        </RouterLink>
      </div>

      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="marketplace.library.title" />
      </h1>

      {purchases.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "marketplace.library.empty" })}
          description={intl.formatMessage({
            id: "marketplace.library.emptyDescription",
          })}
        />
      ) : (
        <div className="space-y-3">
          {purchases.map((purchase) => (
            <Card key={purchase.id} className="p-card-padding">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <Icon
                    icon={Package}
                    size="md"
                    className="text-on-surface-variant"
                  />
                  <div>
                    <RouterLink
                      to={`/marketplace/listings/${purchase.listing_id}`}
                      className="type-title-md text-on-surface hover:text-primary transition-colors"
                    >
                      {purchase.listing_title}
                    </RouterLink>
                    <p className="type-label-sm text-on-surface-variant">
                      <FormattedMessage id="marketplace.library.purchased" />{" "}
                      {new Date(purchase.created_at).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {purchase.refunded && (
                    <Badge variant="secondary">
                      <FormattedMessage id="marketplace.library.refunded" />
                    </Badge>
                  )}
                  {!purchase.refunded && (
                    <RouterLink
                      to={`/marketplace/listings/${purchase.listing_id}`}
                    >
                      <Button variant="secondary" size="sm">
                        <Icon icon={Download} size="sm" className="mr-1" />
                        <FormattedMessage id="marketplace.library.download" />
                      </Button>
                    </RouterLink>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
