import { type ReactNode, Suspense } from "react";
import { Outlet, Link } from "react-router";
import { useIntl } from "react-intl";
import { SkipLink } from "@/components/common";

/**
 * Layout for unauthenticated pages: login, register, recovery, verification.
 *
 * Renders:
 * - Brand link "Homegrown Academy" at top (not h1 — each page owns its h1 via PageTitle)
 * - Centered card container
 * - Outlet/children for page content
 *
 * The card uses a soft shadow for elevation without a hard border.
 */
export function AuthLayout({ children }: { children?: ReactNode }) {
  const intl = useIntl();

  return (
    <>
      <SkipLink />
      <div
        className="min-h-screen bg-surface flex flex-col items-center justify-center px-4 py-8"
        id="main-content"
      >
        <div className="w-full max-w-md">
          {/* Brand — role="banner" region header */}
          <header className="text-center mb-8">
            <Link
              to="/"
              aria-label={intl.formatMessage({ id: "app.name" })}
              className="inline-block focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
            >
              <span className="text-headline-md font-semibold text-primary">
                {intl.formatMessage({ id: "app.name" })}
              </span>
            </Link>
          </header>

          {/* Card */}
          <div className="bg-surface-container-lowest rounded-2xl p-8 shadow-ambient-sm space-y-6">
            <Suspense fallback={null}>
              {children ?? <Outlet />}
            </Suspense>
          </div>
        </div>
      </div>
    </>
  );
}
