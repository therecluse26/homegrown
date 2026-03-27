import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { useIntl, FormattedMessage } from "react-intl";
import { Button, Modal } from "@/components/ui";
import { initLogout } from "@/lib/kratos";
import { apiClient } from "@/api/client";

const WARNING_BEFORE_EXPIRY_MS = 5 * 60 * 1000; // 5 minutes
const SESSION_CHECK_INTERVAL_MS = 60 * 1000; // check every 1 minute

interface SessionTimeoutWarningProps {
  /**
   * ISO string of the session's expiry time (from Kratos session).
   * Pass undefined when no session is active.
   */
  expiresAt: string | undefined;
}

/**
 * Session timeout warning overlay.
 *
 * Shown 5 minutes before the session expires. Includes:
 * - Countdown timer (live region for screen readers)
 * - "Stay logged in" button that refreshes the session
 * - Auto-redirect to /auth/login on timeout
 *
 * @see SPEC §17.1 (session management)
 */
export function SessionTimeoutWarning({ expiresAt }: SessionTimeoutWarningProps) {
  const intl = useIntl();
  const navigate = useNavigate();

  const [isVisible, setIsVisible] = useState(false);
  const [secondsRemaining, setSecondsRemaining] = useState(0);
  const [isExtending, setIsExtending] = useState(false);

  const tickIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const checkIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  function clearTimers() {
    if (tickIntervalRef.current) clearInterval(tickIntervalRef.current);
    if (checkIntervalRef.current) clearInterval(checkIntervalRef.current);
  }

  useEffect(() => {
    if (!expiresAt) {
      clearTimers();
      setIsVisible(false);
      return;
    }

    function checkExpiry() {
      const expiryMs = new Date(expiresAt!).getTime();
      const nowMs = Date.now();
      const remainingMs = expiryMs - nowMs;

      if (remainingMs <= 0) {
        // Session has expired — redirect to login
        clearTimers();
        setIsVisible(false);
        navigate("/auth/login", { replace: true });
        return;
      }

      if (remainingMs <= WARNING_BEFORE_EXPIRY_MS) {
        // Show warning
        setSecondsRemaining(Math.floor(remainingMs / 1000));
        setIsVisible(true);

        // Start per-second countdown if not already running
        if (!tickIntervalRef.current) {
          tickIntervalRef.current = setInterval(() => {
            const remaining = Math.floor(
              (new Date(expiresAt!).getTime() - Date.now()) / 1000,
            );
            if (remaining <= 0) {
              clearTimers();
              setIsVisible(false);
              navigate("/auth/login", { replace: true });
            } else {
              setSecondsRemaining(remaining);
            }
          }, 1000);
        }
      } else {
        setIsVisible(false);
      }
    }

    // Check immediately and then periodically
    checkExpiry();
    checkIntervalRef.current = setInterval(
      checkExpiry,
      SESSION_CHECK_INTERVAL_MS,
    );

    return () => clearTimers();
  }, [expiresAt, navigate]);

  async function handleExtendSession() {
    setIsExtending(true);
    try {
      // Touching the whoami endpoint with credentials refreshes the session cookie
      await apiClient<unknown>("/v1/auth/me");
      setIsVisible(false);
      clearTimers();
    } catch {
      // If extending fails, the session may have already expired
      const { logout_token } = await initLogout().catch(() => ({
        logout_url: "",
        logout_token: "",
      }));
      if (logout_token) {
        await fetch(
          `/self-service/logout?token=${encodeURIComponent(logout_token)}`,
          { credentials: "include" },
        );
      }
      navigate("/auth/login", { replace: true });
    } finally {
      setIsExtending(false);
    }
  }

  const minutesRemaining = Math.floor(secondsRemaining / 60);

  if (!isVisible) return null;

  return (
    <Modal
      open={isVisible}
      onClose={() => {
        // Do not allow dismissing by clicking outside — user must act
      }}
      title={intl.formatMessage({ id: "auth.sessionExpiring" })}
    >
      <div className="space-y-4">
        <p className="text-body-md text-on-surface-variant">
          <FormattedMessage
            id="auth.sessionExpiring.description"
            values={{ minutes: minutesRemaining }}
          />
        </p>

        {/* Live countdown for screen readers */}
        <p
          aria-live="assertive"
          aria-atomic="true"
          className="sr-only"
          role="timer"
        >
          {minutesRemaining > 0
            ? intl.formatMessage(
                { id: "auth.sessionExpiring.description" },
                { minutes: minutesRemaining },
              )
            : `${secondsRemaining} seconds remaining`}
        </p>

        {/* Visual countdown */}
        <div className="flex items-center justify-center rounded-xl bg-surface-container-low px-6 py-4">
          <span
            className="text-display-sm font-semibold tabular-nums text-on-surface"
            aria-hidden="true"
          >
            {minutesRemaining > 0
              ? `${minutesRemaining}:${String(secondsRemaining % 60).padStart(2, "0")}`
              : `0:${String(secondsRemaining).padStart(2, "0")}`}
          </span>
        </div>

        <div className="flex flex-col gap-2">
          <Button
            variant="primary"
            onClick={handleExtendSession}
            loading={isExtending}
            disabled={isExtending}
            className="w-full"
          >
            <FormattedMessage id="auth.extendSession" />
          </Button>
        </div>
      </div>
    </Modal>
  );
}
