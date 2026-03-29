import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { BookMarked, Plus, Check, BookOpen } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  ProgressBar,
  Select,
  Skeleton,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import {
  useReadingLists,
  useReadingProgress,
  useCreateReadingList,
  type ReadingStatus,
} from "@/hooks/use-reading";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Status badge ───────────────────────────────────────────────────────────

const STATUS_CONFIG: Record<
  ReadingStatus,
  { icon: typeof BookOpen; colorClass: string; labelId: string }
> = {
  to_read: {
    icon: BookMarked,
    colorClass: "bg-surface-container-high text-on-surface-variant",
    labelId: "reading.status.toRead",
  },
  in_progress: {
    icon: BookOpen,
    colorClass: "bg-primary-container text-on-primary-container",
    labelId: "reading.status.inProgress",
  },
  completed: {
    icon: Check,
    colorClass: "bg-tertiary-fixed text-on-tertiary-fixed",
    labelId: "reading.status.completed",
  },
};

function StatusBadge({ status }: { status: ReadingStatus }) {
  const intl = useIntl();
  const config = STATUS_CONFIG[status];
  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 type-label-sm rounded-full ${config.colorClass}`}
    >
      <Icon icon={config.icon} size="xs" aria-hidden />
      {intl.formatMessage({ id: config.labelId })}
    </span>
  );
}

// ─── Main page ──────────────────────────────────────────────────────────────

export function ReadingLists() {
  const intl = useIntl();
  const { data: students, isPending: studentsLoading } = useStudents();
  const { toolLabel } = useMethodologyContext();
  const { data: lists, isPending: listsLoading } = useReadingLists();
  const [selectedStudent, setSelectedStudent] = useState("");
  const [showNewList, setShowNewList] = useState(false);
  const [newListName, setNewListName] = useState("");

  const effectiveStudent =
    selectedStudent || (students?.length === 1 ? (students[0]?.id ?? "") : "");

  const createList = useCreateReadingList();
  const {
    data: progressPages,
    isPending: progressLoading,
  } = useReadingProgress(effectiveStudent);

  const progressItems = progressPages?.pages.flatMap((p) => p.data) ?? [];

  function handleCreateList(e: React.FormEvent) {
    e.preventDefault();
    if (!newListName.trim()) return;
    createList.mutate(
      {
        name: newListName.trim(),
        student_id: effectiveStudent || undefined,
      },
      {
        onSuccess: () => {
          setNewListName("");
          setShowNewList(false);
        },
      },
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          {toolLabel("reading-lists", intl.formatMessage({ id: "reading.title" }))}
        </h1>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowNewList(true)}
        >
          <Icon icon={Plus} size="sm" aria-hidden />
          <span className="ml-1.5">
            <FormattedMessage id="reading.newList" />
          </span>
        </Button>
      </div>

      {/* Student selector */}
      <Card className="bg-surface-container-low">
        <div className="flex-1 min-w-[180px]">
          <label
            htmlFor="reading-student"
            className="block type-label-md text-on-surface-variant mb-1.5"
          >
            <FormattedMessage id="reading.student" />
          </label>
          {studentsLoading ? (
            <Skeleton height="h-11" />
          ) : (
            <Select
              id="reading-student"
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
      </Card>

      {/* New list form */}
      {showNewList && (
        <Card className="bg-surface-container-low">
          <h3 className="type-title-sm text-on-surface font-semibold mb-3">
            <FormattedMessage id="reading.newList.title" />
          </h3>
          <form onSubmit={handleCreateList} className="flex gap-2">
            <Input
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              placeholder={intl.formatMessage({
                id: "reading.newList.placeholder",
              })}
              className="flex-1"
              required
            />
            <Button
              variant="primary"
              size="sm"
              type="submit"
              loading={createList.isPending}
              disabled={!newListName.trim()}
            >
              <FormattedMessage id="reading.newList.create" />
            </Button>
            <Button
              variant="tertiary"
              size="sm"
              type="button"
              onClick={() => {
                setShowNewList(false);
                setNewListName("");
              }}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
          </form>
        </Card>
      )}

      {/* Reading lists */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="reading.lists" />
        </h2>
        {listsLoading ? (
          <div className="space-y-3">
            <Skeleton height="h-20" />
            <Skeleton height="h-20" />
          </div>
        ) : !lists || lists.length === 0 ? (
          <EmptyState
            message={intl.formatMessage({ id: "reading.lists.empty" })}
            description={intl.formatMessage({
              id: "reading.lists.empty.description",
            })}
          />
        ) : (
          <div className="space-y-2">
            {lists.map((list) => {
              const pct =
                list.item_count > 0
                  ? Math.round(
                      (list.completed_count / list.item_count) * 100,
                    )
                  : 0;
              return (
                <Card key={list.id} interactive>
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="type-title-sm text-on-surface font-medium">
                      {list.name}
                    </h3>
                    <span className="type-label-sm text-on-surface-variant">
                      {list.completed_count}/{list.item_count}
                    </span>
                  </div>
                  {list.description && (
                    <p className="type-body-sm text-on-surface-variant mb-2">
                      {list.description}
                    </p>
                  )}
                  <ProgressBar value={pct} />
                </Card>
              );
            })}
          </div>
        )}
      </section>

      {/* Current reading */}
      {effectiveStudent && (
        <section>
          <h2 className="type-title-md text-on-surface font-semibold mb-3">
            <FormattedMessage id="reading.currentReading" />
          </h2>
          {progressLoading ? (
            <div className="space-y-3">
              <Skeleton height="h-20" />
              <Skeleton height="h-20" />
            </div>
          ) : progressItems.length === 0 ? (
            <EmptyState
              message={intl.formatMessage({
                id: "reading.progress.empty",
              })}
              description={intl.formatMessage({
                id: "reading.progress.empty.description",
              })}
            />
          ) : (
            <div className="space-y-2">
              {progressItems.map((item) => (
                <Card key={item.id} className="flex items-start gap-3">
                  {item.reading_item.cover_image_url ? (
                    <img
                      src={item.reading_item.cover_image_url}
                      alt=""
                      className="w-12 h-16 rounded-lg object-cover shrink-0"
                    />
                  ) : (
                    <div className="w-12 h-16 rounded-lg bg-surface-container-high flex items-center justify-center shrink-0">
                      <Icon
                        icon={BookMarked}
                        size="md"
                        className="text-on-surface-variant"
                        aria-hidden
                      />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="type-title-sm text-on-surface font-medium">
                      {item.reading_item.title}
                    </p>
                    {item.reading_item.author && (
                      <p className="type-body-sm text-on-surface-variant">
                        {item.reading_item.author}
                      </p>
                    )}
                    <div className="mt-1.5">
                      <StatusBadge status={item.status} />
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </section>
      )}
    </div>
  );
}
