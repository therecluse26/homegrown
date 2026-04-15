import { FormattedMessage, useIntl } from "react-intl";
import { Card, Skeleton, Badge, EmptyState } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useStudentIdentity } from "@/hooks/use-student-identity";
import { useActivityLog } from "@/hooks/use-activities";

export function StudentActivities() {
  const intl = useIntl();
  const { data: identity } = useStudentIdentity();
  const studentId = identity?.student_id ?? "";
  const { data, isPending } = useActivityLog(studentId);

  const activities = data?.pages?.flatMap((p) => p.data) ?? [];

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "studentActivities.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="studentActivities.title" />
      </h1>

      {activities.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "studentActivities.empty" })}
          description={intl.formatMessage({
            id: "studentActivities.emptyDescription",
          })}
        />
      ) : (
        <div className="space-y-3">
          {activities.map((activity) => (
            <Card key={activity.id} className="p-card-padding">
              <div className="flex items-center justify-between mb-2">
                <h3 className="type-title-md text-on-surface">
                  {activity.title}
                </h3>
                {activity.duration_minutes != null && (
                  <span className="type-label-sm text-on-surface-variant">
                    {activity.duration_minutes}{" "}
                    <FormattedMessage id="activityDetail.minutes" />
                  </span>
                )}
              </div>
              <p className="type-label-sm text-on-surface-variant mb-2">
                {new Date(activity.activity_date).toLocaleDateString()}
              </p>
              {activity.description && (
                <p className="type-body-sm text-on-surface line-clamp-2">
                  {activity.description}
                </p>
              )}
              {activity.subject_tags.length > 0 && (
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {activity.subject_tags.map((tag) => (
                    <Badge key={tag} variant="secondary">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
