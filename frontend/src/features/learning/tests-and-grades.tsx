import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Plus, GraduationCap, Calendar } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  Select,
  Skeleton,
  StatCard,
} from "@/components/ui";
import { InfiniteScroll } from "@/components/ui";
import { SubjectPicker } from "@/components/common/subject-picker";
import { useStudents } from "@/hooks/use-family";
import {
  useAssessments,
  useCreateAssessment,
  useGradingScales,
  type ScoreType,
} from "@/hooks/use-assessments";

// ─── Add assessment form ─────────────────────────────────────────────────────

function AddAssessmentForm({
  studentId,
  onClose,
}: {
  studentId: string;
  onClose: () => void;
}) {
  const intl = useIntl();
  const createAssessment = useCreateAssessment(studentId);
  const { data: scales } = useGradingScales();

  const [title, setTitle] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [assessmentDate, setAssessmentDate] = useState(
    new Date().toISOString().slice(0, 10),
  );
  const [scoreType, setScoreType] = useState<ScoreType>("percentage");
  const [scoreValue, setScoreValue] = useState("");
  const [maxValue, setMaxValue] = useState("");
  const [weight, setWeight] = useState("");
  const [gradingScaleId, setGradingScaleId] = useState("");
  const [notes, setNotes] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim() || !scoreValue) return;
    createAssessment.mutate(
      {
        title: title.trim(),
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        assessment_date: assessmentDate,
        score_type: scoreType,
        score_value: Number(scoreValue),
        max_value: maxValue ? Number(maxValue) : undefined,
        weight: weight ? Number(weight) : undefined,
        grading_scale_id: gradingScaleId || undefined,
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: () => {
          setTitle("");
          setSubjectTags([]);
          setScoreValue("");
          setMaxValue("");
          setWeight("");
          setNotes("");
          onClose();
        },
      },
    );
  }

  return (
    <Card className="bg-surface-container-low">
      <h3 className="type-title-sm text-on-surface font-semibold mb-4">
        <FormattedMessage id="grades.add.title" />
      </h3>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label
            htmlFor="assessment-title"
            className="block type-label-md text-on-surface-variant mb-1.5"
          >
            <FormattedMessage id="grades.field.title" />
          </label>
          <Input
            id="assessment-title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={intl.formatMessage({
              id: "grades.field.title.placeholder",
            })}
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="assessment-date"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.field.date" />
            </label>
            <Input
              id="assessment-date"
              type="date"
              value={assessmentDate}
              onChange={(e) => setAssessmentDate(e.target.value)}
              required
            />
          </div>
          <div>
            <label
              htmlFor="score-type"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.field.scoreType" />
            </label>
            <Select
              id="score-type"
              value={scoreType}
              onChange={(e) => setScoreType(e.target.value as ScoreType)}
            >
              <option value="percentage">
                {intl.formatMessage({ id: "grades.scoreType.percentage" })}
              </option>
              <option value="points">
                {intl.formatMessage({ id: "grades.scoreType.points" })}
              </option>
              <option value="letter">
                {intl.formatMessage({ id: "grades.scoreType.letter" })}
              </option>
            </Select>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-4">
          <div>
            <label
              htmlFor="score-value"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.field.score" />
            </label>
            <Input
              id="score-value"
              type="number"
              min="0"
              value={scoreValue}
              onChange={(e) => setScoreValue(e.target.value)}
              required
            />
          </div>
          {scoreType === "points" && (
            <div>
              <label
                htmlFor="max-value"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="grades.field.maxValue" />
              </label>
              <Input
                id="max-value"
                type="number"
                min="1"
                value={maxValue}
                onChange={(e) => setMaxValue(e.target.value)}
              />
            </div>
          )}
          <div>
            <label
              htmlFor="weight"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.field.weight" />
            </label>
            <Input
              id="weight"
              type="number"
              min="0"
              step="0.1"
              value={weight}
              onChange={(e) => setWeight(e.target.value)}
              placeholder="1.0"
            />
          </div>
        </div>

        {scales && scales.length > 0 && (
          <div>
            <label
              htmlFor="grading-scale"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.field.gradingScale" />
            </label>
            <Select
              id="grading-scale"
              value={gradingScaleId}
              onChange={(e) => setGradingScaleId(e.target.value)}
            >
              <option value="">
                {intl.formatMessage({ id: "grades.field.gradingScale.none" })}
              </option>
              {scales.map((scale) => (
                <option key={scale.id} value={scale.id}>
                  {scale.name}
                  {scale.is_default ? " ★" : ""}
                </option>
              ))}
            </Select>
          </div>
        )}

        <div>
          <label className="block type-label-md text-on-surface-variant mb-1.5">
            <FormattedMessage id="grades.field.subjects" />
          </label>
          <SubjectPicker
            value={subjectTags}
            onChange={setSubjectTags}
            allowCustom
          />
        </div>

        <div>
          <label
            htmlFor="assessment-notes"
            className="block type-label-md text-on-surface-variant mb-1.5"
          >
            <FormattedMessage id="grades.field.notes" />
          </label>
          <Input
            id="assessment-notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder={intl.formatMessage({
              id: "grades.field.notes.placeholder",
            })}
          />
        </div>

        <div className="flex gap-2 justify-end">
          <Button variant="tertiary" size="sm" onClick={onClose} type="button">
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            variant="primary"
            size="sm"
            type="submit"
            loading={createAssessment.isPending}
            disabled={!title.trim() || !scoreValue}
          >
            <FormattedMessage id="grades.add.submit" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Score display helper ────────────────────────────────────────────────────

