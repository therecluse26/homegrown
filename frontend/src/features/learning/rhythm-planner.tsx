import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Music, ArrowLeft, Plus, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import { useLogActivity } from "@/hooks/use-activities";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Types ──────────────────────────────────────────────────────────────────

type DayOfWeek = "monday" | "tuesday" | "wednesday" | "thursday" | "friday" | "saturday";
type ActivityCategory =
  | "main_lesson"
  | "arts"
  | "circle_time"
  | "practical_work"
  | "movement"
  | "eurythmy"
  | "storytelling"
  | "handwork"
  | "free_play"
  | "rest";

const DAYS: { value: DayOfWeek; labelId: string; shortId: string }[] = [
  { value: "monday",    labelId: "methodologyTools.rhythm.day.monday",    shortId: "methodologyTools.rhythm.day.monday.short" },
  { value: "tuesday",   labelId: "methodologyTools.rhythm.day.tuesday",   shortId: "methodologyTools.rhythm.day.tuesday.short" },
  { value: "wednesday", labelId: "methodologyTools.rhythm.day.wednesday", shortId: "methodologyTools.rhythm.day.wednesday.short" },
  { value: "thursday",  labelId: "methodologyTools.rhythm.day.thursday",  shortId: "methodologyTools.rhythm.day.thursday.short" },
  { value: "friday",    labelId: "methodologyTools.rhythm.day.friday",    shortId: "methodologyTools.rhythm.day.friday.short" },
  { value: "saturday",  labelId: "methodologyTools.rhythm.day.saturday",  shortId: "methodologyTools.rhythm.day.saturday.short" },
];

const ACTIVITY_CATEGORIES: { value: ActivityCategory; labelId: string; color: string }[] = [
  { value: "main_lesson",     labelId: "methodologyTools.rhythm.category.main_lesson",     color: "bg-primary-container text-on-primary-container" },
  { value: "arts",            labelId: "methodologyTools.rhythm.category.arts",             color: "bg-tertiary-container text-on-tertiary-container" },
  { value: "circle_time",     labelId: "methodologyTools.rhythm.category.circle_time",      color: "bg-secondary-container text-on-secondary-container" },
  { value: "practical_work",  labelId: "methodologyTools.rhythm.category.practical_work",   color: "bg-surface-container text-on-surface" },
  { value: "movement",        labelId: "methodologyTools.rhythm.category.movement",         color: "bg-error-container text-on-error-container" },
  { value: "eurythmy",        labelId: "methodologyTools.rhythm.category.eurythmy",         color: "bg-tertiary-container text-on-tertiary-container" },
  { value: "storytelling",    labelId: "methodologyTools.rhythm.category.storytelling",     color: "bg-secondary-container text-on-secondary-container" },
  { value: "handwork",        labelId: "methodologyTools.rhythm.category.handwork",         color: "bg-primary-container text-on-primary-container" },
  { value: "free_play",       labelId: "methodologyTools.rhythm.category.free_play",        color: "bg-surface-container-low text-on-surface" },
  { value: "rest",            labelId: "methodologyTools.rhythm.category.rest",             color: "bg-surface-container-lowest text-on-surface-variant" },
];

interface TimeBlock {
  id: string;
  startTime: string;
  endTime: string;
  category: ActivityCategory;
  activityName: string;
}

function makeDefaultBlocks(intl: ReturnType<typeof useIntl>): TimeBlock[] {
  return [
    { id: "1", startTime: "08:00", endTime: "09:30", category: "main_lesson",    activityName: intl.formatMessage({ id: "methodologyTools.rhythm.defaultBlock.mainLesson" }) },
    { id: "2", startTime: "09:30", endTime: "10:00", category: "circle_time",    activityName: intl.formatMessage({ id: "methodologyTools.rhythm.defaultBlock.circleTime" }) },
    { id: "3", startTime: "10:00", endTime: "11:00", category: "arts",           activityName: intl.formatMessage({ id: "methodologyTools.rhythm.defaultBlock.artsCrafts" }) },
    { id: "4", startTime: "11:00", endTime: "12:00", category: "practical_work", activityName: intl.formatMessage({ id: "methodologyTools.rhythm.defaultBlock.practicalWork" }) },
    { id: "5", startTime: "14:00", endTime: "15:00", category: "movement",       activityName: intl.formatMessage({ id: "methodologyTools.rhythm.defaultBlock.outdoorMovement" }) },
    { id: "6", startTime: "15:00", endTime: "16:00", category: "free_play",      activityName: intl.formatMessage({ id: "methodologyTools.rhythm.defaultBlock.freePlay" }) },
  ];
}

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "waldorf") return null;
  return (
    <div className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6" role="note">
      <Icon icon={Music} size="sm" className="mt-0.5 shrink-0 text-tertiary" aria-hidden />
      <p className="type-body-sm"><FormattedMessage id="methodologyTools.notPrimary.rhythmPlanner" /></p>
    </div>
  );
}

