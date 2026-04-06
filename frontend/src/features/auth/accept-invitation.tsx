import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui";
import { PageTitle } from "@/components/common";
import { apiClient } from "@/api/client";

/**
 * Co-parent invitation acceptance page — rendered inside AuthLayout.
 *
 * Uses the token from the email link to accept the co-parent role.
 * Route: /auth/accept-invite/:token
 *
 * @see SPEC §3.4 (co-parent invitations)
 * @see 01-iam §7 (POST /families/invites/{token}/accept)
 */
export function AcceptInvitation() {
  const intl = useIntl();
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [isAccepting, setIsAccepting] = useState(false);
  const [error, setError] = useState<string>("");
  const [accepted, setAccepted] = useState(false);

  if (!token) {
    return (
      <>
        <PageTitle
          title={intl.formatMessage({ id: "auth.acceptInvitation.title" })}
        />
        <div className="text-center">
          <p className="text-body-md text-on-surface">
            <FormattedMessage id="auth.acceptInvitation.invalid" />
          </p>
          <Link
            to="/"
            className="mt-4 inline-block text-label-md font-medium text-primary hover:underline"
          >
            <FormattedMessage id="error.goHome" />
          </Link>
        </div>
      </>
    );
  }

  async function handleAccept() {
    if (isAccepting) return;
    setIsAccepting(true);
    setError("");

    try {
      await apiClient(`/v1/families/invites/${token}/accept`, {
        method: "POST",
      });
      setAccepted(true);
      await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
      await queryClient.invalidateQueries({ queryKey: ["family"] });
    } catch {
      setError(intl.formatMessage({ id: "error.generic" }));
      setIsAccepting(false);
    }
  }

  function handleDecline() {
    navigate("/", { replace: true });
  }

  if (accepted) {
    return (
      <>
        <PageTitle
          title={intl.formatMessage({ id: "auth.acceptInvitation.title" })}
        />
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
            <FormattedMessage id="auth.acceptInvitation.accept" />
          </h2>
          <button
            type="button"
            onClick={() => navigate("/", { replace: true })}
            className="text-label-md font-medium text-primary hover:underline"
          >
            <FormattedMessage id="error.goHome" />
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <PageTitle
        title={intl.formatMessage({ id: "auth.acceptInvitation.title" })}
      />

      <div className="space-y-1 text-center">
        <h2 className="text-title-lg font-semibold text-on-surface">
          <FormattedMessage id="auth.acceptInvitation.title" />
        </h2>
        <p className="text-body-sm text-on-surface-variant">
          <FormattedMessage
            id="auth.acceptInvitation.subtitle"
            values={{
              inviterName: (
                <strong className="text-on-surface">
                  <FormattedMessage
                    id="auth.acceptInvitation.aFamilyMember"
                    defaultMessage="a family member"
                  />
                </strong>
              ),
            }}
          />
        </p>
      </div>

      {error && (
        <div
          role="alert"
          aria-live="assertive"
          className="rounded-lg bg-error-container px-4 py-3 text-body-sm text-on-error-container"
        >
          {error}
        </div>
      )}

      <div className="flex flex-col gap-3">
        <Button
          variant="primary"
          onClick={handleAccept}
          loading={isAccepting}
          disabled={isAccepting}
          className="w-full"
        >
          <FormattedMessage id="auth.acceptInvitation.accept" />
        </Button>
        <Button
          variant="tertiary"
          onClick={handleDecline}
          disabled={isAccepting}
          className="w-full"
        >
          <FormattedMessage id="auth.acceptInvitation.decline" />
        </Button>
      </div>
    </>
  );
}
