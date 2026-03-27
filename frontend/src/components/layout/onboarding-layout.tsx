import { type ReactNode } from "react";
import { Outlet } from "react-router";
import { useIntl } from "react-intl";
import { SkipLink } from "@/components/common";
import { ProgressBar } from "@/components/ui";

type OnboardingLayoutProps = {
  children?: ReactNode;
  progress?: number;
};

export function OnboardingLayout({ children, progress }: OnboardingLayoutProps) {
  const intl = useIntl();

  return (
    <>
      <SkipLink />
      <div className="min-h-screen bg-surface">
        <header className="py-6 px-spacing-page-x">
          <div className="max-w-[var(--width-content-narrow)] mx-auto">
            <p className="type-title-md text-primary font-semibold text-center">
              {intl.formatMessage({ id: "app.name", defaultMessage: "Homegrown Academy" })}
            </p>
            {progress !== undefined && (
              <div className="mt-4">
                <ProgressBar value={progress} />
              </div>
            )}
          </div>
        </header>
        <main
          id="main-content"
          className="max-w-[var(--width-content-narrow)] mx-auto px-spacing-page-x pb-12"
        >
          {children ?? <Outlet />}
        </main>
      </div>
    </>
  );
}
