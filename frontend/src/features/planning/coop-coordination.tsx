import { useState, useMemo, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Users,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Select,
  Badge,
  EmptyState,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useMyGroups, useGroupMembers } from "@/hooks/use-social";
import { useCalendar } from "@/hooks/use-planning";
import type { CalendarItem, CalendarSource } from "@/hooks/use-planning";

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

function isSameDay(a: Date, b: Date): boolean {
  return formatDate(a) === formatDate(b);
}

// ─── Source colors (simplified for co-op) ──────────────────────────────────

const SOURCE_BG: Record<CalendarSource, string> = {
  schedule: "bg-surface-container-high",
  activities: "bg-tertiary-container",
  events: "bg-primary-container",
  attendance: "bg-secondary-container",
};

const SOURCE_TEXT: Record<CalendarSource, string> = {
  schedule: "text-on-surface",
  activities: "text-on-tertiary-container",
  events: "text-on-primary-container",
  attendance: "text-on-secondary-container",
};

// ─── Co-op item card ───────────────────────────────────────────────────────

function CoopItemCard({
  item,
  familyName,
}: {
  item: CalendarItem;
  familyName?: string;
}) {
  return (
    <div
      className={`flex items-center gap-2 px-2 py-1.5 rounded-radius-sm ${SOURCE_BG[item.source]} ${SOURCE_TEXT[item.source]} type-label-sm`}
    >
      <span className="truncate flex-1">{item.title}</span>
      {item.start_time && (
        <span className="shrink-0 opacity-75">{item.start_time}</span>
      )}
      {familyName && (
        <Badge variant="secondary" className="shrink-0">
          {familyName}
        </Badge>
      )}
    </div>
  );
}

// ─── Weekly grid for a single family ───────────────────────────────────────

