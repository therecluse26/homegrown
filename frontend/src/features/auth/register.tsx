import { useState, useCallback } from "react";
import { Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Button, Input, FormField } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { register } from "@/lib/hearth-auth";
import type { AuthError } from "@/lib/hearth-auth";

type RegisterFormState = {
  email: string;
  display_name: string;
  family_display_name: string;
  tosAccepted: boolean;
};

/**
 * Registration page — app-orchestrated Hearth registration.
 *
 * Submits to POST /v1/auth/register. The backend creates the Hearth identity
 * and the family + parent records. Hearth then emails an activation link;
 * the user sets their password through Hearth's hosted UI, then logs in via
 * the PKCE flow. [ARCH ADR-020, §10.1]
 */
export function Register() {
  const intl = useIntl();

  const [form, setForm] = useState<RegisterFormState>({
    email: "",
    display_name: "",
    family_display_name: "",
    tosAccepted: false,
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [globalError, setGlobalError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

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
    if (isSubmitting) return;

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

    setIsSubmitting(true);
    setGlobalError("");
    setFieldErrors({});

    try {
      await register({
        email: form.email,
        display_name: form.display_name,
        family_display_name: form.family_display_name,
        // Default; can be updated during onboarding. [§10.1]
        primary_methodology_slug: "charlotte-mason",
      });
      setSuccess(true);
    } catch (err: unknown) {
      const authErr = err as AuthError;
      setGlobalError(
        authErr.message ?? intl.formatMessage({ id: "error.generic" }),
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  if (success) {
    return (
      <>
        <PageTitle title={intl.formatMessage({ id: "auth.register" })} />
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
              <FormattedMessage
                id="auth.register.success.title"
                defaultMessage="Account created!"
              />
            </h2>
            <p className="mt-1 text-body-sm text-on-surface-variant">
              <FormattedMessage
                id="auth.register.success.subtitle"
                defaultMessage="Check your email to set a password, then sign in."
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

  return (
    <>
      <PageTitle title={intl.formatMessage({ id: "auth.register" })} className="text-center" />

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

      <form onSubmit={(e) => void handleSubmit(e)} noValidate className="space-y-4">
        <FormField
          label={intl.formatMessage({ id: "auth.register.name" })}
          error={fieldErrors["display_name"]}
        >
          {({ id, errorId }) => (
            <Input
              id={id}
              type="text"
              name="display_name"
              value={form.display_name}
              onChange={handleChange("display_name")}
              autoComplete="name"
              required
              error={!!fieldErrors["display_name"]}
              aria-describedby={errorId}
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({
            id: "auth.register.familyName",
            defaultMessage: "Family name",
          })}
          error={fieldErrors["family_display_name"]}
        >
          {({ id, errorId }) => (
            <Input
              id={id}
              type="text"
              name="family_display_name"
              placeholder="e.g. The Johnson Family"
              value={form.family_display_name}
              onChange={handleChange("family_display_name")}
              autoComplete="organization"
              required
              error={!!fieldErrors["family_display_name"]}
              aria-describedby={errorId}
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "auth.register.email" })}
          error={fieldErrors["email"]}
        >
          {({ id, errorId }) => (
            <Input
              id={id}
              type="email"
              name="email"
              value={form.email}
              onChange={handleChange("email")}
              autoComplete="email"
              required
              error={!!fieldErrors["email"]}
              aria-describedby={errorId}
            />
          )}
        </FormField>

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
                setForm((prev) => ({ ...prev, tosAccepted: e.target.checked }))
              }
              className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-primary checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
              aria-describedby={fieldErrors["tos"] ? "tos-error" : undefined}
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
