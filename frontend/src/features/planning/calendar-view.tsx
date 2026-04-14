import { useState, useMemo, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink, useParams, useLocation } from "react-router";
import {
  ChevronLeft,
  ChevronRight,
  Plus,
  BookOpen,
  Calendar,
  CheckCircle2,
  CalendarDays,
  Printer,
  Download,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  Select,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useCalendar, useExportSchedule } from "@/hooks/use-planning";
import type { CalendarItem, CalendarSource } from "@/hooks/use-planning";
import { useStudents } from "@/hooks/use-family";

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
  const diff = day === 0 ? -6 : 1 - day; // Monday start
  d.setDate(d.getDate() + diff);
  return d;
}

function getWeekEnd(date: Date): Date {
  return addDays(getWeekStart(date), 6);
}

function isSameDay(a: Date, b: Date): boolean {
  return formatDate(a) === formatDate(b);
}

/** Convert "HH:MM:SS" or "HH:MM" to 12-hour "h:MM AM/PM" format. */
function formatTime(raw: string): string {
  const parts = raw.split(":");
  const hour = parseInt(parts[0] ?? "0", 10);
  const min = parts[1] ?? "00";
  const suffix = hour >= 12 ? "PM" : "AM";
  const display = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
  return `${display}:${min} ${suffix}`;
}

// ─── Source color config ────────────────────────────────────────────────────

const SOURCE_STYLES: Record<
  CalendarSource,
  { bg: string; text: string; border: string; icon: typeof BookOpen; labelId: string }
> = {
  activities: {
    bg: "bg-tertiary-container",
    text: "text-on-tertiary-container",
    border: "border-tertiary",
    icon: BookOpen,
    labelId: "planning.calendar.source.activities",
  },
  events: {
    bg: "bg-primary-container",
    text: "text-on-primary-container",
    border: "border-primary",
    icon: CalendarDays,
    labelId: "planning.calendar.source.events",
  },
  attendance: {
    bg: "bg-secondary-container",
    text: "text-on-secondary-container",
    border: "border-secondary",
    icon: CheckCircle2,
    labelId: "planning.calendar.source.attendance",
  },
  schedule: {
    bg: "bg-surface-container-high",
    text: "text-on-surface",
    border: "border-outline-variant",
    icon: Calendar,
    labelId: "planning.calendar.source.schedule",
  },
};

// ─── Calendar item component ────────────────────────────────────────────────

function CalendarItemCard({ item }: { item: CalendarItem }) {
  const style = SOURCE_STYLES[item.source];

  return (
    <div
      className={`flex items-center gap-2 px-2 py-1.5 rounded-radius-sm ${style.bg} ${style.text} type-label-sm`}
    >
      <Icon icon={style.icon} size="xs" className="shrink-0" />
      <span className="truncate flex-1">{item.title}</span>
      {item.start_time && (
        <span className="shrink-0 opacity-75">{formatTime(item.start_time)}</span>
      )}
      {item.is_completed && (
        <Icon icon={CheckCircle2} size="xs" className="shrink-0 opacity-60" />
      )}
    </div>
  );
}

// ─── Day column (used in week view) ─────────────────────────────────────────

function DayColumn({
  date,
  items,
  isToday,
}: {
  date: Date;
  items: CalendarItem[];
  isToday: boolean;
}) {
  const intl = useIntl();
  const dayName = date.toLocaleDateString(intl.locale, { weekday: "short" });
  const dayNum = date.getDate();

  return (
    <div className="flex-1 min-w-0">
      <div
        className={`text-center pb-2 mb-2 border-b ${
          isToday
            ? "border-primary"
            : "border-outline-variant/10"
        }`}
      >
        <p className="type-label-sm text-on-surface-variant uppercase">
          {dayName}
        </p>
        <p
          className={`type-title-md font-bold ${
            isToday
              ? "text-primary"
              : "text-on-surface"
          }`}
        >
          {dayNum}
        </p>
      </div>
      <div className="space-y-1">
        {items.length === 0 && (
          <p className="type-label-sm text-on-surface-variant text-center py-2 opacity-50">
            —
          </p>
        )}
        {items.map((item) => (
          <CalendarItemCard key={`${item.source}-${item.id}`} item={item} />
        ))}
      </div>
    </div>
  );
}

// ─── Day detail view ────────────────────────────────────────────────────────

