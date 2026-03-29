import { useState, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useSearchParams, Link as RouterLink } from "react-router";
import { Search, Users, ShoppingBag, BookOpen, Star, Calendar, MapPin } from "lucide-react";
import {
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Input,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useSearch } from "@/hooks/use-search";
import type { SearchResult, SearchScope } from "@/hooks/use-search";

// ─── Result renderers ────────────────────────────────────────────────────────

function SocialResult({ result }: { result: SearchResult }) {
  if (result.family) {
    return (
      <RouterLink to={`/family/${result.family.family_id}`}>
        <Card className="p-card-padding hover:bg-surface-container-low transition-colors">
          <div className="flex items-center gap-3">
            <Icon icon={Users} size="md" className="text-on-surface-variant shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="type-title-sm text-on-surface">{result.family.display_name}</p>
              <div className="flex items-center gap-2 mt-0.5">
                {result.family.methodology_name && (
                  <Badge variant="secondary">{result.family.methodology_name}</Badge>
                )}
                {result.family.location_region && (
                  <span className="type-label-sm text-on-surface-variant flex items-center gap-1">
                    <Icon icon={MapPin} size="xs" />
                    {result.family.location_region}
                  </span>
                )}
                {result.family.is_friend && (
                  <Badge variant="primary">
                    <FormattedMessage id="search.result.friend" />
                  </Badge>
                )}
              </div>
            </div>
          </div>
        </Card>
      </RouterLink>
    );
  }

  if (result.group) {
    return (
      <RouterLink to={`/groups/${result.group.group_id}`}>
        <Card className="p-card-padding hover:bg-surface-container-low transition-colors">
          <div className="flex items-center gap-3">
            <Icon icon={Users} size="md" className="text-on-surface-variant shrink-0" />
            <div>
              <p className="type-title-sm text-on-surface">{result.group.name}</p>
              <span className="type-label-sm text-on-surface-variant">
                <FormattedMessage
                  id="search.result.memberCount"
                  values={{ count: result.group.member_count }}
                />
              </span>
            </div>
          </div>
        </Card>
      </RouterLink>
    );
  }

  if (result.event) {
    return (
      <RouterLink to="/events">
        <Card className="p-card-padding hover:bg-surface-container-low transition-colors">
          <div className="flex items-center gap-3">
            <Icon icon={Calendar} size="md" className="text-on-surface-variant shrink-0" />
            <div>
              <p className="type-title-sm text-on-surface">{result.event.title}</p>
              <span className="type-label-sm text-on-surface-variant">
                {new Date(result.event.event_date).toLocaleDateString()}
              </span>
            </div>
          </div>
        </Card>
      </RouterLink>
    );
  }

  return null;
}

function MarketplaceResult({ result }: { result: SearchResult }) {
  const intl = useIntl();
  if (!result.listing) return null;
  const listing = result.listing;
  const price =
    listing.price_cents === 0
      ? intl.formatMessage({ id: "price.free" })
      : intl.formatNumber(listing.price_cents / 100, {
          style: "currency",
          currency: "USD",
        });

  return (
    <RouterLink to={`/marketplace/listings/${listing.listing_id}`}>
      <Card className="p-card-padding hover:bg-surface-container-low transition-colors">
        <div className="flex items-start gap-3">
          <Icon icon={ShoppingBag} size="md" className="text-on-surface-variant shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0">
            <p className="type-title-sm text-on-surface">{listing.title}</p>
            <p className="type-body-sm text-on-surface-variant line-clamp-2 mt-0.5">
              {listing.description_snippet}
            </p>
            <div className="flex items-center gap-3 mt-2">
              <span className="type-title-sm text-primary">{price}</span>
              {listing.rating_count > 0 && (
                <span className="type-label-sm text-on-surface-variant flex items-center gap-1">
                  <Icon icon={Star} size="xs" className="text-warning" />
                  {listing.rating_avg?.toFixed(1)}
                </span>
              )}
              <span className="type-label-sm text-on-surface-variant">
                {listing.publisher_name}
              </span>
              <Badge variant="secondary">
                {intl.formatMessage({
                  id: `marketplace.contentType.${listing.content_type}`,
                  defaultMessage: listing.content_type.replace("_", " "),
                })}
              </Badge>
            </div>
          </div>
        </div>
      </Card>
    </RouterLink>
  );
}

