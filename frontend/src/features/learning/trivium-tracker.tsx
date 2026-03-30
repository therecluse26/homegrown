import { useState, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { Columns3, ArrowLeft } from "lucide-react";
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

type TriviumStage = "grammar" | "logic" | "rhetoric";

const TRIVIUM_STAGES: { value: TriviumStage; labelId: string; descId: string; color: string }[] = [
  {
    value: "grammar",
    labelId: "methodologyTools.trivium.stage.grammar",
    descId: "methodologyTools.trivium.stage.grammar.desc",
    color: "bg-secondary-container text-on-secondary-container",
  },
  {
    value: "logic",
    labelId: "methodologyTools.trivium.stage.logic",
    descId: "methodologyTools.trivium.stage.logic.desc",
    color: "bg-tertiary-container text-on-tertiary-container",
  },
  {
    value: "rhetoric",
    labelId: "methodologyTools.trivium.stage.rhetoric",
    descId: "methodologyTools.trivium.stage.rhetoric.desc",
    color: "bg-primary-container text-on-primary-container",
  },
];

const MEMORIZATION_OPTIONS: { value: string; labelId: string }[] = [
  { value: "recitation",  labelId: "methodologyTools.trivium.memorization.recitation" },
  { value: "definition",  labelId: "methodologyTools.trivium.memorization.definition" },
  { value: "narration",   labelId: "methodologyTools.trivium.memorization.narration" },
  { value: "copywork",    labelId: "methodologyTools.trivium.memorization.copywork" },
  { value: "dictation",   labelId: "methodologyTools.trivium.memorization.dictation" },
];

const COMPOSITION_OPTIONS: { value: string; labelId: string }[] = [
  { value: "essay",        labelId: "methodologyTools.trivium.composition.essay" },
  { value: "debate",       labelId: "methodologyTools.trivium.composition.debate" },
  { value: "presentation", labelId: "methodologyTools.trivium.composition.presentation" },
  { value: "socratic",     labelId: "methodologyTools.trivium.composition.socratic" },
  { value: "research",     labelId: "methodologyTools.trivium.composition.research" },
];

function MethodologyBanner() {
  const { primarySlug } = useMethodologyContext();
  if (primarySlug === "classical") return null;
  return (
    <div className="flex items-start gap-3 p-4 rounded-xl bg-surface-container-low text-on-surface-variant mb-6" role="note">
      <Icon icon={Columns3} size="sm" className="mt-0.5 shrink-0 text-secondary" aria-hidden />
      <p className="type-body-sm"><FormattedMessage id="methodologyTools.notPrimary.triviumTracker" /></p>
    </div>
  );
}

export function TriviumTracker() {
  const intl = useIntl();
  const navigate = useNavigate();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [stage, setStage] = useState<TriviumStage>("grammar");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [topic, setTopic] = useState("");
  const [notes, setNotes] = useState("");
  const [durationMinutes, setDurationMinutes] = useState("");
  const [entryDate, setEntryDate] = useState(new Date().toISOString().slice(0, 10));
  const [vocabulary, setVocabulary] = useState("");
  const [memorization, setMemorization] = useState("");
  const [connections, setConnections] = useState("");
  const [compositionType, setCompositionType] = useState("");

  const effectiveStudent = studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");
  const logActivity = useLogActivity(effectiveStudent);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "methodologyTools.triviumTracker.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !topic.trim()) return;

    const stageLabel = intl.formatMessage({ id: TRIVIUM_STAGES.find(s => s.value === stage)?.labelId ?? "" });
    const descParts: string[] = [
      `Stage: ${stageLabel}`,
      `Topic: ${topic}`,
      vocabulary ? `Vocabulary / Facts: ${vocabulary}` : "",
      memorization ? `Memorization type: ${intl.formatMessage({ id: MEMORIZATION_OPTIONS.find(m => m.value === memorization)?.labelId ?? "" })}` : "",
      connections ? `Connections / Analysis: ${connections}` : "",
      compositionType ? `Composition type: ${intl.formatMessage({ id: COMPOSITION_OPTIONS.find(c => c.value === compositionType)?.labelId ?? "" })}` : "",
      notes ? `Notes: ${notes}` : "",
    ].filter(Boolean);

    logActivity.mutate(
      {
        title: `Trivium (${stageLabel}): ${topic}`,
        description: descParts.join("\n"),
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        tool_id: "trivium-tracker",
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
        <h1 ref={headingRef} tabIndex={-1} className="type-headline-sm text-on-surface font-semibold flex items-center gap-2">
          <Icon icon={Columns3} size="md" className="text-secondary" aria-hidden />
          <FormattedMessage id="methodologyTools.triviumTracker.title" />
        </h1>
      </div>

      <MethodologyBanner />

      {/* Stage selector */}
      <div
        className="grid grid-cols-3 gap-3"
        role="group"
        aria-label={intl.formatMessage({ id: "methodologyTools.trivium.stageLabel" })}
      >
        {TRIVIUM_STAGES.map(s => (
          <button
            key={s.value}
            type="button"
            onClick={() => setStage(s.value)}
            className={`p-4 rounded-xl text-left transition-all focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${
              stage === s.value
                ? `${s.color} ring-2 ring-offset-2 ring-primary`
                : "bg-surface-container-low text-on-surface hover:bg-surface-container"
            }`}
            aria-pressed={stage === s.value}
          >
            <p className="type-label-lg font-semibold">{intl.formatMessage({ id: s.labelId })}</p>
            <p className="type-body-sm mt-1 opacity-75">{intl.formatMessage({ id: s.descId })}</p>
          </button>
        ))}
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {students && students.length > 1 && (
            <div>
              <label htmlFor="tt-student" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.student" />
              </label>
              <Select id="tt-student" value={studentId} onChange={e => setStudentId(e.target.value)} required>
                <option value="">{intl.formatMessage({ id: "methodologyTools.field.selectStudent" })}</option>
                {students.map(s => <option key={s.id} value={s.id ?? ""}>{s.display_name}</option>)}
              </Select>
            </div>
          )}

          <div>
            <label htmlFor="tt-topic" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.trivium.topic" />
              <span className="text-error ml-0.5" aria-hidden="true">*</span>
            </label>
            <Input
              id="tt-topic"
              placeholder={intl.formatMessage(
                { id: "methodologyTools.trivium.topicPlaceholder" },
                { stage: intl.formatMessage({ id: TRIVIUM_STAGES.find(s => s.value === stage)?.labelId ?? "" }) },
              )}
              value={topic}
              onChange={e => setTopic(e.target.value)}
              required
            />
          </div>

          {/* Grammar-stage fields */}
          {stage === "grammar" && (
            <>
              <div>
                <label htmlFor="tt-vocab" className="block type-label-md text-on-surface-variant mb-1.5">
                  <FormattedMessage id="methodologyTools.trivium.grammar.vocabulary" />
                </label>
                <Input
                  id="tt-vocab"
                  placeholder={intl.formatMessage({ id: "methodologyTools.trivium.grammar.vocabularyPlaceholder" })}
                  value={vocabulary}
                  onChange={e => setVocabulary(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="tt-mem" className="block type-label-md text-on-surface-variant mb-1.5">
                  <FormattedMessage id="methodologyTools.trivium.grammar.memorization" />
                </label>
                <Select id="tt-mem" value={memorization} onChange={e => setMemorization(e.target.value)}>
                  <option value="">{intl.formatMessage({ id: "methodologyTools.trivium.memorization.optional" })}</option>
                  {MEMORIZATION_OPTIONS.map(m => (
                    <option key={m.value} value={m.value}>{intl.formatMessage({ id: m.labelId })}</option>
                  ))}
                </Select>
              </div>
            </>
          )}

          {/* Logic-stage fields */}
          {stage === "logic" && (
            <div>
              <label htmlFor="tt-connections" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.trivium.logic.connections" />
              </label>
              <Textarea
                id="tt-connections"
                placeholder={intl.formatMessage({ id: "methodologyTools.trivium.logic.connectionsPlaceholder" })}
                value={connections}
                onChange={e => setConnections(e.target.value)}
                rows={3}
              />
            </div>
          )}

          {/* Rhetoric-stage fields */}
          {stage === "rhetoric" && (
            <div>
              <label htmlFor="tt-composition" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.trivium.rhetoric.composition" />
              </label>
              <Select id="tt-composition" value={compositionType} onChange={e => setCompositionType(e.target.value)}>
                <option value="">{intl.formatMessage({ id: "methodologyTools.trivium.composition.optional" })}</option>
                {COMPOSITION_OPTIONS.map(c => (
                  <option key={c.value} value={c.value}>{intl.formatMessage({ id: c.labelId })}</option>
                ))}
              </Select>
            </div>
          )}

          <div>
            <p className="type-label-md text-on-surface-variant mb-2">
              <FormattedMessage id="methodologyTools.field.subjectTags" />
            </p>
            <SubjectPicker value={subjectTags} onChange={setSubjectTags} />
          </div>

          <div>
            <label htmlFor="tt-notes" className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="methodologyTools.field.notes" />
            </label>
            <Textarea
              id="tt-notes"
              placeholder={intl.formatMessage({ id: "methodologyTools.field.notesPlaceholder" })}
              value={notes}
              onChange={e => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="tt-date" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.date" />
              </label>
              <Input id="tt-date" type="date" value={entryDate} onChange={e => setEntryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="tt-duration" className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="methodologyTools.field.durationMinutes" />
              </label>
              <Input
                id="tt-duration"
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
              disabled={!effectiveStudent || !topic.trim() || logActivity.isPending}
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
