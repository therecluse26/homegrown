import { useState, useRef, useEffect, useCallback } from "react";
import { Link as RouterLink } from "react-router";
import {
  Bell,
  Info,
  UserPlus,
  UserCheck,
  MessageCircle,
  AlertTriangle,
  CalendarX,
  BookOpen,
  ShoppingBag,
  Star,
} from "lucide-react";
import { FormattedMessage, useIntl } from "react-intl";
import { Icon } from "@/components/ui";
import {
  useUnreadCount,
  useNotifications,
  useMarkAllRead,
} from "@/hooks/use-notifications";

const NOTIF_CONFIG: Record<string, { icon: typeof Info; colorClass: string }> =
  {
    friend_request_sent: { icon: UserPlus, colorClass: "text-primary" },
    friend_request_accepted: { icon: UserCheck, colorClass: "text-primary" },
    message_received: { icon: MessageCircle, colorClass: "text-secondary" },
    content_flagged: { icon: AlertTriangle, colorClass: "text-warning" },
    event_cancelled: { icon: CalendarX, colorClass: "text-error" },
    book_completed: { icon: BookOpen, colorClass: "text-tertiary" },
    milestone_achieved: { icon: Star, colorClass: "text-tertiary" },
    activity_streak: { icon: Star, colorClass: "text-tertiary" },
    purchase_completed: { icon: ShoppingBag, colorClass: "text-secondary" },
    purchase_refunded: {
      icon: ShoppingBag,
      colorClass: "text-on-surface-variant",
    },
  };

function getIcon(type: string) {
  return NOTIF_CONFIG[type] ?? { icon: Info, colorClass: "text-on-surface-variant" };
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
  return intl.formatMessage(
    { id: "notifications.time.days" },
    { count: Math.floor(hours / 24) },
  );
}

export function NotificationBell() {
  const intl = useIntl();
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const { data: countData } = useUnreadCount();
  const count = countData?.count ?? 0;

  const { data } = useNotifications({ limit: 5 });
  const recent = data?.notifications?.slice(0, 5) ?? [];

  const markAllRead = useMarkAllRead();

  const close = useCallback(() => setOpen(false), []);

  useEffect(() => {
    if (!open) return;
    function handleOutside(e: MouseEvent) {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(e.target as Node)
      ) {
        close();
      }
    }
    function handleEsc(e: KeyboardEvent) {
      if (e.key === "Escape") close();
    }
    document.addEventListener("mousedown", handleOutside);
    document.addEventListener("keydown", handleEsc);
    return () => {
      document.removeEventListener("mousedown", handleOutside);
      document.removeEventListener("keydown", handleEsc);
    };
  }, [open, close]);

  return (
    <div ref={wrapperRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="relative p-2 min-w-11 min-h-11 flex items-center justify-center rounded-radius-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
        aria-label={intl.formatMessage(
          { id: "notifications.bell.label" },
          { count },
        )}
        aria-expanded={open}
        aria-haspopup="dialog"
      >
        <Icon icon={Bell} size="md" />
        {count > 0 && (
          <span
            aria-hidden="true"
            className="absolute top-1 right-1 flex items-center justify-center min-w-[1.125rem] h-[1.125rem] px-1 rounded-full bg-error text-on-error type-label-sm font-semibold leading-none"
          >
            {count > 99 ? "99+" : count}
          </span>
        )}
        <span className="sr-only" aria-live="polite" aria-atomic="true">
          {count > 0
            ? intl.formatMessage({ id: "notifications.unread.sr" }, { count })
            : ""}
        </span>
      </button>

      {open && (
        <div
          role="dialog"
          aria-label={intl.formatMessage({ id: "notifications.title" })}
          className="absolute right-0 top-full mt-2 w-80 max-h-[28rem] overflow-y-auto bg-surface-container rounded-radius-card shadow-elevation-2 ring-1 ring-outline-variant z-[var(--z-popover)] flex flex-col"
        >
          <div className="flex items-center justify-between px-4 py-3 border-b border-outline-variant sticky top-0 bg-surface-container">
            <span className="type-title-sm text-on-surface font-semibold">
              <FormattedMessage id="notifications.title" />
            </span>
            {count > 0 && (
              <button
                type="button"
                onClick={() => markAllRead.mutate()}
                disabled={markAllRead.isPending}
                className="type-label-sm text-primary hover:text-primary/80 transition-colors disabled:opacity-disabled"
              >
                <FormattedMessage id="notifications.markAllRead" />
              </button>
            )}
          </div>

          {recent.length === 0 ? (
            <p className="px-4 py-8 text-center type-body-sm text-on-surface-variant">
              <FormattedMessage id="notifications.bell.noRecent" />
            </p>
          ) : (
            <ul className="flex flex-col divide-y divide-outline-variant/40">
              {recent.map((n) => {
                const cfg = getIcon(n.notification_type);
                const row = (
                  <div
                    className={`flex items-start gap-3 px-4 py-3 ${!n.is_read ? "bg-surface-container-low" : ""}`}
                  >
                    <span className={`mt-0.5 shrink-0 ${cfg.colorClass}`}>
                      <Icon icon={cfg.icon} size="sm" aria-hidden />
                    </span>
                    <div className="flex-1 min-w-0">
                      <p
                        className={`type-label-md text-on-surface truncate ${!n.is_read ? "font-semibold" : ""}`}
                      >
                        {n.title}
                      </p>
                      <p className="type-label-sm text-on-surface-variant mt-0.5">
                        {formatTimeAgo(n.created_at, intl)}
                      </p>
                    </div>
                    {!n.is_read && (
                      <>
                        <span
                          className="mt-1.5 shrink-0 w-2 h-2 rounded-full bg-primary"
                          aria-hidden
                        />
                        <span className="sr-only">
                          {intl.formatMessage({ id: "notifications.bell.unread.indicator" })}
                        </span>
                      </>
                    )}
                  </div>
                );
                return (
                  <li key={n.id}>
                    {n.action_url ? (
                      <RouterLink
                        to={n.action_url}
                        className="block no-underline hover:bg-surface-container-highest transition-colors"
                        onClick={close}
                      >
                        {row}
                      </RouterLink>
                    ) : (
                      <div className="hover:bg-surface-container-highest transition-colors">
                        {row}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}

          <div className="border-t border-outline-variant/40 px-4 py-2.5 text-center sticky bottom-0 bg-surface-container">
            <RouterLink
              to="/notifications"
              className="type-label-sm text-primary hover:text-primary/80 transition-colors"
              onClick={close}
            >
              <FormattedMessage id="notifications.bell.viewAll" />
            </RouterLink>
          </div>
        </div>
      )}
    </div>
  );
}
