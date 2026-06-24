import { Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Button } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { redirectToLogin } from "@/lib/hearth-auth";

/**
 * Account recovery page.
 *
 * Password reset is handled through Hearth's hosted login UI (the "Forgot
 * password" link on the sign-in page). Direct users there rather than
 * implementing a custom recovery flow. [ARCH ADR-020]
 */
export function AccountRecovery() {
  const intl = useIntl();

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.recovery" })} />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.recovery.title" />
        </h2>
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage
            id="auth.recovery.hearthNote"
            defaultMessage='Password reset is handled through our sign-in page. Click the "Forgot password" link after signing in.'
          />
        </p>
      </div>

      <Button variant="primary" className="w-full" onClick={redirectToLogin}>
        <FormattedMessage id="auth.login.submit" defaultMessage="Go to sign in" />
      </Button>

      <div className="text-center">
        <Link
          to="/auth/login"
          className="text-label-sm text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
        >
          <FormattedMessage id="auth.recovery.backToLogin" />
        </Link>
      </div>
    </>
  );
}
