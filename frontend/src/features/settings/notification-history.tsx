import { useState, useCallback, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  UserPlus,
  UserCheck,
  MessageCircle,
  AlertTriangle,
  CalendarX,
  Info,
  Filter,
  X,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Select,
  Skeleton,
} from "@/components/ui";
import {
  useNotifications,
  useMarkRead,
  useMarkAllRead,
  type NotificationType,
} from "@/hooks/use-notifications";

// ─── Notification type → icon/color mapping ─────────��──────────────────────

const NOTIFICATION_CONFIG: Record<
  string,
  { icon: typeof Info; colorClass: string }
> = {
  friend_request_received: { icon: UserPlus, colorClass: "text-primary" },
  friend_request_accepted: { icon: UserCheck, colorClass: "text-primary" },
  message_received: { icon: MessageCircle, colorClass: "text-secondary" },
  content_flagged: { icon: AlertTriangle, colorClass: "text-warning" },
  event_cancelled: { icon: CalendarX, colorClass: "text-error" },
  system: { icon: Info, colorClass: "text-on-surface-variant" },
};

function getConfig(type: NotificationType) {
  return (
    NOTIFICATION_CONFIG[type] ?? {
      icon: Info,
      colorClass: "text-on-surface-variant",
    }
  );
}

// ─── Notification type options for filter ─────────────────────────────────

const NOTIFICATION_TYPE_OPTIONS: { value: NotificationType; labelId: string }[] =
  [
    { value: "friend_request_received", labelId: "notificationHistory.type.friendRequest" },
    { value: "friend_request_accepted", labelId: "notificationHistory.type.friendAccepted" },
    { value: "message_received", labelId: "notificationHistory.type.message" },
    { value: "content_flagged", labelId: "notificationHistory.type.flagged" },
    { value: "event_cancelled", labelId: "notificationHistory.type.eventCancelled" },
    { value: "purchase_completed", labelId: "notificationHistory.type.purchase" },
    { value: "review_received", labelId: "notificationHistory.type.review" },
    { value: "subscription_created", labelId: "notificationHistory.type.subscriptionCreated" },
    { value: "subscription_cancelled", labelId: "notificationHistory.type.subscriptionCancelled" },
    { value: "subscription_renewed", labelId: "notificationHistory.type.subscriptionRenewed" },
    { value: "streak_milestone", labelId: "notificationHistory.type.streak" },
    { value: "learning_milestone", labelId: "notificationHistory.type.learningMilestone" },
    { value: "attendance_threshold_warning", labelId: "notificationHistory.type.attendance" },
    { value: "payout_completed", labelId: "notificationHistory.type.payout" },
    { value: "system", labelId: "notificationHistory.type.system" },
  ];

// ─── Time formatting ──────────────────────────────────────────────────────

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

// ─── Component ──────────��─────────────────────────────────────────────────

