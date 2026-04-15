import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate, Link as RouterLink } from "react-router";
import { ArrowLeft, Edit2, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  Input,
  Select,
  Textarea,
  Modal,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ResourceNotFound } from "@/components/common/resource-not-found";
import { SubjectPicker } from "@/components/common/subject-picker";
import {
  useAssessment,
  useUpdateAssessment,
  useDeleteAssessment,
  type ScoreType,
} from "@/hooks/use-assessments";

export function GradeDetail() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);

  const searchParams = new URLSearchParams(
    typeof window !== "undefined" ? window.location.search : "",
  );
  const studentId = searchParams.get("studentId") ?? "";

  const { data: assessment, isPending } = useAssessment(
    studentId,
    id ?? "",
  );
  const updateAssessment = useUpdateAssessment(
    assessment?.student_id ?? studentId,
  );
  const deleteAssessment = useDeleteAssessment(
    assessment?.student_id ?? studentId,
  );

  const [editTitle, setEditTitle] = useState("");
  const [editScoreType, setEditScoreType] = useState<ScoreType>("percentage");
  const [editScoreValue, setEditScoreValue] = useState("");
  const [editMaxValue, setEditMaxValue] = useState("");
  const [editTags, setEditTags] = useState<string[]>([]);
  const [editDate, setEditDate] = useState("");
  const [editNotes, setEditNotes] = useState("");

  function startEdit() {
    if (!assessment) return;
    setEditTitle(assessment.title);
    setEditScoreType(assessment.score_type);
    setEditScoreValue(String(assessment.score_value));
    setEditMaxValue(
      assessment.max_value != null ? String(assessment.max_value) : "",
    );
    setEditTags(assessment.subject_tags ?? []);
    setEditDate(assessment.assessment_date?.slice(0, 10) ?? "");
    setEditNotes(assessment.notes ?? "");
    setEditing(true);
  }

  function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!assessment || !editTitle.trim() || !editScoreValue) return;
    updateAssessment.mutate(
      {
        id: assessment.id,
        title: editTitle.trim(),
        subject_tags: editTags.length > 0 ? editTags : undefined,
        assessment_date: editDate ? `${editDate}T00:00:00Z` : undefined,
        score_type: editScoreType,
        score_value: Number(editScoreValue),
        max_value: editMaxValue ? Number(editMaxValue) : undefined,
        notes: editNotes.trim() || undefined,
      },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (!assessment) return;
    deleteAssessment.mutate(assessment.id, {
      onSuccess: () => void navigate("/learning/grades"),
    });
  }

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!assessment) {
    return <ResourceNotFound backTo="/learning/grades" />;
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle title={assessment.title} />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/learning/grades"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="gradeDetail.backToGrades" />
        </RouterLink>
      </div>

      {editing ? (
        <Card>
          <form onSubmit={handleSave} className="space-y-5">
            <div>
              <label
                htmlFor="edit-title"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="gradeNew.assessmentTitle" />
              </label>
              <Input
                id="edit-title"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                required
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label
                  htmlFor="edit-score-type"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="gradeNew.scoreType" />
                </label>
                <Select
                  id="edit-score-type"
                  value={editScoreType}
                  onChange={(e) =>
                    setEditScoreType(e.target.value as ScoreType)
                  }
                >
                  <option value="percentage">
                    {intl.formatMessage({
                      id: "gradeNew.scoreType.percentage",
                    })}
                  </option>
                  <option value="points">
                    {intl.formatMessage({
                      id: "gradeNew.scoreType.points",
                    })}
                  </option>
                  <option value="letter">
                    {intl.formatMessage({
                      id: "gradeNew.scoreType.letter",
                    })}
                  </option>
                </Select>
              </div>
              <div>
                <label
                  htmlFor="edit-score"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="gradeNew.score" />
                </label>
                <Input
                  id="edit-score"
                  type="number"
                  step="0.01"
                  value={editScoreValue}
                  onChange={(e) => setEditScoreValue(e.target.value)}
                  required
                />
              </div>
              {editScoreType === "points" && (
                <div>
                  <label
                    htmlFor="edit-max"
                    className="block type-label-md text-on-surface-variant mb-1.5"
                  >
                    <FormattedMessage id="gradeNew.maxValue" />
                  </label>
                  <Input
                    id="edit-max"
                    type="number"
                    step="0.01"
                    value={editMaxValue}
                    onChange={(e) => setEditMaxValue(e.target.value)}
                  />
                </div>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="edit-date"
                  className="block type-label-md text-on-surface-variant mb-1.5"
                >
                  <FormattedMessage id="gradeNew.date" />
                </label>
                <Input
                  id="edit-date"
                  type="date"
                  value={editDate}
                  onChange={(e) => setEditDate(e.target.value)}
                />
              </div>
            </div>

            <div>
              <label className="block type-label-md text-on-surface-variant mb-1.5">
                <FormattedMessage id="activityLog.field.subjects" />
              </label>
              <SubjectPicker
                value={editTags}
                onChange={setEditTags}
                allowCustom
              />
            </div>

            <div>
              <label
                htmlFor="edit-notes"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="gradeNew.notes" />
              </label>
              <Textarea
                id="edit-notes"
                value={editNotes}
                onChange={(e) => setEditNotes(e.target.value)}
                rows={3}
              />
            </div>

            <div className="flex gap-2 justify-end pt-2">
              <Button
                variant="tertiary"
                size="sm"
                type="button"
                onClick={() => setEditing(false)}
              >
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                variant="primary"
                size="sm"
                type="submit"
                loading={updateAssessment.isPending}
                disabled={!editTitle.trim() || !editScoreValue}
              >
                <FormattedMessage id="common.save" />
              </Button>
            </div>
          </form>
        </Card>
      ) : (
        <Card className="p-card-padding">
          <div className="flex items-center justify-between mb-4">
            <h1 className="type-headline-sm text-on-surface">
              {assessment.title}
            </h1>
            <div className="flex items-center gap-2">
              <Badge variant="secondary">{assessment.score_type}</Badge>
              <Button variant="tertiary" size="sm" onClick={startEdit}>
                <Icon icon={Edit2} size="sm" />
              </Button>
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => setShowDelete(true)}
              >
                <Icon icon={Trash2} size="sm" />
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 mb-4">
            <div>
              <p className="type-label-sm text-on-surface-variant mb-1">
                <FormattedMessage id="gradeDetail.score" />
              </p>
              <p className="type-body-sm text-on-surface">
                {assessment.score_value}
                {assessment.max_value != null &&
                  ` / ${assessment.max_value}`}
              </p>
            </div>
            <div>
              <p className="type-label-sm text-on-surface-variant mb-1">
                <FormattedMessage id="gradeDetail.date" />
              </p>
              <p className="type-body-sm text-on-surface">
                {new Date(assessment.assessment_date).toLocaleDateString()}
              </p>
            </div>
            <div>
              <p className="type-label-sm text-on-surface-variant mb-1">
                <FormattedMessage id="gradeDetail.recorded" />
              </p>
              <p className="type-body-sm text-on-surface">
                {new Date(assessment.created_at).toLocaleDateString()}
              </p>
            </div>
          </div>

          {assessment.subject_tags.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mb-4">
              {assessment.subject_tags.map((tag) => (
                <Badge key={tag} variant="secondary">
                  {tag}
                </Badge>
              ))}
            </div>
          )}

          {assessment.notes && (
            <div className="type-body-sm text-on-surface whitespace-pre-wrap">
              {assessment.notes}
            </div>
          )}
        </Card>
      )}

      <Modal
        open={showDelete}
        onClose={() => setShowDelete(false)}
        title={intl.formatMessage({ id: "gradeDetail.deleteTitle" })}
      >
        <div className="space-y-4">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="gradeDetail.deleteConfirm" />
          </p>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowDelete(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleDelete}
              loading={deleteAssessment.isPending}
            >
              <FormattedMessage id="common.delete" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