function LearningResult({ result }: { result: SearchResult }) {
  const intl = useIntl();
  const item = result.activity ?? result.journal ?? result.reading_item;
  if (!item) return null;

  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-3">
        <Icon icon={BookOpen} size="md" className="text-on-surface-variant shrink-0" />
        <div>
          <p className="type-title-sm text-on-surface">{item.title}</p>
          <div className="flex items-center gap-2 mt-0.5">
            <Badge variant="secondary">
              {intl.formatMessage({
                id: `search.resultType.${result.type}`,
                defaultMessage: result.type.replace("_", " "),
              })}
            </Badge>
            <span className="type-label-sm text-on-surface-variant">
              {item.student_name}
            </span>
          </div>
        </div>
      </div>
    </Card>
  );
}

function ResultItem({ result, scope }: { result: SearchResult; scope: SearchScope }) {
  switch (scope) {
    case "social":
      return <SocialResult result={result} />;
    case "marketplace":
      return <MarketplaceResult result={result} />;
    case "learning":
      return <LearningResult result={result} />;
  }
}

// ─── Search results page ─────────────────────────────────────────────────────

export function SearchResults() {
  const intl = useIntl();
  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const scope = (searchParams.get("scope") as SearchScope) ?? "social";

  const [localQuery, setLocalQuery] = useState(q);

  useEffect(() => {
    setLocalQuery(q);
  }, [q]);

  const { data, isPending } = useSearch(
    q.length >= 2 ? { q, scope } : null,
  );

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (localQuery.trim().length >= 2) {
      setSearchParams({ q: localQuery.trim(), scope });
    }
  };

  const changeScope = (newScope: string) => {
    setSearchParams({ q, scope: newScope });
  };

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "search.title" })}
      />

      {/* Search input */}
      <form onSubmit={handleSearch} className="mb-6">
        <div className="relative">
          <Icon
            icon={Search}
            size="sm"
            className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant"
          />
          <Input
            value={localQuery}
            onChange={(e) => setLocalQuery(e.target.value)}
            placeholder={intl.formatMessage({ id: "search.placeholder" })}
            className="pl-10"
          />
        </div>
      </form>

      {/* Scope tabs */}
      <div className="flex gap-1 bg-surface-container-low rounded-lg p-1" role="tablist">
        {(["social", "marketplace", "learning"] as const).map((s) => (
          <button
            key={s}
            role="tab"
            aria-selected={scope === s}
            onClick={() => changeScope(s)}
            className={`flex-1 rounded-md px-4 py-2 type-label-lg transition-colors ${
              scope === s
                ? "bg-surface-container-lowest text-on-surface shadow-ambient-sm"
                : "text-on-surface-variant hover:text-on-surface hover:bg-surface-container"
            }`}
          >
            {intl.formatMessage({ id: `search.scope.${s}` })}
          </button>
        ))}
      </div>

      {/* Results */}
      <div className="mt-6" aria-live="polite" aria-atomic="false">
        {!q && (
          <EmptyState
            illustration={<Icon icon={Search} size="xl" />}
            message={intl.formatMessage({ id: "search.empty.title" })}
            description={intl.formatMessage({ id: "search.empty.description" })}
          />
        )}

        {isPending && q && (
          <div className="space-y-3">
            {[1, 2, 3, 4].map((n) => (
              <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
            ))}
          </div>
        )}

        {data && (
          <>
            <p className="type-label-md text-on-surface-variant mb-4">
              <FormattedMessage
                id="search.resultCount"
                values={{ count: data.total_count }}
              />
            </p>

            {data.results.length === 0 && (
              <EmptyState
                illustration={<Icon icon={Search} size="xl" />}
                message={intl.formatMessage({
                  id: "search.noResults.title",
                })}
                description={intl.formatMessage({
                  id: "search.noResults.description",
                })}
              />
            )}

            <div className="space-y-3">
              {data.results.map((result, i) => (
                <ResultItem key={i} result={result} scope={scope} />
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
