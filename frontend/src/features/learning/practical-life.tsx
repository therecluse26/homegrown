import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Home, ArrowLeft } from "lucide-react";
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

type LifeSkillCategory =
  | "care_of_self"
  | "care_of_environment"
  | "grace_and_courtesy"
  | "movement_control"
  | "sensorial";

type MasteryLevel = "introduced" | "practicing" | "proficient" | "mastered";

const LIFE_SKILL_CATEGORIES: {
  value: LifeSkillCategory;
  labelId: string;
  descId: string;
  examplesId: string;
}[] = [
  {
    value: "care_of_self",
    labelId: "methodologyTools.practicalLife.cat.care_of_self",
    descId: "methodologyTools.practicalLife.cat.care_of_self.desc",
    examplesId: "methodologyTools.practicalLife.cat.care_of_self.examples",
  },
  {
    value: "care_of_environment",
    labelId: "methodologyTools.practicalLife.cat.care_of_environment",
    descId: "methodologyTools.practicalLife.cat.care_of_environment.desc",
    examplesId: "methodologyTools.practicalLife.cat.care_of_environment.examples",
  },
  {
    value: "grace_and_courtesy",
    labelId: "methodologyTools.practicalLife.cat.grace_and_courtesy",
    descId: "methodologyTools.practicalLife.cat.grace_and_courtesy.desc",
    examplesId: "methodologyTools.practicalLife.cat.grace_and_courtesy.examples",
  },
  {
    value: "movement_control",
    labelId: "methodologyTools.practicalLife.cat.movement_control",
    descId: "methodologyTools.practicalLife.cat.movement_control.desc",
    examplesId: "methodologyTools.practicalLife.cat.movement_control.examples",
  },
  {
    value: "sensorial",
    labelId: "methodologyTools.practicalLife.cat.sensorial",
    descId: "methodologyTools.practicalLife.cat.sensorial.desc",
    examplesId: "methodologyTools.practicalLife.cat.sensorial.examples",
  },
];

const MASTERY_LEVELS: {
  value: MasteryLevel;
  labelId: string;
  descId: string;
  color: string;
}[] = [
  {
    value: "introduced",
    labelId: "methodologyTools.practicalLife.mastery.introduced",
    descId: "methodologyTools.practicalLife.mastery.introduced.desc",
    color: "bg-surface-container text-on-surface",
  },
  {
    value: "practicing",
    labelId: "methodologyTools.practicalLife.mastery.practicing",
    descId: "methodologyTools.practicalLife.mastery.practicing.desc",
    color: "bg-secondary-container text-on-secondary-container",
  },
  {
    value: "proficient",
    labelId: "methodologyTools.practicalLife.mastery.proficient",
    descId: "methodologyTools.practicalLife.mastery.proficient.desc",
    color: "bg-tertiary-container text-on-tertiary-container",
  },
  {
    value: "mastered",
    labelId: "methodologyTools.practicalLife.mastery.mastered",
    descId: "methodologyTools.practicalLife.mastery.mastered.desc",
    color: "bg-primary-container text-on-primary-container",
  },
];

