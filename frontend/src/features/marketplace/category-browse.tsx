import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft } from "lucide-react";
import {
  Card,
  Icon,
  Skeleton,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useBrowseListings } from "@/hooks/use-marketplace";

const CONTENT_TYPES = [
  { value: "", labelId: "marketplace.categories.all" },
  { value: "curriculum", labelId: "marketplace.categories.curriculum" },
  { value: "worksheet", labelId: "marketplace.categories.worksheet" },
  { value: "ebook", labelId: "marketplace.categories.ebook" },
  { value: "course", labelId: "marketplace.categories.course" },
  { value: "printable", labelId: "marketplace.categories.printable" },
  { value: "bundle", labelId: "marketplace.categories.bundle" },
];

export function CategoryBrowse() {
  const intl = useIntl();
  const [contentType, setContentType] = useState("");

  const { data, isPending } = useBrowseListings({
    content_type: contentType || undefined,
  });

  const listings = data?.data ?? [];

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-2 gap-4">
          <Skeleton className="h-48 rounded-radius-md" />
          <Skeleton className="h-48 rounded-radius-md" />
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.categories.title" })}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/marketplace"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="marketplace.categories.backToMarketplace" />
        </RouterLink>
      </div>

      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="marketplace.categories.title" />
      </h1>

      <div className="flex gap-1 flex-wrap bg-surface-container-low rounded-lg p-1">
        {CONTENT_TYPES.map((ct) => (
          <button
            key={ct.value}
            type="button"
            onClick={() => setContentType(ct.value)}
            className={`rounded-md px-4 py-2 type-label-lg transition-colors ${
              contentType === ct.value
                ? "bg-surface-container-lowest text-on-surface shadow-ambient-sm"
                : "text-on-surface-variant hover:text-on-surface hover:bg-surface-container"
            }`}
          >
            {intl.formatMessage({ id: ct.labelId })}
          </button>
        ))}
      </div>

      {listings.length === 0 ? (
        <Card className="p-card-padding">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="marketplace.categories.empty" />
          </p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {listings.map((listing) => (
            <RouterLink
              key={listing.id}
              to={`/marketplace/listings/${listing.id}`}
              className="block"
            >
              <Card className="p-card-padding hover:shadow-elevation-1 transition-shadow h-full">
                {listing.thumbnail_url && (
                  <img
                    src={listing.thumbnail_url}
                    alt=""
                    className="w-full h-32 object-cover rounded-radius-sm mb-3"
                  />
                )}
                <h3 className="type-title-md text-on-surface mb-1">
                  {listing.title}
                </h3>
                <p className="type-body-sm text-on-surface-variant line-clamp-2 mb-2">
                  {listing.description_preview}
                </p>
                <div className="flex items-center justify-between">
                  <span className="type-label-md text-primary">
                    {listing.price_cents === 0
                      ? intl.formatMessage({ id: "marketplace.free" })
                      : `$${(listing.price_cents / 100).toFixed(2)}`}
                  </span>
                  <div className="flex items-center gap-1.5">
                    <Badge variant="secondary">
                      {listing.content_type}
                    </Badge>
                    {listing.rating_count > 0 && (
                      <span className="type-label-sm text-on-surface-variant">
                        {listing.rating_avg.toFixed(1)} ({listing.rating_count})
                      </span>
                    )}
                  </div>
                </div>
              </Card>
            </RouterLink>
          ))}
        </div>
      )}
    </div>
  );
}
