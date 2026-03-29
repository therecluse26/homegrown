import { useState, useMemo, useCallback, useRef, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  Printer,
  CheckCircle2,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Select,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useCalendar } from "@/hooks/use-planning";
import type { CalendarItem, CalendarSource } from "@/hooks/use-planning";
import { useStudents } from "@/hooks/use-family";
import { useFamilyProfile } from "@/hooks/use-family";

// ─── Date helpers ───────────────────────────────────────────────────────────

function formatDate(date: Date): string {
  return date.toISOString().slice(0, 10);
}

function addDays(date: Date, n: number): Date {
  const d = new Date(date);
  d.setDate(d.getDate() + n);
  return d;
}

function getWeekStart(date: Date): Date {
  const d = new Date(date);
  const day = d.getDay();
  const diff = day === 0 ? -6 : 1 - day;
  d.setDate(d.getDate() + diff);
  return d;
}

function getWeekEnd(date: Date): Date {
  return addDays(getWeekStart(date), 6);
}

// ─── Source styling for print ──────────────────────────────────────────────

const SOURCE_LABELS: Record<CalendarSource, string> = {
  schedule: "planning.calendar.source.schedule",
  activities: "planning.calendar.source.activities",
  events: "planning.calendar.source.events",
  attendance: "planning.calendar.source.attendance",
};

// ─── Print-friendly table row ──────────────────────────────────────────────

function ScheduleRow({ item }: { item: CalendarItem }) {
  const intl = useIntl();
  const sourceLabel = intl.formatMessage({ id: SOURCE_LABELS[item.source] });

  return (
    <tr className="print-row border-b border-outline-variant/10 last:border-b-0">
      <td className="py-2 pr-3 type-body-sm text-on-surface-variant align-top whitespace-nowrap">
        {item.start_time ?? "—"}
        {item.end_time ? ` – ${item.end_time}` : ""}
      </td>
      <td className="py-2 pr-3 type-body-md text-on-surface align-top">
        <span className="font-medium">{item.title}</span>
        {item.details.description && (
          <p className="type-body-sm text-on-surface-variant mt-0.5 line-clamp-2">
            {item.details.description}
          </p>
        )}
      </td>
      <td className="py-2 pr-3 type-label-sm text-on-surface-variant align-top whitespace-nowrap">
        {item.student_name ?? "—"}
      </td>
      <td className="py-2 pr-3 type-label-sm text-on-surface-variant align-top whitespace-nowrap">
        {sourceLabel}
      </td>
      <td className="py-2 type-label-sm text-on-surface-variant align-top whitespace-nowrap">
        {item.category
          ? intl.formatMessage({
              id: `planning.schedule.category.${item.category}`,
              defaultMessage: item.category,
            })
          : "—"}
      </td>
      <td className="py-2 pl-2 align-top text-center">
        {item.is_completed && (
          <Icon icon={CheckCircle2} size="sm" className="text-primary" />
        )}
      </td>
    </tr>
  );
}

// ─── Print header ──────────────────────────────────────────────────────────

function PrintHeader({
  familyName,
  dateRange,
  studentName,
}: {
  familyName: string;
  dateRange: string;
  studentName?: string;
}) {
  return (
    <div className="hidden print:block mb-6">
      <h1 className="type-headline-md text-on-surface font-bold">
        {familyName}
      </h1>
      <p className="type-title-sm text-on-surface-variant mt-1">
        <FormattedMessage id="planning.print.subtitle" /> — {dateRange}
        {studentName && ` — ${studentName}`}
      </p>
    </div>
  );
}

// ─── Main component ────────────────────────────────────────────────────────

