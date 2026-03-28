import { useState, useCallback } from "react";
import { useParams, Link as RouterLink } from "react-router";
import {
  Heart,
  MessageCircle,
  ArrowLeft,
  Send,
  CornerDownRight,
  Trash2,
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
import {
  usePostDetail,
  useLikePost,
  useUnlikePost,
  useCreateComment,
  useDeleteComment,
  useDeletePost,
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
  const [showReplyForm, setShowReplyForm] = useState(false);
  const [replyText, setReplyText] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const createComment = useCreateComment(postId);
  const deleteComment = useDeleteComment();

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
            <p className="type-body-sm text-on-surface whitespace-pre-wrap">
              {comment.content}
            </p>
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
                Reply
              </button>
            )}
            <button
              onClick={() => setShowDeleteConfirm(true)}
              className="hover:text-error transition-colors"
              aria-label="Delete comment"
            >
              <Icon icon={Trash2} size="xs" />
            </button>
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
                placeholder="Write a reply..."
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
        title="Delete comment?"
        confirmLabel="Delete"
        destructive
        onConfirm={() => {
          deleteComment.mutate(comment.id, {
            onSuccess: () => setShowDeleteConfirm(false),
          });
        }}
        loading={deleteComment.isPending}
      >
        This cannot be undone.
      </ConfirmationDialog>
    </div>
  );
}

// ─── Post detail page ───────────────────────────────────────────────────────

export function PostDetail() {
  const { postId } = useParams<{ postId: string }>();
  const { data, isPending } = usePostDetail(postId);
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

  if (!data) return null;

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
        Back to feed
      </RouterLink>

      {/* Post */}
      <Card className="p-card-padding">
        <div className="flex items-center gap-3 mb-3">
          <Avatar
            size="md"
            src={post.author_photo_url}
            name={post.author_name}
          />
          <div>
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
              {post.is_edited && " (edited)"}
            </p>
          </div>
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
            aria-label="Delete post"
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
          placeholder="Write a comment..."
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

      {/* Comments */}
      <div className="mt-4 space-y-1">
        {comments.map((comment) => (
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
        title="Delete post?"
        confirmLabel="Delete"
        destructive
        onConfirm={() => deletePost.mutate(post.id)}
        loading={deletePost.isPending}
      >
        This will permanently remove the post and all its comments.
      </ConfirmationDialog>
    </div>
  );
}
