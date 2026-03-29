import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useNavigate, Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { useQueryClient } from "@tanstack/react-query";
import { Turnstile, type TurnstileInstance } from "@marsidev/react-turnstile";
import { Button, Input, Spinner, FormField } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { useAuthContext } from "@/features/auth/auth-provider";
import {
  initRegistrationFlow,
  submitFlow,
  extractCsrfToken,
  extractFieldErrors,
  extractGlobalMessages,
  extractOAuthProviders,
  type KratosFlow,
  type KratosError,
} from "@/lib/kratos";
import { OAuthButton } from "./oauth-button";

// Turnstile site key — use test key in dev, real key in production
const TURNSTILE_SITE_KEY =
  import.meta.env.VITE_TURNSTILE_SITE_KEY ?? "1x00000000000000000000AA";

type RegisterFormState = {
  email: string;
  name: string;
  password: string;
  tosAccepted: boolean;
};

// ─── Password strength ────────────────────────────────────────────────────────

type PasswordStrength = "weak" | "fair" | "strong" | "very-strong";

function measurePasswordStrength(password: string): PasswordStrength | null {
  if (!password) return null;
  let score = 0;
  if (password.length >= 10) score++;
  if (password.length >= 14) score++;
  if (/[A-Z]/.test(password)) score++;
  if (/[0-9]/.test(password)) score++;
  if (/[^A-Za-z0-9]/.test(password)) score++;

  if (score <= 1) return "weak";
  if (score <= 2) return "fair";
  if (score <= 3) return "strong";
  return "very-strong";
}

const STRENGTH_CONFIG: Record<
  PasswordStrength,
  { label: string; widthClass: string; colorClass: string }
> = {
  weak: {
    label: "auth.register.passwordStrength.weak",
    widthClass: "w-1/4",
    colorClass: "bg-error",
  },
  fair: {
    label: "auth.register.passwordStrength.fair",
    widthClass: "w-2/4",
    colorClass: "bg-tertiary",
  },
  strong: {
    label: "auth.register.passwordStrength.strong",
    widthClass: "w-3/4",
    colorClass: "bg-secondary",
  },
  "very-strong": {
    label: "auth.register.passwordStrength.veryStrong",
    widthClass: "w-full",
    colorClass: "bg-primary",
  },
};

function PasswordStrengthIndicator({ password }: { password: string }) {
  const intl = useIntl();
  const strength = useMemo(() => measurePasswordStrength(password), [password]);

  if (!strength) return null;
  const config = STRENGTH_CONFIG[strength];

  const scoreValue =
    strength === "weak"
      ? 25
      : strength === "fair"
        ? 50
        : strength === "strong"
          ? 75
          : 100;

  return (
    <div className="mt-2" aria-live="polite">
      <div
        className="h-1.5 w-full overflow-hidden rounded-full bg-surface-container-highest"
        role="progressbar"
        aria-valuenow={scoreValue}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label="Password strength"
      >
        <div
          className={`h-full transition-all duration-300 ${config.widthClass} ${config.colorClass}`}
        />
      </div>
      <p className="mt-1 text-label-sm text-on-surface-variant">
        {intl.formatMessage({ id: config.label })}
      </p>
    </div>
  );
}

// ─── Register page ────────────────────────────────────────────────────────────

/**
 * Registration page — rendered inside AuthLayout's card.
 *
 * After successful registration, Kratos fires the post-registration webhook
 * which creates the family + parent records in our database, then redirects to /onboarding.
 */
