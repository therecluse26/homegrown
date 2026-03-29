import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { Search, ShoppingCart, Star, Filter } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Input,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useBrowseListings,
  useCuratedSections,
} from "@/hooks/use-marketplace";
import type {
  ListingBrowseResponse,
  BrowseParams,
} from "@/hooks/use-marketplace";

// ─── Listing card ────────────────────────────────────────────────────────────

function ListingCard({ listing }: { listing: ListingBrowseResponse }) {
  const price =
    listing.price_cents === 0
      ? "Free"
      : `$${(listing.price_cents / 100).toFixed(2)}`;

  return (
    <RouterLink
      to={`/marketplace/listings/${listing.id}`}
      className="block hover:opacity-90 transition-opacity"
    >
      <Card className="p-card-padding h-full">
        {listing.thumbnail_url && (
          <div className="w-full h-32 rounded-radius-sm bg-surface-container-low mb-3 overflow-hidden">
            <img
              src={listing.thumbnail_url}
              alt={listing.title}
              loading="lazy"
              className="w-full h-full object-cover"
            />
          </div>
        )}
        {!listing.thumbnail_url && (
          <div className="w-full h-32 rounded-radius-sm bg-surface-container-low mb-3 flex items-center justify-center">
            <Icon
              icon={ShoppingCart}
              size="xl"
              className="text-on-surface-variant"
            />
          </div>
        )}
        <p className="type-title-sm text-on-surface line-clamp-2">
          {listing.title}
        </p>
        <p className="type-body-sm text-on-surface-variant line-clamp-2 mt-1">
          {listing.description_preview}
        </p>
        <div className="flex items-center justify-between mt-3">
          <span className="type-title-sm text-primary">{price}</span>
          {listing.rating_count > 0 && (
            <span className="flex items-center gap-1 type-label-sm text-on-surface-variant">
              <Icon icon={Star} size="xs" className="text-warning" />
              {listing.rating_avg.toFixed(1)} ({listing.rating_count})
            </span>
          )}
        </div>
        <p className="type-label-sm text-on-surface-variant mt-1">
          {listing.publisher_name}
        </p>
      </Card>
    </RouterLink>
  );
}

// ─── Marketplace browse page ─────────────────────────────────────────────────

export function MarketplaceBrowse() {
  const intl = useIntl();
  const [searchQuery, setSearchQuery] = useState("");
  const [params, setParams] = useState<BrowseParams>({});
  const [showFilters, setShowFilters] = useState(false);

  const { data: browseResp, isPending } = useBrowseListings(params);
  const listings = browseResp?.data;
  const { data: curated } = useCuratedSections();

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setParams((p) => ({ ...p, q: searchQuery || undefined }));
  };

  const isFiltered = !!params.q || !!params.content_type || !!params.sort_by;

  return (
    <div className="max-w-content-wide mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.title" })}
      />

      {/* Search bar */}
      <form onSubmit={handleSearch} className="flex gap-2 mb-6">
        <div className="relative flex-1">
          <Icon
            icon={Search}
            size="sm"
            className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant"
          />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={intl.formatMessage({
              id: "marketplace.search.placeholder",
            })}
            className="pl-10"
          />
        </div>
        <Button type="submit" variant="primary">
          <FormattedMessage id="marketplace.search" />
        </Button>
        <Button
          type="button"
          variant="secondary"
          onClick={() => setShowFilters(!showFilters)}
        >
          <Icon icon={Filter} size="sm" />
        </Button>
      </form>

      {/* Filters */}
      {showFilters && (
        <Card className="p-card-padding mb-6">
          <div className="flex flex-wrap gap-4">
            <div>
              <label className="type-label-sm text-on-surface-variant block mb-1">
                <FormattedMessage id="marketplace.filter.sort" />
              </label>
              <select
                value={params.sort_by ?? ""}
                onChange={(e) =>
                  setParams((p) => ({
                    ...p,
                    sort_by: e.target.value || undefined,
                  }))
                }
                className="bg-surface-container-highest rounded-radius-sm px-3 py-2 text-on-surface type-body-sm"
              >
                <option value="">
                  {intl.formatMessage({ id: "marketplace.sort.relevance" })}
                </option>
                <option value="price_asc">
                  {intl.formatMessage({ id: "marketplace.sort.priceAsc" })}
                </option>
                <option value="price_desc">
                  {intl.formatMessage({ id: "marketplace.sort.priceDesc" })}
                </option>
                <option value="rating">
                  {intl.formatMessage({ id: "marketplace.sort.rating" })}
                </option>
                <option value="newest">
                  {intl.formatMessage({ id: "marketplace.sort.newest" })}
                </option>
              </select>
            </div>
            <div>
              <label className="type-label-sm text-on-surface-variant block mb-1">
                <FormattedMessage id="marketplace.filter.contentType" />
              </label>
              <select
                value={params.content_type ?? ""}
                onChange={(e) =>
                  setParams((p) => ({
                    ...p,
                    content_type: e.target.value || undefined,
                  }))
                }
                className="bg-surface-container-highest rounded-radius-sm px-3 py-2 text-on-surface type-body-sm"
              >
                <option value="">All</option>
                <option value="curriculum">Curriculum</option>
                <option value="worksheet">Worksheet</option>
                <option value="unit_study">Unit Study</option>
                <option value="video">Video</option>
                <option value="book_list">Book List</option>
                <option value="lesson_plan">Lesson Plan</option>
                <option value="printable">Printable</option>
                <option value="course">Course</option>
              </select>
            </div>
            {isFiltered && (
              <div className="flex items-end">
                <Button
                  variant="tertiary"
                  size="sm"
                  onClick={() => {
                    setParams({});
                    setSearchQuery("");
                  }}
                >
                  <FormattedMessage id="marketplace.filter.clear" />
                </Button>
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Curated sections (when not searching) */}
      {!isFiltered && curated && curated.length > 0 && (
        <div className="space-y-8 mb-8">
          {curated.map((section) => (
            <div key={section.slug}>
              <h2 className="type-title-lg text-on-surface mb-4">
                {section.display_name}
              </h2>
              {section.description && (
                <p className="type-body-md text-on-surface-variant mb-3">
                  {section.description}
                </p>
              )}
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                {section.listings.map((listing) => (
                  <ListingCard key={listing.id} listing={listing} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Loading */}
      {isPending && (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((n) => (
            <Skeleton key={n} className="h-56 rounded-radius-md" />
          ))}
        </div>
      )}

      {/* Results */}
      {listings && listings.length === 0 && (
        <EmptyState
          illustration={<Icon icon={Search} size="xl" />}
          message={intl.formatMessage({ id: "marketplace.empty.title" })}
          description={intl.formatMessage({
            id: "marketplace.empty.description",
          })}
        />
      )}

      {listings && listings.length > 0 && (
        <>
          {isFiltered && (
            <div className="flex items-center gap-2 mb-4">
              <Badge variant="secondary">
                {listings.length}{" "}
                <FormattedMessage id="marketplace.results" />
              </Badge>
            </div>
          )}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            {listings.map((listing) => (
              <ListingCard key={listing.id} listing={listing} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}
