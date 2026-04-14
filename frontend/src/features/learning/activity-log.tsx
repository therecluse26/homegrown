import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useSearchParams } from "react-router";
import { Plus, Clock, Calendar, ClipboardList } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  Select,
  Skeleton,
} from "@/components/ui";
import { InfiniteScroll } from "@/components/ui";
import { SubjectPicker } from "@/components/common/subject-picker";
import { useStudents } from "@/hooks/use-family";
import { useActivityLog, useLogActivity } from "@/hooks/use-activities";
import { useMethodologyContext } from "@/features/auth/methodology-provider";
import { parseLocalDate } from "@/lib/date-utils";

// ─── Add activity form ──────────────────────────────────────────────────────

function AddActivityForm({
  studentId,
  onClose,
}: {
  studentId: string;
  onClose: () => void;
}) {
  const intl = useIntl();
  const logActivity = useLogActivity(studentId);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [subjectTags, setSubjectTags] = useState<string[]>([]);
  const [durationMinutes, setDurationMinutes] = useState("");
  const [activityDate, setActivityDate] = useState(
    new Date().toISOString().slice(0, 10),
  );

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    logActivity.mutate(
      {
        title: title.trim(),
        description: description.trim() || undefined,
        subject_tags: subjectTags.length > 0 ? subjectTags : undefined,
        duration_minutes: durationMinutes
          ? Number(durationMinutes)
          : undefined,
        activity_date: activityDate || undefined,
      },
      {
        onSuccess: () => {
          setTitle("");
          setDescription("");
          setSubjectTags([]);
          setDurationMinutes("");
          onClose();
        },
      },
    );
  }

  return (
    <Card className="bg-surface-container-low">
      <h3 className="type-title-sm text-on-surface font-semibold mb-4">
        <FormattedMessage id="activityLog.add.title" />
      </h3>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label
            htmlFor="activity-title"
            className="block type-label-md text-on-surface-variant mb-1.5"
          >
            <FormattedMessage id="activityLog.field.title" />
          </label>
          <Input
            id="activity-title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={intl.formatMessage({
              id: "activityLog.field.title.placeholder",
            })}
            required
          />
        </div>

        <div>
          <label
            htmlFor="activity-description"
            className="block type-label-md text-on-surface-variant mb-1.5"
          >
            <FormattedMessage id="activityLog.field.description" />
          </label>
          <Input
            id="activity-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={intl.formatMessage({
              id: "activityLog.field.description.placeholder",
            })}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="activity-duration"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityLog.field.duration" />
            </label>
            <Input
              id="activity-duration"
              type="number"
              min="1"
              value={durationMinutes}
              onChange={(e) => setDurationMinutes(e.target.value)}
              placeholder={intl.formatMessage({
                id: "activityLog.field.duration.placeholder",
              })}
            />
          </div>
          <div>
            <label
              htmlFor="activity-date"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityLog.field.date" />
            </label>
            <Input
              id="activity-date"
              type="date"
              value={activityDate}
              onChange={(e) => setActivityDate(e.target.value)}
            />
          </div>
        </div>

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

        <div className="flex gap-2 justify-end">
          <Button variant="tertiary" size="sm" onClick={onClose} type="button">
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            variant="primary"
            size="sm"
            type="submit"
            loading={logActivity.isPending}
            disabled={!title.trim()}
          >
            <FormattedMessage id="activityLog.add.submit" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Main page ──────────────────────────────────────────────────────────────

