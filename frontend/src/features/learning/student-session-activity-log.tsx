import { FormattedMessage, useIntl } from "react-intl";
import { ArrowLeft, Clock, Eye, FileText, Play } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";

// ─── Types ──────────────────────────────────────────────────────────────────

interface SessionActivity {
  id: string;
  action_type: "page_view" | "content_view" | "quiz_attempt" | "video_watch";
  resource_title: string;
  timestamp: string;
  duration_seconds?: number;
}

const ACTION_ICONS: Record<SessionActivity["action_type"], typeof Eye> = {
  page_view: Eye,
  content_view: FileText,
  quiz_attempt: Play,
  video_watch: Play,
};

// ─── Main component ──────────────────────────────────────────────────────────

export function StudentSessionActivityLog() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { sessionId } = useParams<{ sessionId: string }>();

  // Placeholder — will be backed by a real API when backend session tracking is built
  const activities: SessionActivity[] = [];
  const isPending = false;

  if (!sessionId) {
    return (
      <EmptyState
        message={intl.formatMessage({ id: "sessionLog.noSession" })}
      />
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate(-1)}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="sessionLog.title" />
        </h1>
      </div>

      {/* Activity list */}
      {isPending ? (
        <div className="space-y-2">
          <Skeleton height="h-16" />
          <Skeleton height="h-16" />
          <Skeleton height="h-16" />
        </div>
      ) : activities.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "sessionLog.empty" })}
          description={intl.formatMessage({
            id: "sessionLog.empty.description",
          })}
        />
      ) : (
        <ul className="space-y-2" role="list">
          {activities.map((activity) => {
            const ActivityIcon = ACTION_ICONS[activity.action_type];
            return (
              <li key={activity.id}>
                <Card className="flex items-start gap-3">
                  <div className="shrink-0 mt-0.5 text-primary">
                    <Icon icon={ActivityIcon} size="md" aria-hidden />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="type-title-sm text-on-surface font-medium">
                      {activity.resource_title}
                    </p>
                    <div className="flex items-center gap-3 mt-1">
                      <span className="type-label-sm text-on-surface-variant">
                        {new Date(activity.timestamp).toLocaleTimeString()}
                      </span>
                      {activity.duration_seconds && (
                        <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
                          <Icon icon={Clock} size="xs" aria-hidden />
                          <FormattedMessage
                            id="sessionLog.duration"
                            values={{
                              minutes: Math.round(
                                activity.duration_seconds / 60,
                              ),
                            }}
                          />
                        </span>
                      )}
                    </div>
                  </div>
                </Card>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