export function Register() {
  const { isAuthenticated, isLoading: isAuthLoading } = useAuthContext();
  const intl = useIntl();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [flow, setFlow] = useState<KratosFlow | null>(null);
  const [form, setForm] = useState<RegisterFormState>({
    email: "",
    name: "",
    password: "",
    tosAccepted: false,
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [globalError, setGlobalError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const [captchaToken, setCaptchaToken] = useState<string>("");
  const turnstileRef = useRef<TurnstileInstance | null>(null);

  useEffect(() => {
    if (isAuthLoading || isAuthenticated) return;
    let active = true;
    void initRegistrationFlow()
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

  const handleChange = useCallback(
    (field: keyof Omit<RegisterFormState, "tosAccepted">) =>
      (e: React.ChangeEvent<HTMLInputElement>) => {
        setForm((prev) => ({ ...prev, [field]: e.target.value }));
        setFieldErrors((prev) => ({ ...prev, [field]: "" }));
      },
    [],
  );

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!flow || isSubmitting) return;

    if (!form.tosAccepted) {
      setFieldErrors((prev) => ({
        ...prev,
        tos: intl.formatMessage({
          id: "auth.register.tosRequired",
          defaultMessage: "You must accept the terms to create an account",
        }),
      }));
      return;
    }

    if (!captchaToken) {
      setFieldErrors((prev) => ({
        ...prev,
        captcha: intl.formatMessage({ id: "auth.register.captchaRequired" }),
      }));
      return;
    }

    setIsSubmitting(true);
    setGlobalError("");
    setFieldErrors({});

    const csrfToken = extractCsrfToken(flow);

    try {
      const result = await submitFlow(flow.ui.action, flow.ui.method, {
        "traits.email": form.email,
        "traits.name": form.name,
        password: form.password,
        method: "password",
        csrf_token: csrfToken,
        captcha_token: captchaToken,
      });

      if (result.kind === "success") {
        await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
        navigate("/onboarding", { replace: true });
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
      if (firstGlobal) setGlobalError(firstGlobal.text);
    } catch (err: unknown) {
      const kratosErr = err as KratosError;
      setGlobalError(
        kratosErr.error?.message ?? intl.formatMessage({ id: "error.generic" }),
      );
    } finally {
      setIsSubmitting(false);
      // Reset CAPTCHA so user gets a fresh challenge on retry
      setCaptchaToken("");
      turnstileRef.current?.reset();
    }
  }

  const csrfToken = flow ? extractCsrfToken(flow) : "";
  const oauthAction = flow?.ui.action ?? "";
  const oauthProviders = flow
    ? (extractOAuthProviders(flow) as Array<"google" | "facebook" | "apple">)
    : [];

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.register" })} />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.register.title" />
        </h2>
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage id="auth.register.subtitle" />
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
        <>
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
              label={intl.formatMessage({ id: "auth.register.name" })}
              error={fieldErrors["traits.name"]}
            >
              {({ id, errorId }) => (
                <Input
                  id={id}
                  type="text"
                  name="traits.name"
                  value={form.name}
                  onChange={handleChange("name")}
                  autoComplete="name"
                  required
                  error={!!fieldErrors["traits.name"]}
                  aria-describedby={errorId}
                />
              )}
            </FormField>

            <FormField
              label={intl.formatMessage({ id: "auth.register.email" })}
              error={fieldErrors["traits.email"]}
            >
              {({ id, errorId }) => (
                <Input
                  id={id}
                  type="email"
                  name="traits.email"
                  value={form.email}
                  onChange={handleChange("email")}
                  autoComplete="email"
                  required
                  error={!!fieldErrors["traits.email"]}
                  aria-describedby={errorId}
                />
              )}
            </FormField>

            <div>
              <FormField
                label={intl.formatMessage({ id: "auth.register.password" })}
                error={fieldErrors["password"]}
              >
                {({ id, errorId }) => (
                  <Input
                    id={id}
                    type="password"
                    name="password"
                    value={form.password}
                    onChange={handleChange("password")}
                    autoComplete="new-password"
                    required
                    error={!!fieldErrors["password"]}
                    aria-describedby={errorId}
                  />
                )}
              </FormField>
              <PasswordStrengthIndicator password={form.password} />
            </div>

            {/* ToS acceptance */}
            <div>
              <label
                htmlFor="tos-accept"
                className="flex cursor-pointer select-none items-start gap-3"
              >
                <input
                  id="tos-accept"
                  type="checkbox"
                  name="tos-accept"
                  checked={form.tosAccepted}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      tosAccepted: e.target.checked,
                    }))
                  }
                  className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-primary checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                  aria-describedby={
                    fieldErrors["tos"] ? "tos-error" : undefined
                  }
                />
                <span className="text-body-sm text-on-surface">
                  <FormattedMessage
                    id="auth.register.tosAcceptance"
                    values={{
                      termsLink: (
                        <Link
                          to="/legal/terms"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="font-medium text-primary hover:underline"
                        >
                          <FormattedMessage id="auth.register.terms" />
                        </Link>
                      ),
                      privacyLink: (
                        <Link
                          to="/legal/privacy"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="font-medium text-primary hover:underline"
                        >
                          <FormattedMessage id="auth.register.privacy" />
                        </Link>
                      ),
                    }}
                  />
                </span>
              </label>
              {fieldErrors["tos"] && (
                <p
                  id="tos-error"
                  className="mt-1 text-label-sm text-error"
                  role="alert"
                  aria-live="assertive"
                >
                  {fieldErrors["tos"]}
                </p>
              )}
            </div>

            {/* CAPTCHA verification */}
            <div>
              <Turnstile
                ref={turnstileRef}
                siteKey={TURNSTILE_SITE_KEY}
                onSuccess={(token) => {
                  setCaptchaToken(token);
                  setFieldErrors((prev) => ({ ...prev, captcha: "" }));
                }}
                onError={() => setCaptchaToken("")}
                onExpire={() => setCaptchaToken("")}
                options={{ theme: "light", size: "normal" }}
              />
              {fieldErrors["captcha"] && (
                <p
                  className="mt-1 text-label-sm text-error"
                  role="alert"
                  aria-live="assertive"
                >
                  {fieldErrors["captcha"]}
                </p>
              )}
            </div>

            <Button
              type="submit"
              variant="primary"
              className="w-full"
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              <FormattedMessage id="auth.register.submit" />
            </Button>
          </form>
        </>
      )}

      <p className="text-center text-body-sm text-on-surface-variant">
        <FormattedMessage id="auth.register.haveAccount" />{" "}
        <Link
          to="/auth/login"
          className="font-medium text-primary hover:underline focus-visible:rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
        >
          <FormattedMessage id="auth.register.signIn" />
        </Link>
      </p>
    </>
  );
}