export function SchedulePrint() {
  const intl = useIntl();
  const today = useMemo(() => new Date(), []);
  const printRef = useRef<HTMLDivElement>(null);

  const [startDate, setStartDate] = useState(
    formatDate(getWeekStart(today)),
  );
  const [endDate, setEndDate] = useState(
    formatDate(getWeekEnd(today)),
  );
  const [studentFilter, setStudentFilter] = useState<string | undefined>();

  const { data: students } = useStudents();
  const { data: family } = useFamilyProfile();

  const { data: calendar, isPending } = useCalendar({
    start: startDate,
    end: endDate,
    student_id: studentFilter,
  });

  // Flatten all items grouped by date
  const dayGroups = useMemo(() => {
    if (!calendar?.days) return [];
    return calendar.days
      .filter((d) => d.items.length > 0)
      .map((d) => ({
        date: d.date,
        items: [...d.items].sort((a, b) => {
          // Sort by start_time, items without time go last
          if (!a.start_time && !b.start_time) return 0;
          if (!a.start_time) return 1;
          if (!b.start_time) return -1;
          return a.start_time.localeCompare(b.start_time);
        }),
      }));
  }, [calendar]);

  const studentName = useMemo(() => {
    if (!studentFilter || !students) return undefined;
    return students.find((s) => s.id === studentFilter)?.display_name;
  }, [studentFilter, students]);

  const dateRangeLabel = useMemo(() => {
    const start = new Date(startDate + "T00:00:00");
    const end = new Date(endDate + "T00:00:00");
    return `${start.toLocaleDateString(intl.locale, {
      month: "short",
      day: "numeric",
    })} – ${end.toLocaleDateString(intl.locale, {
      month: "short",
      day: "numeric",
      year: "numeric",
    })}`;
  }, [startDate, endDate, intl.locale]);

  const handlePrint = useCallback(() => {
    window.print();
  }, []);

  // Auto-focus print area after data loads
  useEffect(() => {
    if (calendar && printRef.current) {
      printRef.current.focus();
    }
  }, [calendar]);

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "planning.print.pageTitle" })}
      />

      {/* Controls — hidden when printing */}
      <div className="no-print mb-6">
        <div className="flex items-center gap-3 mb-4">
          <RouterLink to="/calendar">
            <Button variant="tertiary" size="sm">
              <Icon icon={ArrowLeft} size="sm" className="mr-1" />
              <FormattedMessage id="planning.print.backToCalendar" />
            </Button>
          </RouterLink>
        </div>

        <Card className="p-card-padding">
          <h2 className="type-title-sm text-on-surface mb-4">
            <FormattedMessage id="planning.print.configTitle" />
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-4">
            <div>
              <label
                htmlFor="print-start"
                className="type-label-md text-on-surface block mb-1"
              >
                <FormattedMessage id="planning.export.startDate" />
              </label>
              <input
                id="print-start"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            </div>
            <div>
              <label
                htmlFor="print-end"
                className="type-label-md text-on-surface block mb-1"
              >
                <FormattedMessage id="planning.export.endDate" />
              </label>
              <input
                id="print-end"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            </div>
            {students && students.length > 0 && (
              <div>
                <label
                  htmlFor="print-student"
                  className="type-label-md text-on-surface block mb-1"
                >
                  <FormattedMessage id="planning.export.student" />
                </label>
                <Select
                  id="print-student"
                  value={studentFilter ?? ""}
                  onChange={(e) =>
                    setStudentFilter(e.target.value || undefined)
                  }
                >
                  <option value="">
                    {intl.formatMessage({
                      id: "planning.export.allStudents",
                    })}
                  </option>
                  {students.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.display_name}
                    </option>
                  ))}
                </Select>
              </div>
            )}
          </div>
          <div className="flex justify-end">
            <Button
              variant="primary"
              size="sm"
              onClick={handlePrint}
              disabled={isPending}
            >
              <Icon icon={Printer} size="sm" className="mr-1" />
              <FormattedMessage id="planning.print.printButton" />
            </Button>
          </div>
        </Card>
      </div>

      {/* Printable area */}
      <div
        ref={printRef}
        tabIndex={-1}
        className="outline-none"
        data-print-keep
      >
        <PrintHeader
          familyName={family?.display_name ?? ""}
          dateRange={dateRangeLabel}
          studentName={studentName}
        />

        {isPending ? (
          <div className="space-y-3 no-print">
            {[1, 2, 3, 4, 5].map((n) => (
              <Skeleton key={n} className="h-12 w-full rounded-radius-sm" />
            ))}
          </div>
        ) : dayGroups.length === 0 ? (
          <Card className="p-card-padding text-center">
            <p className="type-body-md text-on-surface-variant py-8">
              <FormattedMessage id="planning.print.empty" />
            </p>
          </Card>
        ) : (
          <div className="space-y-6">
            {dayGroups.map((group) => {
              const datePart = group.date.slice(0, 10);
              const dateObj = new Date(datePart + "T12:00:00");
              const dayLabel = dateObj.toLocaleDateString(intl.locale, {
                weekday: "long",
                month: "long",
                day: "numeric",
              });

              return (
                <Card
                  key={group.date}
                  className="p-card-padding break-inside-avoid"
                >
                  <h3 className="type-title-sm text-on-surface font-semibold mb-3 pb-2 border-b border-outline-variant/20">
                    {dayLabel}
                  </h3>
                  <table className="w-full">
                    <thead>
                      <tr className="type-label-sm text-on-surface-variant uppercase tracking-wide">
                        <th className="text-left pb-2 pr-3 font-medium">
                          <FormattedMessage id="planning.print.col.time" />
                        </th>
                        <th className="text-left pb-2 pr-3 font-medium">
                          <FormattedMessage id="planning.print.col.item" />
                        </th>
                        <th className="text-left pb-2 pr-3 font-medium">
                          <FormattedMessage id="planning.print.col.student" />
                        </th>
                        <th className="text-left pb-2 pr-3 font-medium">
                          <FormattedMessage id="planning.print.col.source" />
                        </th>
                        <th className="text-left pb-2 pr-3 font-medium">
                          <FormattedMessage id="planning.print.col.category" />
                        </th>
                        <th className="text-center pb-2 pl-2 font-medium">
                          <FormattedMessage id="planning.print.col.done" />
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {group.items.map((item) => (
                        <ScheduleRow
                          key={`${item.source}-${item.id}`}
                          item={item}
                        />
                      ))}
                    </tbody>
                  </table>
                </Card>
              );
            })}
          </div>
        )}

        {/* Print footer with generation timestamp */}
        <p className="hidden print:block mt-6 type-label-sm text-on-surface-variant text-center">
          <FormattedMessage
            id="planning.print.generatedAt"
            values={{
              date: new Date().toLocaleDateString(intl.locale, {
                year: "numeric",
                month: "long",
                day: "numeric",
                hour: "numeric",
                minute: "2-digit",
              }),
            }}
          />
        </p>
      </div>
    </div>
  );
}
