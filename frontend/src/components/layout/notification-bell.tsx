import { NavLink } from "react-router";
import { Bell } from "lucide-react";
import { useIntl } from "react-intl";
import { Icon } from "@/components/ui";
import { useUnreadCount } from "@/hooks/use-notifications";

export function NotificationBell() {
  const intl = useIntl();
  const { data } = useUnreadCount();
  const count = data?.count ?? 0;

  return (
    <NavLink
      to="/notifications"
      className="relative p-2 rounded-radius-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
      aria-label={intl.formatMessage(
        { id: "notifications.bell.label" },
        { count },
      )}
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
    </NavLink>
  );
}
