import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  BookOpen,
  ClipboardList,
  Play,
  GraduationCap,
  Star,
} from "lucide-react";
import {
  Card,
  EmptyState,
  Icon,
  Skeleton,
  StatCard,
} from "@/components/ui";
import { useStudentSession } from "@/hooks/use-student-session";
import { useStudentProgress } from "@/hooks/use-progress";

// ─── Assignment card ─────────────────────────────────────────────────────────

interface AssignmentItem {
  id: string;
  title: string;
  content_type: string;
  content_id: string;
  is_new: boolean;
}

function AssignmentCard({ item }: { item: AssignmentItem }) {
  const route =
    item.content_type === "quiz"
      ? `/learning/quiz/${item.content_id}`
      : item.content_type === "video"
        ? `/learning/video/${item.content_id}`
        : `/learning/read/${item.content_id}`;

  return (
    <RouterLink to={route} className="block no-underline">
      <Card interactive className="flex items-center gap-3">
        <div className="shrink-0 text-primary">
          <Icon icon={Play} size="md" aria-hidden />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="type-title-sm text-on-surface font-medium">
              {item.title}
            </p>
            {item.is_new && (
              <span className="px-2 py-0.5 bg-primary-container text-on-primary-container type-label-sm rounded-full">
                <FormattedMessage id="student.new" />
              </span>
            )}
          </div>
          <p className="type-label-sm text-on-surface-variant">
            {item.content_type}
          </p>
        </div>
      </Card>
    </RouterLink>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function StudentDashboard() {
  const intl = useIntl();
  const { session } = useStudentSession();
  const studentId = session?.studentId ?? "";

  const { data: summary, isPending: summaryLoading } = useStudentProgress(
    studentId,
  );

  // Placeholder: assignments would come from a dedicated hook
  const assignments: AssignmentItem[] = [];

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Welcome */}
      <div className="flex items-center gap-3">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage
            id="student.welcome"
            values={{ name: session?.studentName ?? "" }}
          />
        </h1>
      </div>

      {/* Quick stats */}
      {summaryLoading ? (
        <div className="grid grid-cols-2 gap-3">
          <Skeleton height="h-24" />
          <Skeleton height="h-24" />
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          <StatCard
            label={intl.formatMessage({ id: "student.stat.activities" })}
            value={String(summary?.total_activities ?? 0)}
          />
          <StatCard
            label={intl.formatMessage({ id: "student.stat.books" })}
            value={String(summary?.books_completed ?? 0)}
          />
        </div>
      )}

      {/* Assignments */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="student.assignments" />
        </h2>
        {assignments.length === 0 ? (
          <EmptyState
            illustration={
              <Icon
                icon={Star}
                size="xl"
                className="text-on-surface-variant"
                aria-hidden
              />
            }
            message={intl.formatMessage({ id: "student.assignments.empty" })}
            description={intl.formatMessage({
              id: "student.assignments.empty.description",
            })}
          />
        ) : (
          <div className="space-y-2">
            {assignments.map((item) => (
              <AssignmentCard key={item.id} item={item} />
            ))}
          </div>
        )}
      </section>

      {/* Quick actions */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="student.explore" />
        </h2>
        <div className="grid grid-cols-3 gap-3">
          <RouterLink to="/learning/reading-lists" className="no-underline">
            <Card
              interactive
              className="flex flex-col items-center gap-2 py-6"
            >
              <Icon
                icon={BookOpen}
                size="lg"
                className="text-primary"
                aria-hidden
              />
              <span className="type-label-md text-on-surface">
                <FormattedMessage id="student.action.reading" />
              </span>
            </Card>
          </RouterLink>
          <RouterLink to="/learning/journals" className="no-underline">
            <Card
              interactive
              className="flex flex-col items-center gap-2 py-6"
            >
              <Icon
                icon={ClipboardList}
                size="lg"
                className="text-primary"
                aria-hidden
              />
              <span className="type-label-md text-on-surface">
                <FormattedMessage id="student.action.journal" />
              </span>
            </Card>
          </RouterLink>
          <RouterLink to="/learning/grades" className="no-underline">
            <Card
              interactive
              className="flex flex-col items-center gap-2 py-6"
            >
              <Icon
                icon={GraduationCap}
                size="lg"
                className="text-primary"
                aria-hidden
              />
              <span className="type-label-md text-on-surface">
                <FormattedMessage id="student.action.grades" />
              </span>
            </Card>
          </RouterLink>
        </div>
      </section>
    </div>
  );
}
