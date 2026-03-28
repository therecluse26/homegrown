import { useState, useEffect, useCallback } from "react";
import { useNavigate, Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { useQueryClient } from "@tanstack/react-query";
import { Button, Input, Spinner, FormField } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { useAuthContext } from "@/features/auth/auth-provider";
import {
  initLoginFlow,
  submitFlow,
  extractCsrfToken,
  extractFieldErrors,
  extractGlobalMessages,
  extractOAuthProviders,
  type KratosFlow,
  type KratosError,
} from "@/lib/kratos";
import { OAuthButton } from "./oauth-button";

type LoginFormState = {
  identifier: string;
  password: string;
};

/**
 * Login page — rendered inside AuthLayout's card.
 *
 * Uses Ory Kratos Browser API:
 * 1. Init login flow (GET /self-service/login/browser)
 * 2. Render form from flow UI nodes
 * 3. Submit to flow's action URL → handle errors or success
 */
export function Login() {
  const { isAuthenticated, isLoading: isAuthLoading } = useAuthContext();
  const intl = useIntl();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [flow, setFlow] = useState<KratosFlow | null>(null);
  const [form, setForm] = useState<LoginFormState>({
    identifier: "",
    password: "",
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [globalError, setGlobalError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const [rateLimitCountdown, setRateLimitCountdown] = useState(0);

  useEffect(() => {
    if (isAuthLoading || isAuthenticated) return;
    let active = true;
    void initLoginFlow()
      .then((f) => {
        if (active) setFlow(f);
      })
      .catch(() => {
        if (active)
          setGlobalError(intl.formatMessage({ id: "error.generic" }));
      })
      .finally(() => {
        if (active) setIsInitializing(false);
      });
    return () => {
      active = false;
    };
  }, [intl, isAuthLoading, isAuthenticated]);

  useEffect(() => {
    if (rateLimitCountdown <= 0) return;
    const timer = window.setInterval(() => {
      setRateLimitCountdown((c) => Math.max(0, c - 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [rateLimitCountdown]);

  const handleChange = useCallback(
    (field: keyof LoginFormState) =>
      (e: React.ChangeEvent<HTMLInputElement>) => {
        setForm((prev) => ({ ...prev, [field]: e.target.value }));
        setFieldErrors((prev) => ({ ...prev, [field]: "" }));
      },
    [],
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!flow || isSubmitting || rateLimitCountdown > 0) return;

    setIsSubmitting(true);
    setGlobalError("");
    setFieldErrors({});

    const csrfToken = extractCsrfToken(flow);

    try {
      const result = await submitFlow(flow.ui.action, flow.ui.method, {
        identifier: form.identifier,
        password: form.password,
        method: "password",
        csrf_token: csrfToken,
      });

      if (result.kind === "success") {
        await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
        navigate("/", { replace: true });
        return;
      }

      if (result.kind === "redirect") {
        window.location.href = result.url;
        return;
      }

      const updatedFlow = result.flow;
      const errors = extractFieldErrors(updatedFlow);
      const globals = extractGlobalMessages(updatedFlow);
      setFieldErrors(errors);
      setFlow(updatedFlow);

      const firstGlobal = globals.find((m) => m.type === "error");
      if (firstGlobal) {
        setGlobalError(
          firstGlobal.id === 4000006
            ? intl.formatMessage({ id: "auth.login.invalidCredentials" })
            : firstGlobal.text,
        );
      }
    } catch (err: unknown) {
      const kratosErr = err as KratosError;
      if (kratosErr.error?.code === 429) {
        const retryAfter = 60;
        setRateLimitCountdown(retryAfter);
        setGlobalError(
          intl.formatMessage(
            { id: "auth.login.tooManyAttempts" },
            { seconds: retryAfter },
          ),
        );
      } else {
        setGlobalError(
          kratosErr.error?.message ??
            intl.formatMessage({ id: "error.generic" }),
        );
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  const csrfToken = flow ? extractCsrfToken(flow) : "";
  const oauthAction = flow?.ui.action ?? "";
  const oauthProviders = flow
    ? (extractOAuthProviders(flow) as Array<"google" | "facebook" | "apple">)
    : [];

  return (
    <>
      {/* Sets document.title and focuses h1 for screen reader route announcements */}
      <PageTitle title={intl.formatMessage({ id: "auth.login" })} />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.login.title" />
        </h2>
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage id="auth.login.subtitle" />
        </p>
      </div>

      {/* Global error */}
      {globalError && (
        <div
          role="alert"
          aria-live="assertive"
          className="rounded-lg bg-error-container px-4 py-3 text-body-sm text-on-error-container"
        >
          {globalError}
        </div>
      )}

      {isInitializing ? (
        <div
          className="flex justify-center py-6"
          role="status"
          aria-live="polite"
          aria-label={intl.formatMessage({ id: "loading.default" })}
        >
          <Spinner size="lg" />
        </div>
      ) : (
        <>
          {/* OAuth buttons */}
          {oauthProviders.length > 0 && (
            <>
              <div className="space-y-3">
                {oauthProviders.map((provider) => (
                  <OAuthButton
                    key={provider}
                    provider={provider}
                    kratosActionUrl={oauthAction}
                    csrfToken={csrfToken}
                    disabled={isSubmitting}
                  />
                ))}
              </div>
              <div className="flex items-center gap-3">
                <div className="h-px flex-1 bg-outline-variant" />
                <span className="text-label-sm text-on-surface-variant">
                  <FormattedMessage id="common.or" />
                </span>
                <div className="h-px flex-1 bg-outline-variant" />
              </div>
            </>
          )}

          <form onSubmit={handleSubmit} noValidate className="space-y-4">
            <FormField
              label={intl.formatMessage({ id: "auth.login.email" })}
              error={fieldErrors["identifier"]}
            >
              {({ id, errorId }) => (
                <Input
                  id={id}
                  type="email"
                  name="identifier"
                  value={form.identifier}
                  onChange={handleChange("identifier")}
                  autoComplete="email"
                  required
                  error={!!fieldErrors["identifier"]}
                  aria-describedby={errorId}
                />
              )}
            </FormField>

            <FormField
              label={intl.formatMessage({ id: "auth.login.password" })}
              error={fieldErrors["password"]}
            >
              {({ id, errorId }) => (
                <Input
                  id={id}
                  type="password"
                  name="password"
                  value={form.password}
                  onChange={handleChange("password")}
                  autoComplete="current-password"
                  required
                  error={!!fieldErrors["password"]}
                  aria-describedby={errorId}
                />
              )}
            </FormField>

            <div className="flex justify-end">
              <Link
                to="/auth/recovery"
                className="text-label-sm text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
              >
                <FormattedMessage id="auth.login.forgotPassword" />
              </Link>
            </div>

            <Button
              type="submit"
              variant="primary"
              className="w-full"
              loading={isSubmitting}
              disabled={isSubmitting || rateLimitCountdown > 0}
            >
              {rateLimitCountdown > 0
                ? intl.formatMessage(
                    { id: "auth.login.tooManyAttempts" },
                    { seconds: rateLimitCountdown },
                  )
                : intl.formatMessage({ id: "auth.login.submit" })}
            </Button>
          </form>
        </>
      )}

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
