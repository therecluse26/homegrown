import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { Users, ArrowLeft, LogOut } from "lucide-react";
import { ResourceNotFound } from "@/components/common/resource-not-found";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Avatar,
  Badge,
  Tabs,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useGroupDetail,
  useGroupMembers,
  useGroupPosts,
  useJoinGroup,
  useLeaveGroup,
} from "@/hooks/use-social";
import type { PostResponse } from "@/hooks/use-social";

// ─── Inline post card (simplified for group context) ────────────────────────

function GroupPostCard({ post }: { post: PostResponse }) {
  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-3 mb-2">
        <Avatar
          size="sm"
          src={post.author_photo_url}
          name={post.author_name}
        />
        <div>
          <p className="type-title-sm text-on-surface">{post.author_name}</p>
          <p className="type-label-sm text-on-surface-variant">
            {new Date(post.created_at).toLocaleDateString()}
          </p>
        </div>
      </div>
      {post.content && (
        <p className="type-body-md text-on-surface whitespace-pre-wrap">
          {post.content}
        </p>
      )}
    </Card>
  );
}

// ─── Group detail page ──────────────────────────────────────────────────────

export function GroupDetail() {
  const intl = useIntl();
  const { groupId } = useParams<{ groupId: string }>();
  const [showLeaveConfirm, setShowLeaveConfirm] = useState(false);

  const { data: group, isPending } = useGroupDetail(groupId);
  const { data: members } = useGroupMembers(groupId);
  const { data: posts } = useGroupPosts(groupId);
  const joinGroup = useJoinGroup();
  const leaveGroup = useLeaveGroup();

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!group) return <ResourceNotFound backTo="/groups" />;

  const summary = group.summary;

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={summary.name} />

      {/* Back link */}
      <RouterLink
        to="/groups"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="social.groups.title" />
      </RouterLink>

      {/* Group header card */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-start gap-4">
          <div className="w-16 h-16 rounded-radius-lg bg-secondary-container flex items-center justify-center shrink-0">
            <Icon
              icon={Users}
              size="xl"
              className="text-on-secondary-container"
            />
          </div>
          <div className="flex-1">
            <h2 className="type-headline-sm text-on-surface">
              {summary.name}
            </h2>
            {summary.description && (
              <p className="type-body-md text-on-surface-variant mt-1">
                {summary.description}
              </p>
            )}
            <div className="flex items-center gap-3 mt-3">
              <span className="type-label-md text-on-surface-variant">
                <FormattedMessage
                  id="social.groups.members"
                  values={{ count: summary.member_count }}
                />
              </span>
              {summary.methodology_name && (
                <Badge variant="secondary">{summary.methodology_name}</Badge>
              )}
            </div>
          </div>

          {/* Join/Leave actions */}
          <div className="shrink-0">
            {summary.is_member ? (
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => setShowLeaveConfirm(true)}
              >
                <Icon icon={LogOut} size="sm" className="mr-1" />
                <FormattedMessage id="social.groups.leave" />
              </Button>
            ) : (
              <Button
                variant="primary"
                size="sm"
                onClick={() => joinGroup.mutate(summary.id)}
                disabled={joinGroup.isPending}
              >
                <FormattedMessage id="social.groups.join" />
              </Button>
            )}
          </div>
        </div>
      </Card>

      {/* Content tabs */}
      <Tabs
        defaultTab="posts"
        tabs={[
          {
            id: "posts",
            label: intl.formatMessage({ id: "social.groups.detail.posts" }),
            content: (
              <div className="mt-6 space-y-4">
                {posts && posts.length === 0 && (
                  <EmptyState
                    illustration={<Icon icon={Users} size="xl" />}
                    message="No posts yet"
                    description="Be the first to share in this group."
                  />
                )}
                {posts?.map((post) => (
                  <GroupPostCard key={post.id} post={post} />
                ))}
              </div>
            ),
          },
          {
            id: "members",
            label: `${intl.formatMessage({ id: "social.groups.detail.members" })} (${members?.length ?? 0})`,
            content: (
              <div className="mt-6 space-y-2">
                {members?.map((member) => (
                  <Card
                    key={member.family_id}
                    className="p-card-padding flex items-center gap-3"
                  >
                    <Avatar size="md" name={member.display_name} />
                    <div className="flex-1">
                      <p className="type-title-sm text-on-surface">
                        {member.display_name}
                      </p>
                      <p className="type-label-sm text-on-surface-variant capitalize">
                        {member.role}
                      </p>
                    </div>
                    {member.role === "admin" && (
                      <Badge variant="primary">Admin</Badge>
                    )}
                  </Card>
                ))}
              </div>
            ),
          },
        ]}
      />

      {/* Leave confirmation */}
      <ConfirmationDialog
        open={showLeaveConfirm}
        onClose={() => setShowLeaveConfirm(false)}
        title="Leave group?"
        confirmLabel="Leave"
        destructive
        onConfirm={() => {
          leaveGroup.mutate(summary.id, {
            onSuccess: () => setShowLeaveConfirm(false),
          });
        }}
        loading={leaveGroup.isPending}
      >
        You can rejoin later if the group allows it.
      </ConfirmationDialog>
    </div>
  );
}
