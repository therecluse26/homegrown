import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Lightbulb, ArrowLeft, X } from "lucide-react";
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

type ExplorationMethod = "reading" | "watching" | "doing" | "talking" | "observing" | "creating" | "playing";

const EXPLORATION_METHODS: { value: ExplorationMethod; labelId: string }[] = [
  { value: "reading",    labelId: "methodologyTools.interestLed.method.reading" },
  { value: "watching",   labelId: "methodologyTools.interestLed.method.watching" },
  { value: "doing",      labelId: "methodologyTools.interestLed.method.doing" },
  { value: "talking",    labelId: "methodologyTools.interestLed.method.talking" },
  { value: "observing",  labelId: "methodologyTools.interestLed.method.observing" },
  { value: "creating",   labelId: "methodologyTools.interestLed.method.creating" },
  { value: "playing",    labelId: "methodologyTools.interestLed.method.playing" },
];

// ─── Methodology gate banner ─────────────────────────────────────────────────

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "unschooling") return null;
  return (
    <div
      className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6"
      role="note"
    >
      <Icon icon={Lightbulb} size="sm" className="mt-0.5 shrink-0 text-tertiary" aria-hidden />
      <p className="type-body-sm">
        <FormattedMessage id="methodologyTools.notPrimary.interestLedLog" />
      </p>
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function InterestLedLog() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [interest, setInterest] = useState("");
  const [howExplored, setHowExplored] = useState<ExplorationMethod[]>(["doing"]);
  const [activityDescription, setActivityDescription] = useState("");
  const [newResource, setNewResource] = useState("");
  const [resourceList, setResourceList] = useState<string[]>([]);
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [durationMinutes, setDurationMinutes] = useState("");
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.interestLedLog.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function toggleExploration(method: ExplorationMethod) {
    setHowExplored(prev =>
      prev.includes(method) ? prev.filter(m => m !== method) : [...prev, method],
    );
  }

  function addResource() {
    if (!newResource.trim()) return;
    setResourceList(prev => [...prev, newResource.trim()]);
    setNewResource("");
  }

  function removeResource(index: number) {
    setResourceList(prev => prev.filter((_, i) => i !== index));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !interest.trim()) return;

    const explorationLabels = howExplored
      .map(m => intl.formatMessage({ id: EXPLORATION_METHODS.find(opt => opt.value === m)?.labelId ?? "" }))
      .join(", ");

    const descParts: string[] = [
      `Interest: ${interest}`,
      howExplored.length > 0 ? `How explored: ${explorationLabels}` : "",
      activityDescription ? `Description: ${activityDescription}` : "",
      resourceList.length > 0 ? `Resources: ${resourceList.join(", ")}` : "",
    ].filter(Boolean);

    logActivity.mutate(
      {
        title: `Interest-Led: ${interest}`,
        description: descParts.join("\n"),
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        tool_id: "interest-led-log",
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
          <Icon icon={Lightbulb} size="md" className="text-tertiary" aria-hidden />
          <FormattedMessage id="methodologyTools.interestLedLog.title" />
        </h1>
      </div>

      <MethodologyBanner />

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {students && students.length > 1 && (
            <div>
              <label htmlFor="ill-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select id="ill-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
              </Select>
            </div>
          )}

          <div>
            <label htmlFor="ill-interest" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.interestLed.interest" />
              <span className="text-error ml-0.5" aria-hidden="true">*</span>
            </label>
            <Input
              id="ill-interest"
              placeholder={intl.formatMessage({ id: "methodologyTools.interestLed.interestPlaceholder" })}
              value={interest}
              onChange={e => setInterest(e.target.value)}
              required
            />
          </div>

          {/* How it was explored */}
          <fieldset>
            <legend className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.interestLed.howExplored" />
            </legend>
            <div className="flex flex-wrap gap-2">
              {EXPLORATION_METHODS.map(method => {
                const selected = howExplored.includes(method.value);
                return (
                  <button
                    key={method.value}
                    type="button"
                    onClick={() => toggleExploration(method.value)}
                    className={`px-3 py-1.5 rounded-full type-label-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring ${
                      selected
                        ? "bg-primary text-on-primary"
                        : "bg-surface-container-low text-on-surface hover:bg-surface-container"
                    }`}
                    aria-pressed={selected}
                  >
                    {intl.formatMessage({ id: method.labelId })}
                  </button>
                );
              })}
            </div>
          </fieldset>

          <div>
            <label htmlFor="ill-description" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.interestLed.description" />
            </label>
            <Textarea
              id="ill-description"
              placeholder={intl.formatMessage({ id: "methodologyTools.interestLed.descriptionPlaceholder" })}
              value={activityDescription}
              onChange={e => setActivityDescription(e.target.value)}
              rows={4}
            />
          </div>

          {/* Resources used */}
          <div>
            <p className="type-label-md text-on-surface-variant font-medium mb-2">
              <FormattedMessage id="methodologyTools.interestLed.resources" />
            </p>
            {resourceList.length > 0 && (
              <ul className="flex flex-wrap gap-2 mb-3" aria-label={intl.formatMessage({ id: "methodologyTools.interestLed.resourcesList" })}>
                {resourceList.map((res, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-1 px-3 py-1 rounded-full bg-secondary-container text-on-secondary-container type-label-sm"
                  >
                    {res}
                    <button
                      type="button"
                      onClick={() => removeResource(i)}
                      className="ml-1 hover:text-error transition-colors focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus-ring"
                      aria-label={intl.formatMessage({ id: "methodologyTools.interestLed.removeResource" }, { resource: res })}
                    >
                      <Icon icon={X} size="xs" aria-hidden />
                    </button>
                  </li>
                ))}
              </ul>
            )}
            <div className="flex gap-2">
              <Input
                placeholder={intl.formatMessage({ id: "methodologyTools.interestLed.resourcePlaceholder" })}
                value={newResource}
                onChange={e => setNewResource(e.target.value)}
                onKeyDown={e => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addResource();
                  }
                }}
              />
              <Button
                variant="secondary"
                type="button"
                onClick={addResource}
                disabled={!newResource.trim()}
              >
                <FormattedMessage id="methodologyTools.interestLed.addResource" />
              </Button>
            </div>
          </div>

          {/* Subject connections */}
          <div>
            <p className="type-label-md text-on-surface-variant font-medium mb-1">
              <FormattedMessage id="methodologyTools.interestLed.subjectConnections" />
            </p>
            <p className="type-body-sm text-on-surface-variant mb-3">
              <FormattedMessage id="methodologyTools.interestLed.subjectConnectionsHint" />
            </p>
            <SubjectPicker value={subjectTags} onChange={setSubjectTags} />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="ill-date" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.date" />
              </label>
              <Input id="ill-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="ill-duration" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.durationMinutes" />
              </label>
              <Input
                id="ill-duration"
                type="number"
                min="1"
                max="480"
                placeholder="60"
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
              disabled={!effectiveStudent || !interest.trim() || logActivity.isPending}
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
