import {
  useState,
  useMemo,
  useCallback,
  type KeyboardEvent,
} from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Icon } from "./icon";

type DatePickerProps = {
  /** Currently selected date (YYYY-MM-DD) */
  value?: string;
  /** Called with the new date string (YYYY-MM-DD) */
  onChange: (date: string) => void;
  /** Accessible label */
  label?: string;
  className?: string;
};

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate();
}

function getFirstDayOfWeek(year: number, month: number): number {
  return new Date(year, month, 1).getDay();
}

function formatDate(year: number, month: number, day: number): string {
  return `${String(year)}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

const WEEKDAYS = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"] as const;
const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
] as const;

export function DatePicker({
  value,
  onChange,
  label = "Select a date",
  className = "",
}: DatePickerProps) {
  const selectedDate = value ? new Date(value) : null;
  const today = new Date();

  const [viewYear, setViewYear] = useState(
    selectedDate?.getFullYear() ?? today.getFullYear(),
  );
  const [viewMonth, setViewMonth] = useState(
    selectedDate?.getMonth() ?? today.getMonth(),
  );
  const [focusedDay, setFocusedDay] = useState(
    selectedDate?.getDate() ?? today.getDate(),
  );

  const daysInMonth = useMemo(
    () => getDaysInMonth(viewYear, viewMonth),
    [viewYear, viewMonth],
  );
  const firstDay = useMemo(
    () => getFirstDayOfWeek(viewYear, viewMonth),
    [viewYear, viewMonth],
  );

  const goToPrevMonth = useCallback(() => {
    if (viewMonth === 0) {
      setViewMonth(11);
      setViewYear((y) => y - 1);
    } else {
      setViewMonth((m) => m - 1);
    }
    setFocusedDay(1);
  }, [viewMonth]);

  const goToNextMonth = useCallback(() => {
    if (viewMonth === 11) {
      setViewMonth(0);
      setViewYear((y) => y + 1);
    } else {
      setViewMonth((m) => m + 1);
    }
    setFocusedDay(1);
  }, [viewMonth]);

  const handleDayKeyDown = useCallback(
    (e: KeyboardEvent) => {
      let newDay = focusedDay;

      switch (e.key) {
        case "ArrowRight":
          e.preventDefault();
          newDay = focusedDay + 1;
          if (newDay > daysInMonth) {
            goToNextMonth();
            return;
          }
          break;
        case "ArrowLeft":
          e.preventDefault();
          newDay = focusedDay - 1;
          if (newDay < 1) {
            goToPrevMonth();
            return;
          }
          break;
        case "ArrowDown":
          e.preventDefault();
          newDay = focusedDay + 7;
          if (newDay > daysInMonth) {
            goToNextMonth();
            return;
          }
          break;
        case "ArrowUp":
          e.preventDefault();
          newDay = focusedDay - 7;
          if (newDay < 1) {
            goToPrevMonth();
            return;
          }
          break;
        case "Enter":
        case " ":
          e.preventDefault();
          onChange(formatDate(viewYear, viewMonth, focusedDay));
          return;
        case "Escape":
          e.preventDefault();
          return;
        default:
          return;
      }

      setFocusedDay(newDay);
    },
    [focusedDay, daysInMonth, viewYear, viewMonth, onChange, goToNextMonth, goToPrevMonth],
  );

  const isSelected = (day: number) =>
    selectedDate !== null &&
    selectedDate.getFullYear() === viewYear &&
    selectedDate.getMonth() === viewMonth &&
    selectedDate.getDate() === day;

  const isToday = (day: number) =>
    today.getFullYear() === viewYear &&
    today.getMonth() === viewMonth &&
    today.getDate() === day;

  return (
    <div
      className={`inline-block rounded-xl bg-surface-container-lowest p-4 shadow-ambient-md ${className}`}
      role="group"
      aria-label={label}
    >
      {/* Month navigation */}
      <div className="flex items-center justify-between mb-3">
        <button
          type="button"
          onClick={goToPrevMonth}
          className="rounded-full p-1.5 hover:bg-surface-container-low focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          aria-label="Previous month"
        >
          <Icon icon={ChevronLeft} size="sm" />
        </button>
        <span className="type-title-sm text-on-surface">
          {MONTHS[viewMonth]} {viewYear}
        </span>
        <button
          type="button"
          onClick={goToNextMonth}
          className="rounded-full p-1.5 hover:bg-surface-container-low focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          aria-label="Next month"
        >
          <Icon icon={ChevronRight} size="sm" />
        </button>
      </div>

      {/* Weekday headers */}
      <div className="grid grid-cols-7 gap-0" role="row">
        {WEEKDAYS.map((day) => (
          <div
            key={day}
            className="flex h-8 items-center justify-center type-label-sm text-on-surface-variant"
            role="columnheader"
          >
            {day}
          </div>
        ))}
      </div>

      {/* Day grid */}
      { }
      <div
        className="grid grid-cols-7 gap-0"
        role="grid"
        aria-label={`${MONTHS[viewMonth]} ${String(viewYear)}`}
        onKeyDown={handleDayKeyDown}
      >
        {/* Empty cells for offset */}
        {Array.from({ length: firstDay }).map((_, i) => (
          <div key={`empty-${String(i)}`} className="h-9" />
        ))}

        {Array.from({ length: daysInMonth }).map((_, i) => {
          const day = i + 1;
          const selected = isSelected(day);
          const todayDay = isToday(day);
          const focused = day === focusedDay;

          return (
            <button
              key={day}
              type="button"
              tabIndex={focused ? 0 : -1}
              role="gridcell"
              aria-selected={selected}
              className={`flex h-9 w-9 items-center justify-center rounded-full type-body-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${
                selected
                  ? "bg-primary text-on-primary font-medium"
                  : todayDay
                    ? "bg-surface-container-high text-on-surface font-medium"
                    : "text-on-surface hover:bg-surface-container-low"
              }`}
              onClick={() => {
                setFocusedDay(day);
                onChange(formatDate(viewYear, viewMonth, day));
              }}
            >
              {day}
            </button>
          );
        })}
      </div>
    </div>
  );
}
