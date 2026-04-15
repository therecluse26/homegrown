import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { ArrowLeft } from "lucide-react";
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
import { PageTitle } from "@/components/common/page-title";
import { useStudents } from "@/hooks/use-family";
import {
  useCreateAssessment,
  useGradingScales,
  type ScoreType,
} from "@/hooks/use-assessments";

export function GradeNew() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: students, isPending: studentsLoading } = useStudents();
  const { data: scales } = useGradingScales();

  const [studentId, setStudentId] = useState("");
  const [title, setTitle] = useState("");
  const [scoreType, setScoreType] = useState<ScoreType>("percentage");
  const [scoreValue, setScoreValue] = useState("");
  const [maxValue, setMaxValue] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [assessmentDate, setAssessmentDate] = useState(
    new Date().toISOString().slice(0, 10),
  );
  const [gradingScaleId, setGradingScaleId] = useState("");
  const [notes, setNotes] = useState("");

  const effectiveStudent =
    studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const createAssessment = useCreateAssessment(effectiveStudent);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !title.trim() || !scoreValue) return;
    createAssessment.mutate(
      {
        title: title.trim(),
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        assessment_date: `${assessmentDate}T00:00:00Z`,
        score_type: scoreType,
        score_value: Number(scoreValue),
        max_value: maxValue ? Number(maxValue) : undefined,
        grading_scale_id: gradingScaleId || undefined,
        notes: notes.trim() || undefined,
      },
      {
        onSuccess: () => void navigate("/learning/grades"),
      },
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "gradeNew.title" })}
      />

      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/learning/grades")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="gradeNew.title" />
        </h1>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Student selector */}
          <div>
            <label
              htmlFor="grade-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journals.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="grade-student"
                value={effectiveStudent}
                onChange={(e) => setStudentId(e.target.value)}
                required
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

          {/* Title */}
          <div>
            <label
              htmlFor="grade-title"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="gradeNew.assessmentTitle" />
            </label>
            <Input
              id="grade-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={intl.formatMessage({
                id: "gradeNew.assessmentTitle.placeholder",
              })}
              required
            />
          </div>

          {/* Score type + value */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <label
                htmlFor="score-type"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="gradeNew.scoreType" />
              </label>
              <Select
                id="score-type"
                value={scoreType}
                onChange={(e) => setScoreType(e.target.value as ScoreType)}
              >
                <option value="percentage">
                  {intl.formatMessage({ id: "gradeNew.scoreType.percentage" })}
                </option>
                <option value="points">
                  {intl.formatMessage({ id: "gradeNew.scoreType.points" })}
                </option>
                <option value="letter">
                  {intl.formatMessage({ id: "gradeNew.scoreType.letter" })}
                </option>
              </Select>
            </div>
            <div>
              <label
                htmlFor="score-value"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="gradeNew.score" />
              </label>
              <Input
                id="score-value"
                type="number"
                step="0.01"
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
                  <FormattedMessage id="gradeNew.maxValue" />
                </label>
                <Input
                  id="max-value"
                  type="number"
                  step="0.01"
                  value={maxValue}
                  onChange={(e) => setMaxValue(e.target.value)}
                />
              </div>
            )}
          </div>

          {/* Date + Grading Scale */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="grade-date"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="gradeNew.date" />
              </label>
              <Input
                id="grade-date"
                type="date"
                value={assessmentDate}
                onChange={(e) => setAssessmentDate(e.target.value)}
              />
            </div>
            {scales && scales.length > 0 && (
              <div>
                <label
                  htmlFor="grading-scale"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="gradeNew.gradingScale" />
                </label>
                <Select
                  id="grading-scale"
                  value={gradingScaleId}
                  onChange={(e) => setGradingScaleId(e.target.value)}
                >
                  <option value="">
                    {intl.formatMessage({ id: "gradeNew.noScale" })}
                  </option>
                  {scales.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </Select>
              </div>
            )}
          </div>

          {/* Subjects */}
          <div>
            <label className="block type-label-md text-on-surface-variant mb-1.5">
              <FormattedMessage id="activityLog.field.subjects" />
            </label>
            <SubjectPicker
              value={subjectTags}
              onChange={setSubjectTags}
              allowCustom
            />
          </div>

          {/* Notes */}
          <div>
            <label
              htmlFor="grade-notes"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="gradeNew.notes" />
            </label>
            <Textarea
              id="grade-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          {/* Actions */}
          <div className="flex gap-2 justify-end pt-2">
            <Button
              variant="tertiary"
              size="sm"
              type="button"
              onClick={() => void navigate("/learning/grades")}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              size="sm"
              type="submit"
              loading={createAssessment.isPending}
              disabled={!effectiveStudent || !title.trim() || !scoreValue}
            >
              <FormattedMessage id="gradeNew.save" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
