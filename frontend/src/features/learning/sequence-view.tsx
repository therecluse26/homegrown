import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  CheckCircle,
  Circle,
  Lock,
  Play,
  SkipForward,
  Unlock,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  ProgressBar,
  Skeleton,
} from "@/components/ui";
import { ConfirmationDialog } from "@/components/ui";
import { useState } from "react";
import { useStudents } from "@/hooks/use-family";
import {
  useSequenceDef,
  useSequenceProgress,
  useUpdateSequenceProgress,
  type SequenceItemResponse,
} from "@/hooks/use-sequences";

// ─── Item status helpers ─────────────────────────────────────────────────────

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

const STATUS_ICONS: Record<ItemStatus, typeof Circle> = {
  completed: CheckCircle,
  current: Play,
  locked: Lock,
  available: Circle,
};

const STATUS_COLORS: Record<ItemStatus, string> = {
  completed: "text-primary",
  current: "text-primary bg-primary-container",
  locked: "text-on-surface-variant opacity-50",
  available: "text-on-surface-variant",
};

// ─── Content type label ──────────────────────────────────────────────────────

function contentTypeRoute(type: string, id: string): string {
  switch (type) {
    case "quiz":
      return `/learning/quiz/${id}`;
    case "video":
      return `/learning/video/${id}`;
    case "reading":
      return `/learning/read/${id}`;
    default:
      return `/learning/read/${id}`;
  }
}

// ─── Main component ──────────────────────────────────────────────────────────

