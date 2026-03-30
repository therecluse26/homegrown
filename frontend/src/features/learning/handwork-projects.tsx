import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Scissors, ArrowLeft } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
  Textarea,
} from "@/components/ui";
import { FileUpload } from "@/components/ui/file-upload";
import { useStudents } from "@/hooks/use-family";
import { useLogActivity } from "@/hooks/use-activities";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Types ─────��────────────────────────────────────────────────────────────

type CraftType =
  | "knitting"
  | "weaving"
  | "woodwork"
  | "sewing"
  | "felting"
  | "drawing"
  | "painting"
  | "sculpting"
  | "watercolor"
  | "embroidery"
  | "beeswax_modeling"
  | "basketry"
  | "other";

type ProjectStatus = "just_started" | "in_progress" | "finishing" | "completed";

const CRAFT_TYPES: { value: CraftType; labelId: string }[] = [
  { value: "knitting",         labelId: "methodologyTools.handwork.craft.knitting" },
  { value: "weaving",          labelId: "methodologyTools.handwork.craft.weaving" },
  { value: "woodwork",         labelId: "methodologyTools.handwork.craft.woodwork" },
  { value: "sewing",           labelId: "methodologyTools.handwork.craft.sewing" },
  { value: "felting",          labelId: "methodologyTools.handwork.craft.felting" },
  { value: "drawing",          labelId: "methodologyTools.handwork.craft.drawing" },
  { value: "painting",         labelId: "methodologyTools.handwork.craft.painting" },
  { value: "watercolor",       labelId: "methodologyTools.handwork.craft.watercolor" },
  { value: "sculpting",        labelId: "methodologyTools.handwork.craft.sculpting" },
  { value: "embroidery",       labelId: "methodologyTools.handwork.craft.embroidery" },
  { value: "beeswax_modeling", labelId: "methodologyTools.handwork.craft.beeswax_modeling" },
  { value: "basketry",         labelId: "methodologyTools.handwork.craft.basketry" },
  { value: "other",            labelId: "methodologyTools.handwork.craft.other" },
];

const PROJECT_STATUSES: { value: ProjectStatus; labelId: string; color: string }[] = [
  { value: "just_started", labelId: "methodologyTools.handwork.projectStatus.just_started", color: "bg-surface-container text-on-surface" },
  { value: "in_progress",  labelId: "methodologyTools.handwork.projectStatus.in_progress",  color: "bg-secondary-container text-on-secondary-container" },
  { value: "finishing",    labelId: "methodologyTools.handwork.projectStatus.finishing",     color: "bg-tertiary-container text-on-tertiary-container" },
  { value: "completed",    labelId: "methodologyTools.handwork.projectStatus.completed",     color: "bg-primary-container text-on-primary-container" },
];

// ─── Methodology gate banner ──────────────���──────────────────────────────────

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "waldorf") return null;
  return (
    <div
      className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6"
      role="note"
    >
      <Icon icon={Scissors} size="sm" className="mt-0.5 shrink-0 text-secondary" aria-hidden />
      <p className="type-body-sm">
        <FormattedMessage id="methodologyTools.notPrimary.handworkProjects" />
      </p>
    </div>
  );
}

// ──�� Main component ───────────��──────────────────────────────────────────────

