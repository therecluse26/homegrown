import { type ReactNode, Suspense } from "react";
import { NavLink, Outlet, useLocation } from "react-router";
import {
  LayoutDashboard,
  Users,
  Shield,
  Flag,
  FileText,
  Settings,
  ArrowLeft,
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
  { to: "/admin/flags", icon: Flag, label: "Feature Flags" },
  { to: "/admin/audit", icon: FileText, label: "Audit Log" },
  { to: "/admin/methodologies", icon: Settings, label: "Methodologies" },
];

export function AdminShell({ children }: { children?: ReactNode }) {
  const intl = useIntl();
  const location = useLocation();

  return (
    <>
      <SkipLink />
      <div className="min-h-screen bg-surface flex">
        <nav
          aria-label="Admin navigation"
          className="hidden lg:flex flex-col fixed top-0 left-0 h-full bg-surface-container-low z-[var(--z-sticky)]"
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
        <div className="flex-1 lg:pl-[var(--width-sidebar)]">
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