function DayDetailView({
  date,
  items,
}: {
  date: Date;
  items: CalendarItem[];
}) {
  const intl = useIntl();

  // Group items by source
  const grouped = useMemo(() => {
    const groups: Record<CalendarSource, CalendarItem[]> = {
      schedule: [],
      activities: [],
      attendance: [],
      events: [],
    };
    for (const item of items) {
      groups[item.source].push(item);
    }
    return groups;
  }, [items]);

  return (
    <div>
      <h2 className="type-title-md text-on-surface mb-4">
        {date.toLocaleDateString(intl.locale, {
          weekday: "long",
          month: "long",
          day: "numeric",
          year: "numeric",
        })}
      </h2>

      {items.length === 0 && (
        <p className="type-body-md text-on-surface-variant text-center py-8">
          <FormattedMessage id="planning.calendar.empty" />
        </p>
      )}

      {(["schedule", "activities", "events", "attendance"] as CalendarSource[]).map(
        (source) => {
          const sourceItems = grouped[source];
          if (sourceItems.length === 0) return null;
          const style = SOURCE_STYLES[source];
          return (
            <div key={source} className="mb-6">
              <h3
                className={`flex items-center gap-2 type-label-lg ${style.text} mb-2`}
              >
                <Icon icon={style.icon} size="sm" />
                <FormattedMessage id={style.labelId} />
                <Badge variant="secondary">{sourceItems.length}</Badge>
              </h3>
              <div className="space-y-2">
                {sourceItems.map((item) => (
                  <Card
                    key={`${item.source}-${item.id}`}
                    className={`p-card-padding border-l-4 ${style.border}`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex-1 min-w-0">
                        <p className="type-title-sm text-on-surface">
                          {item.title}
                        </p>
                        {item.student_name && (
                          <p className="type-label-sm text-on-surface-variant mt-0.5">
                            {item.student_name}
                          </p>
                        )}
                        {item.details.description && (
                          <p className="type-body-sm text-on-surface-variant mt-1 line-clamp-2">
                            {item.details.description}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center gap-2 shrink-0 type-label-sm text-on-surface-variant">
                        {item.start_time && (
                          <span>
                            {formatTime(item.start_time)}
                            {item.end_time && ` – ${formatTime(item.end_time)}`}
                          </span>
                        )}
                        {item.is_completed && (
                          <Icon
                            icon={CheckCircle2}
                            size="sm"
                            className="text-primary"
                          />
                        )}
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          );
        },
      )}
    </div>
  );
}

// ─── Color legend ───────────────────────────────────────────────────────────

function ColorLegend() {
  return (
    <div className="flex flex-wrap gap-3">
      {(Object.entries(SOURCE_STYLES) as [CalendarSource, (typeof SOURCE_STYLES)[CalendarSource]][]).map(
        ([source, style]) => (
          <div key={source} className="flex items-center gap-1.5">
            <span
              className={`w-3 h-3 rounded-full ${style.bg}`}
              aria-hidden
            />
            <span className="type-label-sm text-on-surface-variant">
              <FormattedMessage id={style.labelId} />
            </span>
          </div>
        ),
      )}
    </div>
  );
}

// ─── Export panel ───────────────────────────────────────────────────────────

function ExportPanel({
  defaultStart,
  defaultEnd,
  students,
  onClose,
}: {
  defaultStart: string;
  defaultEnd: string;
  students?: { id?: string; display_name?: string }[];
  onClose: () => void;
}) {
  const intl = useIntl();
  const exportSchedule = useExportSchedule();
  const [format, setFormat] = useState<"csv" | "ical">("ical");
  const [startDate, setStartDate] = useState(defaultStart);
  const [endDate, setEndDate] = useState(defaultEnd);
  const [studentId, setStudentId] = useState("");

  const handleExport = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      exportSchedule.mutate(
        {
          format,
          start_date: startDate,
          end_date: endDate,
          student_id: studentId || undefined,
        },
        {
          onSuccess: (data) => {
            window.open(data.download_url, "_blank", "noopener,noreferrer");
            onClose();
          },
        },
      );
    },
    [format, startDate, endDate, studentId, exportSchedule, onClose],
  );

  return (
    <Card className="p-card-padding">
      <h3 className="type-title-sm text-on-surface mb-4">
        <FormattedMessage id="planning.export.title" />
      </h3>
      <form onSubmit={handleExport} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="type-label-md text-on-surface block mb-1">
              <FormattedMessage id="planning.export.format" />
            </label>
            <div className="flex gap-2">
              {(["ical", "csv"] as const).map((f) => (
                <button
                  key={f}
                  type="button"
                  onClick={() => setFormat(f)}
                  className={`px-3 py-1.5 rounded-radius-sm type-label-md transition-colors ${
                    format === f
                      ? "bg-primary text-on-primary"
                      : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
                  }`}
                >
                  {f.toUpperCase()}
                </button>
              ))}
            </div>
          </div>

          {students && students.length > 0 && (
            <div>
              <label
                htmlFor="export-student"
                className="type-label-md text-on-surface block mb-1"
              >
                <FormattedMessage id="planning.export.student" />
              </label>
              <select
                id="export-student"
                value={studentId}
                onChange={(e) => setStudentId(e.target.value)}
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              >
                <option value="">
                  {intl.formatMessage({ id: "planning.export.allStudents" })}
                </option>
                {students.filter((s) => s.id).map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.display_name ?? s.id}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="export-start"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="planning.export.startDate" />
            </label>
            <input
              id="export-start"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          </div>
          <div>
            <label
              htmlFor="export-end"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="planning.export.endDate" />
            </label>
            <input
              id="export-end"
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="tertiary" size="sm" onClick={onClose}>
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={!startDate || !endDate || exportSchedule.isPending}
          >
            <Icon icon={Download} size="sm" className="mr-1" />
            <FormattedMessage id="planning.export.download" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Calendar page ──────────────────────────────────────────────────────────

type ViewMode = "week" | "day";

export function CalendarView() {
  const intl = useIntl();
  const { date: dateParam } = useParams<{ date: string }>();
  const location = useLocation();
  const today = useMemo(() => new Date(), []);

  // Derive initial view mode from the URL path segment
  const initialViewMode = useMemo((): ViewMode => {
    if (location.pathname.includes("/calendar/day/")) return "day";
    if (location.pathname.includes("/calendar/week/")) return "week";
    return "week";
  }, [location.pathname]);

  // Parse :date param into a Date, falling back to today
  const initialDate = useMemo(() => {
    if (dateParam) {
      const parsed = new Date(dateParam + "T00:00:00");
      if (!isNaN(parsed.getTime())) return parsed;
    }
    return today;
  }, [dateParam, today]);

  const [viewMode, setViewMode] = useState<ViewMode>(initialViewMode);
  const [selectedDate, setSelectedDate] = useState(initialDate);
  const [studentFilter, setStudentFilter] = useState<string | undefined>();

  const { data: students } = useStudents();
  const [showExport, setShowExport] = useState(false);

  // Compute date range for the query
  const dateRange = useMemo(() => {
    if (viewMode === "day") {
      return {
        start: formatDate(selectedDate),
        end: formatDate(addDays(selectedDate, 1)),
      };
    }
    return {
      start: formatDate(getWeekStart(selectedDate)),
      end: formatDate(getWeekEnd(selectedDate)),
    };
  }, [viewMode, selectedDate]);

  const { data: calendar, isPending } = useCalendar({
    start: dateRange.start,
    end: dateRange.end,
    student_id: studentFilter,
  });

  // Navigation
  const navigate = useCallback(
    (direction: -1 | 1) => {
      setSelectedDate((prev) =>
        addDays(prev, direction * (viewMode === "week" ? 7 : 1)),
      );
    },
    [viewMode],
  );

  const goToToday = useCallback(() => {
    setSelectedDate(new Date());
  }, []);

  // Get items for a specific date
  const getItemsForDate = useCallback(
    (date: Date): CalendarItem[] => {
      if (!calendar?.days) return [];
      const dateStr = formatDate(date);
      const day = calendar.days.find((d) => d.date.startsWith(dateStr));
      return day?.items ?? [];
    },
    [calendar],
  );

  // Week dates array
  const weekDates = useMemo(() => {
    if (viewMode !== "week") return [];
    const start = getWeekStart(selectedDate);
    return Array.from({ length: 7 }, (_, i) => addDays(start, i));
  }, [viewMode, selectedDate]);

  // Header date text
  const headerText = useMemo(() => {
    if (viewMode === "day") {
      return selectedDate.toLocaleDateString(intl.locale, {
        weekday: "long",
        month: "long",
        day: "numeric",
        year: "numeric",
      });
    }
    const start = getWeekStart(selectedDate);
    const end = getWeekEnd(selectedDate);
    const sameMonth = start.getMonth() === end.getMonth();
    if (sameMonth) {
      return `${start.toLocaleDateString(intl.locale, { month: "long", day: "numeric" })} – ${end.getDate()}, ${end.getFullYear()}`;
    }
    return `${start.toLocaleDateString(intl.locale, { month: "short", day: "numeric" })} – ${end.toLocaleDateString(intl.locale, { month: "short", day: "numeric", year: "numeric" })}`;
  }, [viewMode, selectedDate, intl.locale]);

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "planning.calendar.pageTitle" })}
      />

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-3">
          {/* View toggle */}
          <div className="flex bg-surface-container-low rounded-radius-sm">
            <button
              onClick={() => setViewMode("week")}
              className={`px-3 py-1.5 type-label-md rounded-radius-sm transition-colors ${
                viewMode === "week"
                  ? "bg-primary text-on-primary"
                  : "text-on-surface-variant hover:bg-surface-container-high"
              }`}
            >
              <FormattedMessage id="planning.calendar.view.week" />
            </button>
            <button
              onClick={() => setViewMode("day")}
              className={`px-3 py-1.5 type-label-md rounded-radius-sm transition-colors ${
                viewMode === "day"
                  ? "bg-primary text-on-primary"
                  : "text-on-surface-variant hover:bg-surface-container-high"
              }`}
            >
              <FormattedMessage id="planning.calendar.view.day" />
            </button>
          </div>

          {/* Student filter */}
          {students && students.length > 0 && (
            <Select
              value={studentFilter ?? ""}
              onChange={(e) =>
                setStudentFilter(e.target.value || undefined)
              }
              className="w-36"
              aria-label={intl.formatMessage({
                id: "planning.calendar.studentFilter",
              })}
            >
              <option value="">
                {intl.formatMessage({
                  id: "planning.calendar.allStudents",
                })}
              </option>
              {students.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.display_name}
                </option>
              ))}
            </Select>
          )}
        </div>

        <div className="flex items-center gap-2">
          <RouterLink to="/schedule/new">
            <Button variant="primary" size="sm">
              <Icon icon={Plus} size="sm" className="mr-1" />
              <FormattedMessage id="planning.calendar.addItem" />
            </Button>
          </RouterLink>
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setShowExport((v) => !v)}
            aria-label={intl.formatMessage({ id: "planning.export.title" })}
          >
            <Icon icon={Download} size="sm" />
          </Button>
          <RouterLink to="/planning/print">
            <Button variant="tertiary" size="sm">
              <Icon icon={Printer} size="sm" />
            </Button>
          </RouterLink>
        </div>
      </div>

      {/* Export panel */}
      {showExport && (
        <div className="mb-4">
          <ExportPanel
            defaultStart={dateRange.start}
            defaultEnd={dateRange.end}
            students={students}
            onClose={() => setShowExport(false)}
          />
        </div>
      )}

      {/* Date navigation */}
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={() => navigate(-1)}
          className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-low transition-colors touch-target"
          aria-label={intl.formatMessage({
            id: "planning.calendar.previous",
          })}
        >
          <Icon icon={ChevronLeft} size="md" />
        </button>

        <div className="flex items-center gap-3">
          <h2 className="type-title-md text-on-surface font-semibold" aria-live="polite" aria-atomic="true">
            {headerText}
          </h2>
          <button
            onClick={goToToday}
            className="px-2 py-1 type-label-sm text-primary hover:bg-primary-container/30 rounded-radius-sm transition-colors"
          >
            <FormattedMessage id="planning.calendar.today" />
          </button>
        </div>

        <button
          onClick={() => navigate(1)}
          className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-surface-container-low transition-colors touch-target"
          aria-label={intl.formatMessage({
            id: "planning.calendar.next",
          })}
        >
          <Icon icon={ChevronRight} size="md" />
        </button>
      </div>

      {/* Calendar body */}
      <Card className="p-card-padding mb-4">
        {isPending ? (
          <div className="space-y-3">
            {[1, 2, 3, 4, 5].map((n) => (
              <Skeleton key={n} className="h-12 w-full rounded-radius-sm" />
            ))}
          </div>
        ) : viewMode === "week" ? (
          <div className="flex gap-2 overflow-x-auto">
            {weekDates.map((date) => (
              <DayColumn
                key={formatDate(date)}
                date={date}
                items={getItemsForDate(date)}
                isToday={isSameDay(date, today)}
              />
            ))}
          </div>
        ) : (
          <DayDetailView
            date={selectedDate}
            items={getItemsForDate(selectedDate)}
          />
        )}
      </Card>

      {/* Color legend */}
      <ColorLegend />
    </div>
  );
}
