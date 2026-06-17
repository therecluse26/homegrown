import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink, useParams } from "react-router";
import { MapPin, UserPlus, Lock, MessageCircle, UserMinus, Pencil } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Avatar,
  Badge,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useFamilyProfileView,
  useSendFriendRequest,
  useUnfriend,
  useFamilyPosts,
} from "@/hooks/use-social";
import { useAuth } from "@/hooks/use-auth";

export function FamilyProfile() {
  const intl = useIntl();
  const { familyId } = useParams<{ familyId: string }>();
  const { user } = useAuth();
  const { data: profile, isPending } = useFamilyProfileView(familyId);
  const { data: posts } = useFamilyPosts(familyId);
  const sendRequest = useSendFriendRequest();
  const unfriend = useUnfriend();
  const [showUnfriendConfirm, setShowUnfriendConfirm] = useState(false);

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-32 w-full rounded-md" />
        <Skeleton className="h-24 w-full rounded-md" />
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="max-w-content-narrow mx-auto">
        <Card className="p-card-padding text-center">
          <Icon icon={Lock} size="xl" className="text-on-surface-variant mx-auto mb-3" />
          <p className="type-body-md text-on-surface-variant">
            <FormattedMessage id="social.profile.friendsOnly" />
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage(
          { id: "social.profile.title" },
          { name: profile.display_name ?? "Family" },
        )}
      />

      <Card className="p-card-padding">
        <div className="flex items-start gap-4">
          <Avatar
            size="xl"
            src={profile.profile_photo_url}
            name={profile.display_name ?? "?"}
          />
          <div className="flex-1">
            <h2 className="type-headline-sm text-on-surface">
              {profile.display_name}
            </h2>

            {profile.bio && (
              <p className="type-body-md text-on-surface-variant mt-2">
                {profile.bio}
              </p>
            )}

            <div className="flex flex-wrap gap-4 mt-4">
              {profile.location_region && profile.location_visible && (
                <span className="flex items-center gap-1 type-label-md text-on-surface-variant">
                  <Icon icon={MapPin} size="xs" />
                  {profile.location_region}
                </span>
              )}

              {profile.methodology_names && profile.methodology_names.length > 0 && (
                <div className="flex gap-1.5">
                  {profile.methodology_names.map((name) => (
                    <Badge key={name} variant="secondary">
                      {name}
                    </Badge>
                  ))}
                </div>
              )}
            </div>

            {/* Children */}
            {profile.children && profile.children.length > 0 && (
              <div className="mt-4">
                <p className="type-label-md text-on-surface-variant mb-1">
                  <FormattedMessage id="social.profile.children" />
                </p>
                <div className="flex flex-wrap gap-2">
                  {profile.children.map((child) => (
                    <Badge key={child.display_name} variant="default">
                      {child.display_name}
                      {child.grade_level && ` (${child.grade_level})`}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {/* Actions */}
            <div className="flex flex-wrap gap-2 mt-4">
              {profile.family_id === user?.family_id && (
                <RouterLink to="/settings">
                  <Button variant="secondary" size="sm">
                    <Icon icon={Pencil} size="sm" className="mr-1" />
                    <FormattedMessage id="social.profile.editProfile" />
                  </Button>
                </RouterLink>
              )}

              {profile.is_friend && profile.family_id !== user?.family_id && (
                <>
                  <RouterLink
                    to={`/messages?start=${profile.family_id}`}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-button bg-secondary-container text-on-secondary-container type-label-md hover:opacity-90 transition-opacity"
                  >
                    <Icon icon={MessageCircle} size="sm" />
                    <FormattedMessage id="social.friends.message" />
                  </RouterLink>
                  <Button
                    variant="tertiary"
                    size="sm"
                    onClick={() => setShowUnfriendConfirm(true)}
                  >
                    <Icon icon={UserMinus} size="sm" className="mr-1" />
                    <FormattedMessage id="social.friends.unfriend" />
                  </Button>
                </>
              )}

              {!profile.is_friend && profile.friendship_status !== "pending" && profile.family_id !== user?.family_id && (
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => sendRequest.mutate(profile.family_id)}
                  disabled={sendRequest.isPending}
                >
                  <Icon icon={UserPlus} size="sm" className="mr-1" />
                  <FormattedMessage id="social.profile.addFriend" />
                </Button>
              )}

              {profile.friendship_status === "pending" && (
                <Badge variant="secondary">
                  <FormattedMessage id="social.friends.request.sent" />
                </Badge>
              )}
            </div>
          </div>
        </div>
      </Card>

      {/* Public posts */}
      {posts && posts.length > 0 && (
        <div className="mt-6 space-y-3">
          <h2 className="type-title-md text-on-surface">
            <FormattedMessage id="social.profile.posts" defaultMessage="Posts" />
          </h2>
          {posts.map((post) => (
            <Card key={post.id} className="p-card-padding">
              <div className="flex items-start gap-3">
                <Avatar
                  size="sm"
                  src={post.author_photo_url}
                  name={post.author_name}
                />
                <div className="flex-1 min-w-0">
                  <p className="type-label-md text-on-surface font-semibold">{post.author_name}</p>
                  <p className="type-label-sm text-on-surface-variant">
                    {new Date(post.created_at).toLocaleDateString()}
                  </p>
                  {post.content && (
                    <p className="type-body-md text-on-surface mt-1">{post.content}</p>
                  )}
                  <div className="flex gap-4 mt-2 type-label-sm text-on-surface-variant">
                    <span>{post.likes_count} <FormattedMessage id="social.post.likes" defaultMessage="likes" /></span>
                    <span>{post.comments_count} <FormattedMessage id="social.post.comments" defaultMessage="comments" /></span>
                  </div>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <ConfirmationDialog
        open={showUnfriendConfirm}
        onClose={() => setShowUnfriendConfirm(false)}
        title={intl.formatMessage(
          { id: "social.friends.unfriend.title" },
          { name: profile.display_name ?? "" },
        )}
        confirmLabel={intl.formatMessage({ id: "social.friends.unfriend" })}
        destructive
        onConfirm={() => {
          unfriend.mutate(profile.family_id, {
            onSuccess: () => setShowUnfriendConfirm(false),
          });
        }}
        loading={unfriend.isPending}
      >
        {intl.formatMessage({ id: "social.friends.unfriend.description" })}
      </ConfirmationDialog>
    </div>
  );
}
