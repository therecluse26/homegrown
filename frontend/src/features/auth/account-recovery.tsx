import { useState, useEffect } from "react";
import { Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Button, Input, Spinner, FormField } from "@/components/ui";
import { PageTitle } from "@/components/common";
import {
  initRecoveryFlow,
  submitFlow,
  extractCsrfToken,
  extractFieldErrors,
  extractGlobalMessages,
  type KratosFlow,
  type KratosError,
} from "@/lib/kratos";

/**
 * Account recovery (forgot password) page — rendered inside AuthLayout.
 *
 * Submits email to Kratos, which emails a recovery link.
 * Always shows success message (avoids email enumeration).
 */
export function AccountRecovery() {
  const intl = useIntl();

  const [flow, setFlow] = useState<KratosFlow | null>(null);
  const [email, setEmail] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [globalError, setGlobalError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    let active = true;
    void initRecoveryFlow()
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
  }, [intl]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!flow || isSubmitting) return;

    setIsSubmitting(true);
    setGlobalError("");
    setFieldErrors({});

    const csrfToken = extractCsrfToken(flow);

    try {
      const result = await submitFlow(flow.ui.action, flow.ui.method, {
        email,
        method: "link",
        csrf_token: csrfToken,
      });

      if (result.kind === "success") {
        setSuccess(true);
        return;
      }

      if (result.kind === "redirect") {
        window.location.href = result.url;
        return;
      }

      const updatedFlow = result.flow;
      const globals = extractGlobalMessages(updatedFlow);
      setFieldErrors(extractFieldErrors(updatedFlow));
      setFlow(updatedFlow);

      if ((updatedFlow.state as string) === "sent_email") {
        setSuccess(true);
        return;
      }

      const successMsg = globals.find((m) => m.type === "info");
      if (successMsg) {
        setSuccess(true);
        return;
      }

      const errMsg = globals.find((m) => m.type === "error");
      if (errMsg) setGlobalError(errMsg.text);
    } catch (err: unknown) {
      const kratosErr = err as KratosError;
      setGlobalError(
        kratosErr.error?.message ?? intl.formatMessage({ id: "error.generic" }),
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  if (success) {
    return (
      <>
        <PageTitle title={intl.formatMessage({ id: "auth.recovery" })} />
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
              <FormattedMessage id="auth.recovery.success" />
            </h2>
            <p className="mt-1 text-body-sm text-on-surface-variant">{email}</p>
          </div>
          <Link
            to="/auth/login"
            className="text-label-md font-medium text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          >
            <FormattedMessage id="auth.recovery.backToLogin" />
          </Link>
        </div>
      </>
    );
  }

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.recovery" })} />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.recovery.title" />
        </h2>
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage id="auth.recovery.subtitle" />
        </p>
      </div>

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
        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <FormField
            label={intl.formatMessage({ id: "auth.recovery.email" })}
            error={fieldErrors["email"]}
          >
            {({ id, errorId }) => (
              <Input
                id={id}
                type="email"
                name="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  setFieldErrors({});
                }}
                autoComplete="email"
                required
                error={!!fieldErrors["email"]}
                aria-describedby={errorId}
              />
            )}
          </FormField>

          <Button
            type="submit"
            variant="primary"
            className="w-full"
            loading={isSubmitting}
            disabled={isSubmitting}
          >
            <FormattedMessage id="auth.recovery.submit" />
          </Button>

          <div className="text-center">
            <Link
              to="/auth/login"
              className="text-label-sm text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
            >
              <FormattedMessage id="auth.recovery.backToLogin" />
            </Link>
          </div>
        </form>
      )}
    </>
  );
}
