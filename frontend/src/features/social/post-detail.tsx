import { useState, useCallback } from "react";
import { useParams, Link as RouterLink, useNavigate } from "react-router";
import { FormattedMessage, useIntl } from "react-intl";
import {
  Heart,
  MessageCircle,
  ArrowLeft,
  Send,
  CornerDownRight,
  Trash2,
  Pencil,
  Check,
  X,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Avatar,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ReportButton } from "@/components/common/report-button";
import {
  usePostDetail,
  useLikePost,
  useUnlikePost,
  useCreateComment,
  useDeleteComment,
  useDeletePost,
  useUpdateComment,
} from "@/hooks/use-social";
import type { CommentResponse } from "@/hooks/use-social";

// ─── Comment component ──────────────────────────────────────────────────────

function Comment({
  comment,
  postId,
  depth = 0,
}: {
  comment: CommentResponse;
  postId: string;
  depth?: number;
}) {
  const intl = useIntl();
  const [showReplyForm, setShowReplyForm] = useState(false);
  const [replyText, setReplyText] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editText, setEditText] = useState(comment.content);
  const createComment = useCreateComment(postId);
  const deleteComment = useDeleteComment();
  const updateComment = useUpdateComment(comment.id);

  const handleReply = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!replyText.trim()) return;
      createComment.mutate(
        { content: replyText.trim(), parent_comment_id: comment.id },
        {
          onSuccess: () => {
            setReplyText("");
            setShowReplyForm(false);
          },
        },
      );
    },
    [replyText, comment.id, createComment],
  );

  const handleEditSave = useCallback(() => {
    if (!editText.trim() || editText.trim() === comment.content) {
      setIsEditing(false);
      setEditText(comment.content);
      return;
    }
    updateComment.mutate(
      { content: editText.trim() },
      {
        onSuccess: () => setIsEditing(false),
      },
    );
  }, [editText, comment.content, updateComment]);

  return (
    <div className={depth > 0 ? "ml-8 mt-2" : "mt-3"}>
      <div className="flex items-start gap-2.5">
        <Avatar
          size="sm"
          src={comment.author_photo_url}
          name={comment.author_name}
        />
        <div className="flex-1 min-w-0">
          <div className="bg-surface-container-low rounded-radius-md px-3 py-2">
            <p className="type-label-md font-semibold text-on-surface">
              {comment.author_name}
            </p>
            {isEditing ? (
              <div className="flex items-center gap-2 mt-1">
                <input
                  type="text"
                  value={editText}
                  onChange={(e) => setEditText(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleEditSave();
                    if (e.key === "Escape") {
                      setIsEditing(false);
                      setEditText(comment.content);
                    }
                  }}
                  className="flex-1 bg-surface-container-highest rounded-radius-sm px-2 py-1 text-on-surface type-body-sm focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                  autoFocus
                />
                <button
                  onClick={handleEditSave}
                  disabled={updateComment.isPending}
                  className="text-primary hover:text-primary/80 transition-colors"
                  aria-label={intl.formatMessage({ id: "common.save" })}
                >
                  <Icon icon={Check} size="xs" />
                </button>
                <button
                  onClick={() => {
                    setIsEditing(false);
                    setEditText(comment.content);
                  }}
                  className="text-on-surface-variant hover:text-on-surface transition-colors"
                  aria-label={intl.formatMessage({ id: "common.cancel" })}
                >
                  <Icon icon={X} size="xs" />
                </button>
              </div>
            ) : (
              <p className="type-body-sm text-on-surface whitespace-pre-wrap">
                {comment.content}
              </p>
            )}
          </div>
          <div className="flex items-center gap-3 mt-1 type-label-sm text-on-surface-variant">
            <span>
              {new Date(comment.created_at).toLocaleDateString()}
            </span>
            {depth === 0 && (
              <button
                onClick={() => setShowReplyForm(!showReplyForm)}
                className="hover:text-primary transition-colors"
              >
                <FormattedMessage id="social.post.comment.reply" />
              </button>
            )}
            <button
              onClick={() => {
                setEditText(comment.content);
                setIsEditing(true);
              }}
              className="hover:text-primary transition-colors"
              aria-label={intl.formatMessage({
                id: "social.post.comment.edit",
              })}
            >
              <Icon icon={Pencil} size="xs" />
            </button>
            <button
              onClick={() => setShowDeleteConfirm(true)}
              className="hover:text-error transition-colors"
              aria-label={intl.formatMessage({
                id: "social.post.comment.delete",
              })}
            >
              <Icon icon={Trash2} size="xs" />
            </button>
            <ReportButton targetType="comment" targetId={comment.id} />
          </div>

          {/* Reply form */}
          {showReplyForm && (
            <form
              onSubmit={handleReply}
              className="flex items-center gap-2 mt-2"
            >
              <Icon
                icon={CornerDownRight}
                size="sm"
                className="text-on-surface-variant shrink-0"
              />
              <input
                type="text"
                value={replyText}
                onChange={(e) => setReplyText(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "social.post.comment.replyPlaceholder",
                })}
                className="flex-1 bg-surface-container-highest rounded-radius-sm px-3 py-1.5 text-on-surface type-body-sm placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
              <Button
                type="submit"
                variant="primary"
                size="sm"
                disabled={!replyText.trim() || createComment.isPending}
              >
                <Icon icon={Send} size="xs" />
              </Button>
            </form>
          )}

          {/* Nested replies */}
          {comment.replies?.map((reply) => (
            <Comment
              key={reply.id}
              comment={reply}
              postId={postId}
              depth={depth + 1}
            />
          ))}
        </div>
      </div>

      <ConfirmationDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        title={intl.formatMessage({ id: "social.post.comment.delete.title" })}
        confirmLabel={intl.formatMessage({ id: "common.delete" })}
        destructive
        onConfirm={() => {
          deleteComment.mutate(comment.id, {
            onSuccess: () => setShowDeleteConfirm(false),
          });
        }}
        loading={deleteComment.isPending}
      >
        <FormattedMessage id="social.post.comment.delete.description" />
      </ConfirmationDialog>
    </div>
  );
}

