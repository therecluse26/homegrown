import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Heart, ArrowLeft, CheckCircle2, MinusCircle, XCircle } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
  Textarea,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import { useLogActivity } from "@/hooks/use-activities";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Types ──────────────────────────────────────────────────────────────────

type HabitCheckIn = "yes" | "partial" | "no";

const CHARLOTTE_MASON_HABITS: { key: string; labelId: string }[] = [
  { key: "attention",     labelId: "methodologyTools.habit.cm.attention" },
  { key: "obedience",     labelId: "methodologyTools.habit.cm.obedience" },
  { key: "truthfulness",  labelId: "methodologyTools.habit.cm.truthfulness" },
  { key: "gentleness",    labelId: "methodologyTools.habit.cm.gentleness" },
  { key: "kindness",      labelId: "methodologyTools.habit.cm.kindness" },
  { key: "diligence",     labelId: "methodologyTools.habit.cm.diligence" },
  { key: "punctuality",   labelId: "methodologyTools.habit.cm.punctuality" },
  { key: "orderliness",   labelId: "methodologyTools.habit.cm.orderliness" },
  { key: "patience",      labelId: "methodologyTools.habit.cm.patience" },
  { key: "selfControl",   labelId: "methodologyTools.habit.cm.selfControl" },
  { key: "perseverance",  labelId: "methodologyTools.habit.cm.perseverance" },
  { key: "courtesy",      labelId: "methodologyTools.habit.cm.courtesy" },
];

const CHECK_IN_OPTIONS: {
  value: HabitCheckIn;
  labelId: string;
  icon: typeof CheckCircle2;
  activeClass: string;
}[] = [
  { value: "yes",     labelId: "methodologyTools.habit.checkIn.yes",     icon: CheckCircle2, activeClass: "bg-success-container text-on-surface" },
  { value: "partial", labelId: "methodologyTools.habit.checkIn.partial", icon: MinusCircle,  activeClass: "bg-warning-container text-on-surface" },
  { value: "no",      labelId: "methodologyTools.habit.checkIn.no",      icon: XCircle,      activeClass: "bg-error-container text-on-error-container" },
];

interface HabitEntry {
  habit: string;
  checkIn: HabitCheckIn;
}

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "charlotte-mason") return null;
  return (
    <div className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6" role="note">
      <Icon icon={Heart} size="sm" className="mt-0.5 shrink-0 text-tertiary" aria-hidden />
      <p className="type-body-sm"><FormattedMessage id="methodologyTools.notPrimary.habitTracking" /></p>
    </div>
  );
}

