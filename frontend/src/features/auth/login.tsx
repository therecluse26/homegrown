import { useState, useEffect } from "react";
import { Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Button, Spinner } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { useAuthContext } from "@/features/auth/auth-provider";
import { redirectToLogin } from "@/lib/hearth-auth";

/**
 * Login page — initiates the Hearth PKCE login flow.
 *
 * Redirects the browser to GET /v1/auth/login, which sets a PKCE state
 * cookie and redirects onward to Hearth's hosted login UI. After the user
 * authenticates, the backend sets an HttpOnly sid cookie and redirects back
 * to the SPA. [ARCH ADR-020]
 */
export function Login() {
  const { isAuthenticated, isLoading } = useAuthContext();
  const intl = useIntl();
  const [isRedirecting, setIsRedirecting] = useState(false);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      setIsRedirecting(true);
      redirectToLogin();
    }
  }, [isLoading, isAuthenticated]);

  if (isRedirecting) {
    return (
      <div className="flex flex-col items-center gap-3 py-6" role="status" aria-live="polite">
        <Spinner size="lg" />
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage id="auth.login.redirecting" defaultMessage="Taking you to sign in…" />
        </p>
      </div>
    );
  }

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.login" })} className="text-center" />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.login.title" />
        </h2>
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage id="auth.login.subtitle" />
        </p>
      </div>

      <Button
        variant="primary"
        className="w-full"
        onClick={redirectToLogin}
      >
        <FormattedMessage id="auth.login.submit" />
      </Button>

      <p className="text-center text-body-sm text-on-surface-variant">
        <FormattedMessage id="auth.login.noAccount" />{" "}
        <Link
          to="/auth/register"
          className="font-medium text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
        >
          <FormattedMessage id="auth.login.createAccount" />
        </Link>
      </p>
    </>
  );
}
