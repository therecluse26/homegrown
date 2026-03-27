import { type ReactNode } from "react";
import { NavLink, Outlet, useLocation } from "react-router";
import { ArrowLeft, BookOpen, Home, Trophy } from "lucide-react";
import { useIntl } from "react-intl";
import { Icon } from "@/components/ui";
import { SkipLink } from "@/components/common";

type NavItem = {
  to: string;
  icon: typeof Home;
  label: string;
  end?: boolean;
};

const studentNavItems: NavItem[] = [
  { to: "/student", icon: Home, label: "Home", end: true },
  { to: "/student/quiz", icon: BookOpen, label: "Quizzes" },
  { to: "/student/progress", icon: Trophy, label: "Progress" },
];

export function StudentShell({ children }: { children?: ReactNode }) {
  const intl = useIntl();
  const location = useLocation();

  return (
    <>
      <SkipLink />
      <div className="min-h-screen bg-surface" data-context="student">
        <header className="flex items-center gap-3 p-spacing-page-x py-4 bg-surface-container-low/80 backdrop-blur-[20px] sticky top-0 z-[var(--z-sticky)]">
          <NavLink
            to="/"
            className="flex items-center gap-2 px-3 py-2 rounded-radius-xl text-primary type-label-lg hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
          >
            <Icon icon={ArrowLeft} size="md" />
            <span>{intl.formatMessage({ id: "common.back", defaultMessage: "Back to Parent" })}</span>
          </NavLink>
        </header>
        <div className="max-w-[var(--width-content)] mx-auto px-spacing-page-x">
          <main id="main-content" key={location.pathname}>
            {children ?? <Outlet />}
          </main>
        </div>
        <nav
          aria-label="Student navigation"
          className="fixed bottom-0 left-0 right-0 bg-surface-container-low/80 backdrop-blur-[20px] z-[var(--z-sticky)]"
        >
          <ul className="flex justify-around items-center h-16">
            {studentNavItems.map((item) => (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    `flex flex-col items-center gap-0.5 px-4 py-2 min-w-[4rem] rounded-radius-xl transition-colors duration-[var(--duration-normal)] ${
                      isActive ? "text-primary" : "text-on-surface-variant"
                    }`
                  }
                >
                  <Icon icon={item.icon} size="lg" />
                  <span className="type-label-md">{item.label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className="h-20" />
      </div>
    </>
  );
}
