import { useState, useEffect, useRef, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  Clock,
  Shield,
  UserRound,
  AlertTriangle,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  ConfirmationDialog,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";

// ─── Types ──────────────────────────────────────────────────────────────────

type SessionDuration = "1h" | "2h" | "4h" | "end-of-day";

const DURATION_MINUTES: Record<SessionDuration, number | null> = {
  "1h": 60,
  "2h": 120,
  "4h": 240,
  "end-of-day": null, // computed from current time to midnight
};

// ─── Duration picker ─────────────────────────────────────────────────────────

function DurationOption({
  duration,
  labelId,
  selected,
  onSelect,
}: {
  duration: SessionDuration;
  labelId: string;
  selected: boolean;
  onSelect: (d: SessionDuration) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(duration)}
      className={`flex items-center gap-3 p-4 rounded-xl transition-colors text-left ${
        selected
          ? "bg-primary-container text-on-primary-container"
          : "bg-surface-container-lowest text-on-surface hover:bg-surface-container-low"
      }`}
    >
      <Icon icon={Clock} size="md" aria-hidden />
      <span className="type-title-sm font-medium">
        <FormattedMessage id={labelId} />
      </span>
    </button>
  );
}

// ─── Timeout warning overlay ─────────────────────────────────────────────────

function TimeoutWarning({
  minutesRemaining,
  onExtend,
  onEnd,
}: {
  minutesRemaining: number;
  onExtend: () => void;
  onEnd: () => void;
}) {
  return (
    <div className="fixed inset-0 z-[var(--z-modal)] flex items-center justify-center bg-scrim/50 backdrop-blur-sm">
      <Card className="max-w-sm mx-auto text-center space-y-4 p-6">
        <div className="text-warning mx-auto w-fit">
          <Icon icon={AlertTriangle} size="xl" aria-hidden />
        </div>
        <h2 className="type-title-lg text-on-surface font-semibold">
          <FormattedMessage id="session.timeout.title" />
        </h2>
        <p className="type-body-md text-on-surface-variant">
          <FormattedMessage
            id="session.timeout.message"
            values={{ minutes: minutesRemaining }}
          />
        </p>
        <div className="flex gap-3 justify-center">
          <Button variant="primary" onClick={onExtend}>
            <FormattedMessage id="session.timeout.extend" />
          </Button>
          <Button variant="secondary" onClick={onEnd}>
            <FormattedMessage id="session.timeout.end" />
          </Button>
        </div>
      </Card>
    </div>
  );
}

// ─── Session timer hook ──────────────────────────────────────────────────────

function useSessionTimer(
  expiresAt: Date | null,
  onWarning: () => void,
  onExpire: () => void,
) {
  const warningFired = useRef(false);
  const expireFired = useRef(false);

  useEffect(() => {
    if (!expiresAt) return;

    const checkTimer = () => {
      const remaining = expiresAt.getTime() - Date.now();
      const minutesLeft = remaining / 60_000;

      if (minutesLeft <= 5 && !warningFired.current) {
        warningFired.current = true;
        onWarning();
      }
      if (minutesLeft <= 0 && !expireFired.current) {
        expireFired.current = true;
        onExpire();
      }
    };

    const interval = setInterval(checkTimer, 10_000);
    checkTimer();

    return () => clearInterval(interval);
  }, [expiresAt, onWarning, onExpire]);
}

// ─── Main component ──────────────────────────────────────────────────────────

