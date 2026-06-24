import { Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { PageTitle } from "@/components/common";

/**
 * Email verification page.
 *
 * With Hearth BFF, email verification is handled through the activation link
 * Hearth sends after account creation. The user clicks that link, sets their
 * password in Hearth's hosted UI, and then logs in via the PKCE flow. This
 * page is kept as a graceful landing for any stale verification links. [ARCH ADR-020]
 */
export function EmailVerification() {
  const intl = useIntl();

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.verification" })} />

      <div className="flex flex-col items-center gap-4 text-center">
        <div
          className="flex h-16 w-16 items-center justify-center rounded-full bg-primary-container"
          aria-hidden="true"
        >
          <svg
            className="h-8 w-8 text-on-primary-container"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
        </div>

        <div>
          <h2 className="text-title-lg font-semibold text-on-surface">
            <FormattedMessage id="auth.verification.title" />
          </h2>
          <p className="mt-1 text-body-sm text-on-surface-variant">
            <FormattedMessage
              id="auth.verification.hearthNote"
              defaultMessage="Check the activation email we sent you and follow the link to set your password. Once set, you can sign in below."
            />
          </p>
        </div>

        <Link
          to="/auth/login"
          className="text-label-md font-medium text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
        >
          <FormattedMessage id="auth.login.submit" defaultMessage="Sign in" />
        </Link>
      </div>
    </>
  );
}
