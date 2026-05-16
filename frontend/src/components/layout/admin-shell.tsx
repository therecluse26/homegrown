import { type ReactNode, Suspense, useEffect, useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router";
import {
  LayoutDashboard,
  Users,
  Shield,
  Flag,
  FileText,
  Settings,
  Server,
  AlertTriangle,
  ArrowLeft,
  Menu,
  X,
} from "lucide-react";
import { useIntl } from "react-intl";
import { Icon, Spinner } from "@/components/ui";
import { SkipLink } from "@/components/common";

type NavItem = {
  to: string;
  icon: typeof LayoutDashboard;
  label: string;
  end?: boolean;
};

const adminNavItems: NavItem[] = [
  { to: "/admin", icon: LayoutDashboard, label: "Dashboard", end: true },
  { to: "/admin/users", icon: Users, label: "Users" },
  { to: "/admin/moderation", icon: Shield, label: "Moderation" },
  { to: "/admin/reports", icon: AlertTriangle, label: "Safety Reports" },
  { to: "/admin/flags", icon: Flag, label: "Feature Flags" },
  { to: "/admin/audit", icon: FileText, label: "Audit Log" },
  { to: "/admin/methodologies", icon: Settings, label: "Methodologies" },
  { to: "/admin/system", icon: Server, label: "System" },
];

export function AdminShell({ children }: { children?: ReactNode }) {
  const intl = useIntl();
  const location = useLocation();
  const [navOpen, setNavOpen] = useState(false);

  // Close drawer on route change so mobile users don't see stale state
  useEffect(() => {
    setNavOpen(false);
  }, [location.pathname]);

  return (
    <>
      <SkipLink />
      <div className="min-h-screen bg-surface flex">
        {/* Small-viewport top bar with hamburger */}
        <header className="lg:hidden fixed top-0 left-0 right-0 h-14 bg-surface-container-low border-b border-outline-variant flex items-center px-4 gap-3 z-[var(--z-sticky)]">
          <button
            type="button"
            aria-label={intl.formatMessage({
              id: "nav.admin.toggle",
              defaultMessage: "Toggle admin navigation",
            })}
            aria-expanded={navOpen}
            aria-controls="admin-nav"
            onClick={() => setNavOpen((prev) => !prev)}
            className="p-2 -ml-2 rounded-radius-button text-on-surface hover:bg-surface-container-high focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <Icon icon={navOpen ? X : Menu} size="md" />
          </button>
          <p className="type-title-md text-primary font-semibold">
            {intl.formatMessage({ id: "nav.admin", defaultMessage: "Admin" })}
          </p>
        </header>

        {/* Mobile backdrop */}
        {navOpen && (
          <button
            type="button"
            aria-label={intl.formatMessage({
              id: "nav.admin.close",
              defaultMessage: "Close admin navigation",
            })}
            onClick={() => setNavOpen(false)}
            className="lg:hidden fixed inset-0 bg-scrim/40 z-[calc(var(--z-sticky)+1)]"
          />
        )}

        <nav
          id="admin-nav"
          aria-label="Admin navigation"
          className={`flex flex-col fixed top-0 left-0 h-full bg-surface-container-low z-[calc(var(--z-sticky)+2)] transition-transform duration-[var(--duration-normal)] lg:translate-x-0 ${
            navOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
          }`}
          style={{ width: "var(--width-sidebar)" }}
        >
          <div className="p-card-padding">
            <NavLink
              to="/"
              className="flex items-center gap-2 text-on-surface-variant type-label-lg hover:text-primary transition-colors duration-[var(--duration-normal)]"
            >
              <Icon icon={ArrowLeft} size="sm" />
              <span>{intl.formatMessage({ id: "common.back", defaultMessage: "Back" })}</span>
            </NavLink>
            <p className="type-title-md text-primary font-semibold mt-3">
              {intl.formatMessage({ id: "nav.admin", defaultMessage: "Admin" })}
            </p>
          </div>
          <ul className="flex flex-col gap-1 px-3 flex-1">
            {adminNavItems.map((item) => (
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
                  <span>{item.label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className="flex-1 pt-14 lg:pt-0 lg:pl-[var(--width-sidebar)]">
          <div className="max-w-[var(--width-content)] mx-auto px-spacing-page-x lg:px-spacing-page-x-lg py-6">
            <main id="main-content" key={location.pathname}>
              <Suspense
                fallback={
                  <div className="flex items-center justify-center py-12">
                    <Spinner size="lg" className="text-primary" />
                  </div>
                }
              >
                {children ?? <Outlet />}
              </Suspense>
            </main>
          </div>
        </div>
      </div>
    </>
  );
}