// ─── Post detail page ───────────────────────────────────────────────────────

export function PostDetail() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { postId } = useParams<{ postId: string }>();
  const { data, isPending, error } = usePostDetail(postId);
  const likePost = useLikePost();
  const unlikePost = useUnlikePost();
  const deletePost = useDeletePost();
  const createComment = useCreateComment(postId ?? "");
  const [commentText, setCommentText] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const handleComment = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!commentText.trim()) return;
      createComment.mutate(
        { content: commentText.trim() },
        { onSuccess: () => setCommentText("") },
      );
    },
    [commentText, createComment],
  );

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-24" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="max-w-content-narrow mx-auto">
        <PageTitle title={intl.formatMessage({ id: "social.post.notFound.title" })} />
        <RouterLink
          to="/"
          className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="social.post.backToFeed" />
        </RouterLink>
        <Card className="p-card-padding text-center">
          <p className="type-body-md text-on-surface-variant">
            <FormattedMessage id="social.post.notFound" />
          </p>
        </Card>
      </div>
    );
  }

  const { post, comments } = data;

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={`${post.author_name}'s post`} />

      {/* Back */}
      <RouterLink
        to="/"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="social.post.backToFeed" />
      </RouterLink>

      {/* Post */}
      <Card className="p-card-padding">
        <div className="flex items-center gap-3 mb-3">
          <Avatar
            size="md"
            src={post.author_photo_url}
            name={post.author_name}
          />
          <div className="flex-1 min-w-0">
            <p className="type-title-sm text-on-surface">
              {post.author_name}
            </p>
            <p className="type-label-sm text-on-surface-variant">
              {new Date(post.created_at).toLocaleDateString(undefined, {
                year: "numeric",
                month: "long",
                day: "numeric",
                hour: "2-digit",
                minute: "2-digit",
              })}
              {post.is_edited && (
                <span className="ml-1">
                  (<FormattedMessage id="social.post.edited" />)
                </span>
              )}
            </p>
          </div>
          <ReportButton targetType="post" targetId={post.id} />
        </div>

        {post.content && (
          <p className="type-body-md text-on-surface whitespace-pre-wrap mb-4">
            {post.content}
          </p>
        )}

        {/* Actions */}
        <div className="flex items-center gap-4 pt-3 border-t border-outline-variant/10">
          <button
            onClick={() =>
              post.is_liked_by_me
                ? unlikePost.mutate(post.id)
                : likePost.mutate(post.id)
            }
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-radius-sm transition-colors ${
              post.is_liked_by_me
                ? "text-error bg-error-container/50"
                : "text-on-surface-variant hover:bg-surface-container-low"
            }`}
            aria-pressed={post.is_liked_by_me}
          >
            <Icon icon={Heart} size="sm" />
            <span className="type-label-md">
              {post.likes_count > 0 && post.likes_count}
            </span>
          </button>

          <span className="flex items-center gap-1.5 text-on-surface-variant">
            <Icon icon={MessageCircle} size="sm" />
            <span className="type-label-md">{comments.length}</span>
          </span>

          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="ml-auto text-on-surface-variant hover:text-error transition-colors p-1.5 rounded-radius-sm"
            aria-label={intl.formatMessage({
              id: "social.post.delete",
            })}
          >
            <Icon icon={Trash2} size="sm" />
          </button>
        </div>
      </Card>

      {/* Comment form */}
      <form onSubmit={handleComment} className="flex items-center gap-2 mt-4">
        <input
          type="text"
          value={commentText}
          onChange={(e) => setCommentText(e.target.value)}
          placeholder={intl.formatMessage({
            id: "social.post.comment.placeholder",
          })}
          className="flex-1 bg-surface-container-highest rounded-radius-md px-4 py-2.5 text-on-surface type-body-md placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
        />
        <Button
          type="submit"
          variant="primary"
          size="md"
          disabled={!commentText.trim() || createComment.isPending}
        >
          <Icon icon={Send} size="sm" />
        </Button>
      </form>

      {/* Comments — only render top-level; replies are nested via Comment component */}
      <div className="mt-4 space-y-1">
        {comments
          .filter((c) => !c.parent_comment_id)
          .map((comment) => (
            <Comment
              key={comment.id}
              comment={comment}
              postId={post.id}
            />
          ))}
      </div>

      <ConfirmationDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        title={intl.formatMessage({ id: "social.post.delete.title" })}
        confirmLabel={intl.formatMessage({ id: "common.delete" })}
        destructive
        onConfirm={() =>
          deletePost.mutate(post.id, {
            onSuccess: () => navigate("/"),
          })
        }
        loading={deletePost.isPending}
      >
        <FormattedMessage id="social.post.delete.description" />
      </ConfirmationDialog>
    </div>
  );
}
