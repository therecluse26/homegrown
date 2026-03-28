import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  CheckCircle,
  Circle,
  Lock,
  Play,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  ProgressBar,
  Skeleton,
} from "@/components/ui";
import { useStudentSession } from "@/hooks/use-student-session";
import {
  useSequenceDef,
  useSequenceProgress,
  type SequenceItemResponse,
} from "@/hooks/use-sequences";

// ─── Item status ─────────────────────────────────────────────────────────────

type ItemStatus = "completed" | "current" | "locked" | "available";

function getItemStatus(
  item: SequenceItemResponse,
  currentIndex: number,
  completions: Record<string, unknown>,
  isLinear: boolean,
): ItemStatus {
  if (completions[item.id]) return "completed";
  if (item.sort_order === currentIndex) return "current";
  if (isLinear && item.unlock_after_previous && item.sort_order > currentIndex)
    return "locked";
  return "available";
}

function contentRoute(type: string, id: string): string {
  if (type === "quiz") return `/student/quiz/${id}`;
  if (type === "video") return `/student/video/${id}`;
  return `/student/read/${id}`;
}

// ─── Main component ──────────────────────────────────────────────────────────

export function StudentSequence() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { progressId } = useParams<{ progressId: string }>();
  const { session: studentSession } = useStudentSession();
  const studentId = studentSession?.studentId ?? "";

  const { data: progress, isPending: progressLoading } = useSequenceProgress(
    studentId,
    progressId ?? "",
  );
  const { data: sequenceDef, isPending: defLoading } = useSequenceDef(
    progress?.sequence_def_id ?? "",
  );

  const items = sequenceDef?.items ?? [];
  const completions = (progress?.item_completions ?? {}) as Record<string, unknown>;
  const completedCount = items.filter((i) => completions[i.id]).length;
  const pct = items.length > 0 ? Math.round((completedCount / items.length) * 100) : 0;

  if (!progressId) {
    return <EmptyState message={intl.formatMessage({ id: "sequence.notFound" })} />;
  }

  if (progressLoading || defLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-20" />
        <Skeleton height="h-20" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => void navigate(-1)}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {sequenceDef?.title ?? ""}
        </h1>
      </div>

      {/* Progress bar */}
      <Card className="bg-surface-container-low">
        <div className="flex items-center justify-between mb-2">
          <span className="type-label-md text-on-surface-variant">
            <FormattedMessage id="sequence.progress" />
          </span>
          <span className="type-label-sm text-on-surface-variant">
            {completedCount}/{items.length}
          </span>
        </div>
        <ProgressBar value={pct} />
        {progress?.status === "completed" && (
          <p className="mt-2 type-label-sm text-primary font-medium">
            <FormattedMessage id="sequence.completed" />
          </p>
        )}
      </Card>

      {/* Items */}
      {items.length === 0 ? (
        <EmptyState message={intl.formatMessage({ id: "sequence.empty" })} />
      ) : (
        <div className="space-y-2">
          {items.map((item) => {
            const status = getItemStatus(
              item,
              progress?.current_item_index ?? 0,
              completions,
              sequenceDef?.is_linear ?? false,
            );
            const isLocked = status === "locked";
            const isCompleted = status === "completed";

            const iconMap = {
              completed: CheckCircle,
              current: Play,
              locked: Lock,
              available: Circle,
            };
            const StatusIcon = iconMap[status];

            return (
              <Card
                key={item.id}
                className={`flex items-center gap-3 ${isLocked ? "opacity-50" : ""}`}
              >
                <div
                  className={`shrink-0 w-10 h-10 rounded-full flex items-center justify-center ${
                    isCompleted
                      ? "text-primary"
                      : status === "current"
                        ? "text-primary bg-primary-container"
                        : "text-on-surface-variant"
                  }`}
                >
                  <Icon icon={StatusIcon} size="md" aria-hidden />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="type-title-sm text-on-surface font-medium">
                    <FormattedMessage
                      id="sequence.itemTitle"
                      values={{ index: item.sort_order + 1, type: item.content_type }}
                    />
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {item.content_type}
                  </p>
                </div>
                {(status === "current" || status === "available") && (
                  <RouterLink
                    to={contentRoute(item.content_type, item.content_id)}
                    className="no-underline shrink-0"
                  >
                    <Button variant="primary" size="sm">
                      <Icon icon={Play} size="sm" aria-hidden />
                      <span className="ml-1">
                        <FormattedMessage id="sequence.start" />
                      </span>
                    </Button>
                  </RouterLink>
                )}
                {isCompleted && (
                  <Icon icon={CheckCircle} size="md" className="text-primary shrink-0" aria-hidden />
                )}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