// ─── Methodology gate banner ─────────────────────────────────────────────────

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "montessori") return null;
  return (
    <div
      className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6"
      role="note"
    >
      <Icon icon={Home} size="sm" className="mt-0.5 shrink-0 text-primary" aria-hidden />
      <p className="type-body-sm">
        <FormattedMessage id="methodologyTools.notPrimary.practicalLife" />
      </p>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function PracticalLife() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [category, setCategory] = useState<LifeSkillCategory>("care_of_self");
  const [activityName, setActivityName] = useState("");
  const [masteryLevel, setMasteryLevel] = useState<MasteryLevel>("practicing");
  const [observations, setObservations] = useState("");
  const [durationMinutes, setDurationMinutes] = useState("");
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  const categoryInfo = LIFE_SKILL_CATEGORIES.find(c => c.value === category);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.practicalLife.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !activityName.trim()) return;

    const categoryLabel = intl.formatMessage({ id: LIFE_SKILL_CATEGORIES.find(c => c.value === category)?.labelId ?? "" });
    const masteryLabel = intl.formatMessage({ id: MASTERY_LEVELS.find(m => m.value === masteryLevel)?.labelId ?? "" });

    const descParts: string[] = [
      `Category: ${categoryLabel}`,
      `Mastery level: ${masteryLabel}`,
      observations ? `Observations: ${observations}` : "",
    ].filter(Boolean);

    logActivity.mutate(
      {
        title: `Practical Life: ${activityName}`,
        description: descParts.join("\n"),
        subject_tags: ["practical_life", "life_skills"],
        tool_id: "practical-life",
        duration_minutes: durationMinutes ? Number(durationMinutes) : undefined,
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
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-sm text-on-surface font-semibold flex items-center gap-2"
        >
          <Icon icon={Home} size="md" className="text-primary" aria-hidden />
          <FormattedMessage id="methodologyTools.practicalLife.title" />
        </h1>
      </div>

      <MethodologyBanner />

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {students && students.length > 1 && (
            <div>
              <label htmlFor="pl-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select id="pl-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
              </Select>
            </div>
          )}

          {/* Life skill category */}
          <div>
            <p className="type-label-md text-on-surface-variant font-medium mb-3">
              <FormattedMessage id="methodologyTools.practicalLife.category" />
            </p>
            <div
              className="grid grid-cols-1 sm:grid-cols-2 gap-2"
              role="radiogroup"
              aria-label={intl.formatMessage({ id: "methodologyTools.practicalLife.category" })}
            >
              {LIFE_SKILL_CATEGORIES.map(cat => (
                <label
                  key={cat.value}
                  className={`p-3 rounded-xl cursor-pointer transition-all ${
                    category === cat.value
                      ? "bg-primary-container text-on-primary-container ring-2 ring-offset-1 ring-primary"
                      : "bg-surface-container-low text-on-surface hover:bg-surface-container"
                  }`}
                >
                  <input
                    type="radio"
                    name="pl-category"
                    value={cat.value}
                    checked={category === cat.value}
                    onChange={() => setCategory(cat.value)}
                    className="sr-only"
                  />
                  <p className="type-label-md font-semibold">{intl.formatMessage({ id: cat.labelId })}</p>
                  <p className="type-body-sm mt-0.5 opacity-75">{intl.formatMessage({ id: cat.descId })}</p>
                </label>
              ))}
            </div>
          </div>

          {/* Example activities for selected category */}
          {categoryInfo && (
            <div className="p-3 rounded-xl bg-secondary-container/30 text-on-surface-variant">
              <p className="type-label-sm font-medium mb-1">
                <FormattedMessage id="methodologyTools.practicalLife.examples" />
              </p>
              <p className="type-body-sm">{intl.formatMessage({ id: categoryInfo.examplesId })}</p>
            </div>
          )}

          <div>
            <label htmlFor="pl-activity" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.practicalLife.activityName" />
              <span className="text-error ml-0.5" aria-hidden="true">*</span>
            </label>
            <Input
              id="pl-activity"
              placeholder={intl.formatMessage({ id: "methodologyTools.practicalLife.activityNamePlaceholder" })}
              value={activityName}
              onChange={e => setActivityName(e.target.value)}
              required
            />
          </div>

          {/* Mastery level */}
          <fieldset>
            <legend className="type-label-md text-on-surface-variant mb-3">
              <FormattedMessage id="methodologyTools.practicalLife.masteryLevel" />
            </legend>
            <div className="grid grid-cols-2 gap-2" role="radiogroup">
              {MASTERY_LEVELS.map(level => (
                <label
                  key={level.value}
                  className={`p-3 rounded-xl cursor-pointer transition-all ${
                    masteryLevel === level.value
                      ? `${level.color} ring-2 ring-offset-1 ring-primary`
                      : "bg-surface-container-low text-on-surface hover:bg-surface-container"
                  }`}
                >
                  <input
                    type="radio"
                    name="pl-mastery"
                    value={level.value}
                    checked={masteryLevel === level.value}
                    onChange={() => setMasteryLevel(level.value)}
                    className="sr-only"
                  />
                  <p className="type-label-md font-semibold">{intl.formatMessage({ id: level.labelId })}</p>
                  <p className="type-body-sm mt-0.5 opacity-75">{intl.formatMessage({ id: level.descId })}</p>
                </label>
              ))}
            </div>
          </fieldset>

          <div>
            <label htmlFor="pl-observations" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.practicalLife.observations" />
            </label>
            <Textarea
              id="pl-observations"
              placeholder={intl.formatMessage({ id: "methodologyTools.practicalLife.observationsPlaceholder" })}
              value={observations}
              onChange={e => setObservations(e.target.value)}
              rows={3}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="pl-date" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.date" />
              </label>
              <Input id="pl-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="pl-duration" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.durationMinutes" />
              </label>
              <Input
                id="pl-duration"
                type="number"
                min="1"
                max="480"
                placeholder="15"
                value={durationMinutes}
                onChange={e => setDurationMinutes(e.target.value)}
              />
            </div>
          </div>

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button variant="tertiary" type="button" onClick={() => void navigate("/learning")}>
              <FormattedMessage id="action.cancel" />
            </Button>
            <Button
              variant="primary"
              type="submit"
              disabled={!effectiveStudent || !activityName.trim() || logActivity.isPending}
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
