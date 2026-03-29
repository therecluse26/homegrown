import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  Heart,
  MessageCircle,
  MoreHorizontal,
  Image,
  Award,
  Calendar,
  Star,
  Share2,
  Trash2,
  Users,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Avatar,
  DropdownMenu,
  DropdownMenuItem,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useFeed,
  useCreatePost,
  useLikePost,
  useUnlikePost,
  useDeletePost,
} from "@/hooks/use-social";
import type { PostResponse, PostType, CreatePostCommand } from "@/hooks/use-social";

// ─── Post type icon mapping ─────────────────────────────────────────────────

const POST_TYPE_ICONS: Record<PostType, typeof Heart> = {
  text: MessageCircle,
  photo: Image,
  milestone: Award,
  event_share: Calendar,
  marketplace_review: Star,
  resource_share: Share2,
};

// ─── Post composer ──────────────────────────────────────────────────────────

function PostComposer() {
  const intl = useIntl();
  const [content, setContent] = useState("");
  const [postType, setPostType] = useState<PostType>("text");
  const createPost = useCreatePost();

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!content.trim()) return;

      const command: CreatePostCommand = {
        post_type: postType,
        content: content.trim(),
      };

      createPost.mutate(command, {
        onSuccess: () => {
          setContent("");
          setPostType("text");
        },
      });
    },
    [content, postType, createPost],
  );

  return (
    <Card className="p-card-padding">
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="flex gap-3">
          <Avatar size="md" name="You" />
          <div className="flex-1">
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={intl.formatMessage({
                id: "social.feed.composer.placeholder",
              })}
              className="w-full min-h-[80px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              aria-label={intl.formatMessage({
                id: "social.feed.composer.label",
              })}
            />
          </div>
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {(
              [
                "text",
                "photo",
                "milestone",
                "event_share",
                "resource_share",
              ] as PostType[]
            ).map((type) => {
              const TypeIcon = POST_TYPE_ICONS[type];
              return (
                <button
                  key={type}
                  type="button"
                  onClick={() => setPostType(type)}
                  className={`p-2 rounded-radius-sm transition-colors touch-target ${
                    postType === type
                      ? "bg-primary-container text-on-primary-container"
                      : "text-on-surface-variant hover:bg-surface-container-low"
                  }`}
                  aria-label={intl.formatMessage(
                    { id: "social.feed.composer.type" },
                    { type },
                  )}
                  aria-pressed={postType === type}
                >
                  <Icon icon={TypeIcon} size="sm" />
                </button>
              );
            })}
          </div>

          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={!content.trim() || createPost.isPending}
          >
            {createPost.isPending ? (
              <FormattedMessage id="common.posting" />
            ) : (
              <FormattedMessage id="social.feed.composer.post" />
            )}
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Post card ──────────────────────────────────────────────────────────────

