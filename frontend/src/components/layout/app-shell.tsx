import { type ReactNode } from "react";
import { NavLink, Outlet, useLocation } from "react-router";
import {
  Home,
  BookOpen,
  Users,
  ShoppingBag,
  Calendar,
  Settings,
  Search,
  Bell,
} from "lucide-react";
import { useIntl } from "react-intl";
import { Icon } from "@/components/ui";
import { SkipLink } from "@/components/common";
import { useAuthContext } from "@/features/auth/auth-provider";

type NavItem = {
  to: string;
  icon: typeof Home;
  labelId: string;
  end?: boolean;
};

const navItems: NavItem[] = [
  { to: "/", icon: Home, labelId: "nav.home", end: true },
  { to: "/learning", icon: BookOpen, labelId: "nav.learning" },
  { to: "/friends", icon: Users, labelId: "nav.social" },
  { to: "/marketplace", icon: ShoppingBag, labelId: "nav.marketplace" },
  { to: "/calendar", icon: Calendar, labelId: "nav.calendar" },
  { to: "/settings", icon: Settings, labelId: "nav.settings" },
];

function SidebarNav() {
  const intl = useIntl();

  return (
    <nav
      aria-label={intl.formatMessage({ id: "nav.home", defaultMessage: "Main navigation" })}
      className="hidden lg:flex flex-col fixed top-0 left-0 h-full bg-surface-container-low/80 backdrop-blur-[20px] z-[var(--z-sticky)]"
      style={{ width: "var(--width-sidebar)" }}
    >
      <div className="p-card-padding">
        <p className="type-title-md text-primary font-semibold">
          {intl.formatMessage({ id: "app.name", defaultMessage: "Homegrown Academy" })}
        </p>
      </div>
      <ul className="flex flex-col gap-1 px-3 flex-1">
        {navItems.map((item) => (
          <li key={item.to}>
            <NavLink
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-radius-button type-label-lg text-on-surface-variant transition-colors duration-[var(--duration-normal)] ${
                  isActive
                    ? "bg-primary/10 text-primary font-semibold"
                    : "hover:bg-surface-container-high"
                }`
              }
            >
              <Icon icon={item.icon} size="md" />
              <span>{intl.formatMessage({ id: item.labelId })}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}

function BottomNav() {
  const intl = useIntl();

  return (
    <nav
      aria-label="Main navigation"
      className="lg:hidden fixed bottom-0 left-0 right-0 bg-surface-container-low/80 backdrop-blur-[20px] z-[var(--z-sticky)] safe-area-pb"
    >
      <ul className="flex justify-around items-center h-16">
        {navItems.slice(0, 5).map((item) => (
          <li key={item.to}>
            <NavLink
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 px-3 py-1.5 min-w-[3rem] rounded-radius-button transition-colors duration-[var(--duration-normal)] ${
                  isActive ? "text-primary" : "text-on-surface-variant"
                }`
              }
            >
              <Icon icon={item.icon} size="md" />
              <span className="type-label-sm">
                {intl.formatMessage({ id: item.labelId })}
              </span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}

function Header() {
  const intl = useIntl();
  const { user } = useAuthContext();

  return (
    <header className="flex items-center justify-between py-4 lg:py-6">
      <div className="lg:hidden">
        <p className="type-title-md text-primary font-semibold">
          {intl.formatMessage({ id: "app.name", defaultMessage: "Homegrown Academy" })}
        </p>
      </div>
      <div className="flex items-center gap-3 ml-auto">
        <NavLink
          to="/search"
          className="p-2 rounded-radius-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
          aria-label={intl.formatMessage({ id: "nav.search", defaultMessage: "Search" })}
        >
          <Icon icon={Search} size="md" />
        </NavLink>
        <NavLink
          to="/notifications"
          className="p-2 rounded-radius-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
          aria-label="Notifications"
        >
          <Icon icon={Bell} size="md" />
        </NavLink>
        <div className="type-label-md text-on-surface-variant">
          {user?.display_name ?? ""}
        </div>
      </div>
    </header>
  );
}

export function AppShell({ children }: { children?: ReactNode }) {
  const location = useLocation();

  return (
    <>
      <SkipLink />
      <SidebarNav />
      <BottomNav />
      <div
        className="min-h-screen bg-surface lg:pl-[var(--width-sidebar)]"
        data-context="parent"
      >
        <div className="max-w-[var(--width-content)] mx-auto px-spacing-page-x lg:px-spacing-page-x-lg">
          <Header />
          <main id="main-content" key={location.pathname}>
            {children ?? <Outlet />}
          </main>
          <div className="h-20 lg:h-8" />
        </div>
      </div>
    </>
  );
}
