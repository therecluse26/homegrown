import { useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  UserPlus,
  UserCheck,
  MessageCircle,
  AlertTriangle,
  CalendarX,
  Info,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import {
  useNotifications,
  useMarkRead,
  useMarkAllRead,
  type NotificationType,
} from "@/hooks/use-notifications";

// ─── Notification type → icon/color mapping ────────────────────────────────

const NOTIFICATION_CONFIG: Record<
  string,
  { icon: typeof Info; colorClass: string }
> = {
  friend_request_received: {
    icon: UserPlus,
    colorClass: "text-primary",
  },
  friend_request_accepted: {
    icon: UserCheck,
    colorClass: "text-primary",
  },
  message_received: {
    icon: MessageCircle,
    colorClass: "text-secondary",
  },
  content_flagged: {
    icon: AlertTriangle,
    colorClass: "text-warning",
  },
  event_cancelled: {
    icon: CalendarX,
    colorClass: "text-error",
  },
  system: {
    icon: Info,
    colorClass: "text-on-surface-variant",
  },
};

function getConfig(type: NotificationType) {
  return (
    NOTIFICATION_CONFIG[type] ?? {
      icon: Info,
      colorClass: "text-on-surface-variant",
    }
  );
}

function formatTimeAgo(dateStr: string, intl: ReturnType<typeof useIntl>) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1)
    return intl.formatMessage({ id: "notifications.time.justNow" });
  if (minutes < 60)
    return intl.formatMessage(
      { id: "notifications.time.minutes" },
      { count: minutes },
    );
  const hours = Math.floor(minutes / 60);
  if (hours < 24)
    return intl.formatMessage(
      { id: "notifications.time.hours" },
      { count: hours },
    );
  const days = Math.floor(hours / 24);
  return intl.formatMessage(
    { id: "notifications.time.days" },
    { count: days },
  );
}

export function NotificationCenter() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data, isPending, error } = useNotifications();
  const markRead = useMarkRead();
  const markAllRead = useMarkAllRead();

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "notifications.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const notifications = data?.notifications ?? [];
  const unreadCount = data?.unread_count ?? 0;

  function handleMarkRead(id: string) {
    markRead.mutate(id);
  }

  function handleMarkAllRead() {
    markAllRead.mutate();
  }

  if (isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <div className="flex items-center justify-between mb-6">
          <Skeleton height="h-8" width="w-48" />
        </div>
        <div className="flex flex-col gap-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-md text-on-surface font-semibold mb-6 outline-none">
          <FormattedMessage id="notifications.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-center justify-between mb-6">
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-md text-on-surface font-semibold outline-none">
          <FormattedMessage id="notifications.title" />
        </h1>
        {unreadCount > 0 && (
          <Button
            variant="tertiary"
            size="sm"
            onClick={handleMarkAllRead}
            loading={markAllRead.isPending}
          >
            <FormattedMessage id="notifications.markAllRead" />
          </Button>
        )}
      </div>

      {notifications.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "notifications.empty" })}
          description={intl.formatMessage({
            id: "notifications.empty.description",
          })}
        />
      ) : (
        <ul className="flex flex-col gap-2" role="list">
          {notifications.map((notification) => {
            const config = getConfig(notification.type);
            const content = (
              <Card
                className={`flex items-start gap-3 transition-colors ${
                  notification.read
                    ? ""
                    : "bg-surface-container-low ring-1 ring-primary/10"
                }`}
                interactive={!!notification.deep_link}
              >
                <div
                  className={`mt-0.5 shrink-0 ${config.colorClass}`}
                >
                  <Icon icon={config.icon} size="md" aria-hidden />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="type-title-sm text-on-surface font-medium">
                    {notification.title}
                  </p>
                  <p className="type-body-sm text-on-surface-variant line-clamp-2">
                    {notification.body}
                  </p>
                  <p className="type-label-sm text-on-surface-variant mt-1">
                    {formatTimeAgo(notification.created_at, intl)}
                  </p>
                </div>
                {!notification.read && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      handleMarkRead(notification.id);
                    }}
                    className="shrink-0 p-1.5 rounded-full text-on-surface-variant hover:text-primary hover:bg-primary-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                    aria-label={intl.formatMessage({
                      id: "notifications.markRead",
                    })}
                  >
                    <span className="block w-2.5 h-2.5 rounded-full bg-primary" />
                  </button>
                )}
              </Card>
            );

            return (
              <li key={notification.id}>
                {notification.deep_link ? (
                  <RouterLink
                    to={notification.deep_link}
                    className="block no-underline"
                  >
                    {content}
                  </RouterLink>
                ) : (
                  content
                )}
              </li>
            );
          })}
        </ul>
      )}

      {notifications.length > 0 && (
        <div className="mt-6 text-center">
          <RouterLink
            to="/settings/notifications"
            className="type-label-md text-primary hover:text-primary-container transition-colors"
          >
            <FormattedMessage id="notifications.managePreferences" />
          </RouterLink>
        </div>
      )}
    </div>
  );
}