export function NotificationHistory() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const [page, setPage] = useState(1);
  const [typeFilter, setTypeFilter] = useState<NotificationType | "">("");
  const [readFilter, setReadFilter] = useState<"" | "true" | "false">("");
  const [showFilters, setShowFilters] = useState(false);

  const queryParams = {
    page,
    type: typeFilter || undefined,
    read: readFilter === "" ? undefined : readFilter === "true",
  };

  const { data, isPending, error } = useNotifications(queryParams);
  const markRead = useMarkRead();
  const markAllRead = useMarkAllRead();

  const notifications = data?.notifications ?? [];
  const total = data?.total ?? 0;
  const unreadCount = data?.unread_count ?? 0;

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "notificationHistory.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const hasActiveFilters = typeFilter !== "" || readFilter !== "";
  const pageSize = 20;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleClearFilters = useCallback(() => {
    setTypeFilter("");
    setReadFilter("");
    setPage(1);
  }, []);

  function handleMarkRead(id: string) {
    markRead.mutate(id);
  }

  function handleMarkAllRead() {
    markAllRead.mutate();
  }

  // ─── Loading state ────────────────────────────────────────────────────

  if (isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <div className="flex items-center justify-between mb-6">
          <Skeleton height="h-8" width="w-56" />
        </div>
        <div className="flex flex-col gap-3">
          <Skeleton height="h-16" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </div>
    );
  }

  // ─── Error state ──────────���───────────────────────────────────────────

  if (error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-md text-on-surface font-semibold outline-none mb-6">
          <FormattedMessage id="notificationHistory.title" />
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
    <div className="mx-auto max-w-3xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-6 gap-4">
        <div>
          <h1 ref={headingRef} tabIndex={-1} className="type-headline-md text-on-surface font-semibold outline-none">
            <FormattedMessage id="notificationHistory.title" />
          </h1>
          <p className="type-body-sm text-on-surface-variant mt-1">
            <FormattedMessage
              id="notificationHistory.description"
              values={{ total }}
            />
          </p>
        </div>
        <div className="flex items-center gap-2">
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
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setShowFilters((prev) => !prev)}
            aria-expanded={showFilters}
            aria-controls="notification-filters"
          >
            <Icon icon={Filter} size="sm" aria-hidden />
            <span className="ml-1.5">
              <FormattedMessage id="notificationHistory.filters" />
            </span>
            {hasActiveFilters && (
              <span className="ml-1.5 inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary text-on-primary type-label-sm">
                {(typeFilter ? 1 : 0) + (readFilter ? 1 : 0)}
              </span>
            )}
          </Button>
        </div>
      </div>

      {/* Filter bar */}
      {showFilters && (
        <Card
          id="notification-filters"
          className="mb-4 bg-surface-container-low"
        >
          <div className="flex flex-wrap items-end gap-4">
            {/* Type filter */}
            <div className="flex-1 min-w-[180px]">
              <label
                htmlFor="filter-type"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="notificationHistory.filter.type" />
              </label>
              <Select
                id="filter-type"
                value={typeFilter}
                onChange={(e) => {
                  setTypeFilter(
                    e.target.value as NotificationType | "",
                  );
                  setPage(1);
                }}
              >
                <option value="">
                  {intl.formatMessage({
                    id: "notificationHistory.filter.allTypes",
                  })}
                </option>
                {NOTIFICATION_TYPE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {intl.formatMessage({ id: opt.labelId })}
                  </option>
                ))}
              </Select>
            </div>

            {/* Read/unread filter */}
            <div className="flex-1 min-w-[140px]">
              <label
                htmlFor="filter-read"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="notificationHistory.filter.status" />
              </label>
              <Select
                id="filter-read"
                value={readFilter}
                onChange={(e) => {
                  setReadFilter(
                    e.target.value as "" | "true" | "false",
                  );
                  setPage(1);
                }}
              >
                <option value="">
                  {intl.formatMessage({
                    id: "notificationHistory.filter.allStatus",
                  })}
                </option>
                <option value="false">
                  {intl.formatMessage({
                    id: "notificationHistory.filter.unread",
                  })}
                </option>
                <option value="true">
                  {intl.formatMessage({
                    id: "notificationHistory.filter.read",
                  })}
                </option>
              </Select>
            </div>

            {/* Clear filters */}
            {hasActiveFilters && (
              <Button
                variant="tertiary"
                size="sm"
                onClick={handleClearFilters}
                className="mb-0.5"
              >
                <Icon icon={X} size="sm" aria-hidden />
                <span className="ml-1">
                  <FormattedMessage id="notificationHistory.filter.clear" />
                </span>
              </Button>
            )}
          </div>
        </Card>
      )}

      {/* Notification list */}
      {notifications.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({
            id: hasActiveFilters
              ? "notificationHistory.empty.filtered"
              : "notificationHistory.empty",
          })}
          description={intl.formatMessage({
            id: hasActiveFilters
              ? "notificationHistory.empty.filtered.description"
              : "notificationHistory.empty.description",
          })}
          action={
            hasActiveFilters ? (
              <Button variant="secondary" size="sm" onClick={handleClearFilters}>
                <FormattedMessage id="notificationHistory.filter.clear" />
              </Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <ul
            className="flex flex-col gap-2"
            role="list"
            aria-label={intl.formatMessage({
              id: "notificationHistory.list.label",
            })}
          >
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

          {/* Pagination */}
          {totalPages > 1 && (
            <nav
              className="flex items-center justify-center gap-2 mt-6"
              aria-label={intl.formatMessage({
                id: "notificationHistory.pagination.label",
              })}
            >
              <Button
                variant="tertiary"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                <FormattedMessage id="notificationHistory.pagination.prev" />
              </Button>
              <span className="type-label-md text-on-surface-variant px-3">
                <FormattedMessage
                  id="notificationHistory.pagination.info"
                  values={{ page, totalPages }}
                />
              </span>
              <Button
                variant="tertiary"
                size="sm"
                disabled={page >= totalPages}
                onClick={() =>
                  setPage((p) => Math.min(totalPages, p + 1))
                }
              >
                <FormattedMessage id="notificationHistory.pagination.next" />
              </Button>
            </nav>
          )}
        </>
      )}

      {/* Footer link */}
      <div className="mt-6 flex justify-center gap-4">
        <RouterLink
          to="/notifications"
          className="type-label-md text-primary hover:text-primary-container transition-colors"
        >
          <FormattedMessage id="notificationHistory.backToRecent" />
        </RouterLink>
        <RouterLink
          to="/settings/notifications"
          className="type-label-md text-primary hover:text-primary-container transition-colors"
        >
          <FormattedMessage id="notifications.managePreferences" />
        </RouterLink>
      </div>
    </div>
  );
}
