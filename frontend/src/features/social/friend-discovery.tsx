import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  UserPlus,
  Search,
  MapPin,
  Users,
  ArrowLeft,
  Compass,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Avatar,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useDiscoverFamilies,
  useDiscoverGroups,
  useDiscoverEvents,
  useMyProfile,
  useSendFriendRequest,
} from "@/hooks/use-social";
import type {
  DiscoverableFamilyResponse,
  GroupSummaryResponse,
  EventSummaryResponse,
} from "@/hooks/use-social";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Family suggestion card ────────────────────────────────────────────────

function FamilySuggestionCard({
  family,
}: {
  family: DiscoverableFamilyResponse;
}) {
  const sendRequest = useSendFriendRequest();
  const [sent, setSent] = useState(false);

  return (
    <Card className="p-card-padding flex items-center gap-4">
      <RouterLink to={`/family/${family.family_id}`} className="shrink-0">
        <Avatar
          size="lg"
          src={family.profile_photo_url}
          name={family.display_name}
        />
      </RouterLink>
      <div className="flex-1 min-w-0">
        <RouterLink
          to={`/family/${family.family_id}`}
          className="type-title-sm text-on-surface hover:text-primary transition-colors"
        >
          {family.display_name}
        </RouterLink>
        {family.methodology_names && family.methodology_names.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1">
            {family.methodology_names.map((name) => (
              <Badge key={name} variant="secondary">
                {name}
              </Badge>
            ))}
          </div>
        )}
        {family.location_region && (
          <p className="flex items-center gap-1 type-label-sm text-on-surface-variant mt-1">
            <Icon icon={MapPin} size="xs" />
            {family.location_region}
          </p>
        )}
      </div>
      <div className="shrink-0">
        {sent ? (
          <Badge variant="secondary">
            <FormattedMessage id="social.discover.requestSent" />
          </Badge>
        ) : (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              sendRequest.mutate(family.family_id, {
                onSuccess: () => setSent(true),
              });
            }}
            disabled={sendRequest.isPending}
          >
            <Icon icon={UserPlus} size="sm" className="mr-1" />
            <FormattedMessage id="social.discover.addFriend" />
          </Button>
        )}
      </div>
    </Card>
  );
}

// ─── Group suggestion card ──────────────────────────────────────────────────

function GroupSuggestionCard({ group }: { group: GroupSummaryResponse }) {
  return (
    <RouterLink to={`/groups/${group.id}`}>
      <Card className="p-card-padding flex items-center gap-4 hover:bg-surface-container-low transition-colors">
        <div className="w-12 h-12 rounded-radius-md bg-secondary-container flex items-center justify-center shrink-0">
          <Icon icon={Users} size="md" className="text-on-secondary-container" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="type-title-sm text-on-surface">{group.name}</p>
          {group.description && (
            <p className="type-body-sm text-on-surface-variant line-clamp-1 mt-0.5">
              {group.description}
            </p>
          )}
          <div className="flex items-center gap-3 mt-1 type-label-sm text-on-surface-variant">
            <span className="flex items-center gap-1">
              <Icon icon={Users} size="xs" />
              <FormattedMessage
                id="social.groups.memberCount"
                values={{ count: group.member_count }}
              />
            </span>
            {group.methodology_name && (
              <Badge variant="secondary">{group.methodology_name}</Badge>
            )}
          </div>
        </div>
      </Card>
    </RouterLink>
  );
}

// ─── Nearby event card ─────────────────────────────────────────────────────

