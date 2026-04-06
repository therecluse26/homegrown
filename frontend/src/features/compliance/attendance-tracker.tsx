import { FormattedMessage, useIntl } from "react-intl";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Icon,
  ProgressBar,
  Select,
  Skeleton,
} from "@/components/ui";
import { TierGate } from "@/components/common/tier-gate";
import {
  useAttendance,
  useAttendanceSummary,
  useRecordAttendance,
  type AttendanceStatus,
  type PaceStatus,
} from "@/hooks/use-compliance";
import { useStudents } from "@/hooks/use-family";
import { useAuth } from "@/hooks/use-auth";
import { useState, useEffect, useRef, useMemo } from "react";

// ─── Helpers ────────────────────────────────────────────────────────────────

const STATUS_COLORS: Record<AttendanceStatus, string> = {
  present: "bg-primary",
  partial: "bg-secondary",
  absent: "bg-error",
  excused: "bg-outline-variant",
};

const STATUS_LABELS: Record<AttendanceStatus, string> = {
  present: "compliance.attendance.present",
  absent: "compliance.attendance.absent",
  partial: "compliance.attendance.partial",
  excused: "compliance.attendance.excused",
};

function getPaceVariant(pace: PaceStatus): "primary" | "secondary" | "error" {
  switch (pace) {
    case "ahead":
      return "primary";
    case "on_track":
      return "secondary";
    case "behind":
      return "error";
  }
}

function getPaceLabelId(pace: PaceStatus): string {
  switch (pace) {
    case "ahead":
      return "compliance.attendance.pace.ahead";
    case "on_track":
      return "compliance.attendance.pace.onTrack";
    case "behind":
      return "compliance.attendance.pace.behind";
  }
}

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate();
}

function getFirstDayOfMonth(year: number, month: number): number {
  return new Date(year, month, 1).getDay();
}

// ─── Calendar heatmap ──────────────────────────────────────────────────────

