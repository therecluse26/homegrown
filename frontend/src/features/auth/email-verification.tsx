import { useState, useEffect, useRef } from "react";
import { useSearchParams, Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { useQueryClient } from "@tanstack/react-query";
import { Button, Input, Spinner, FormField } from "@/components/ui";
import { PageTitle } from "@/components/common";
import {
  initVerificationFlow,
  getFlow,
  submitFlow,
  extractCsrfToken,
  extractFieldErrors,
  extractGlobalMessages,
  type KratosFlow,
  type KratosError,
} from "@/lib/kratos";

const RESEND_COOLDOWN_SECONDS = 60;

/**
 * Email verification page — rendered inside AuthLayout.
 *
 * Handles both initial verification state and URL ?flow=xxx re-entry.
 * After successful verification, invalidates auth query.
 */
export function EmailVerification() {
  const intl = useIntl();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();

  const [flow, setFlow] = useState<KratosFlow | null>(null);
  const [code, setCode] = useState("");
  const [email, setEmail] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [globalError, setGlobalError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const [success, setSuccess] = useState(false);
  const [resendCountdown, setResendCountdown] = useState(0);
  const resendTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    let active = true;
    const flowId = searchParams.get("flow");
    const init = flowId
      ? getFlow("verification", flowId)
      : initVerificationFlow();

    void init
      .then((f) => {
        if (active) {
          setFlow(f);
          const emailNode = f.ui.nodes.find(
            (n) => n.attributes.name === "email",
          );
          if (typeof emailNode?.attributes.value === "string") {
            setEmail(emailNode.attributes.value);
          }
        }
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
  }, [intl, searchParams]);

  useEffect(
    () => () => {
      if (resendTimerRef.current) clearInterval(resendTimerRef.current);
    },
    [],
  );

  function startResendCooldown() {
    setResendCountdown(RESEND_COOLDOWN_SECONDS);
    resendTimerRef.current = setInterval(() => {
      setResendCountdown((c) => {
        if (c <= 1) {
          if (resendTimerRef.current) clearInterval(resendTimerRef.current);
          return 0;
        }
        return c - 1;
      });
    }, 1000);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!flow || isSubmitting) return;

    setIsSubmitting(true);
    setGlobalError("");
    setFieldErrors({});

    const csrfToken = extractCsrfToken(flow);

    try {
      const result = await submitFlow(flow.ui.action, flow.ui.method, {
        code,
        method: "code",
        csrf_token: csrfToken,
      });

      if (result.kind === "success") {
        setSuccess(true);
        await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
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

      if ((updatedFlow.state as string) === "passed_challenge") {
        setSuccess(true);
        await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
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

  async function handleResend() {
    if (!flow || resendCountdown > 0) return;

    setGlobalError("");
    const csrfToken = extractCsrfToken(flow);

    try {
      const result = await submitFlow(flow.ui.action, flow.ui.method, {
        email,
        method: "code",
        csrf_token: csrfToken,
      });

      if (result.kind === "flow") setFlow(result.flow);
      startResendCooldown();
    } catch {
      setGlobalError(intl.formatMessage({ id: "error.generic" }));
    }
  }

  if (success) {
    return (
      <>
        <PageTitle title={intl.formatMessage({ id: "auth.verification" })} />
        <div className="flex flex-col items-center gap-4 text-center" role="status">
          <div
            className="flex h-16 w-16 items-center justify-center rounded-full bg-primary-container"
            aria-hidden="true"
          >
            <svg
              className="h-8 w-8 text-on-primary-container"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M5 13l4 4L19 7"
              />
            </svg>
          </div>
          <h2 className="text-title-lg font-semibold text-on-surface">
            <FormattedMessage id="auth.verification.success" />
          </h2>
          <Link
            to="/"
            className="text-label-md font-medium text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
          >
            <FormattedMessage id="error.goHome" />
          </Link>
        </div>
      </>
    );
  }

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.verification" })} />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.verification.title" />
        </h2>
        {email && (
          <p className="text-body-sm text-on-surface-variant">
            <FormattedMessage
              id="auth.verification.subtitle"
              values={{ email }}
            />
          </p>
        )}
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
            label={intl.formatMessage({ id: "auth.verification.code" })}
            error={fieldErrors["code"]}
          >
            {({ id, errorId }) => (
              <Input
                id={id}
                type="text"
                name="code"
                value={code}
                onChange={(e) => {
                  setCode(e.target.value);
                  setFieldErrors({});
                }}
                autoComplete="one-time-code"
                inputMode="numeric"
                required
                error={!!fieldErrors["code"]}
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
            <FormattedMessage id="auth.verification.submit" />
          </Button>

          <div className="text-center">
            <button
              type="button"
              onClick={handleResend}
              disabled={resendCountdown > 0}
              className="text-label-sm text-primary hover:underline disabled:cursor-not-allowed disabled:opacity-disabled focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
            >
              {resendCountdown > 0 ? (
                <FormattedMessage
                  id="auth.verification.resendIn"
                  values={{ seconds: resendCountdown }}
                />
              ) : (
                <FormattedMessage id="auth.verification.resend" />
              )}
            </button>
          </div>
        </form>
      )}
    </>
  );
}
