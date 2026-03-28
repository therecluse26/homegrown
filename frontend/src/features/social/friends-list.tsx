import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  UserPlus,
  MessageCircle,
  UserMinus,
  Search,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Avatar,
  Tabs,
  Badge,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useFriends,
  useIncomingFriendRequests,
  useOutgoingFriendRequests,
  useAcceptFriendRequest,
  useRejectFriendRequest,
  useUnfriend,
  useDiscoverFamilies,
  useSendFriendRequest,
} from "@/hooks/use-social";
import type { FriendResponse, FriendRequestResponse } from "@/hooks/use-social";

// ─── Friend card ────────────────────────────────────────────────────────────

function FriendCard({ friend }: { friend: FriendResponse }) {
  const intl = useIntl();
  const unfriend = useUnfriend();
  const [showUnfriendConfirm, setShowUnfriendConfirm] = useState(false);

  return (
    <Card className="p-card-padding flex items-center gap-4">
      <RouterLink to={`/family/${friend.family_id}`} className="shrink-0">
        <Avatar
          size="lg"
          src={friend.profile_photo_url}
          name={friend.display_name}
        />
      </RouterLink>

      <div className="flex-1 min-w-0">
        <RouterLink
          to={`/family/${friend.family_id}`}
          className="type-title-sm text-on-surface hover:text-primary transition-colors"
        >
          {friend.display_name}
        </RouterLink>
        {friend.methodology_names && friend.methodology_names.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1">
            {friend.methodology_names.map((name) => (
              <Badge key={name} variant="secondary">
                {name}
              </Badge>
            ))}
          </div>
        )}
        <p className="type-label-sm text-on-surface-variant mt-0.5">
          <FormattedMessage
            id="social.friends.since"
            values={{
              date: new Date(friend.friends_since).toLocaleDateString(),
            }}
          />
        </p>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <RouterLink
          to={`/messages?start=${friend.family_id}`}
          className="p-2 rounded-radius-sm bg-secondary-container text-on-secondary-container hover:opacity-90 transition-opacity"
          aria-label={intl.formatMessage({ id: "social.friends.message" })}
        >
          <Icon icon={MessageCircle} size="sm" />
        </RouterLink>
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => setShowUnfriendConfirm(true)}
          aria-label={intl.formatMessage({ id: "social.friends.unfriend" })}
        >
          <Icon icon={UserMinus} size="sm" />
        </Button>
      </div>

      <ConfirmationDialog
        open={showUnfriendConfirm}
        onClose={() => setShowUnfriendConfirm(false)}
        title={intl.formatMessage(
          { id: "social.friends.unfriend.title" },
          { name: friend.display_name },
        )}
        confirmLabel={intl.formatMessage({ id: "social.friends.unfriend" })}
        destructive
        onConfirm={() => {
          unfriend.mutate(friend.family_id, {
            onSuccess: () => setShowUnfriendConfirm(false),
          });
        }}
        loading={unfriend.isPending}
      >
        {intl.formatMessage({ id: "social.friends.unfriend.description" })}
      </ConfirmationDialog>
    </Card>
  );
}

// ─── Incoming request card ──────────────────────────────────────────────────

function IncomingRequestCard({
  request,
}: {
  request: FriendRequestResponse;
}) {
  const accept = useAcceptFriendRequest();
  const reject = useRejectFriendRequest();

  return (
    <Card className="p-card-padding flex items-center gap-4">
      <Avatar
        size="md"
        src={request.profile_photo_url}
        name={request.display_name}
      />
      <div className="flex-1 min-w-0">
        <p className="type-title-sm text-on-surface">{request.display_name}</p>
        <p className="type-label-sm text-on-surface-variant">
          {new Date(request.created_at).toLocaleDateString()}
        </p>
      </div>
      <div className="flex gap-2 shrink-0">
        <Button
          variant="primary"
          size="sm"
          onClick={() => accept.mutate(request.friendship_id)}
          disabled={accept.isPending}
        >
          <FormattedMessage id="social.friends.request.accept" />
        </Button>
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => reject.mutate(request.friendship_id)}
          disabled={reject.isPending}
        >
          <FormattedMessage id="social.friends.request.decline" />
        </Button>
      </div>
    </Card>
  );
}

// ─── Outgoing request card ──────────────────────────────────────────────────

