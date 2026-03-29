import { FormattedMessage, useIntl } from "react-intl";
import { Monitor, Smartphone, Globe, LogOut } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { useSessions, useRevokeSession, useRevokeAllSessions } from "@/hooks/use-sessions";
import { useState, useEffect, useRef } from "react";

// ─── Helpers ────────────────────────────────────────────────────────────────

function getDeviceIcon(device: string) {
  const lower = device.toLowerCase();
  if (lower.includes("mobile") || lower.includes("phone"))
    return Smartphone;
  if (lower.includes("desktop") || lower.includes("laptop"))
    return Monitor;
  return Globe;
}

function formatRelativeTime(
  dateStr: string,
  intl: ReturnType<typeof useIntl>,
) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return intl.formatMessage({ id: "sessions.time.justNow" });
  if (minutes < 60)
    return intl.formatMessage(
      { id: "sessions.time.minutes" },
      { count: minutes },
    );
  const hours = Math.floor(minutes / 60);
  if (hours < 24)
    return intl.formatMessage(
      { id: "sessions.time.hours" },
      { count: hours },
    );
  const days = Math.floor(hours / 24);
  return intl.formatMessage({ id: "sessions.time.days" }, { count: days });
}

// ─── Component ──────────────────────────────────────────────────────────────

export function SessionManagement() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const sessions = useSessions();
  const revokeSession = useRevokeSession();
  const revokeAll = useRevokeAllSessions();

  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [showRevokeAll, setShowRevokeAll] = useState(false);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "sessions.title" })} — Homegrown Academy`;
    headingRef.current?.focus();
  }, [intl]);

  if (sessions.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </div>
    );
  }

  if (sessions.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-md text-on-surface font-semibold outline-none mb-6">
          <FormattedMessage id="sessions.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const sessionList = sessions.data ?? [];
  const otherSessions = sessionList.filter((s) => !s.is_current);

  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-center justify-between mb-6">
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-md text-on-surface font-semibold outline-none">
          <FormattedMessage id="sessions.title" />
        </h1>
        {otherSessions.length > 0 && (
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setShowRevokeAll(true)}
            className="text-error"
          >
            <Icon icon={LogOut} size="xs" aria-hidden className="mr-1.5" />
            <FormattedMessage id="sessions.revokeAll" />
          </Button>
        )}
      </div>

      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="sessions.description" />
      </p>

      {sessionList.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "sessions.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {sessionList.map((session) => {
            const DeviceIcon = getDeviceIcon(session.device);
            return (
              <li key={session.id}>
                <Card className="flex items-center justify-between">
                  <div className="flex items-start gap-3">
                    <Icon
                      icon={DeviceIcon}
                      size="md"
                      aria-hidden
                      className="text-on-surface-variant mt-0.5 shrink-0"
                    />
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="type-title-sm text-on-surface font-medium">
                          {session.browser || session.device}
                        </p>
                        {session.is_current && (
                          <Badge variant="primary">
                            <FormattedMessage id="sessions.current" />
                          </Badge>
                        )}
                      </div>
                      <p className="type-body-sm text-on-surface-variant">
                        {session.ip_address} ·{" "}
                        {formatRelativeTime(session.last_active, intl)}
                      </p>
                    </div>
                  </div>
                  {!session.is_current && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      onClick={() => setRevokeTarget(session.id)}
                      className="text-error shrink-0"
                    >
                      <FormattedMessage id="sessions.revoke" />
                    </Button>
                  )}
                </Card>
              </li>
            );
          })}
        </ul>
      )}

      {/* Revoke single session */}
      <ConfirmationDialog
        open={!!revokeTarget}
        onClose={() => setRevokeTarget(null)}
        onConfirm={() => {
          if (revokeTarget) {
            void revokeSession.mutateAsync(revokeTarget).then(() => {
              setRevokeTarget(null);
            });
          }
        }}
        title={intl.formatMessage({ id: "sessions.revoke.title" })}
        confirmLabel={intl.formatMessage({ id: "sessions.revoke.confirm" })}
        destructive
        loading={revokeSession.isPending}
      >
        <FormattedMessage id="sessions.revoke.description" />
      </ConfirmationDialog>

      {/* Revoke all sessions */}
      <ConfirmationDialog
        open={showRevokeAll}
        onClose={() => setShowRevokeAll(false)}
        onConfirm={() => {
          void revokeAll.mutateAsync().then(() => {
            setShowRevokeAll(false);
          });
        }}
        title={intl.formatMessage({ id: "sessions.revokeAll.title" })}
        confirmLabel={intl.formatMessage({
          id: "sessions.revokeAll.confirm",
        })}
        destructive
        loading={revokeAll.isPending}
      >
        <FormattedMessage id="sessions.revokeAll.description" />
      </ConfirmationDialog>
    </div>
  );
}
