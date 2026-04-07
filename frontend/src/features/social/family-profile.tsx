import { FormattedMessage, useIntl } from "react-intl";
import { useParams } from "react-router";
import { MapPin, UserPlus, Lock } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Avatar,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useFamilyProfileView,
  useSendFriendRequest,
} from "@/hooks/use-social";
import { useAuth } from "@/hooks/use-auth";

export function FamilyProfile() {
  const intl = useIntl();
  const { familyId } = useParams<{ familyId: string }>();
  const { user } = useAuth();
  const { data: profile, isPending } = useFamilyProfileView(familyId);
  const sendRequest = useSendFriendRequest();

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-32 w-full rounded-radius-md" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
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
            {!profile.is_friend && profile.friendship_status !== "pending" && profile.family_id !== user?.family_id && (
              <div className="mt-4">
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => sendRequest.mutate(profile.family_id)}
                  disabled={sendRequest.isPending}
                >
                  <Icon icon={UserPlus} size="sm" className="mr-1" />
                  <FormattedMessage id="social.profile.addFriend" />
                </Button>
              </div>
            )}

            {profile.friendship_status === "pending" && (
              <div className="mt-4">
                <Badge variant="secondary">
                  <FormattedMessage id="social.friends.request.sent" />
                </Badge>
              </div>
            )}
          </div>
        </div>
      </Card>
    </div>
  );
}
