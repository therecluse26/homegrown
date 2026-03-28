import { useState, useRef, useEffect, useCallback } from "react";
import { useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, Send } from "lucide-react";
import {
  Button,
  Icon,
  Skeleton,
  Avatar,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useMessages,
  useConversations,
  useSendMessage,
  useMarkConversationRead,
} from "@/hooks/use-social";
import { useAuth } from "@/hooks/use-auth";
import type { MessageResponse } from "@/hooks/use-social";

// ─── Message bubble ─────────────────────────────────────────────────────────

function MessageBubble({
  message,
  isMine,
}: {
  message: MessageResponse;
  isMine: boolean;
}) {
  return (
    <div
      className={`flex ${isMine ? "justify-end" : "justify-start"} mb-2`}
    >
      <div
        className={`max-w-[75%] px-4 py-2.5 rounded-radius-lg ${
          isMine
            ? "bg-primary text-on-primary rounded-br-radius-xs"
            : "bg-surface-container-high text-on-surface rounded-bl-radius-xs"
        }`}
      >
        {!isMine && (
          <p className="type-label-sm font-semibold mb-0.5">
            {message.sender_name}
          </p>
        )}
        <p className="type-body-md whitespace-pre-wrap">{message.content}</p>
        <p
          className={`type-label-sm mt-1 ${
            isMine ? "text-on-primary/70" : "text-on-surface-variant"
          }`}
        >
          {new Date(message.created_at).toLocaleTimeString(undefined, {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      </div>
    </div>
  );
}

// ─── Conversation page ──────────────────────────────────────────────────────

export function Conversation() {
  const intl = useIntl();
  const { conversationId } = useParams<{ conversationId: string }>();
  const { user } = useAuth();
  const [messageText, setMessageText] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const { data: messages, isPending } = useMessages(conversationId);
  const { data: conversations } = useConversations();
  const sendMessage = useSendMessage(conversationId ?? "");
  const markRead = useMarkConversationRead(conversationId ?? "");

  // Find the other participant's name
  const conversation = conversations?.find((c) => c.id === conversationId);
  const otherName = conversation?.other_parent_name ?? "";

  // Mark as read on mount
  useEffect(() => {
    if (conversationId && conversation && conversation.unread_count > 0) {
      markRead.mutate();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [conversationId]);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!messageText.trim()) return;

      sendMessage.mutate(
        { content: messageText.trim() },
        { onSuccess: () => setMessageText("") },
      );
    },
    [messageText, sendMessage],
  );

  return (
    <div className="max-w-content-narrow mx-auto flex flex-col h-[calc(100vh-8rem)]">
      <PageTitle title={otherName || intl.formatMessage({ id: "social.messages.title" })} />

      {/* Header */}
      <div className="flex items-center gap-3 pb-4 border-b border-outline-variant/10">
        <RouterLink
          to="/messages"
          className="p-2 rounded-radius-sm hover:bg-surface-container-low transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
        </RouterLink>
        <Avatar size="md" name={otherName || "?"} />
        <p className="type-title-sm text-on-surface">{otherName}</p>
      </div>

      {/* Messages area */}
      <div className="flex-1 overflow-y-auto py-4 space-y-1">
        {isPending && (
          <div className="space-y-3 p-4">
            {[1, 2, 3].map((n) => (
              <div key={n} className={`flex ${n % 2 === 0 ? "justify-end" : "justify-start"}`}>
                <Skeleton className="h-12 w-48 rounded-radius-lg" />
              </div>
            ))}
          </div>
        )}

        {messages?.map((msg) => (
          <MessageBubble
            key={msg.id}
            message={msg}
            isMine={msg.sender_parent_id === user?.parent_id}
          />
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Composer */}
      <form
        onSubmit={handleSend}
        className="flex items-end gap-2 pt-3 border-t border-outline-variant/10"
      >
        <textarea
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSend(e);
            }
          }}
          placeholder={intl.formatMessage({
            id: "social.messages.composer.placeholder",
          })}
          className="flex-1 min-h-[44px] max-h-32 resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
          aria-label={intl.formatMessage({
            id: "social.messages.composer.placeholder",
          })}
          rows={1}
        />
        <Button
          type="submit"
          variant="primary"
          size="md"
          disabled={!messageText.trim() || sendMessage.isPending}
          aria-label={intl.formatMessage({
            id: "social.messages.composer.send",
          })}
        >
          <Icon icon={Send} size="sm" />
        </Button>
      </form>
    </div>
  );
}
