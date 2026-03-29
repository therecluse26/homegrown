import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  BookOpen,
  ClipboardList,
  PenTool,
  BookMarked,
  BarChart3,
  GraduationCap,
  Plus,
  Flame,
  Trophy,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  StatCard,
} from "@/components/ui";
import { TierGate } from "@/components/common/tier-gate";
import { ParentEducationPanel } from "@/components/common/parent-education-panel";
import { useStudents } from "@/hooks/use-family";
import { useStudentProgress, useStreak } from "@/hooks/use-progress";
import { useAuth } from "@/hooks/use-auth";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

// ─── Streak milestone thresholds ─────────────────────────────────────────────

const MILESTONES = [7, 14, 30, 60, 100] as const;

function StreakBadge({ studentId }: { studentId: string }) {
  const intl = useIntl();
  const { data: streak } = useStreak(studentId);

  if (!streak || streak.current_streak === 0) return null;

  // Find the highest milestone reached
  const highestMilestone = MILESTONES.filter(
    (m) => streak.current_streak >= m,
  ).at(-1);

  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-tertiary-container text-on-tertiary-container">
        <Icon icon={Flame} size="sm" aria-hidden />
        <span className="type-label-md font-semibold">
          <FormattedMessage
            id="learning.streak.days"
            values={{ count: streak.current_streak }}
          />
        </span>
      </div>
      {highestMilestone && (
        <div
          className="flex items-center gap-1 px-2 py-1 rounded-full bg-primary-container text-on-primary-container"
          title={intl.formatMessage(
            { id: "learning.streak.milestone" },
            { days: highestMilestone },
          )}
        >
          <Icon icon={Trophy} size="xs" aria-hidden />
          <span className="type-label-sm">{highestMilestone}</span>
        </div>
      )}
    </div>
  );
}

// ─── Student progress card ──────────────────────────────────────────────────

function StudentProgressCard({
  studentId,
  studentName,
}: {
  studentId: string;
  studentName: string;
}) {
  const intl = useIntl();
  const { data: progress, isPending } = useStudentProgress(studentId);

  if (isPending) {
    return (
      <Card className="space-y-3">
        <Skeleton height="h-6" width="w-32" />
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h3 className="type-title-md text-on-surface font-semibold">
            {studentName}
          </h3>
          <StreakBadge studentId={studentId} />
        </div>
        <RouterLink
          to={`/learning/progress/${studentId}`}
          className="type-label-md text-primary hover:text-primary-container transition-colors"
        >
          <FormattedMessage id="learning.viewProgress" />
        </RouterLink>
      </div>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatCard
          label={intl.formatMessage({ id: "learning.stat.activities" })}
          value={String(progress?.total_activities ?? 0)}
        />
        <StatCard
          label={intl.formatMessage({ id: "learning.stat.hours" })}
          value={String(Math.round((progress?.total_hours ?? 0) * 10) / 10)}
        />
        <StatCard
          label={intl.formatMessage({ id: "learning.stat.books" })}
          value={String(progress?.books_completed ?? 0)}
        />
        <StatCard
          label={intl.formatMessage({ id: "learning.stat.journals" })}
          value={String(progress?.journal_entries ?? 0)}
        />
      </div>
    </Card>
  );
}

// ─── Quick action card ──────────────────────────────────────────────────────

function QuickAction({
  icon,
  label,
  to,
}: {
  icon: typeof BookOpen;
  label: string;
  to: string;
}) {
  return (
    <RouterLink
      to={to}
      className="flex flex-col items-center gap-2 p-4 rounded-xl bg-surface-container-lowest hover:bg-surface-container-low transition-colors text-center no-underline group focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
    >
      <div className="p-3 rounded-full bg-primary-container text-on-primary-container group-hover:bg-primary group-hover:text-on-primary transition-colors">
        <Icon icon={icon} size="lg" aria-hidden />
      </div>
      <span className="type-label-md text-on-surface">{label}</span>
    </RouterLink>
  );
}

// ─── Main dashboard ─────────────────────────────────────────────────────────

export function LearningDashboard() {
  const intl = useIntl();
  const { tier } = useAuth();
  const { data: students, isPending: studentsLoading } = useStudents();
  const { toolLabel } = useMethodologyContext();

  return (
    <div className="mx-auto max-w-content-narrow space-y-8">
      {/* Page heading */}
      <div className="flex items-center justify-between">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="learning.title" />
        </h1>
      </div>

      {/* Quick actions grid */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-4">
          <FormattedMessage id="learning.quickActions" />
        </h2>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          <QuickAction
            icon={ClipboardList}
            label={toolLabel("activities", intl.formatMessage({ id: "learning.action.logActivity" }))}
            to="/learning/activities"
          />
          <QuickAction
            icon={PenTool}
            label={toolLabel("journaling", intl.formatMessage({ id: "learning.action.journal" }))}
            to="/learning/journals"
          />
          <QuickAction
            icon={BookMarked}
            label={toolLabel("reading-lists", intl.formatMessage({ id: "learning.action.reading" }))}
            to="/learning/reading-lists"
          />
          <QuickAction
            icon={BarChart3}
            label={toolLabel("progress-tracking", intl.formatMessage({ id: "learning.action.progress" }))}
            to="/learning/progress/select"
          />
          <QuickAction
            icon={GraduationCap}
            label={toolLabel("tests-grades", intl.formatMessage({ id: "learning.action.grades" }))}
            to="/learning/grades"
          />
          <QuickAction
            icon={Plus}
            label={intl.formatMessage({ id: "learning.action.addNew" })}
            to="/learning/activities?new=1"
          />
        </div>
      </section>

      {/* Methodology guidance */}
      <ParentEducationPanel
        toolName={intl.formatMessage({ id: "learning.title" })}
        guidance={intl.formatMessage({ id: "learning.guidance" })}
      />

      {/* Per-student progress overview */}
      <section>
        <h2 className="type-title-md text-on-surface font-semibold mb-4">
          <FormattedMessage id="learning.studentProgress" />
        </h2>
        {studentsLoading ? (
          <div className="space-y-4">
            <Skeleton height="h-32" />
            <Skeleton height="h-32" />
          </div>
        ) : !students || students.length === 0 ? (
          <EmptyState
            message={intl.formatMessage({
              id: "learning.noStudents",
            })}
            description={intl.formatMessage({
              id: "learning.noStudents.description",
            })}
            action={
              <RouterLink to="/settings">
                <Button variant="primary" size="sm">
                  <FormattedMessage id="learning.addStudent" />
                </Button>
              </RouterLink>
            }
          />
        ) : (
          <div className="space-y-4">
            {students.map((student) => (
              <StudentProgressCard
                key={student.id}
                studentId={student.id ?? ""}
                studentName={student.display_name ?? ""}
              />
            ))}
          </div>
        )}
      </section>

      {/* Premium features gate */}
      {tier === "free" && (
        <TierGate featureName="Advanced Learning Analytics" />
      )}
    </div>
  );
}
