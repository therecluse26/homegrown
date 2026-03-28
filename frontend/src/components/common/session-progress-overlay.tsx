import { useState, useEffect } from "react";
import { FormattedMessage } from "react-intl";
import { Clock, ChevronUp, ChevronDown, Activity } from "lucide-react";
import { Card, Icon, ProgressBar } from "@/components/ui";
import { useStudentSession } from "@/hooks/use-student-session";

/**
 * Glassmorphism progress overlay for student sessions.
 * Shows session progress, time remaining, and current activity.
 * Collapsible to a minimal bar to avoid obscuring content.
 */
export function SessionProgressOverlay() {
  const { session } = useStudentSession();
  const [collapsed, setCollapsed] = useState(false);
  const [minutesRemaining, setMinutesRemaining] = useState<number | null>(null);
  const [progressPct, setProgressPct] = useState(100);

  useEffect(() => {
    if (!session?.expiresAt) return;

    const update = () => {
      const now = Date.now();
      const expiresAt = new Date(session.expiresAt!).getTime();
      const startedAt = new Date(session.startedAt).getTime();
      const total = expiresAt - startedAt;
      const remaining = expiresAt - now;

      setMinutesRemaining(Math.max(0, Math.ceil(remaining / 60_000)));
      setProgressPct(
        Math.max(0, Math.min(100, ((total - remaining) / total) * 100)),
      );
    };

    update();
    const interval = setInterval(update, 30_000);
    return () => clearInterval(interval);
  }, [session?.expiresAt, session?.startedAt]);

  if (!session) return null;

  if (collapsed) {
    return (
      <button
        type="button"
        onClick={() => setCollapsed(false)}
        className="fixed bottom-20 right-4 z-[var(--z-sticky)] px-3 py-2 rounded-full bg-secondary-container/90 backdrop-blur-[20px] text-on-secondary-container flex items-center gap-2 shadow-elevation-1 transition-all hover:bg-secondary-container"
      >
        <Icon icon={Clock} size="sm" aria-hidden />
        {minutesRemaining !== null && (
          <span className="type-label-sm font-medium">
            <FormattedMessage
              id="session.overlay.minutesLeft"
              values={{ minutes: minutesRemaining }}
            />
          </span>
        )}
        <Icon icon={ChevronUp} size="xs" aria-hidden />
      </button>
    );
  }

  return (
    <div className="fixed bottom-20 left-4 right-4 z-[var(--z-sticky)]">
      <Card className="bg-secondary-container/80 backdrop-blur-[20px] shadow-elevation-2">
        <div className="flex items-start justify-between gap-3">
          <div className="flex-1 min-w-0 space-y-2">
            {/* Student name & session label */}
            <div className="flex items-center gap-2">
              <Icon
                icon={Activity}
                size="sm"
                className="text-on-secondary-container"
                aria-hidden
              />
              <span className="type-label-md text-on-secondary-container font-medium">
                <FormattedMessage
                  id="session.overlay.label"
                  values={{ name: session.studentName }}
                />
              </span>
            </div>

            {/* Time remaining */}
            {minutesRemaining !== null && (
              <div className="flex items-center gap-2">
                <Icon
                  icon={Clock}
                  size="xs"
                  className="text-on-secondary-container/70"
                  aria-hidden
                />
                <span className="type-label-sm text-on-secondary-container/80">
                  <FormattedMessage
                    id="session.overlay.timeRemaining"
                    values={{ minutes: minutesRemaining }}
                  />
                </span>
              </div>
            )}

            {/* Progress bar */}
            {session.expiresAt && (
              <ProgressBar value={progressPct} />
            )}
          </div>

          {/* Collapse button */}
          <button
            type="button"
            onClick={() => setCollapsed(true)}
            className="shrink-0 p-1 rounded-full hover:bg-on-secondary-container/10 transition-colors text-on-secondary-container"
            aria-label="Minimize"
          >
            <Icon icon={ChevronDown} size="sm" aria-hidden />
          </button>
        </div>
      </Card>
    </div>
  );
}
