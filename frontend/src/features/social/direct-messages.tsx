import { useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { MessageCircle, Circle } from "lucide-react";
import {
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Avatar,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useConversations } from "@/hooks/use-social";
import type { ConversationSummaryResponse } from "@/hooks/use-social";

// ─── Conversation list item ─────────────────────────────────────────────────

function ConversationItem({
  conversation,
}: {
  conversation: ConversationSummaryResponse;
}) {
  const timeAgo = formatTimeAgo(conversation.updated_at);

  return (
    <RouterLink
      to={`/messages/${conversation.id}`}
      className="flex items-center gap-3 p-card-padding rounded-radius-md hover:bg-surface-container-low transition-colors"
    >
      <div className="relative shrink-0">
        <Avatar size="lg" name={conversation.other_parent_name} />
        {conversation.unread_count > 0 && (
          <div className="absolute -top-0.5 -right-0.5 w-4 h-4 bg-primary rounded-full flex items-center justify-center">
            <span className="type-label-sm text-on-primary text-[10px]">
              {conversation.unread_count > 9
                ? "9+"
                : conversation.unread_count}
            </span>
          </div>
        )}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <p
            className={`type-title-sm truncate ${
              conversation.unread_count > 0
                ? "text-on-surface font-semibold"
                : "text-on-surface"
            }`}
          >
            {conversation.other_parent_name}
          </p>
          <span className="type-label-sm text-on-surface-variant shrink-0 ml-2">
            {timeAgo}
          </span>
        </div>
        {conversation.last_message_preview && (
          <p
            className={`type-body-sm truncate mt-0.5 ${
              conversation.unread_count > 0
                ? "text-on-surface"
                : "text-on-surface-variant"
            }`}
          >
            {conversation.last_message_preview}
          </p>
        )}
      </div>

      {conversation.unread_count > 0 && (
        <Circle className="w-2 h-2 fill-primary text-primary shrink-0" />
      )}
    </RouterLink>
  );
}

// ─── Direct Messages page ───────────────────────────────────────────────────

export function DirectMessages() {
  const intl = useIntl();
  const { data: conversations, isPending } = useConversations();

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "social.messages.title" })} />

      {isPending && (
        <div className="space-y-2">
          {[1, 2, 3, 4].map((n) => (
            <div key={n} className="flex items-center gap-3 p-card-padding">
              <Skeleton className="w-12 h-12 rounded-full" />
              <div className="flex-1">
                <Skeleton className="h-4 w-32 mb-1.5" />
                <Skeleton className="h-3 w-48" />
              </div>
            </div>
          ))}
        </div>
      )}

      {conversations && conversations.length === 0 && (
        <EmptyState
          illustration={<Icon icon={MessageCircle} size="xl" />}
          message={intl.formatMessage({
            id: "social.messages.empty.title",
          })}
          description={intl.formatMessage({
            id: "social.messages.empty.description",
          })}
        />
      )}

      {conversations && conversations.length > 0 && (
        <Card>
          {conversations.map((conv) => (
            <ConversationItem key={conv.id} conversation={conv} />
          ))}
        </Card>
      )}
    </div>
  );
}

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "now";
  if (diffMins < 60) return `${diffMins}m`;
  if (diffHours < 24) return `${diffHours}h`;
  if (diffDays < 7) return `${diffDays}d`;
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}
