import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  Download,
  Loader2,
  BookOpen,
  PenTool,
  ClipboardList,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  ProgressBar,
  Skeleton,
  StatCard,
} from "@/components/ui";
import { InfiniteScroll } from "@/components/ui";
import {
  useStudentProgress,
  useSubjectProgress,
  useProgressTimeline,
  type TimelineEntryType,
} from "@/hooks/use-progress";
import { useStudents } from "@/hooks/use-family";
import { useRequestExport } from "@/hooks/use-data-lifecycle";

// ─── Timeline entry icon mapping ────────────────────────────────────────────

const TIMELINE_ICONS: Record<TimelineEntryType, typeof ClipboardList> = {
  activity: ClipboardList,
  journal: PenTool,
  reading_completed: BookOpen,
};

// ─── Main component ─────────────────────────────────────────────────────────

export function ProgressView() {
  const intl = useIntl();
  const { studentId } = useParams<{ studentId: string }>();
  const { data: students } = useStudents();
  const student = students?.find((s) => s.id === studentId);

  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const requestExport = useRequestExport();

  const dateParams = {
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
  };

  const { data: summary, isPending: summaryLoading } = useStudentProgress(
    studentId ?? "",
    dateParams,
  );
  const { data: subjectProgress, isPending: subjectsLoading } =
    useSubjectProgress(studentId ?? "", dateParams);
  const {
    data: timelinePages,
    isPending: timelineLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useProgressTimeline(studentId ?? "", dateParams);

  const timeline = timelinePages?.pages.flatMap((p) => p.data) ?? [];

  if (!studentId) {
    return (
      <EmptyState
        message={intl.formatMessage({ id: "progress.noStudent" })}
      />
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <RouterLink to="/learning" className="no-underline">
          <Button variant="tertiary" size="sm">
            <Icon icon={ArrowLeft} size="sm" aria-hidden />
            <span className="ml-1">
              <FormattedMessage id="common.back" />
            </span>
          </Button>
        </RouterLink>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage
            id="progress.title"
            values={{ name: student?.display_name ?? "" }}
          />
        </h1>
      </div>

      {/* Date range filter */}
      <Card className="bg-surface-container-low">
        <div className="flex flex-wrap items-end gap-4">
          <div className="min-w-[140px]">
            <label
              htmlFor="progress-from"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="progress.dateFrom" />
            </label>
            <Input
              id="progress-from"
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
            />
          </div>
          <div className="min-w-[140px]">
            <label
              htmlFor="progress-to"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="progress.dateTo" />
            </label>
            <Input
              id="progress-to"
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
            />
          </div>
          <Button
            variant="tertiary"
            size="sm"
            disabled={requestExport.isPending}
            onClick={() =>
              requestExport.mutate({
                format: "csv",
                domains: ["learning"],
              })
            }
          >
            {requestExport.isPending ? (
              <Icon icon={Loader2} size="sm" className="animate-spin" aria-hidden />
            ) : (
              <Icon icon={Download} size="sm" aria-hidden />
            )}
            <span className="ml-1">
              {requestExport.isSuccess ? (
                <FormattedMessage id="progress.exportStarted" />
              ) : (
                <FormattedMessage id="progress.export" />
              )}
            </span>
          </Button>
        </div>
      </Card>

      {/* Summary stats */}
      {summaryLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Skeleton height="h-24" />
          <Skeleton height="h-24" />
          <Skeleton height="h-24" />
          <Skeleton height="h-24" />
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <StatCard
            label={intl.formatMessage({ id: "learning.stat.activities" })}
            value={String(summary?.total_activities ?? 0)}
          />
          <StatCard
            label={intl.formatMessage({ id: "learning.stat.hours" })}
            value={String(
              Math.round((summary?.total_hours ?? 0) * 10) / 10,
            )}
          />
          <StatCard
            label={intl.formatMessage({ id: "learning.stat.books" })}
            value={String(summary?.books_completed ?? 0)}
          />
          <StatCard
            label={intl.formatMessage({ id: "learning.stat.journals" })}
            value={String(summary?.journal_entries ?? 0)}
          />
        </div>
      )}

      {/* Hours by subject */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="progress.bySubject" />
        </h2>
        {subjectsLoading ? (
          <div className="space-y-2">
            <Skeleton height="h-12" />
            <Skeleton height="h-12" />
            <Skeleton height="h-12" />
          </div>
        ) : !subjectProgress || subjectProgress.length === 0 ? (
          <EmptyState
            message={intl.formatMessage({
              id: "progress.noSubjects",
            })}
          />
        ) : (
          <div className="space-y-3">
            {subjectProgress.map((sp) => {
              const maxHours = Math.max(
                ...subjectProgress.map((s) => s.total_hours),
                1,
              );
              const pct = Math.round((sp.total_hours / maxHours) * 100);
              return (
                <Card key={sp.subject_slug} className="flex items-center gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <span className="type-title-sm text-on-surface font-medium">
                        {sp.subject_name}
                      </span>
                      <span className="type-label-sm text-on-surface-variant">
                        <FormattedMessage
                          id="progress.hours"
                          values={{
                            hours:
                              Math.round(sp.total_hours * 10) / 10,
                          }}
                        />
                      </span>
                    </div>
                    <ProgressBar value={pct} />
                    <div className="flex gap-4 mt-1 type-label-sm text-on-surface-variant">
                      <span>
                        <FormattedMessage
                          id="progress.activities"
                          values={{ count: sp.activity_count }}
                        />
                      </span>
                      <span>
                        <FormattedMessage
                          id="progress.journals"
                          values={{ count: sp.journal_count }}
                        />
                      </span>
                      <span>
                        <FormattedMessage
                          id="progress.books"
                          values={{ count: sp.books_completed }}
                        />
                      </span>
                    </div>
                  </div>
                </Card>
              );
            })}
          </div>
        )}
      </section>

      {/* Timeline */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="progress.timeline" />
        </h2>
        {timelineLoading ? (
          <div className="space-y-2">
            <Skeleton height="h-16" />
            <Skeleton height="h-16" />
            <Skeleton height="h-16" />
          </div>
        ) : timeline.length === 0 ? (
          <EmptyState
            message={intl.formatMessage({
              id: "progress.timeline.empty",
            })}
          />
        ) : (
          <>
            <ul className="space-y-2" role="list">
              {timeline.map((entry) => {
                const EntryIcon =
                  TIMELINE_ICONS[entry.entry_type] ?? ClipboardList;
                return (
                  <li key={entry.id}>
                    <Card className="flex items-start gap-3">
                      <div className="shrink-0 mt-0.5 text-primary">
                        <Icon icon={EntryIcon} size="md" aria-hidden />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="type-title-sm text-on-surface font-medium">
                          {entry.title}
                        </p>
                        {entry.description && (
                          <p className="type-body-sm text-on-surface-variant line-clamp-1">
                            {entry.description}
                          </p>
                        )}
                        <p className="type-label-sm text-on-surface-variant mt-1">
                          {new Date(entry.date).toLocaleDateString()}
                        </p>
                      </div>
                    </Card>
                  </li>
                );
              })}
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
      </section>
    </div>
  );
}