function NearbyEventCard({ event }: { event: EventSummaryResponse }) {
  const eventDate = new Date(event.event_date);
  return (
    <RouterLink to={`/events/${event.id}`}>
      <Card className="p-card-padding flex items-center gap-4 hover:bg-surface-container-low transition-colors">
        <div className="w-12 h-12 rounded-radius-md bg-primary-container text-on-primary-container flex flex-col items-center justify-center shrink-0">
          <span className="type-label-sm uppercase">
            {eventDate.toLocaleDateString(undefined, { month: "short" })}
          </span>
          <span className="type-title-sm font-bold">
            {eventDate.getDate()}
          </span>
        </div>
        <div className="flex-1 min-w-0">
          <p className="type-title-sm text-on-surface">{event.title}</p>
          <div className="flex items-center gap-3 mt-1 type-label-sm text-on-surface-variant">
            {event.location_name && (
              <span className="flex items-center gap-1">
                <Icon icon={MapPin} size="xs" />
                {event.location_name}
                {event.location_region && `, ${event.location_region}`}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Icon icon={Users} size="xs" />
              {event.attendee_count}
            </span>
          </div>
        </div>
      </Card>
    </RouterLink>
  );
}

// ─── Friend discovery page ──────────────────────────────────────────────────

export function FriendDiscovery() {
  const intl = useIntl();
  const [searchQuery, setSearchQuery] = useState("");
  const [methodologyFilter, setMethodologyFilter] = useState<
    string | undefined
  >();

  const methodology = useMethodologyContext();
  const activeSlug = methodology?.primarySlug;
  const { data: profile } = useMyProfile();
  const locationEnabled = profile?.location_visible ?? false;
  const locationRegion = profile?.location_region;

  const { data: families, isPending: familiesLoading } = useDiscoverFamilies({
    methodology_slug: methodologyFilter,
  });
  const { data: groups, isPending: groupsLoading } = useDiscoverGroups({
    methodology_slug: methodologyFilter,
  });
  const { data: nearbyEvents } = useDiscoverEvents(
    locationEnabled && locationRegion
      ? { location_region: locationRegion }
      : undefined,
  );

  // Client-side name search filtering
  const filteredFamilies = families?.filter((f) =>
    searchQuery
      ? f.display_name.toLowerCase().includes(searchQuery.toLowerCase())
      : true,
  );

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "social.discover.title" })} />

      {/* Back link */}
      <RouterLink
        to="/friends"
        className="inline-flex items-center gap-1 mb-6 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="social.discover.backToFriends" />
      </RouterLink>

      {/* Search + filter bar */}
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <div className="relative flex-1">
          <Icon
            icon={Search}
            size="sm"
            className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant"
          />
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={intl.formatMessage({
              id: "social.discover.searchPlaceholder",
            })}
            className="w-full pl-10 pr-4 py-2.5 bg-surface-container-highest rounded-radius-md text-on-surface type-body-md placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
          />
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setMethodologyFilter(undefined)}
            className={`px-3 py-2 rounded-radius-sm type-label-md transition-colors ${
              !methodologyFilter
                ? "bg-primary text-on-primary"
                : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
            }`}
          >
            <FormattedMessage id="social.discover.filter.all" />
          </button>
          {activeSlug && (
            <button
              onClick={() => setMethodologyFilter(activeSlug)}
              className={`px-3 py-2 rounded-radius-sm type-label-md transition-colors ${
                methodologyFilter === activeSlug
                  ? "bg-primary text-on-primary"
                  : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
              }`}
            >
              <FormattedMessage id="social.discover.filter.myMethodology" />
            </button>
          )}
        </div>
      </div>

      {/* Methodology matches section */}
      <section className="mb-8">
        <h2 className="type-title-md text-on-surface mb-4">
          <FormattedMessage id="social.discover.familySuggestions" />
        </h2>

        {familiesLoading && (
          <div className="space-y-3">
            {[1, 2, 3].map((n) => (
              <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
            ))}
          </div>
        )}

        {filteredFamilies && filteredFamilies.length === 0 && !familiesLoading && (
          <EmptyState
            illustration={<Icon icon={Compass} size="xl" />}
            message={intl.formatMessage({ id: "social.discover.noFamilies" })}
            description={intl.formatMessage({
              id: "social.discover.noFamilies.description",
            })}
          />
        )}

        <div className="space-y-3">
          {filteredFamilies?.map((family) => (
            <FamilySuggestionCard
              key={family.family_id}
              family={family}
            />
          ))}
        </div>
      </section>

      {/* Nearby section — shown when location sharing is enabled */}
      {locationEnabled && (
        <section className="mb-8">
          <h2 className="type-title-md text-on-surface mb-1">
            <FormattedMessage id="social.discover.nearby.title" />
          </h2>
          <p className="type-body-sm text-on-surface-variant mb-4">
            <FormattedMessage
              id="social.discover.nearby.description"
              values={{ region: locationRegion ?? "" }}
            />
          </p>

          {/* Nearby families — those with location_region shown with pin */}
          {filteredFamilies && filteredFamilies.filter((f) => f.location_region).length > 0 && (
            <div className="mb-4">
              <h3 className="type-label-lg text-on-surface-variant mb-2">
                <FormattedMessage id="social.discover.nearby.families" />
              </h3>
              <div className="space-y-3">
                {filteredFamilies
                  .filter((f) => f.location_region)
                  .map((family) => (
                    <FamilySuggestionCard
                      key={family.family_id}
                      family={family}
                    />
                  ))}
              </div>
            </div>
          )}

          {/* Nearby events */}
          {nearbyEvents && nearbyEvents.length > 0 && (
            <div>
              <h3 className="type-label-lg text-on-surface-variant mb-2">
                <FormattedMessage id="social.discover.nearby.events" />
              </h3>
              <div className="space-y-3">
                {nearbyEvents.map((event) => (
                  <NearbyEventCard key={event.id} event={event} />
                ))}
              </div>
            </div>
          )}

          {(!filteredFamilies || filteredFamilies.filter((f) => f.location_region).length === 0) &&
            (!nearbyEvents || nearbyEvents.length === 0) && (
              <p className="type-body-sm text-on-surface-variant text-center py-4">
                <FormattedMessage id="social.discover.nearby.none" />
              </p>
            )}
        </section>
      )}

      {/* Groups section */}
      <section>
        <h2 className="type-title-md text-on-surface mb-4">
          <FormattedMessage id="social.discover.groupSuggestions" />
        </h2>

        {groupsLoading && (
          <div className="space-y-3">
            {[1, 2].map((n) => (
              <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
            ))}
          </div>
        )}

        {groups && groups.length === 0 && !groupsLoading && (
          <p className="type-body-md text-on-surface-variant text-center py-4">
            <FormattedMessage id="social.discover.noGroups" />
          </p>
        )}

        <div className="space-y-3">
          {groups?.map((group) => (
            <GroupSuggestionCard key={group.id} group={group} />
          ))}
        </div>
      </section>
    </div>
  );
}