function ScoreDisplay({
  scoreType,
  scoreValue,
  maxValue,
}: {
  scoreType: ScoreType;
  scoreValue: number;
  maxValue?: number;
}) {
  switch (scoreType) {
    case "percentage":
      return <span>{scoreValue}%</span>;
    case "points":
      return (
        <span>
          {scoreValue}
          {maxValue ? `/${maxValue}` : ""}
        </span>
      );
    case "letter":
      // Score value maps to a letter grade string
      return <span>{scoreValue}</span>;
    default:
      return <span>{scoreValue}</span>;
  }
}

// ─── Main page ───────────────────────────────────────────────────────────────

export function TestsAndGrades() {
  const intl = useIntl();
  const { data: students, isPending: studentsLoading } = useStudents();
  const [selectedStudent, setSelectedStudent] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [subjectFilter, setSubjectFilter] = useState("");

  const effectiveStudent =
    selectedStudent || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const {
    data: pages,
    isPending: assessmentsLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useAssessments(effectiveStudent, {
    subject: subjectFilter || undefined,
  });

  const assessments = pages?.pages.flatMap((p) => p.data) ?? [];

  // Compute summary stats
  const totalAssessments = assessments.length;
  const avgScore =
    assessments.length > 0
      ? Math.round(
          assessments
            .filter((a) => a.score_type === "percentage")
            .reduce((sum, a) => sum + a.score_value, 0) /
            Math.max(
              assessments.filter((a) => a.score_type === "percentage").length,
              1,
            ),
        )
      : 0;
  const subjectCount = new Set(assessments.flatMap((a) => a.subject_tags)).size;

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="grades.title" />
        </h1>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowForm(true)}
          disabled={!effectiveStudent}
        >
          <Icon icon={Plus} size="sm" aria-hidden />
          <span className="ml-1.5">
            <FormattedMessage id="grades.add" />
          </span>
        </Button>
      </div>

      {/* Student selector + filters */}
      <Card className="bg-surface-container-low">
        <div className="flex flex-wrap items-end gap-4">
          <div className="flex-1 min-w-[180px]">
            <label
              htmlFor="grades-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="grades-student"
                value={effectiveStudent}
                onChange={(e) => setSelectedStudent(e.target.value)}
              >
                <option value="">
                  {intl.formatMessage({ id: "activityLog.selectStudent" })}
                </option>
                {students?.map((s) => (
                  <option key={s.id} value={s.id ?? ""}>
                    {s.display_name}
                  </option>
                ))}
              </Select>
            )}
          </div>
          <div className="min-w-[140px]">
            <label
              htmlFor="grades-subject"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="grades.filter.subject" />
            </label>
            <Input
              id="grades-subject"
              value={subjectFilter}
              onChange={(e) => setSubjectFilter(e.target.value)}
              placeholder={intl.formatMessage({
                id: "activityLog.filter.subject.placeholder",
              })}
            />
          </div>
        </div>
      </Card>

      {/* Add form */}
      {showForm && effectiveStudent && (
        <AddAssessmentForm
          studentId={effectiveStudent}
          onClose={() => setShowForm(false)}
        />
      )}

      {/* Summary stats */}
      {effectiveStudent && !assessmentsLoading && assessments.length > 0 && (
        <div className="grid grid-cols-3 gap-3">
          <StatCard
            label={intl.formatMessage({ id: "grades.stat.total" })}
            value={String(totalAssessments)}
          />
          <StatCard
            label={intl.formatMessage({ id: "grades.stat.average" })}
            value={`${avgScore}%`}
          />
          <StatCard
            label={intl.formatMessage({ id: "grades.stat.subjects" })}
            value={String(subjectCount)}
          />
        </div>
      )}

      {/* Assessment list */}
      {!effectiveStudent ? (
        <EmptyState
          message={intl.formatMessage({
            id: "activityLog.selectStudentFirst",
          })}
          description={intl.formatMessage({
            id: "activityLog.selectStudentFirst.description",
          })}
        />
      ) : assessmentsLoading ? (
        <div className="space-y-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      ) : assessments.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "grades.empty" })}
          description={intl.formatMessage({ id: "grades.empty.description" })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowForm(true)}
            >
              <FormattedMessage id="grades.add" />
            </Button>
          }
        />
      ) : (
        <>
          <ul className="space-y-2" role="list">
            {assessments.map((assessment) => (
              <li key={assessment.id}>
                <Card interactive className="flex items-start gap-3">
                  <div className="shrink-0 mt-0.5 text-primary">
                    <Icon icon={GraduationCap} size="md" aria-hidden />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <p className="type-title-sm text-on-surface font-medium">
                        {assessment.title}
                      </p>
                      <span className="type-title-sm text-primary font-semibold">
                        <ScoreDisplay
                          scoreType={assessment.score_type}
                          scoreValue={assessment.score_value}
                          maxValue={assessment.max_value}
                        />
                      </span>
                    </div>
                    <div className="flex flex-wrap items-center gap-3 mt-1">
                      <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
                        <Icon icon={Calendar} size="xs" aria-hidden />
                        {new Date(
                          assessment.assessment_date,
                        ).toLocaleDateString()}
                      </span>
                      {assessment.subject_tags?.map((tag) => (
                        <span
                          key={tag}
                          className="px-2 py-0.5 bg-primary-container text-on-primary-container type-label-sm rounded-full"
                        >
                          {tag}
                        </span>
                      ))}
                      {assessment.weight && (
                        <span className="type-label-sm text-on-surface-variant">
                          <FormattedMessage
                            id="grades.weight"
                            values={{ weight: assessment.weight }}
                          />
                        </span>
                      )}
                    </div>
                    {assessment.notes && (
                      <p className="type-body-sm text-on-surface-variant mt-1 line-clamp-1">
                        {assessment.notes}
                      </p>
                    )}
                  </div>
                </Card>
              </li>
            ))}
          </ul>

          <InfiniteScroll
            onLoadMore={() => void fetchNextPage()}
            loading={isFetchingNextPage}
            hasMore={!!hasNextPage}
          >
            <span />
          </InfiniteScroll>
        </>
      )}
    </div>
  );
}
