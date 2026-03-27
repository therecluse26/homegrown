import { type ReactNode } from "react";
import { Outlet } from "react-router";
import { useIntl } from "react-intl";
import { SkipLink } from "@/components/common";

export function AuthLayout({ children }: { children?: ReactNode }) {
  const intl = useIntl();

  return (
    <>
      <SkipLink />
      <div className="min-h-screen bg-surface flex flex-col items-center justify-center px-spacing-page-x py-8">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="type-headline-lg text-primary">
              {intl.formatMessage({ id: "app.name", defaultMessage: "Homegrown Academy" })}
            </h1>
          </div>
          <div
            className="bg-surface-container-lowest rounded-radius-lg p-spacing-card-padding shadow-ambient-sm"
            id="main-content"
          >
            {children ?? <Outlet />}
          </div>
        </div>
      </div>
    </>
  );
}
