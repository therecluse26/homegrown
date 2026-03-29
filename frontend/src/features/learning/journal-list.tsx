import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { PenTool, Calendar, Plus } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Select,
  Skeleton,
} from "@/components/ui";
import { InfiniteScroll } from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import {
  useJournalEntries,
  type JournalEntryType,
} from "@/hooks/use-journals";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

const ENTRY_TYPE_COLORS: Record<JournalEntryType, string> = {
  freeform: "bg-tertiary-fixed text-on-tertiary-fixed",
  narration: "bg-primary-container text-on-primary-container",
  reflection: "bg-secondary-container text-on-secondary-container",
};

export function JournalList() {
  const intl = useIntl();
  const { data: students, isPending: studentsLoading } = useStudents();
  const { toolLabel } = useMethodologyContext();
  const [selectedStudent, setSelectedStudent] = useState("");
  const [typeFilter, setTypeFilter] = useState<JournalEntryType | "">("");

  const effectiveStudent =
    selectedStudent || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const {
    data: pages,
    isPending,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useJournalEntries(effectiveStudent, {
    entry_type: typeFilter || undefined,
  });

  const entries = pages?.pages.flatMap((p) => p.data) ?? [];

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          {toolLabel("journaling", intl.formatMessage({ id: "journals.title" }))}
        </h1>
        <RouterLink to="/learning/journals/new" className="no-underline">
          <Button variant="primary" size="sm" disabled={!effectiveStudent}>
            <Icon icon={Plus} size="sm" aria-hidden />
            <span className="ml-1.5">
              <FormattedMessage id="journals.new" />
            </span>
          </Button>
        </RouterLink>
      </div>

      {/* Filters */}
      <Card className="bg-surface-container-low">
        <div className="flex flex-wrap items-end gap-4">
          <div className="flex-1 min-w-[180px]">
            <label
              htmlFor="journal-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journals.student" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="journal-student"
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
          <div className="min-w-[160px]">
            <label
              htmlFor="journal-type"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="journals.filter.type" />
            </label>
            <Select
              id="journal-type"
              value={typeFilter}
              onChange={(e) =>
                setTypeFilter(e.target.value as JournalEntryType | "")
              }
            >
              <option value="">
                {intl.formatMessage({ id: "journals.filter.allTypes" })}
              </option>
              <option value="freeform">
                {intl.formatMessage({ id: "journals.type.freeform" })}
              </option>
              <option value="narration">
                {intl.formatMessage({ id: "journals.type.narration" })}
              </option>
              <option value="reflection">
                {intl.formatMessage({ id: "journals.type.reflection" })}
              </option>
            </Select>
          </div>
        </div>
      </Card>

      {/* Entry list */}
      {!effectiveStudent ? (
        <EmptyState
          message={intl.formatMessage({
            id: "activityLog.selectStudentFirst",
          })}
          description={intl.formatMessage({
            id: "activityLog.selectStudentFirst.description",
          })}
        />
      ) : isPending ? (
        <div className="space-y-3">
          <Skeleton height="h-24" />
          <Skeleton height="h-24" />
          <Skeleton height="h-24" />
        </div>
      ) : entries.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "journals.empty" })}
          description={intl.formatMessage({
            id: "journals.empty.description",
          })}
          action={
            <RouterLink to="/learning/journals/new" className="no-underline">
              <Button variant="primary" size="sm">
                <FormattedMessage id="journals.new" />
              </Button>
            </RouterLink>
          }
        />
      ) : (
        <>
          <ul className="space-y-2" role="list">
            {entries.map((entry) => (
              <li key={entry.id}>
                <RouterLink
                  to={`/learning/journals/${entry.id}`}
                  className="block no-underline"
                >
                  <Card interactive className="flex items-start gap-3">
                    <div className="shrink-0 mt-0.5 text-primary">
                      <Icon icon={PenTool} size="md" aria-hidden />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <p className="type-title-sm text-on-surface font-medium">
                          {entry.title ||
                            intl.formatMessage({
                              id: "journals.untitled",
                            })}
                        </p>
                        <span
                          className={`px-2 py-0.5 type-label-sm rounded-full ${
                            ENTRY_TYPE_COLORS[entry.entry_type] ?? ""
                          }`}
                        >
                          {intl.formatMessage({
                            id: `journals.type.${entry.entry_type}`,
                          })}
                        </span>
                      </div>
                      <p className="type-body-sm text-on-surface-variant line-clamp-2">
                        {entry.content}
                      </p>
                      <div className="flex items-center gap-2 mt-2">
                        <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
                          <Icon icon={Calendar} size="xs" aria-hidden />
                          {new Date(entry.entry_date).toLocaleDateString()}
                        </span>
                        {entry.subject_tags?.map((tag) => (
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
                </RouterLink>
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
