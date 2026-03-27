import { useState, useMemo, useCallback, type ReactNode } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Icon } from "./icon";

type CalendarEvent = {
  id: string;
  date: string; // YYYY-MM-DD
  label: string;
};

type CalendarProps = {
  /** Events to display on the calendar */
  events?: CalendarEvent[];
  /** Render custom content for a day cell */
  renderDay?: (date: string, events: CalendarEvent[]) => ReactNode;
  /** Called when a date is clicked */
  onDateClick?: (date: string) => void;
  className?: string;
};

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"] as const;
const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
] as const;

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate();
}

function getFirstDayOfWeek(year: number, month: number): number {
  return new Date(year, month, 1).getDay();
}

function formatDate(year: number, month: number, day: number): string {
  return `${String(year)}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

export function Calendar({
  events = [],
  renderDay,
  onDateClick,
  className = "",
}: CalendarProps) {
  const today = new Date();
  const [viewYear, setViewYear] = useState(today.getFullYear());
  const [viewMonth, setViewMonth] = useState(today.getMonth());

  const daysInMonth = useMemo(
    () => getDaysInMonth(viewYear, viewMonth),
    [viewYear, viewMonth],
  );
  const firstDay = useMemo(
    () => getFirstDayOfWeek(viewYear, viewMonth),
    [viewYear, viewMonth],
  );

  const eventsByDate = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const event of events) {
      const existing = map.get(event.date) ?? [];
      existing.push(event);
      map.set(event.date, existing);
    }
    return map;
  }, [events]);

  const goToPrevMonth = useCallback(() => {
    if (viewMonth === 0) {
      setViewMonth(11);
      setViewYear((y) => y - 1);
    } else {
      setViewMonth((m) => m - 1);
    }
  }, [viewMonth]);

  const goToNextMonth = useCallback(() => {
    if (viewMonth === 11) {
      setViewMonth(0);
      setViewYear((y) => y + 1);
    } else {
      setViewMonth((m) => m + 1);
    }
  }, [viewMonth]);

  const isToday = (day: number) =>
    today.getFullYear() === viewYear &&
    today.getMonth() === viewMonth &&
    today.getDate() === day;

  return (
    <div className={`${className}`}>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <button
          type="button"
          onClick={goToPrevMonth}
          className="rounded-full p-2 hover:bg-surface-container-low focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          aria-label="Previous month"
        >
          <Icon icon={ChevronLeft} size="md" />
        </button>
        <h2 className="type-title-lg text-on-surface">
          {MONTHS[viewMonth]} {viewYear}
        </h2>
        <button
          type="button"
          onClick={goToNextMonth}
          className="rounded-full p-2 hover:bg-surface-container-low focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          aria-label="Next month"
        >
          <Icon icon={ChevronRight} size="md" />
        </button>
      </div>

      {/* Weekday headers */}
      <div className="grid grid-cols-7 gap-px">
        {WEEKDAYS.map((day) => (
          <div
            key={day}
            className="py-2 text-center type-label-md text-on-surface-variant"
          >
            {day}
          </div>
        ))}
      </div>

      {/* Day grid */}
      <div className="grid grid-cols-7 gap-px bg-surface-container-low">
        {/* Empty cells */}
        {Array.from({ length: firstDay }).map((_, i) => (
          <div
            key={`empty-${String(i)}`}
            className="min-h-24 bg-surface p-1"
          />
        ))}

        {Array.from({ length: daysInMonth }).map((_, i) => {
          const day = i + 1;
          const dateStr = formatDate(viewYear, viewMonth, day);
          const dayEvents = eventsByDate.get(dateStr) ?? [];
          const todayDay = isToday(day);

          return (
            <button
              key={day}
              type="button"
              className={`min-h-24 bg-surface p-1 text-left transition-colors hover:bg-surface-container-low focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-focus-ring ${
                todayDay ? "bg-surface-container-lowest" : ""
              }`}
              onClick={() => onDateClick?.(dateStr)}
            >
              <span
                className={`inline-flex h-7 w-7 items-center justify-center rounded-full type-body-sm ${
                  todayDay
                    ? "bg-primary text-on-primary font-medium"
                    : "text-on-surface"
                }`}
              >
                {day}
              </span>

              {renderDay ? (
                renderDay(dateStr, dayEvents)
              ) : (
                <div className="mt-1 flex flex-col gap-0.5">
                  {dayEvents.slice(0, 2).map((event) => (
                    <div
                      key={event.id}
                      className="truncate rounded-sm bg-primary/10 px-1 type-label-sm text-primary"
                    >
                      {event.label}
                    </div>
                  ))}
                  {dayEvents.length > 2 && (
                    <span className="type-label-sm text-on-surface-variant">
                      +{dayEvents.length - 2} more
                    </span>
                  )}
                </div>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