function AttendanceHeatmap({
  entries,
  year,
  month,
  onDayClick,
}: {
  entries: { date: string; status: AttendanceStatus; auto_generated: boolean }[];
  year: number;
  month: number;
  onDayClick: (date: string) => void;
}) {
  const intl = useIntl();
  const daysInMonth = getDaysInMonth(year, month);
  const firstDay = getFirstDayOfMonth(year, month);

  const entryMap = useMemo(() => {
    const map = new Map<string, { status: AttendanceStatus; auto_generated: boolean }>();
    for (const entry of entries) {
      map.set(entry.date, { status: entry.status, auto_generated: entry.auto_generated });
    }
    return map;
  }, [entries]);

  const dayNames = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"];

  return (
    <div>
      <div className="grid grid-cols-7 gap-1 mb-1">
        {dayNames.map((d) => (
          <div
            key={d}
            className="type-label-sm text-on-surface-variant text-center py-1"
          >
            {d}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-1">
        {/* Empty cells for days before month starts */}
        {Array.from({ length: firstDay }, (_, i) => (
          <div key={`empty-${i}`} className="aspect-square" />
        ))}
        {/* Day cells */}
        {Array.from({ length: daysInMonth }, (_, i) => {
          const day = i + 1;
          const dateStr = `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
          const entry = entryMap.get(dateStr);
          const isToday =
            dateStr ===
            new Date().toISOString().split("T")[0];

          return (
            <button
              key={day}
              type="button"
              onClick={() => onDayClick(dateStr)}
              className={`aspect-square rounded-radius-sm flex items-center justify-center type-label-sm relative transition-colors touch-target ${
                entry
                  ? `${STATUS_COLORS[entry.status]} text-on-primary`
                  : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
              } ${isToday ? "ring-2 ring-primary ring-offset-1" : ""}`}
              aria-label={`${intl.formatDate(dateStr, { month: "long", day: "numeric" })}${entry ? ` — ${intl.formatMessage({ id: STATUS_LABELS[entry.status] })}` : ""}`}
            >
              {day}
              {entry?.auto_generated && (
                <span className="absolute bottom-0.5 right-0.5 w-1.5 h-1.5 rounded-radius-full bg-tertiary" />
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AttendanceTracker() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier } = useAuth();
  const students = useStudents();
  const now = new Date();
  const [selectedStudentId, setSelectedStudentId] = useState("");
  const summaries = useAttendanceSummary(selectedStudentId);
  const recordAttendance = useRecordAttendance();
  const [currentYear, setCurrentYear] = useState(now.getFullYear());
  const [currentMonth, setCurrentMonth] = useState(now.getMonth());
  const [selectedDate, setSelectedDate] = useState<string | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<AttendanceStatus>("present");

  const monthStr = `${currentYear}-${String(currentMonth + 1).padStart(2, "0")}`;
  const attendance = useAttendance(selectedStudentId, monthStr);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "compliance.attendance.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // Auto-select first student
  useEffect(() => {
    const firstStudent = students.data?.[0];
    if (!selectedStudentId && firstStudent?.id) {
      setSelectedStudentId(firstStudent.id);
    }
  }, [students.data, selectedStudentId]);

  const handlePrevMonth = () => {
    if (currentMonth === 0) {
      setCurrentYear((y) => y - 1);
      setCurrentMonth(11);
    } else {
      setCurrentMonth((m) => m - 1);
    }
  };

  const handleNextMonth = () => {
    if (currentMonth === 11) {
      setCurrentYear((y) => y + 1);
      setCurrentMonth(0);
    } else {
      setCurrentMonth((m) => m + 1);
    }
  };

  const handleRecord = () => {
    if (!selectedDate || !selectedStudentId) return;
    recordAttendance.mutate(
      {
        student_id: selectedStudentId,
        date: selectedDate,
        status: selectedStatus,
      },
      {
        onSuccess: () => setSelectedDate(null),
      },
    );
  };

  // Tier gate
  if (tier === "free") {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.attendance.title" />
        </h1>
        <TierGate featureName="Compliance Tracking" />
      </div>
    );
  }

  if (students.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-96" />
      </div>
    );
  }

  const studentList = students.data ?? [];
  const studentSummary = summaries.data ?? null;
  const attendanceEntries = attendance.data ?? [];

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="compliance.attendance.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="compliance.attendance.description" />
      </p>

      {/* Student selector */}
      <div className="mb-4">
        <Select
          value={selectedStudentId}
          onChange={(e) => setSelectedStudentId(e.target.value)}
        >
          {studentList.map((s) => (
            <option key={s.id} value={s.id}>
              {s.display_name}
            </option>
          ))}
        </Select>
      </div>

      {/* Summary card */}
      {studentSummary && (
        <Card className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <Icon icon={CalendarDays} size="md" className="text-primary" aria-hidden />
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id="compliance.attendance.pace.label" />
              </p>
            </div>
            <Badge variant={getPaceVariant(studentSummary.pace)}>
              <FormattedMessage id={getPaceLabelId(studentSummary.pace)} />
            </Badge>
          </div>
          <ProgressBar
            value={
              studentSummary.days_required > 0
                ? (studentSummary.days_present / studentSummary.days_required) * 100
                : 0
            }
          />
          <p className="type-body-sm text-on-surface-variant mt-1" aria-live="polite">
            <FormattedMessage
              id="compliance.attendance.summary"
              values={{
                count: studentSummary.days_present,
                required: studentSummary.days_required,
                percentage:
                  studentSummary.days_required > 0
                    ? Math.round(
                        (studentSummary.days_present /
                          studentSummary.days_required) *
                          100,
                      )
                    : 0,
              }}
            />
          </p>
        </Card>
      )}

      {/* Calendar */}
      <Card className="mb-4">
        {/* Month navigation */}
        <div className="flex items-center justify-between mb-4">
          <Button variant="tertiary" size="sm" onClick={handlePrevMonth}>
            <Icon icon={ChevronLeft} size="sm" aria-hidden />
          </Button>
          <h2 className="type-title-md text-on-surface font-semibold">
            {intl.formatDate(new Date(currentYear, currentMonth), {
              month: "long",
              year: "numeric",
            })}
          </h2>
          <Button variant="tertiary" size="sm" onClick={handleNextMonth}>
            <Icon icon={ChevronRight} size="sm" aria-hidden />
          </Button>
        </div>

        {attendance.isPending ? (
          <Skeleton height="h-64" />
        ) : (
          <AttendanceHeatmap
            entries={attendanceEntries}
            year={currentYear}
            month={currentMonth}
            onDayClick={(date) => setSelectedDate(date)}
          />
        )}

        {/* Legend */}
        <div className="flex items-center gap-4 mt-4 pt-3 border-t border-outline-variant/20">
          <p className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="compliance.attendance.legend" />:
          </p>
          {(Object.keys(STATUS_COLORS) as AttendanceStatus[]).map(
            (status) => (
              <div key={status} className="flex items-center gap-1">
                <span
                  className={`w-3 h-3 rounded-radius-sm ${STATUS_COLORS[status]}`}
                />
                <span className="type-label-sm text-on-surface-variant">
                  <FormattedMessage id={STATUS_LABELS[status]} />
                </span>
              </div>
            ),
          )}
          <div className="flex items-center gap-1">
            <span className="relative w-3 h-3 rounded-radius-sm bg-surface-container-high">
              <span className="absolute bottom-0 right-0 w-1.5 h-1.5 rounded-radius-full bg-tertiary" />
            </span>
            <span className="type-label-sm text-on-surface-variant">
              <FormattedMessage id="compliance.attendance.autoGenerated" />
            </span>
          </div>
        </div>
      </Card>

      {/* Record attendance for selected day */}
      {selectedDate && (
        <Card>
          <h3 className="type-title-sm text-on-surface font-medium mb-3">
            {intl.formatDate(selectedDate, {
              weekday: "long",
              month: "long",
              day: "numeric",
            })}
          </h3>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="type-label-md text-on-surface font-medium mb-1 block">
                <FormattedMessage id="compliance.attendance.status" />
              </label>
              <Select
                value={selectedStatus}
                onChange={(e) =>
                  setSelectedStatus(e.target.value as AttendanceStatus)
                }
              >
                <option value="present">
                  {intl.formatMessage({
                    id: "compliance.attendance.present",
                  })}
                </option>
                <option value="absent">
                  {intl.formatMessage({
                    id: "compliance.attendance.absent",
                  })}
                </option>
                <option value="partial">
                  {intl.formatMessage({
                    id: "compliance.attendance.partial",
                  })}
                </option>
                <option value="excused">
                  {intl.formatMessage({
                    id: "compliance.attendance.excused",
                  })}
                </option>
              </Select>
            </div>
            <Button
              variant="primary"
              onClick={handleRecord}
              disabled={recordAttendance.isPending}
            >
              <FormattedMessage id="compliance.setup.save" />
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}
