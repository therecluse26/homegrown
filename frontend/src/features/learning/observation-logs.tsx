import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Eye, ArrowLeft } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
  Textarea,
} from "@/components/ui";
import { SubjectPicker } from "@/components/common/subject-picker";
import { useStudents } from "@/hooks/use-family";
import { useLogActivity } from "@/hooks/use-activities";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Types ──────────────────────────────────────────────────────────────────

type ConcentrationLevel = "distracted" | "somewhat_focused" | "focused" | "deeply_absorbed";

const CONCENTRATION_LEVELS: {
  value: ConcentrationLevel;
  labelId: string;
  descId: string;
  color: string;
}[] = [
  {
    value: "distracted",
    labelId: "methodologyTools.observation.level.distracted",
    descId: "methodologyTools.observation.level.distracted.desc",
    color: "bg-error-container text-on-error-container",
  },
  {
    value: "somewhat_focused",
    labelId: "methodologyTools.observation.level.somewhat_focused",
    descId: "methodologyTools.observation.level.somewhat_focused.desc",
    color: "bg-surface-container text-on-surface",
  },
  {
    value: "focused",
    labelId: "methodologyTools.observation.level.focused",
    descId: "methodologyTools.observation.level.focused.desc",
    color: "bg-secondary-container text-on-secondary-container",
  },
  {
    value: "deeply_absorbed",
    labelId: "methodologyTools.observation.level.deeply_absorbed",
    descId: "methodologyTools.observation.level.deeply_absorbed.desc",
    color: "bg-primary-container text-on-primary-container",
  },
];

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "montessori") return null;
  return (
    <div className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6" role="note">
      <Icon icon={Eye} size="sm" className="mt-0.5 shrink-0 text-primary" aria-hidden />
      <p className="type-body-sm"><FormattedMessage id="methodologyTools.notPrimary.observationLogs" /></p>
    </div>
  );
}

export function ObservationLogs() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [workChosen, setWorkChosen] = useState("");
  const [materials, setMaterials] = useState("");
  const [durationMinutes, setDurationMinutes] = useState("");
  const [concentration, setConcentration] = useState<ConcentrationLevel>("focused");
  const [observations, setObservations] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.observationLogs.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !workChosen.trim()) return;

    const concentrationLabel = intl.formatMessage({ id: CONCENTRATION_LEVELS.find(c => c.value === concentration)?.labelId ?? "" });
    const descParts: string[] = [
      `Work chosen: ${workChosen}`,
      materials ? `Materials: ${materials}` : "",
      `Concentration: ${concentrationLabel}`,
      observations ? `Observations: ${observations}` : "",
    ].filter(Boolean);

    logActivity.mutate(
      {
        title: `Observation: ${workChosen}`,
        description: descParts.join("\n"),
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        tool_id: "observation-logs",
        duration_minutes: durationMinutes ? Number(durationMinutes) : undefined,
        activity_date: entryDate || undefined,
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
          <Icon icon={Eye} size="md" className="text-primary" aria-hidden />
          <FormattedMessage id="methodologyTools.observationLogs.title" />
        </h1>
      </div>

      <MethodologyBanner />

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {students && students.length > 1 && (
            <div>
              <label htmlFor="ol-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select id="ol-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
              </Select>
            </div>
          )}

          <div>
            <label htmlFor="ol-work" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.observation.workChosen" />
              <span className="text-error ml-0.5" aria-hidden="true">*</span>
            </label>
            <Input
              id="ol-work"
              placeholder={intl.formatMessage({ id: "methodologyTools.observation.workChosenPlaceholder" })}
              value={workChosen}
              onChange={e => setWorkChosen(e.target.value)}
              required
            />
          </div>

          <div>
            <label htmlFor="ol-materials" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.observation.materials" />
            </label>
            <Input
              id="ol-materials"
              placeholder={intl.formatMessage({ id: "methodologyTools.observation.materialsPlaceholder" })}
              value={materials}
              onChange={e => setMaterials(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="ol-date" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.date" />
              </label>
              <Input id="ol-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="ol-duration" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.durationMinutes" />
              </label>
              <Input
                id="ol-duration"
                type="number"
                min="1"
                max="480"
                placeholder="20"
                value={durationMinutes}
                onChange={e => setDurationMinutes(e.target.value)}
              />
            </div>
          </div>

          {/* Concentration level */}
          <fieldset>
            <legend className="type-label-md text-on-surface-variant mb-3">
              <FormattedMessage id="methodologyTools.observation.concentration" />
            </legend>
            <div className="grid grid-cols-2 gap-2" role="radiogroup">
              {CONCENTRATION_LEVELS.map(level => (
                <label
                  key={level.value}
                  className={`p-3 rounded-xl cursor-pointer transition-all ${
                    concentration === level.value
                      ? `${level.color} ring-2 ring-offset-1 ring-primary`
                      : "bg-surface-container-low text-on-surface hover:bg-surface-container"
                  }`}
                >
                  <input
                    type="radio"
                    name="ol-concentration"
                    value={level.value}
                    checked={concentration === level.value}
                    onChange={() => setConcentration(level.value)}
                    className="sr-only"
                  />
                  <p className="type-label-md font-semibold">{intl.formatMessage({ id: level.labelId })}</p>
                  <p className="type-body-sm mt-0.5 opacity-75">{intl.formatMessage({ id: level.descId })}</p>
                </label>
              ))}
            </div>
          </fieldset>

          <div>
            <label htmlFor="ol-observations" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.observation.notes" />
            </label>
            <Textarea
              id="ol-observations"
              placeholder={intl.formatMessage({ id: "methodologyTools.observation.notesPlaceholder" })}
              value={observations}
              onChange={e => setObservations(e.target.value)}
              rows={4}
            />
          </div>

          <div>
            <p className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.field.subjectTags" />
            </p>
            <SubjectPicker value={subjectTags} onChange={setSubjectTags} />
          </div>

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button variant="tertiary" type="button" onClick={() => void navigate("/learning")}>
              <FormattedMessage id="action.cancel" />
            </Button>
            <Button
              variant="primary"
              type="submit"
              disabled={!effectiveStudent || !workChosen.trim() || logActivity.isPending}
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