function PostCard({ post }: { post: PostResponse }) {
  const intl = useIntl();
  const likePost = useLikePost();
  const unlikePost = useUnlikePost();
  const deletePost = useDeletePost();
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const handleLikeToggle = useCallback(() => {
    if (post.is_liked_by_me) {
      unlikePost.mutate(post.id);
    } else {
      likePost.mutate(post.id);
    }
  }, [post.id, post.is_liked_by_me, likePost, unlikePost]);

  const TypeIcon = POST_TYPE_ICONS[post.post_type] ?? MessageCircle;
  const timeAgo = formatTimeAgo(post.created_at, intl);

  return (
    <Card className="p-card-padding">
      {/* Post header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <Avatar
            size="md"
            src={post.author_photo_url}
            name={post.author_name}
          />
          <div>
            <p className="type-title-sm text-on-surface">
              {post.author_name}
            </p>
            <div className="flex items-center gap-2 type-label-sm text-on-surface-variant">
              <Icon icon={TypeIcon} size="xs" aria-hidden />
              <span>{timeAgo}</span>
              {post.is_edited && (
                <span className="text-on-surface-variant">
                  <FormattedMessage id="social.post.edited" />
                </span>
              )}
              {post.group_name && (
                <span className="text-on-surface-variant">
                  &middot; {post.group_name}
                </span>
              )}
            </div>
          </div>
        </div>

        <DropdownMenu
          trigger={
            <button
              className="p-1.5 rounded-radius-sm hover:bg-surface-container-low text-on-surface-variant"
              aria-label={intl.formatMessage({
                id: "social.post.actions",
              })}
            >
              <Icon icon={MoreHorizontal} size="sm" />
            </button>
          }
        >
          <DropdownMenuItem
            destructive
            onClick={() => setShowDeleteConfirm(true)}
          >
            <Icon icon={Trash2} size="sm" />
            <FormattedMessage id="social.post.delete" />
          </DropdownMenuItem>
        </DropdownMenu>
      </div>

      {/* Post content */}
      {post.content && (
        <p className="type-body-md text-on-surface mb-4 whitespace-pre-wrap">
          {post.content}
        </p>
      )}

      {/* Post actions */}
      <div className="flex items-center gap-4 pt-3 border-t border-outline-variant/10">
        <button
          onClick={handleLikeToggle}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-radius-sm transition-colors ${
            post.is_liked_by_me
              ? "text-error bg-error-container/50"
              : "text-on-surface-variant hover:bg-surface-container-low"
          }`}
          aria-label={intl.formatMessage(
            { id: "social.post.like.toggle" },
            { liked: post.is_liked_by_me },
          )}
          aria-pressed={post.is_liked_by_me}
        >
          <Icon icon={Heart} size="sm" />
          <span className="type-label-md">
            {post.likes_count > 0 ? post.likes_count : ""}
          </span>
        </button>

        <RouterLink
          to={`/post/${post.id}`}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-low transition-colors"
        >
          <Icon icon={MessageCircle} size="sm" />
          <span className="type-label-md">
            {post.comments_count > 0 ? post.comments_count : ""}
          </span>
        </RouterLink>
      </div>

      {/* Delete confirmation */}
      <ConfirmationDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        title={intl.formatMessage({ id: "social.post.delete.title" })}
        confirmLabel={intl.formatMessage({ id: "common.delete" })}
        destructive
        onConfirm={() => {
          deletePost.mutate(post.id, {
            onSuccess: () => setShowDeleteConfirm(false),
          });
        }}
        loading={deletePost.isPending}
      >
        {intl.formatMessage({ id: "social.post.delete.description" })}
      </ConfirmationDialog>
    </Card>
  );
}

// ─── Feed page ──────────────────────────────────────────────────────────────

export function Feed() {
  const intl = useIntl();
  const [offset, setOffset] = useState(0);
  const { data, isPending, isError } = useFeed({ offset, limit: 20 });

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "social.feed.title" })} />

      <div className="flex flex-col gap-6">
        {/* Post composer */}
        <PostComposer />

        {/* Feed content */}
        {isPending && (
          <div className="flex flex-col gap-4" aria-busy="true">
            {[1, 2, 3].map((n) => (
              <Card key={n} className="p-card-padding">
                <div className="flex items-center gap-3 mb-4">
                  <Skeleton className="w-10 h-10 rounded-full" />
                  <div className="flex-1">
                    <Skeleton className="h-4 w-32 mb-1" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                </div>
                <Skeleton className="h-16 w-full" />
              </Card>
            ))}
          </div>
        )}

        {isError && (
          <Card className="p-card-padding text-center">
            <p className="type-body-md text-on-surface-variant">
              <FormattedMessage id="social.feed.error" />
            </p>
            <Button
              variant="secondary"
              size="sm"
              className="mt-3"
              onClick={() => setOffset(0)}
            >
              <FormattedMessage id="common.retry" />
            </Button>
          </Card>
        )}

        {data && data.posts.length === 0 && (
          <EmptyState
            illustration={<Icon icon={Users} size="xl" />}
            message={intl.formatMessage({ id: "social.feed.empty.title" })}
            description={intl.formatMessage({
              id: "social.feed.empty.description",
            })}
            action={
              <RouterLink
                to="/friends"
                className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-on-primary rounded-radius-button type-label-md touch-target"
              >
                <FormattedMessage id="social.feed.empty.cta" />
              </RouterLink>
            }
          />
        )}

        {/* Post list with aria-live for new posts */}
        <div aria-live="polite" aria-relevant="additions">
          {data?.posts.map((post) => <PostCard key={post.id} post={post} />)}
        </div>

        {/* Pagination */}
        {data && data.posts.length >= 20 && (
          <div className="flex justify-center gap-3 py-4">
            {offset > 0 && (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setOffset(Math.max(0, offset - 20))}
              >
                <FormattedMessage id="common.previous" />
              </Button>
            )}
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setOffset(offset + 20)}
            >
              <FormattedMessage id="common.next" />
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatTimeAgo(
  dateStr: string,
  intl: ReturnType<typeof useIntl>,
): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1)
    return intl.formatMessage({ id: "common.time.justNow" });
  if (diffMins < 60)
    return intl.formatMessage(
      { id: "common.time.minutesAgo" },
      { count: diffMins },
    );
  if (diffHours < 24)
    return intl.formatMessage(
      { id: "common.time.hoursAgo" },
      { count: diffHours },
    );
  if (diffDays < 7)
    return intl.formatMessage(
      { id: "common.time.daysAgo" },
      { count: diffDays },
    );
  return date.toLocaleDateString();
}