export function StudentSessionLauncher() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: students, isPending } = useStudents();

  // Session setup state
  const [selectedStudentId, setSelectedStudentId] = useState<string | null>(
    null,
  );
  const [duration, setDuration] = useState<SessionDuration>("1h");
  const [showAgeGate, setShowAgeGate] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  // Active session state
  const [activeSession, setActiveSession] = useState<{
    studentId: string;
    studentName: string;
    expiresAt: Date | null;
  } | null>(null);
  const [showTimeoutWarning, setShowTimeoutWarning] = useState(false);

  const selectedStudent = students?.find((s) => s.id === selectedStudentId);

  // Age gate: students 10+ need a simple verification
  const currentYear = new Date().getFullYear();

  const handleStartSession = useCallback(() => {
    if (!selectedStudentId || !selectedStudent) return;

    // Calculate expiry
    let expiresAt: Date | null = null;
    const durationMins = DURATION_MINUTES[duration];
    if (durationMins) {
      expiresAt = new Date(Date.now() + durationMins * 60_000);
    } else {
      // End of day
      const eod = new Date();
      eod.setHours(23, 59, 59, 999);
      expiresAt = eod;
    }

    setActiveSession({
      studentId: selectedStudentId,
      studentName: selectedStudent.display_name ?? "",
      expiresAt,
    });
    setShowConfirm(false);

    // Navigate to student shell
    void navigate("/student");
  }, [selectedStudentId, selectedStudent, duration, navigate]);

  const handleStudentSelect = (studentId: string) => {
    setSelectedStudentId(studentId);
    const student = students?.find((s) => s.id === studentId);
    const age = student?.birth_year
      ? currentYear - student.birth_year
      : null;
    if (age !== null && age >= 10) {
      setShowAgeGate(true);
    } else {
      setShowConfirm(true);
    }
  };

  const handleAgeGateConfirm = () => {
    setShowAgeGate(false);
    setShowConfirm(true);
  };

  const handleEndSession = useCallback(() => {
    setActiveSession(null);
    setShowTimeoutWarning(false);
    void navigate("/learning");
  }, [navigate]);

  const handleExtendSession = useCallback(() => {
    if (!activeSession) return;
    // Extend by 30 minutes
    const newExpiry = new Date(
      (activeSession.expiresAt?.getTime() ?? Date.now()) + 30 * 60_000,
    );
    setActiveSession({ ...activeSession, expiresAt: newExpiry });
    setShowTimeoutWarning(false);
  }, [activeSession]);

  const handleWarning = useCallback(() => {
    setShowTimeoutWarning(true);
  }, []);

  useSessionTimer(
    activeSession?.expiresAt ?? null,
    handleWarning,
    handleEndSession,
  );

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <RouterLink to="/learning" className="no-underline">
          <Button variant="tertiary" size="sm">
            <Icon icon={ArrowLeft} size="sm" aria-hidden />
            <span className="ml-1">
              <FormattedMessage id="common.back" />
            </span>
          </Button>
        </RouterLink>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="session.launcher.title" />
        </h1>
      </div>

      {/* Active session banner */}
      {activeSession && (
        <Card className="bg-primary-container">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Icon
                icon={Shield}
                size="md"
                className="text-on-primary-container"
                aria-hidden
              />
              <div>
                <p className="type-title-sm text-on-primary-container font-medium">
                  <FormattedMessage
                    id="session.active"
                    values={{ name: activeSession.studentName }}
                  />
                </p>
                {activeSession.expiresAt && (
                  <p className="type-label-sm text-on-primary-container/80">
                    <FormattedMessage
                      id="session.expiresAt"
                      values={{
                        time: activeSession.expiresAt.toLocaleTimeString([], {
                          hour: "2-digit",
                          minute: "2-digit",
                        }),
                      }}
                    />
                  </p>
                )}
              </div>
            </div>
            <Button variant="secondary" size="sm" onClick={handleEndSession}>
              <FormattedMessage id="session.end" />
            </Button>
          </div>
        </Card>
      )}

      {/* Student selector */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="session.selectStudent" />
        </h2>
        {isPending ? (
          <div className="space-y-2">
            <Skeleton height="h-16" />
            <Skeleton height="h-16" />
          </div>
        ) : !students || students.length === 0 ? (
          <EmptyState
            message={intl.formatMessage({ id: "session.noStudents" })}
            description={intl.formatMessage({
              id: "session.noStudents.description",
            })}
          />
        ) : (
          <div className="space-y-2">
            {students.map((student) => (
              <button
                key={student.id}
                type="button"
                onClick={() => handleStudentSelect(student.id ?? "")}
                className="w-full text-left"
              >
                <Card
                  interactive
                  className="flex items-center gap-3"
                >
                  <div className="p-2 rounded-full bg-primary-container text-on-primary-container">
                    <Icon icon={UserRound} size="md" aria-hidden />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="type-title-sm text-on-surface font-medium">
                      {student.display_name}
                    </p>
                    {student.birth_year && (
                      <p className="type-label-sm text-on-surface-variant">
                        <FormattedMessage
                          id="session.studentAge"
                          values={{
                            age: currentYear - student.birth_year,
                          }}
                        />
                      </p>
                    )}
                  </div>
                </Card>
              </button>
            ))}
          </div>
        )}
      </section>

      {/* Age gate dialog */}
      {showAgeGate && (
        <ConfirmationDialog
          open={showAgeGate}
          onClose={() => setShowAgeGate(false)}
          onConfirm={handleAgeGateConfirm}
          title={intl.formatMessage({ id: "session.ageGate.title" })}
          confirmLabel={intl.formatMessage({ id: "session.ageGate.confirm" })}
        >
          <p className="type-body-md text-on-surface-variant">
            <FormattedMessage
              id="session.ageGate.message"
              values={{ name: selectedStudent?.display_name ?? "" }}
            />
          </p>
        </ConfirmationDialog>
      )}

      {/* Session configuration dialog */}
      {showConfirm && selectedStudent && (
        <ConfirmationDialog
          open={showConfirm}
          onClose={() => setShowConfirm(false)}
          onConfirm={handleStartSession}
          title={intl.formatMessage(
            { id: "session.confirm.title" },
            { name: selectedStudent.display_name ?? "" },
          )}
          confirmLabel={intl.formatMessage({ id: "session.confirm.start" })}
        >
          <div className="space-y-4">
            <p className="type-body-md text-on-surface-variant">
              <FormattedMessage id="session.confirm.description" />
            </p>
            <div className="space-y-2">
              <p className="type-label-md text-on-surface font-medium">
                <FormattedMessage id="session.duration.label" />
              </p>
              <div className="grid grid-cols-2 gap-2">
                <DurationOption
                  duration="1h"
                  labelId="session.duration.1h"
                  selected={duration === "1h"}
                  onSelect={setDuration}
                />
                <DurationOption
                  duration="2h"
                  labelId="session.duration.2h"
                  selected={duration === "2h"}
                  onSelect={setDuration}
                />
                <DurationOption
                  duration="4h"
                  labelId="session.duration.4h"
                  selected={duration === "4h"}
                  onSelect={setDuration}
                />
                <DurationOption
                  duration="end-of-day"
                  labelId="session.duration.endOfDay"
                  selected={duration === "end-of-day"}
                  onSelect={setDuration}
                />
              </div>
            </div>
          </div>
        </ConfirmationDialog>
      )}

      {/* Timeout warning overlay */}
      {showTimeoutWarning && (
        <TimeoutWarning
          minutesRemaining={5}
          onExtend={handleExtendSession}
          onEnd={handleEndSession}
        />
      )}
    </div>
  );
}