export function HandworkProjects() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [projectName, setProjectName] = useState("");
  const [craftType, setCraftType] = useState<CraftType>("knitting");
  const [materials, setMaterials] = useState("");
  const [techniques, setTechniques] = useState("");
  const [progressNotes, setProgressNotes] = useState("");
  const [status, setStatus] = useState<ProjectStatus>("in_progress");
  const [durationMinutes, setDurationMinutes] = useState("");
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));
  const [photoFiles, setPhotoFiles] = useState<File[]>([]);

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.handworkProjects.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !projectName.trim()) return;

    const craftLabel = intl.formatMessage({ id: CRAFT_TYPES.find(c => c.value === craftType)?.labelId ?? "" });
    const statusLabel = intl.formatMessage({ id: PROJECT_STATUSES.find(s => s.value === status)?.labelId ?? "" });

    const descParts: string[] = [
      `Craft: ${craftLabel}`,
      `Status: ${statusLabel}`,
      materials ? `Materials: ${materials}` : "",
      techniques ? `Techniques: ${techniques}` : "",
      progressNotes ? `Progress notes: ${progressNotes}` : "",
      photoFiles.length > 0 ? `Photos: ${photoFiles.map(f => f.name).join(", ")}` : "",
    ].filter(Boolean);

    logActivity.mutate(
      {
        title: `Handwork: ${projectName} (${craftLabel})`,
        description: descParts.join("\n"),
        subject_tags: ["arts", "handwork"],
        tool_id: "handwork-projects",
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
          <Icon icon={Scissors} size="md" className="text-secondary" aria-hidden />
          <FormattedMessage id="methodologyTools.handworkProjects.title" />
        </h1>
      </div>

      <MethodologyBanner />

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {students && students.length > 1 && (
            <div>
              <label htmlFor="hp-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select id="hp-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
              </Select>
            </div>
          )}

          <div>
            <label htmlFor="hp-name" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.handwork.projectName" />
              <span className="text-error ml-0.5" aria-hidden="true">*</span>
            </label>
            <Input
              id="hp-name"
              placeholder={intl.formatMessage({ id: "methodologyTools.handwork.projectNamePlaceholder" })}
              value={projectName}
              onChange={e => setProjectName(e.target.value)}
              required
            />
          </div>

          <div>
            <label htmlFor="hp-craft" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.handwork.craftType" />
            </label>
            <Select id="hp-craft" value={craftType} onChange={e => setCraftType(e.target.value as CraftType)}>
              {CRAFT_TYPES.map(c => <option key={c.value} value={c.value}>{intl.formatMessage({ id: c.labelId })}</option>)}
            </Select>
          </div>

          {/* Project status */}
          <fieldset>
            <legend className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.handwork.status" />
            </legend>
            <div className="flex flex-wrap gap-2" role="radiogroup">
              {PROJECT_STATUSES.map(s => (
                <label
                  key={s.value}
                  className={`px-3 py-2 rounded-xl cursor-pointer type-label-md transition-all ${
                    status === s.value
                      ? `${s.color} ring-2 ring-offset-1 ring-primary`
                      : "bg-surface-container-low text-on-surface hover:bg-surface-container"
                  }`}
                >
                  <input
                    type="radio"
                    name="hp-status"
                    value={s.value}
                    checked={status === s.value}
                    onChange={() => setStatus(s.value)}
                    className="sr-only"
                  />
                  {intl.formatMessage({ id: s.labelId })}
                </label>
              ))}
            </div>
          </fieldset>

          <div>
            <label htmlFor="hp-materials" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.handwork.materials" />
            </label>
            <Input
              id="hp-materials"
              placeholder={intl.formatMessage({ id: "methodologyTools.handwork.materialsPlaceholder" })}
              value={materials}
              onChange={e => setMaterials(e.target.value)}
            />
          </div>

          <div>
            <label htmlFor="hp-techniques" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.handwork.techniques" />
            </label>
            <Input
              id="hp-techniques"
              placeholder={intl.formatMessage({ id: "methodologyTools.handwork.techniquesPlaceholder" })}
              value={techniques}
              onChange={e => setTechniques(e.target.value)}
            />
          </div>

          <div>
            <label htmlFor="hp-notes" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.handwork.progressNotes" />
            </label>
            <Textarea
              id="hp-notes"
              placeholder={intl.formatMessage({ id: "methodologyTools.handwork.progressNotesPlaceholder" })}
              value={progressNotes}
              onChange={e => setProgressNotes(e.target.value)}
              rows={3}
            />
          </div>

          {/* Project photos */}
          <div>
            <p className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.handwork.photos" />
            </p>
            <FileUpload
              accept="image/*"
              multiple
              onFiles={files => setPhotoFiles(prev => [...prev, ...files])}
            />
            {photoFiles.length > 0 && (
              <ul className="flex flex-wrap gap-2 mt-2">
                {photoFiles.map((f, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-1 px-3 py-1 rounded-full bg-secondary-container text-on-secondary-container type-label-sm"
                  >
                    {f.name}
                    <button
                      type="button"
                      onClick={() => setPhotoFiles(prev => prev.filter((_, idx) => idx !== i))}
                      className="ml-1 hover:text-error transition-colors"
                      aria-label={`Remove ${f.name}`}
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="hp-date" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.date" />
              </label>
              <Input id="hp-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="hp-duration" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.durationMinutes" />
              </label>
              <Input
                id="hp-duration"
                type="number"
                min="1"
                max="480"
                placeholder="45"
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
              disabled={!effectiveStudent || !projectName.trim() || logActivity.isPending}
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
