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
import { useLogActivity } from "@/hooks/use-activities";

export function ActivityNew() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: students, isPending: studentsLoading } = useStudents();

  const [studentId, setStudentId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [durationMinutes, setDurationMinutes] = useState("");
  const [activityDate, setActivityDate] = useState(
    new Date().toISOString().slice(0, 10),
  );

  const effectiveStudent =
    studentId || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const logActivity = useLogActivity(effectiveStudent);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!effectiveStudent || !title.trim()) return;
    logActivity.mutate(
      {
        title: title.trim(),
        description: description.trim() || undefined,
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        duration_minutes: durationMinutes
          ? Number(durationMinutes)
          : undefined,
        activity_date: activityDate
          ? `${activityDate}T00:00:00Z`
          : undefined,
      },
      {
        onSuccess: () => void navigate("/learning/activities"),
      },
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "activityNew.title" })}
      />

      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/learning/activities")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="activityNew.title" />
        </h1>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Student selector */}
          <div>
            <label
              htmlFor="activity-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journals.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="activity-student"
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
              htmlFor="activity-title"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityNew.activityTitle" />
            </label>
            <Input
              id="activity-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={intl.formatMessage({
                id: "activityNew.activityTitle.placeholder",
              })}
              required
            />
          </div>

          {/* Description */}
          <div>
            <label
              htmlFor="activity-desc"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityNew.description" />
            </label>
            <Textarea
              id="activity-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={5}
            />
          </div>

          {/* Date + Duration */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="activity-date"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="activityNew.date" />
              </label>
              <Input
                id="activity-date"
                type="date"
                value={activityDate}
                onChange={(e) => setActivityDate(e.target.value)}
              />
            </div>
            <div>
              <label
                htmlFor="activity-duration"
                className="block type-label-md text-on-surface-variant mb-1.5"
              >
                <FormattedMessage id="activityNew.duration" />
              </label>
              <Input
                id="activity-duration"
                type="number"
                min={0}
                value={durationMinutes}
                onChange={(e) => setDurationMinutes(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "activityNew.duration.placeholder",
                })}
              />
            </div>
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

          {/* Actions */}
          <div className="flex gap-2 justify-end pt-2">
            <Button
              variant="tertiary"
              size="sm"
              type="button"
              onClick={() => void navigate("/learning/activities")}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              size="sm"
              type="submit"
              loading={logActivity.isPending}
              disabled={!effectiveStudent || !title.trim()}
            >
              <FormattedMessage id="activityNew.save" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