function OutgoingRequestCard({
  request,
}: {
  request: FriendRequestResponse;
}) {
  return (
    <Card className="p-card-padding flex items-center gap-4">
      <Avatar
        size="md"
        src={request.profile_photo_url}
        name={request.display_name}
      />
      <div className="flex-1 min-w-0">
        <p className="type-title-sm text-on-surface">{request.display_name}</p>
        <p className="type-label-sm text-on-surface-variant">
          <FormattedMessage id="social.friends.request.sent" />
        </p>
      </div>
    </Card>
  );
}

// ─── Discovery section ──────────────────────────────────────────────────────

function DiscoverSection() {
  const { data: families, isPending } = useDiscoverFamilies();
  const sendRequest = useSendFriendRequest();

  if (isPending) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((n) => (
          <Skeleton key={n} className="h-16 w-full rounded-radius-md" />
        ))}
      </div>
    );
  }

  if (!families || families.length === 0) return null;

  return (
    <div className="space-y-3">
      <h2 className="type-title-md text-on-surface">
        <FormattedMessage id="social.friends.discover.title" />
      </h2>
      {families.map((family) => (
        <Card key={family.family_id} className="p-card-padding flex items-center gap-4">
          <Avatar
            size="md"
            src={family.profile_photo_url}
            name={family.display_name}
          />
          <div className="flex-1 min-w-0">
            <p className="type-title-sm text-on-surface">
              {family.display_name}
            </p>
            {family.methodology_names && family.methodology_names.length > 0 && (
              <p className="type-label-sm text-on-surface-variant">
                {family.methodology_names.join(", ")}
              </p>
            )}
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => sendRequest.mutate(family.family_id)}
            disabled={sendRequest.isPending}
          >
            <Icon icon={UserPlus} size="sm" className="mr-1" />
            <FormattedMessage id="social.friends.discover.sendRequest" />
          </Button>
        </Card>
      ))}
    </div>
  );
}

// ─── Friends list page ──────────────────────────────────────────────────────

export function FriendsList() {
  const intl = useIntl();
  const [searchQuery, setSearchQuery] = useState("");
  const { data: friends, isPending: friendsPending } = useFriends();
  const { data: incoming } = useIncomingFriendRequests();
  const { data: outgoing } = useOutgoingFriendRequests();

  const filteredFriends = friends?.filter((f) =>
    f.display_name.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "social.friends.title" })} />

      <Tabs
        tabs={[
          {
            id: "all",
            label: `${intl.formatMessage({ id: "social.friends.tabs.all" })} (${friends?.length ?? 0})`,
            content: (
              <div className="space-y-6 mt-6">
                {/* Search */}
                <div className="relative">
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
                      id: "social.friends.search.placeholder",
                    })}
                    className="w-full pl-10 pr-4 py-2.5 bg-surface-container-highest rounded-radius-md text-on-surface type-body-md placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                  />
                </div>

                {/* Friend list */}
                {friendsPending && (
                  <div className="space-y-3">
                    {[1, 2, 3].map((n) => (
                      <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
                    ))}
                  </div>
                )}

                {filteredFriends && filteredFriends.length === 0 && !friendsPending && (
                  <EmptyState
                    illustration={<Icon icon={UserPlus} size="xl" />}
                    message={intl.formatMessage({ id: "social.friends.empty.title" })}
                    description={intl.formatMessage({ id: "social.friends.empty.description" })}
                  />
                )}

                <div className="space-y-3">
                  {filteredFriends?.map((friend) => (
                    <FriendCard key={friend.family_id} friend={friend} />
                  ))}
                </div>

                {/* Discovery section */}
                <DiscoverSection />
              </div>
            ),
          },
          {
            id: "incoming",
            label: `${intl.formatMessage({ id: "social.friends.tabs.incoming" })} (${incoming?.length ?? 0})`,
            content: (
              <div className="space-y-3 mt-6">
                {incoming && incoming.length === 0 && (
                  <p className="type-body-md text-on-surface-variant text-center py-8">
                    No incoming requests
                  </p>
                )}
                {incoming?.map((req) => (
                  <IncomingRequestCard key={req.friendship_id} request={req} />
                ))}
              </div>
            ),
          },
          {
            id: "outgoing",
            label: intl.formatMessage({ id: "social.friends.tabs.outgoing" }),
            content: (
              <div className="space-y-3 mt-6">
                {outgoing && outgoing.length === 0 && (
                  <p className="type-body-md text-on-surface-variant text-center py-8">
                    No sent requests
                  </p>
                )}
                {outgoing?.map((req) => (
                  <OutgoingRequestCard key={req.friendship_id} request={req} />
                ))}
              </div>
            ),
          },
        ]}
        defaultTab="all"
      />
    </div>
  );
}
