import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, Send, BellOff, Bell, Search, X } from "lucide-react";
import {
  Button,
  Icon,
  Skeleton,
  Avatar,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useMessages,
  useConversations,
  useSendMessage,
  useMarkConversationRead,
  useMuteConversation,
  useUnmuteConversation,
} from "@/hooks/use-social";
import { useAuth } from "@/hooks/use-auth";
import type { MessageResponse } from "@/hooks/use-social";

// ─── Message bubble ─────────────────────────────────────────────────────────

function MessageBubble({
  message,
  isMine,
  searchHighlight,
}: {
  message: MessageResponse;
  isMine: boolean;
  searchHighlight?: string;
}) {
  const content = message.content;

  // Highlight matching search text
  const renderedContent = useMemo(() => {
    if (!searchHighlight || !searchHighlight.trim()) return content;
    const regex = new RegExp(
      `(${searchHighlight.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
      "gi",
    );
    const parts = content.split(regex);
    return parts.map((part, i) =>
      regex.test(part) ? (
        <mark
          key={i}
          className="bg-tertiary-fixed text-on-tertiary-fixed rounded-sm px-0.5"
        >
          {part}
        </mark>
      ) : (
        part
      ),
    );
  }, [content, searchHighlight]);

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
        <p className="type-body-md whitespace-pre-wrap">{renderedContent}</p>
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
  const [showSearch, setShowSearch] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const { data: messages, isPending } = useMessages(conversationId);
  const { data: conversations } = useConversations();
  const sendMessage = useSendMessage(conversationId ?? "");
  const markRead = useMarkConversationRead(conversationId ?? "");
  const muteConversation = useMuteConversation(conversationId ?? "");
  const unmuteConversation = useUnmuteConversation(conversationId ?? "");

  const conversation = conversations?.find((c) => c.id === conversationId);
  const otherName = conversation?.other_parent_name ?? "";
  const isMuted = conversation?.is_muted ?? false;

  // Filter messages by search query
  const filteredMessages = useMemo(() => {
    if (!messages) return undefined;
    if (!searchQuery.trim()) return messages;
    const q = searchQuery.toLowerCase();
    return messages.filter((m) => m.content.toLowerCase().includes(q));
  }, [messages, searchQuery]);

  const searchMatchCount = useMemo(() => {
    if (!searchQuery.trim() || !messages) return 0;
    const q = searchQuery.toLowerCase();
    return messages.filter((m) => m.content.toLowerCase().includes(q)).length;
  }, [messages, searchQuery]);

  // Mark as read on mount
  useEffect(() => {
    if (conversationId && conversation && conversation.unread_count > 0) {
      markRead.mutate();
    }
  }, [conversationId]); // Only re-run when conversation changes, not on every markRead ref update

  // Scroll to bottom when messages change (but not when searching)
  useEffect(() => {
    if (!showSearch) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, showSearch]);

  // Ctrl+F to toggle search
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === "f") {
        e.preventDefault();
        setShowSearch((prev) => !prev);
        if (!showSearch) {
          setTimeout(() => searchInputRef.current?.focus(), 0);
        }
      }
      if (e.key === "Escape" && showSearch) {
        setShowSearch(false);
        setSearchQuery("");
      }
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [showSearch]);

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

  const handleMuteToggle = useCallback(() => {
    if (isMuted) {
      unmuteConversation.mutate();
    } else {
      muteConversation.mutate();
    }
  }, [isMuted, muteConversation, unmuteConversation]);

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
        <div className="flex-1 min-w-0">
          <p className="type-title-sm text-on-surface">{otherName}</p>
          {isMuted && (
            <Badge variant="secondary">
              <FormattedMessage id="social.messages.muted" />
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={() => {
              setShowSearch((prev) => !prev);
              if (!showSearch) {
                setTimeout(() => searchInputRef.current?.focus(), 0);
              } else {
                setSearchQuery("");
              }
            }}
            className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-low transition-colors"
            aria-label={intl.formatMessage({
              id: "social.messages.search",
            })}
          >
            <Icon icon={Search} size="sm" />
          </button>
          <button
            onClick={handleMuteToggle}
            disabled={muteConversation.isPending || unmuteConversation.isPending}
            className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-low transition-colors"
            aria-label={intl.formatMessage({
              id: isMuted
                ? "social.messages.unmute"
                : "social.messages.mute",
            })}
          >
            <Icon icon={isMuted ? Bell : BellOff} size="sm" />
          </button>
        </div>
      </div>

      {/* Search bar */}
      {showSearch && (
        <div className="flex items-center gap-2 py-2 px-1 border-b border-outline-variant/10">
          <Icon
            icon={Search}
            size="sm"
            className="text-on-surface-variant shrink-0"
          />
          <input
            ref={searchInputRef}
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={intl.formatMessage({
              id: "social.messages.search.placeholder",
            })}
            className="flex-1 bg-transparent text-on-surface type-body-sm placeholder:text-on-surface-variant focus:outline-none"
            aria-label={intl.formatMessage({
              id: "social.messages.search.placeholder",
            })}
          />
          {searchQuery && (
            <span className="type-label-sm text-on-surface-variant shrink-0">
              <FormattedMessage
                id="social.messages.search.count"
                values={{ count: searchMatchCount }}
              />
            </span>
          )}
          <button
            onClick={() => {
              setShowSearch(false);
              setSearchQuery("");
            }}
            className="p-1 rounded-radius-sm text-on-surface-variant hover:text-on-surface transition-colors"
            aria-label={intl.formatMessage({ id: "common.close" })}
          >
            <Icon icon={X} size="xs" />
          </button>
        </div>
      )}

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

        {filteredMessages?.map((msg) => (
          <MessageBubble
            key={msg.id}
            message={msg}
            isMine={msg.sender_parent_id === user?.parent_id}
            searchHighlight={searchQuery}
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
