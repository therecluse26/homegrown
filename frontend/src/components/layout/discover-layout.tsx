import { Suspense } from "react";
import { Outlet, Link, NavLink } from "react-router";
import { SkipLink } from "@/components/common";
import { Button } from "@/components/ui";

/**
 * Public layout for discovery pages (quiz, state guides).
 * No authentication required. Provides brand header, secondary nav,
 * and a CTA to register.
 */
export function DiscoverLayout() {
  return (
    <>
      <SkipLink />
      <div className="min-h-screen bg-surface flex flex-col" id="main-content">
        {/* Header */}
        <header className="border-b border-outline-variant bg-surface-container-lowest">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
            <Link
              to="/discover"
              className="focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
            >
              <span className="type-title-lg font-semibold text-primary">
                Homegrown Academy
              </span>
            </Link>

            {/* Secondary nav */}
            <nav
              aria-label="Discovery navigation"
              className="hidden items-center gap-1 sm:flex"
            >
              <NavLink
                to="/discover/quiz"
                className={({ isActive }) =>
                  `rounded-button px-3 py-1.5 type-label-md transition-colors ${
                    isActive
                      ? "bg-secondary-container text-on-secondary-container"
                      : "text-on-surface-variant hover:bg-surface-container-low"
                  }`
                }
              >
                Methodology Quiz
              </NavLink>
              <NavLink
                to="/discover/states"
                className={({ isActive }) =>
                  `rounded-button px-3 py-1.5 type-label-md transition-colors ${
                    isActive
                      ? "bg-secondary-container text-on-secondary-container"
                      : "text-on-surface-variant hover:bg-surface-container-low"
                  }`
                }
              >
                State Guides
              </NavLink>
            </nav>

            {/* CTA */}
            <Link to="/auth/register" tabIndex={-1}>
              <Button variant="primary" size="sm">
                Sign Up Free
              </Button>
            </Link>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1">
          <div className="mx-auto max-w-3xl px-4 py-8">
            <Suspense fallback={null}>
              <Outlet />
            </Suspense>
          </div>
        </main>

        {/* Footer */}
        <footer className="border-t border-outline-variant py-6 text-center">
          <p className="type-body-sm text-on-surface-variant">
            &copy; {new Date().getFullYear()} Homegrown Academy &middot;{" "}
            <Link
              to="/legal/privacy"
              className="text-primary hover:underline"
            >
              Privacy
            </Link>{" "}
            &middot;{" "}
            <Link to="/legal/terms" className="text-primary hover:underline">
              Terms
            </Link>
          </p>
        </footer>
      </div>
    </>
  );
}