export function SequenceView() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { progressId } = useParams<{ progressId: string }>();
  const { data: students } = useStudents();

  const studentId = students?.[0]?.id ?? "";

  const { data: progress, isPending: progressLoading } = useSequenceProgress(
    studentId,
    progressId ?? "",
  );
  const { data: sequenceDef, isPending: defLoading } = useSequenceDef(
    progress?.sequence_def_id ?? "",
  );
  const updateProgress = useUpdateSequenceProgress(studentId);

  const [confirmSkip, setConfirmSkip] = useState<string | null>(null);
  const [confirmUnlock, setConfirmUnlock] = useState<string | null>(null);

  const items = sequenceDef?.items ?? [];
  const completions = (progress?.item_completions ?? {}) as Record<
    string,
    unknown
  >;
  const completedCount = items.filter((item) => completions[item.id]).length;
  const progressPct =
    items.length > 0 ? Math.round((completedCount / items.length) * 100) : 0;

  function handleComplete(itemId: string) {
    if (!progressId) return;
    updateProgress.mutate({
      progressId,
      complete_item_id: itemId,
    });
  }

  function handleSkip(itemId: string) {
    if (!progressId) return;
    updateProgress.mutate({
      progressId,
      skip_item_id: itemId,
    });
    setConfirmSkip(null);
  }

  function handleUnlock(itemId: string) {
    if (!progressId) return;
    updateProgress.mutate({
      progressId,
      unlock_item_id: itemId,
    });
    setConfirmUnlock(null);
  }

  if (!progressId) {
    return (
      <EmptyState
        message={intl.formatMessage({ id: "sequence.notFound" })}
      />
    );
  }

  if (progressLoading || defLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-20" />
        <Skeleton height="h-20" />
        <Skeleton height="h-20" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/learning")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {sequenceDef?.title ?? ""}
        </h1>
      </div>

      {/* Progress summary */}
      <Card className="bg-surface-container-low">
        <div className="flex items-center justify-between mb-2">
          <span className="type-label-md text-on-surface-variant">
            <FormattedMessage id="sequence.progress" />
          </span>
          <span className="type-label-sm text-on-surface-variant">
            <FormattedMessage
              id="sequence.progressCount"
              values={{ completed: completedCount, total: items.length }}
            />
          </span>
        </div>
        <ProgressBar value={progressPct} />
        {progress?.status === "completed" && (
          <p className="mt-2 type-label-sm text-primary font-medium">
            <FormattedMessage id="sequence.completed" />
          </p>
        )}
      </Card>

      {/* Sequence items */}
      {items.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "sequence.empty" })}
        />
      ) : (
        <div className="space-y-2">
          {items.map((item) => {
            const status = getItemStatus(
              item,
              progress?.current_item_index ?? 0,
              completions,
              sequenceDef?.is_linear ?? false,
            );
            const StatusIcon = STATUS_ICONS[status];
            const isLocked = status === "locked";
            const isCompleted = status === "completed";

            return (
              <Card
                key={item.id}
                className={`flex items-center gap-3 ${
                  isLocked ? "opacity-60" : ""
                }`}
              >
                {/* Status icon */}
                <div
                  className={`shrink-0 w-10 h-10 rounded-full flex items-center justify-center ${STATUS_COLORS[status]}`}
                >
                  <Icon icon={StatusIcon} size="md" aria-hidden />
                </div>

                {/* Item info */}
                <div className="flex-1 min-w-0">
                  <p className="type-title-sm text-on-surface font-medium">
                    <FormattedMessage
                      id="sequence.itemTitle"
                      values={{
                        index: item.sort_order + 1,
                        type: item.content_type,
                      }}
                    />
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {item.content_type}
                    {item.is_required && (
                      <span className="ml-2">
                        <FormattedMessage id="sequence.required" />
                      </span>
                    )}
                  </p>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 shrink-0">
                  {/* Go to content */}
                  {(status === "current" || status === "available") && (
                    <RouterLink
                      to={contentTypeRoute(item.content_type, item.content_id)}
                      className="no-underline"
                    >
                      <Button variant="primary" size="sm">
                        <Icon icon={Play} size="sm" aria-hidden />
                        <span className="ml-1">
                          <FormattedMessage id="sequence.start" />
                        </span>
                      </Button>
                    </RouterLink>
                  )}

                  {/* Mark complete (parent) */}
                  {(status === "current" || status === "available") && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      onClick={() => handleComplete(item.id)}
                      loading={updateProgress.isPending}
                    >
                      <Icon icon={CheckCircle} size="sm" aria-hidden />
                    </Button>
                  )}

                  {/* Skip (parent override) */}
                  {!isCompleted && !isLocked && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      onClick={() => setConfirmSkip(item.id)}
                    >
                      <Icon icon={SkipForward} size="sm" aria-hidden />
                    </Button>
                  )}

                  {/* Unlock (parent override) */}
                  {isLocked && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      onClick={() => setConfirmUnlock(item.id)}
                    >
                      <Icon icon={Unlock} size="sm" aria-hidden />
                    </Button>
                  )}

                  {/* Completed indicator */}
                  {isCompleted && (
                    <span className="type-label-sm text-primary">
                      <Icon icon={CheckCircle} size="md" aria-hidden />
                    </span>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}

      {/* Description */}
      {sequenceDef?.description && (
        <Card>
          <p className="type-body-md text-on-surface-variant">
            {sequenceDef.description}
          </p>
        </Card>
      )}

      {/* Confirmation dialogs */}
      {confirmSkip && (
        <ConfirmationDialog
          open
          title={intl.formatMessage({ id: "sequence.skip.title" })}
          confirmLabel={intl.formatMessage({ id: "sequence.skip.confirm" })}
          onConfirm={() => handleSkip(confirmSkip)}
          onClose={() => setConfirmSkip(null)}
        >
          {intl.formatMessage({ id: "sequence.skip.message" })}
        </ConfirmationDialog>
      )}

      {confirmUnlock && (
        <ConfirmationDialog
          open
          title={intl.formatMessage({ id: "sequence.unlock.title" })}
          confirmLabel={intl.formatMessage({ id: "sequence.unlock.confirm" })}
          onConfirm={() => handleUnlock(confirmUnlock)}
          onClose={() => setConfirmUnlock(null)}
        >
          {intl.formatMessage({ id: "sequence.unlock.message" })}
        </ConfirmationDialog>
      )}
    </div>
  );
}
