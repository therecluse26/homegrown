import { FormattedMessage, useIntl } from "react-intl";
import { Card, Skeleton, Badge, EmptyState, Icon } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useStudentIdentity } from "@/hooks/use-student-identity";
import { useReadingProgress } from "@/hooks/use-reading";
import { BookOpen } from "lucide-react";

export function StudentReadingList() {
  const intl = useIntl();
  const { data: identity } = useStudentIdentity();
  const studentId = identity?.student_id ?? "";
  const { data, isPending } = useReadingProgress(studentId);

  const items = data?.pages?.flatMap((p) => p.data) ?? [];

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
        title={intl.formatMessage({ id: "studentReadingList.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="studentReadingList.title" />
      </h1>

      {items.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "studentReadingList.empty" })}
          description={intl.formatMessage({
            id: "studentReadingList.emptyDescription",
          })}
        />
      ) : (
        <Card className="p-card-padding">
          <div className="space-y-2">
            {items.map((item) => (
              <div
                key={item.id}
                className="flex items-center justify-between py-3 border-b border-outline-variant/10 last:border-0"
              >
                <div className="flex items-center gap-3">
                  <Icon
                    icon={BookOpen}
                    size="sm"
                    className="text-on-surface-variant"
                  />
                  <div>
                    <p className="type-body-sm text-on-surface">
                      {item.reading_item.title}
                    </p>
                    {item.reading_item.author && (
                      <p className="type-label-sm text-on-surface-variant">
                        {item.reading_item.author}
                      </p>
                    )}
                  </div>
                </div>
                <Badge
                  variant={
                    item.status === "completed" ? "primary" : "secondary"
                  }
                >
                  {item.status === "completed"
                    ? intl.formatMessage({ id: "studentReadingList.completed" })
                    : item.status === "in_progress"
                      ? intl.formatMessage({ id: "studentReadingList.reading" })
                      : intl.formatMessage({ id: "studentReadingList.toRead" })}
                </Badge>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
