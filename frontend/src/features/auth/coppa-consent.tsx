import { useState } from "react";
import { Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Button } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { useConsent } from "@/hooks/use-consent";

/**
 * COPPA parental consent gate component.
 *
 * Shown after registration, before adding any student profiles.
 * Parent must explicitly acknowledge and consent to student data collection.
 *
 * Status flow: registered → noticed → consented
 * This component handles the final "consented" step.
 *
 * @see SPEC §7.3 (COPPA consent flow)
 */
export function CoppaConsent() {
  const intl = useIntl();
  const { provideConsent, isConsenting, consentError } = useConsent();
  const [acknowledged, setAcknowledged] = useState(false);

  async function handleConsent() {
    if (!acknowledged) return;
    await provideConsent({
      coppa_notice_acknowledged: true,
      method: "explicit",
      verification_token: "", // Token only required for micro-charge COPPA method (P2)
    });
  }

  const errorMessage =
    consentError instanceof Error
      ? consentError.message
      : consentError
        ? intl.formatMessage({ id: "error.generic" })
        : null;

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-surface px-4">
      <PageTitle title={intl.formatMessage({ id: "coppa.title" })} />

      <div className="w-full max-w-lg">
        <div className="rounded-2xl bg-surface-container-lowest p-8 shadow-ambient-sm">
          {/* Icon */}
          <div className="mb-6 flex justify-center">
            <div
              className="flex h-16 w-16 items-center justify-center rounded-full bg-secondary-container"
              aria-hidden="true"
            >
              <svg
                className="h-8 w-8 text-on-secondary-container"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z"
                />
              </svg>
            </div>
          </div>

          <h1 className="mb-2 text-center text-title-lg font-semibold text-on-surface">
            <FormattedMessage id="coppa.title" />
          </h1>
          <p className="mb-6 text-center text-body-md text-on-surface-variant">
            <FormattedMessage id="coppa.subtitle" />
          </p>

          {/* What we collect */}
          <div className="mb-6 rounded-xl bg-surface-container-low p-4">
            <p className="text-body-md text-on-surface">
              <FormattedMessage id="coppa.description" />
            </p>
          </div>

          {/* Error */}
          {errorMessage && (
            <div
              role="alert"
              aria-live="assertive"
              className="mb-4 rounded-lg bg-error-container px-4 py-3 text-body-sm text-on-error-container"
            >
              {errorMessage}
            </div>
          )}

          {/* Acknowledgment checkbox */}
          <label
            htmlFor="coppa-acknowledge"
            className="mb-6 flex cursor-pointer select-none items-start gap-3"
          >
            <input
              id="coppa-acknowledge"
              type="checkbox"
              checked={acknowledged}
              onChange={(e) => setAcknowledged(e.target.checked)}
              className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-primary checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            />
            <span className="text-body-md text-on-surface">
              <FormattedMessage id="coppa.acknowledge" />
            </span>
          </label>

          {/* Privacy link */}
          <p className="mb-6 text-body-sm text-on-surface-variant">
            <FormattedMessage
              id="coppa.privacy"
              values={{
                privacyLink: (
                  <Link
                    to="/legal/privacy"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="font-medium text-primary hover:underline"
                  >
                    Privacy Policy
                  </Link>
                ),
              }}
            />
          </p>

          <Button
            variant="primary"
            onClick={handleConsent}
            loading={isConsenting}
            disabled={!acknowledged || isConsenting}
            className="w-full"
          >
            <FormattedMessage id="coppa.submit" />
          </Button>
        </div>
      </div>
    </div>
  );
}
