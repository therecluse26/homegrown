import { FormattedMessage, useIntl } from "react-intl";
import { Card, Skeleton, Badge, EmptyState } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useStudentIdentity } from "@/hooks/use-student-identity";
import { useJournalEntries } from "@/hooks/use-journals";

export function StudentJournal() {
  const intl = useIntl();
  const { data: identity } = useStudentIdentity();
  const studentId = identity?.student_id ?? "";
  const { data, isPending } = useJournalEntries(studentId);

  const entries = data?.pages?.flatMap((p) => p.data) ?? [];

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
        title={intl.formatMessage({ id: "studentJournal.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="studentJournal.title" />
      </h1>

      {entries.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "studentJournal.empty" })}
          description={intl.formatMessage({
            id: "studentJournal.emptyDescription",
          })}
        />
      ) : (
        <div className="space-y-3">
          {entries.map((entry) => (
            <Card key={entry.id} className="p-card-padding">
              <div className="flex items-center justify-between mb-2">
                <h3 className="type-title-md text-on-surface">
                  {entry.title ?? intl.formatMessage({ id: "journalDetail.untitled" })}
                </h3>
                <Badge variant="secondary">{entry.entry_type}</Badge>
              </div>
              <p className="type-label-sm text-on-surface-variant mb-2">
                {new Date(entry.entry_date).toLocaleDateString()}
              </p>
              <p className="type-body-sm text-on-surface line-clamp-3">
                {entry.content}
              </p>
              {entry.subject_tags.length > 0 && (
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {entry.subject_tags.map((tag) => (
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