function FamilyWeekRow({
  familyName,
  weekDates,
  items,
  today,
}: {
  familyName: string;
  weekDates: Date[];
  items: CalendarItem[];
  today: Date;
}) {
  const intl = useIntl();

  const itemsByDate = useMemo(() => {
    const map = new Map<string, CalendarItem[]>();
    for (const item of items) {
      const dateKey = item.date.slice(0, 10);
      const existing = map.get(dateKey) ?? [];
      existing.push(item);
      map.set(dateKey, existing);
    }
    return map;
  }, [items]);

  return (
    <Card className="p-card-padding mb-4">
      <h3 className="type-title-sm text-on-surface font-semibold mb-3 flex items-center gap-2">
        <Icon icon={Users} size="sm" className="text-on-surface-variant" />
        {familyName}
        <Badge variant="secondary">
          <FormattedMessage
            id="planning.coop.sharedItems"
            values={{ count: items.length }}
          />
        </Badge>
      </h3>
      <div className="flex gap-2 overflow-x-auto">
        {weekDates.map((date) => {
          const dateStr = formatDate(date);
          const dayItems = itemsByDate.get(dateStr) ?? [];
          const isToday = isSameDay(date, today);
          const dayName = date.toLocaleDateString(intl.locale, {
            weekday: "short",
          });
          const dayNum = date.getDate();

          return (
            <div key={dateStr} className="flex-1 min-w-0">
              <div
                className={`text-center pb-2 mb-2 border-b ${
                  isToday ? "border-primary" : "border-outline-variant/10"
                }`}
              >
                <p className="type-label-sm text-on-surface-variant uppercase">
                  {dayName}
                </p>
                <p
                  className={`type-title-md font-bold ${
                    isToday ? "text-primary" : "text-on-surface"
                  }`}
                >
                  {dayNum}
                </p>
              </div>
              <div className="space-y-1">
                {dayItems.length === 0 && (
                  <p className="type-label-sm text-on-surface-variant text-center py-2 opacity-50">
                    —
                  </p>
                )}
                {dayItems.map((item) => (
                  <CoopItemCard key={`${item.source}-${item.id}`} item={item} />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
}

// ─── Overlap detection ─────────────────────────────────────────────────────

interface TimeOverlap {
  date: string;
  time: string;
  items: { title: string; familyName: string }[];
}

function detectOverlaps(
  familyItems: { familyName: string; items: CalendarItem[] }[],
): TimeOverlap[] {
  // Group all items by date + start_time
  const slotMap = new Map<
    string,
    { title: string; familyName: string }[]
  >();

  for (const { familyName, items } of familyItems) {
    for (const item of items) {
      if (!item.start_time) continue;
      const key = `${item.date.slice(0, 10)}|${item.start_time}`;
      const existing = slotMap.get(key) ?? [];
      existing.push({ title: item.title, familyName });
      slotMap.set(key, existing);
    }
  }

  // Only keep slots with items from 2+ families
  const overlaps: TimeOverlap[] = [];
  for (const [key, entries] of slotMap) {
    const families = new Set(entries.map((e) => e.familyName));
    if (families.size >= 2) {
      const parts = key.split("|");
      overlaps.push({ date: parts[0] ?? "", time: parts[1] ?? "", items: entries });
    }
  }

  return overlaps.sort((a, b) => {
    const dateCompare = a.date.localeCompare(b.date);
    if (dateCompare !== 0) return dateCompare;
    return a.time.localeCompare(b.time);
  });
}

// ─── Main component ────────────────────────────────────────────────────────

export function CoopCoordination() {
  const intl = useIntl();
  const today = useMemo(() => new Date(), []);
  const [selectedDate, setSelectedDate] = useState(today);
  const [selectedGroupId, setSelectedGroupId] = useState<string | undefined>();

  const { data: myGroups, isPending: groupsPending } = useMyGroups();
  const { data: members, isPending: membersPending } = useGroupMembers(
    selectedGroupId,
  );

  // Select first group by default
  const activeGroupId = selectedGroupId ?? myGroups?.[0]?.summary.id;
  const activeGroup = myGroups?.find(
    (g) => g.summary.id === activeGroupId,
  );

  // Week calculations
  const weekStart = useMemo(
    () => getWeekStart(selectedDate),
    [selectedDate],
  );
  const weekEnd = useMemo(
    () => getWeekEnd(selectedDate),
    [selectedDate],
  );
  const weekDates = useMemo(
    () => Array.from({ length: 7 }, (_, i) => addDays(weekStart, i)),
    [weekStart],
  );

  // Fetch calendar for the current family (own schedule)
  const { data: ownCalendar, isPending: calendarPending } = useCalendar({
    start: formatDate(weekStart),
    end: formatDate(weekEnd),
  });

  // Flatten own calendar items
  const ownItems = useMemo(() => {
    if (!ownCalendar?.days) return [];
    return ownCalendar.days.flatMap((d) =>
      d.items.filter((item) => item.source === "schedule"),
    );
  }, [ownCalendar]);

  // Build family schedule data — in production this would come from a
  // backend endpoint that returns co-op member schedules. For now we
  // show our own schedule and placeholder entries for other members.
  const familySchedules = useMemo(() => {
    const schedules: { familyName: string; items: CalendarItem[] }[] = [];

    // Own family schedule
    schedules.push({
      familyName: intl.formatMessage({ id: "planning.coop.myFamily" }),
      items: ownItems,
    });

    // Other member families — show as empty until backend co-op endpoint exists
    if (members) {
      for (const member of members) {
        if (member.status !== "active") continue;
        // Skip own family (already added above)
        schedules.push({
          familyName: member.display_name,
          items: [], // Populated by backend co-op API when available
        });
      }
    }

    return schedules;
  }, [ownItems, members, intl]);

  // Detect time overlaps
  const overlaps = useMemo(
    () => detectOverlaps(familySchedules),
    [familySchedules],
  );

  // Navigation
  const navigate = useCallback((direction: -1 | 1) => {
    setSelectedDate((prev) => addDays(prev, direction * 7));
  }, []);

  const goToToday = useCallback(() => {
    setSelectedDate(new Date());
  }, []);

  // Header text
  const headerText = useMemo(() => {
    const sameMonth = weekStart.getMonth() === weekEnd.getMonth();
    if (sameMonth) {
      return `${weekStart.toLocaleDateString(intl.locale, { month: "long", day: "numeric" })} – ${weekEnd.getDate()}, ${weekEnd.getFullYear()}`;
    }
    return `${weekStart.toLocaleDateString(intl.locale, { month: "short", day: "numeric" })} – ${weekEnd.toLocaleDateString(intl.locale, { month: "short", day: "numeric", year: "numeric" })}`;
  }, [weekStart, weekEnd, intl.locale]);

  const isPending = groupsPending || membersPending || calendarPending;

  // No groups state
  if (!groupsPending && (!myGroups || myGroups.length === 0)) {
    return (
      <div className="max-w-content mx-auto">
        <PageTitle
          title={intl.formatMessage({ id: "planning.coop.pageTitle" })}
        />
        <div className="flex items-center gap-3 mb-6">
          <RouterLink to="/calendar">
            <Button variant="tertiary" size="sm">
              <Icon icon={ArrowLeft} size="sm" className="mr-1" />
              <FormattedMessage id="planning.coop.backToCalendar" />
            </Button>
          </RouterLink>
        </div>
        <EmptyState
          illustration={<Icon icon={Users} size="xl" />}
          message={intl.formatMessage({ id: "planning.coop.noGroups" })}
          action={
            <RouterLink to="/groups">
              <Button variant="primary" size="sm">
                <FormattedMessage id="planning.coop.noGroupsAction" />
              </Button>
            </RouterLink>
          }
        />
      </div>
    );
  }

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "planning.coop.pageTitle" })}
      />

      {/* Top bar */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-3">
          <RouterLink to="/calendar">
            <Button variant="tertiary" size="sm">
              <Icon icon={ArrowLeft} size="sm" className="mr-1" />
              <FormattedMessage id="planning.coop.backToCalendar" />
            </Button>
          </RouterLink>

          {myGroups && myGroups.length > 1 && (
            <Select
              value={activeGroupId ?? ""}
              onChange={(e) => setSelectedGroupId(e.target.value || undefined)}
              className="w-48"
              aria-label={intl.formatMessage({
                id: "planning.coop.selectGroup",
              })}
            >
              {myGroups.map((g) => (
                <option key={g.summary.id} value={g.summary.id}>
                  {g.summary.name}
                </option>
              ))}
            </Select>
          )}
        </div>

        {activeGroup && (
          <div className="flex items-center gap-2">
            <Icon icon={Users} size="sm" className="text-on-surface-variant" />
            <span className="type-label-md text-on-surface">
              {activeGroup.summary.name}
            </span>
            <Badge variant="secondary">
              {activeGroup.summary.member_count}
            </Badge>
          </div>
        )}
      </div>

      {/* Week navigation */}
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
          <h2
            className="type-title-md text-on-surface font-semibold"
            aria-live="polite"
            aria-atomic="true"
          >
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

      {/* Loading state */}
      {isPending ? (
        <div className="space-y-4">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-40 w-full rounded-radius-md" />
          ))}
        </div>
      ) : (
        <>
          {/* Overlap alerts */}
          {overlaps.length > 0 && (
            <Card className="p-card-padding mb-4 bg-secondary-container">
              <h3 className="type-title-sm text-on-secondary-container font-semibold mb-2">
                <FormattedMessage
                  id="planning.coop.overlapCount"
                  values={{ count: overlaps.length }}
                />
              </h3>
              <div className="space-y-2">
                {overlaps.map((overlap) => {
                  const dateObj = new Date(overlap.date + "T00:00:00");
                  const dayLabel = dateObj.toLocaleDateString(intl.locale, {
                    weekday: "short",
                    month: "short",
                    day: "numeric",
                  });
                  return (
                    <div
                      key={`${overlap.date}-${overlap.time}`}
                      className="flex items-center gap-2 type-body-sm text-on-secondary-container"
                    >
                      <span className="font-medium">
                        {dayLabel} {overlap.time}
                      </span>
                      <span>—</span>
                      <span>
                        {overlap.items.map((i) => i.familyName).join(", ")}
                      </span>
                    </div>
                  );
                })}
              </div>
            </Card>
          )}

          {/* Family schedule rows */}
          {familySchedules.length === 0 ? (
            <Card className="p-card-padding text-center">
              <p className="type-body-md text-on-surface-variant py-8">
                <FormattedMessage id="planning.coop.empty" />
              </p>
            </Card>
          ) : (
            familySchedules.map((fs) => (
              <FamilyWeekRow
                key={fs.familyName}
                familyName={fs.familyName}
                weekDates={weekDates}
                items={fs.items}
                today={today}
              />
            ))
          )}
        </>
      )}
    </div>
  );
}