export function ActivityLog() {
  const intl = useIntl();
  const [searchParams] = useSearchParams();
  const { data: students, isPending: studentsLoading } = useStudents();
  const { toolLabel } = useMethodologyContext();
  const [selectedStudent, setSelectedStudent] = useState("");
  const [showForm, setShowForm] = useState(searchParams.get("new") === "1");
  const [subjectFilter, setSubjectFilter] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  // Auto-select first student if only one exists
  const effectiveStudent =
    selectedStudent || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const {
    data: activityPages,
    isPending: activitiesLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useActivityLog(effectiveStudent, {
    subject: subjectFilter || undefined,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
  });

  const activities = activityPages?.pages.flatMap((p) => p.data) ?? [];

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          {toolLabel("activities", intl.formatMessage({ id: "activityLog.title" }))}
        </h1>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowForm(true)}
          disabled={!effectiveStudent}
        >
          <Icon icon={Plus} size="sm" aria-hidden />
          <span className="ml-1.5">
            <FormattedMessage id="activityLog.add" />
          </span>
        </Button>
      </div>

      {/* Student selector + filters */}
      <Card className="bg-surface-container-low">
        <div className="flex flex-wrap items-end gap-4">
          <div className="flex-1 min-w-[180px]">
            <label
              htmlFor="student-select"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityLog.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="student-select"
                value={effectiveStudent}
                onChange={(e) => setSelectedStudent(e.target.value)}
              >
                <option value="">
                  {intl.formatMessage({
                    id: "activityLog.selectStudent",
                  })}
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
              htmlFor="filter-subject"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityLog.filter.subject" />
            </label>
            <Input
              id="filter-subject"
              value={subjectFilter}
              onChange={(e) => setSubjectFilter(e.target.value)}
              placeholder={intl.formatMessage({
                id: "activityLog.filter.subject.placeholder",
              })}
            />
          </div>

          <div className="min-w-[140px]">
            <label
              htmlFor="filter-from"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityLog.filter.from" />
            </label>
            <Input
              id="filter-from"
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
            />
          </div>

          <div className="min-w-[140px]">
            <label
              htmlFor="filter-to"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="activityLog.filter.to" />
            </label>
            <Input
              id="filter-to"
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
            />
          </div>
        </div>
      </Card>

      {/* Add activity form */}
      {showForm && effectiveStudent && (
        <AddActivityForm
          studentId={effectiveStudent}
          onClose={() => setShowForm(false)}
        />
      )}

      {/* Activity list */}
      {!effectiveStudent ? (
        <EmptyState
          message={intl.formatMessage({
            id: "activityLog.selectStudentFirst",
          })}
          description={intl.formatMessage({
            id: "activityLog.selectStudentFirst.description",
          })}
        />
      ) : activitiesLoading ? (
        <div className="space-y-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      ) : activities.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "activityLog.empty" })}
          description={intl.formatMessage({
            id: "activityLog.empty.description",
          })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowForm(true)}
            >
              <FormattedMessage id="activityLog.add" />
            </Button>
          }
        />
      ) : (
        <>
          <ul className="space-y-2" role="list">
            {activities.map((activity) => (
              <li key={activity.id}>
                <Card interactive className="flex items-start gap-3">
                  <div className="shrink-0 mt-0.5 text-primary">
                    <Icon icon={ClipboardList} size="md" aria-hidden />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="type-title-sm text-on-surface font-medium">
                      {activity.title}
                    </p>
                    {activity.description && (
                      <p className="type-body-sm text-on-surface-variant line-clamp-2 mt-0.5">
                        {activity.description}
                      </p>
                    )}
                    <div className="flex flex-wrap items-center gap-3 mt-2">
                      {activity.duration_minutes && (
                        <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
                          <Icon icon={Clock} size="xs" aria-hidden />
                          <FormattedMessage
                            id="activityLog.duration"
                            values={{
                              minutes: activity.duration_minutes,
                            }}
                          />
                        </span>
                      )}
                      <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
                        <Icon icon={Calendar} size="xs" aria-hidden />
                        {parseLocalDate(
                          activity.activity_date,
                        ).toLocaleDateString()}
                      </span>
                      {activity.subject_tags?.map((tag) => (
                        <span
                          key={tag}
                          className="px-2 py-0.5 bg-primary-container text-on-primary-container type-label-sm rounded-full"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
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