function TimeBlockRow({
  block,
  onChange,
  onDelete,
  index,
}: {
  block: TimeBlock;
  onChange: (id: string, updates: Partial<TimeBlock>) => void;
  onDelete: (id: string) => void;
  index: number;
}) {
  const intl = useIntl();
  return (
    <div className="flex items-center gap-2 p-3 rounded-xl bg-surface-container-low">
      <div className="flex items-center gap-1 shrink-0">
        <Input
          type="time"
          value={block.startTime}
          onChange={e => onChange(block.id, { startTime: e.target.value })}
          aria-label={intl.formatMessage({ id: "methodologyTools.rhythm.startTime" }, { index: index + 1 })}
          className="w-24 text-sm"
        />
        <span className="type-body-sm text-on-surface-variant">–</span>
        <Input
          type="time"
          value={block.endTime}
          onChange={e => onChange(block.id, { endTime: e.target.value })}
          aria-label={intl.formatMessage({ id: "methodologyTools.rhythm.endTime" }, { index: index + 1 })}
          className="w-24 text-sm"
        />
      </div>

      <Select
        value={block.category}
        onChange={e => onChange(block.id, { category: e.target.value as ActivityCategory })}
        aria-label={intl.formatMessage({ id: "methodologyTools.rhythm.categoryLabel" })}
        className="flex-1"
      >
        {ACTIVITY_CATEGORIES.map(c => (
          <option key={c.value} value={c.value}>{intl.formatMessage({ id: c.labelId })}</option>
        ))}
      </Select>

      <Input
        value={block.activityName}
        onChange={e => onChange(block.id, { activityName: e.target.value })}
        placeholder={intl.formatMessage({ id: ACTIVITY_CATEGORIES.find(c => c.value === block.category)?.labelId ?? "" })}
        aria-label={intl.formatMessage({ id: "methodologyTools.rhythm.activityName" }, { index: index + 1 })}
        className="flex-1"
      />

      <button
        type="button"
        onClick={() => onDelete(block.id)}
        className="p-2 rounded-lg text-on-surface-variant hover:text-error hover:bg-error-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        aria-label={intl.formatMessage({ id: "methodologyTools.rhythm.deleteBlock" })}
      >
        <Icon icon={Trash2} size="sm" aria-hidden />
      </button>
    </div>
  );
}