function HabitRow({
  entry,
  onChange,
  onRemove,
}: {
  entry: HabitEntry;
  onChange: (updated: HabitEntry) => void;
  onRemove: () => void;
}) {
  const intl = useIntl();
  return (
    <div className="flex items-center gap-3 p-3 rounded-xl bg-surface-container-low">
      <span className="type-label-md text-on-surface flex-1 min-w-0 truncate">{entry.habit}</span>
      <div
        className="flex gap-1 shrink-0"
        role="group"
        aria-label={intl.formatMessage({ id: "methodologyTools.habit.checkInFor" }, { habit: entry.habit })}
      >
        {CHECK_IN_OPTIONS.map(opt => {
          const isSelected = entry.checkIn === opt.value;
          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => onChange({ ...entry, checkIn: opt.value })}
              className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg type-label-sm transition-all focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring ${
                isSelected ? opt.activeClass : "bg-surface-container text-on-surface-variant hover:bg-surface-container-high"
              }`}
              aria-pressed={isSelected}
            >
              <Icon icon={opt.icon} size="xs" aria-hidden />
              <span className="hidden sm:inline">{intl.formatMessage({ id: opt.labelId })}</span>
            </button>
          );
        })}
      </div>
      <button
        type="button"
        onClick={onRemove}
        className="p-1 rounded text-on-surface-variant hover:text-error transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring"
        aria-label={intl.formatMessage({ id: "methodologyTools.habit.removeHabit" }, { habit: entry.habit })}
      >
        <Icon icon={XCircle} size="xs" aria-hidden />
      </button>
    </div>
  );
}

export function HabitTracking() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  // Resolve CM habit names for initial entries
  const resolvedHabits = CHARLOTTE_MASON_HABITS.map(h => intl.formatMessage({ id: h.labelId }));

  const [studentId, setStudentId] = useState("");
  const [habitEntries, setHabitEntries] = useState<HabitEntry[]>([
    { habit: resolvedHabits[0] ?? "Attention", checkIn: "yes" },
    { habit: resolvedHabits[5] ?? "Diligence", checkIn: "yes" },
  ]);
  const [customHabit, setCustomHabit] = useState("");
  const [parentNotes, setParentNotes] = useState("");
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.habitTracking.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function addHabit(name: string) {
    const trimmed = name.trim();
    if (!trimmed || habitEntries.some(e => e.habit === trimmed)) return;
    setHabitEntries(prev => [...prev, { habit: trimmed, checkIn: "yes" }]);
    setCustomHabit("");
  }

  function updateEntry(index: number, updated: HabitEntry) {
    setHabitEntries(prev => prev.map((e, i) => (i === index ? updated : e)));
  }

  function removeEntry(index: number) {
    setHabitEntries(prev => prev.filter((_, i) => i !== index));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || habitEntries.length === 0) return;

    const checkInLines = habitEntries
      .map(entry => {
        const opt = CHECK_IN_OPTIONS.find(o => o.value === entry.checkIn);
        const optLabel = opt ? intl.formatMessage({ id: opt.labelId }) : entry.checkIn;
        return `${entry.habit}: ${optLabel}`;
      })
      .join("\n");

    const successCount = habitEntries.filter(e => e.checkIn === "yes").length;

    logActivity.mutate(
      {
        title: `Habit Check-in (${successCount}/${habitEntries.length} habits)`,
        description: `Habit check-in:\n${checkInLines}${parentNotes ? `\n\nNotes: ${parentNotes}` : ""}`,
        subject_tags: ["character_education"],
        tool_id: "habit-tracking",
        activity_date: entryDate ? `${entryDate}T00:00:00Z` : undefined,
      },
      { onSuccess: () => { void navigate("/learning/activities"); } },
    );
  }

  if (studentsLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-4" aria-busy="true">
        <Skeleton height="h-8" width="w-48" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => void navigate("/learning")}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
        </Button>
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-sm text-on-surface font-semibold flex items-center gap-2">
          <Icon icon={Heart} size="md" className="text-tertiary" aria-hidden />
          <FormattedMessage id="methodologyTools.habitTracking.title" />
        </h1>
      </div>

      <MethodologyBanner />

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {students && students.length > 1 && (
            <div>
              <label htmlFor="ht-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select id="ht-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
              </Select>
            </div>
          )}

          <div>
            <label htmlFor="ht-date" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.field.date" />
            </label>
            <Input id="ht-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
          </div>

          {habitEntries.length > 0 && (
            <div className="space-y-2">
              <p className="type-label-md text-on-surface-variant font-medium">
                <FormattedMessage id="methodologyTools.habit.todaysCheckIn" />
              </p>
              {habitEntries.map((entry, i) => (
                <HabitRow
                  key={`${entry.habit}-${i}`}
                  entry={entry}
                  onChange={updated => updateEntry(i, updated)}
                  onRemove={() => removeEntry(i)}
                />
              ))}
            </div>
          )}

          <div>
            <p className="type-label-md text-on-surface-variant font-medium mb-2">
              <FormattedMessage id="methodologyTools.habit.addHabit" />
            </p>
            <div className="flex flex-wrap gap-2 mb-3">
              {CHARLOTTE_MASON_HABITS
                .map(h => ({ ...h, label: intl.formatMessage({ id: h.labelId }) }))
                .filter(h => !habitEntries.some(e => e.habit === h.label))
                .map(habit => (
                  <button
                    key={habit.key}
                    type="button"
                    onClick={() => addHabit(habit.label)}
                    className="px-3 py-1.5 rounded-full type-label-sm bg-surface-container-low text-on-surface hover:bg-primary-container hover:text-on-primary-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring"
                  >
                    + {habit.label}
                  </button>
                ))}
            </div>
            <div className="flex gap-2">
              <Input
                placeholder={intl.formatMessage({ id: "methodologyTools.habit.customPlaceholder" })}
                value={customHabit}
                onChange={e => setCustomHabit(e.target.value)}
                onKeyDown={e => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addHabit(customHabit);
                  }
                }}
              />
              <Button
                variant="secondary"
                type="button"
                onClick={() => addHabit(customHabit)}
                disabled={!customHabit.trim()}
              >
                <FormattedMessage id="methodologyTools.habit.addCustom" />
              </Button>
            </div>
          </div>

          <div>
            <label htmlFor="ht-notes" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.habit.parentNotes" />
            </label>
            <Textarea
              id="ht-notes"
              placeholder={intl.formatMessage({ id: "methodologyTools.habit.parentNotesPlaceholder" })}
              value={parentNotes}
              onChange={e => setParentNotes(e.target.value)}
              rows={3}
            />
          </div>

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button variant="tertiary" type="button" onClick={() => void navigate("/learning")}>
              <FormattedMessage id="action.cancel" />
            </Button>
            <Button
              variant="primary"
              type="submit"
              disabled={!effectiveStudent || habitEntries.length === 0 || logActivity.isPending}
              loading={logActivity.isPending}
            >
              <FormattedMessage id="methodologyTools.action.saveEntry" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