export function RhythmPlanner() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [selectedDay, setSelectedDay] = useState<DayOfWeek>("monday");
  const [templateName, setTemplateName] = useState("");
  const [blocks, setBlocks] = useState<TimeBlock[]>(() => makeDefaultBlocks(intl));
  const [weekStartDate, setWeekStartDate] = useState(new Date().toISOString().slice(0, 10));

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.rhythmPlanner.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function addBlock() {
    setBlocks(prev => [
      ...prev,
      { id: String(Date.now()), startTime: "09:00", endTime: "10:00", category: "main_lesson", activityName: "" },
    ]);
  }

  function updateBlock(id: string, updates: Partial<TimeBlock>) {
    setBlocks(prev => prev.map(b => (b.id === id ? { ...b, ...updates } : b)));
  }

  function deleteBlock(id: string) {
    setBlocks(prev => prev.filter(b => b.id !== id));
  }

  function handleSave() {
    if (!effectiveStudent || blocks.length === 0) return;

    const dayLabel = intl.formatMessage({ id: DAYS.find(d => d.value === selectedDay)?.labelId ?? "" });
    const sortedBlocks = [...blocks].sort((a, b) => a.startTime.localeCompare(b.startTime));
    const blockLines = sortedBlocks.map(b => {
      const catLabel = intl.formatMessage({ id: ACTIVITY_CATEGORIES.find(c => c.value === b.category)?.labelId ?? "" });
      return `${b.startTime}–${b.endTime}: ${b.activityName || catLabel} (${catLabel})`;
    });

    const descParts: string[] = [
      templateName ? `Template: ${templateName}` : "",
      `Week of: ${weekStartDate}`,
      `Day: ${dayLabel}`,
      "",
      ...blockLines,
    ].filter(part => part !== undefined);

    logActivity.mutate(
      {
        title: `Rhythm Plan: ${dayLabel}${templateName ? ` — ${templateName}` : ""}`,
        description: descParts.join("\n"),
        subject_tags: ["rhythm", "planning"],
        tool_id: "rhythm-planner",
        activity_date: weekStartDate ? `${weekStartDate}T00:00:00Z` : undefined,
      },
      { onSuccess: () => { void navigate("/learning/activities"); } },
    );
  }

  if (studentsLoading) {
    return (
      <div className="mx-auto max-w-content space-y-4" aria-busy="true">
        <Skeleton height="h-8" width="w-48" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => void navigate("/learning")}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
        </Button>
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-sm text-on-surface font-semibold flex items-center gap-2">
          <Icon icon={Music} size="md" className="text-tertiary" aria-hidden />
          <FormattedMessage id="methodologyTools.rhythmPlanner.title" />
        </h1>
      </div>

      <MethodologyBanner />

      {/* Student selector */}
      {students && students.length > 1 && (
        <div>
          <label htmlFor="rp-student" className="block type-label-md text-on-surface-variant mb-1.5">
            <FormattedMessage id="methodologyTools.field.student" />
          </label>
          <Select id="rp-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
            <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
            {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
          </Select>
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label htmlFor="rp-name" className="block type-label-md text-on-surface-variant mb-1.5">
            <FormattedMessage id="methodologyTools.rhythm.templateName" />
          </label>
          <Input
            id="rp-name"
            placeholder={intl.formatMessage({ id: "methodologyTools.rhythm.templateNamePlaceholder" })}
            value={templateName}
            onChange={e => setTemplateName(e.target.value)}
          />
        </div>
        <div>
          <label htmlFor="rp-week" className="block type-label-md text-on-surface-variant mb-1.5">
            <FormattedMessage id="methodologyTools.rhythm.weekOf" />
          </label>
          <Input id="rp-week" type="date" value={weekStartDate} onChange={e => setWeekStartDate(e.target.value)} />
        </div>
      </div>

      {/* Day selector */}
      <div
        className="flex gap-2 overflow-x-auto pb-1"
        role="group"
        aria-label={intl.formatMessage({ id: "methodologyTools.rhythm.dayLabel" })}
      >
        {DAYS.map(day => (
          <button
            key={day.value}
            type="button"
            onClick={() => setSelectedDay(day.value)}
            className={`px-4 py-2 rounded-xl type-label-md font-medium shrink-0 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${
              selectedDay === day.value
                ? "bg-primary text-on-primary"
                : "bg-surface-container-low text-on-surface hover:bg-surface-container"
            }`}
            aria-pressed={selectedDay === day.value}
          >
            {intl.formatMessage({ id: day.shortId })}
          </button>
        ))}
      </div>

      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="type-title-md text-on-surface font-semibold">
            {intl.formatMessage({ id: DAYS.find(d => d.value === selectedDay)?.labelId ?? "" })}{" "}
            <FormattedMessage id="methodologyTools.rhythm.rhythmTitle" />
          </h2>
          <Button variant="secondary" size="sm" type="button" onClick={addBlock}>
            <Icon icon={Plus} size="sm" aria-hidden />
            <FormattedMessage id="methodologyTools.rhythm.addBlock" />
          </Button>
        </div>

        <div className="space-y-2">
          {[...blocks]
            .sort((a, b) => a.startTime.localeCompare(b.startTime))
            .map((block, index) => (
              <TimeBlockRow
                key={block.id}
                block={block}
                onChange={updateBlock}
                onDelete={deleteBlock}
                index={index}
              />
            ))}
        </div>

        <div className="flex items-center justify-end gap-3 pt-4 mt-4">
          <Button variant="tertiary" type="button" onClick={() => void navigate("/learning")}>
            <FormattedMessage id="action.cancel" />
          </Button>
          <Button
            variant="primary"
            type="button"
            onClick={handleSave}
            disabled={!effectiveStudent || blocks.length === 0 || logActivity.isPending}
            loading={logActivity.isPending}
          >
            <FormattedMessage id="methodologyTools.rhythm.saveRhythm" />
          </Button>
        </div>
      </Card>

      <Card>
        <h2 className="type-label-lg text-on-surface font-semibold mb-3">
          <FormattedMessage id="methodologyTools.rhythm.legend" />
        </h2>
        <div className="flex flex-wrap gap-2">
          {ACTIVITY_CATEGORIES.map(cat => (
            <span key={cat.value} className={`px-3 py-1 rounded-full type-label-sm ${cat.color}`}>
              {intl.formatMessage({ id: cat.labelId })}
            </span>
          ))}
        </div>
      </Card>
    </div>
  );
}
