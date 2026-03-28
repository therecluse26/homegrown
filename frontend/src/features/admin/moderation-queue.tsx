import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Shield, Check, X, AlertTriangle } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Tabs,
  Modal,
  Input,
  FormField,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useAdminModerationQueue,
  useAdminTakeModerationAction,
  useAdminAppeals,
  useAdminResolveAppeal,
} from "@/hooks/use-admin";
import type { ModerationQueueItem, AdminAppealResponse } from "@/hooks/use-admin";

// ─── Queue item card ─────────────────────────────────────────────────────────

function QueueItemCard({ item }: { item: ModerationQueueItem }) {
  const intl = useIntl();
  const takeAction = useAdminTakeModerationAction();
  const [showActionModal, setShowActionModal] = useState(false);
  const [selectedAction, setSelectedAction] = useState("");
  const [reason, setReason] = useState("");

  const handleAction = () => {
    takeAction.mutate(
      { itemId: item.id, action: selectedAction, reason: reason || undefined },
      { onSuccess: () => setShowActionModal(false) },
    );
  };

  return (
    <>
      <Card className="p-card-padding">
        <div className="flex items-start gap-3">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-2">
              <Badge variant="secondary">{item.content_type}</Badge>
              <Badge
                variant={
                  item.status === "pending" ? "default" : "primary"
                }
              >
                {item.status}
              </Badge>
              <span className="type-label-sm text-on-surface-variant">
                {new Date(item.created_at).toLocaleDateString()}
              </span>
            </div>
            <p className="type-body-sm text-on-surface mb-1">{item.reason}</p>
            <p className="type-label-sm text-on-surface-variant">
              Content ID: {item.content_id}
            </p>
          </div>
          {item.status === "pending" && (
            <div className="flex gap-1 shrink-0">
              <button
                onClick={() => {
                  setSelectedAction("approve");
                  setShowActionModal(true);
                }}
                className="p-2 rounded-radius-sm text-primary hover:bg-primary-container/30"
                aria-label="Approve"
              >
                <Icon icon={Check} size="sm" />
              </button>
              <button
                onClick={() => {
                  setSelectedAction("reject");
                  setShowActionModal(true);
                }}
                className="p-2 rounded-radius-sm text-error hover:bg-error-container/30"
                aria-label="Reject"
              >
                <Icon icon={X} size="sm" />
              </button>
              <button
                onClick={() => {
                  setSelectedAction("escalate");
                  setShowActionModal(true);
                }}
                className="p-2 rounded-radius-sm text-warning hover:bg-warning-container/30"
                aria-label="Escalate"
              >
                <Icon icon={AlertTriangle} size="sm" />
              </button>
            </div>
          )}
        </div>
      </Card>

      <Modal
        open={showActionModal}
        onClose={() => setShowActionModal(false)}
        title={`${selectedAction.charAt(0).toUpperCase() + selectedAction.slice(1)} content`}
      >
        <div className="space-y-4">
          <FormField
            label={intl.formatMessage({ id: "admin.moderation.reason" })}
          >
            {({ id }) => (
              <Input
                id={id}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Optional reason"
              />
            )}
          </FormField>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowActionModal(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleAction}
              disabled={takeAction.isPending}
            >
              {selectedAction === "approve" ? "Approve" : selectedAction === "reject" ? "Reject" : "Escalate"}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
}

// ─── Appeal card ─────────────────────────────────────────────────────────────

function AppealCard({ appeal }: { appeal: AdminAppealResponse }) {
  const intl = useIntl();
  const resolveAppeal = useAdminResolveAppeal();
  const [showResolveModal, setShowResolveModal] = useState(false);
  const [resolution, setResolution] = useState("");
  const [resolveStatus, setResolveStatus] = useState("denied");

  const handleResolve = () => {
    if (!resolution.trim()) return;
    resolveAppeal.mutate(
      {
        appealId: appeal.id,
        status: resolveStatus,
        resolution_text: resolution,
      },
      { onSuccess: () => setShowResolveModal(false) },
    );
  };

  return (
    <>
      <Card className="p-card-padding">
        <div className="flex items-start gap-3">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-2">
              <Badge variant="secondary">Appeal</Badge>
              <Badge
                variant={appeal.status === "pending" ? "default" : "primary"}
              >
                {appeal.status}
              </Badge>
              <span className="type-label-sm text-on-surface-variant">
                {new Date(appeal.created_at).toLocaleDateString()}
              </span>
            </div>
            <p className="type-body-sm text-on-surface mb-2">
              {appeal.appeal_text}
            </p>
            <div className="bg-surface-container-low rounded-radius-sm p-2">
              <p className="type-label-sm text-on-surface-variant mb-1">
                Original action:
              </p>
              <p className="type-body-sm text-on-surface">
                {appeal.original_action.action_type} — {appeal.original_action.reason}
              </p>
            </div>
          </div>
          {appeal.status === "pending" && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowResolveModal(true)}
            >
              <FormattedMessage id="admin.moderation.resolve" />
            </Button>
          )}
        </div>
      </Card>

      <Modal
        open={showResolveModal}
        onClose={() => setShowResolveModal(false)}
        title={intl.formatMessage({ id: "admin.moderation.resolveAppeal" })}
      >
        <div className="space-y-4">
          <FormField label="Decision">
            {({ id }) => (
              <select
                id={id}
                value={resolveStatus}
                onChange={(e) => setResolveStatus(e.target.value)}
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md"
              >
                <option value="denied">Deny appeal</option>
                <option value="approved">Grant appeal</option>
              </select>
            )}
          </FormField>
          <FormField
            label={intl.formatMessage({
              id: "admin.moderation.resolutionText",
            })}
            required
          >
            {({ id }) => (
              <textarea
                id={id}
                value={resolution}
                onChange={(e) => setResolution(e.target.value)}
                required
                className="w-full min-h-[80px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowResolveModal(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleResolve}
              disabled={!resolution.trim() || resolveAppeal.isPending}
            >
              <FormattedMessage id="admin.moderation.resolve" />
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
}

// ─── Moderation queue page ───────────────────────────────────────────────────

export function ModerationQueue() {
  const intl = useIntl();
  const { data: queue, isPending: queuePending } =
    useAdminModerationQueue("pending");
  const { data: appeals, isPending: appealsPending } =
    useAdminAppeals("pending");

  const queueItems = queue?.data ?? [];
  const appealItems = appeals?.data ?? [];

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "admin.moderation.title" })}
      />

      <Tabs
        defaultTab="queue"
        tabs={[
          {
            id: "queue",
            label: `${intl.formatMessage({ id: "admin.moderation.queue" })} (${queueItems.length})`,
            content: (
              <div className="mt-6 space-y-3">
                {queuePending && (
                  <div className="space-y-3">
                    {[1, 2, 3].map((n) => (
                      <Skeleton
                        key={n}
                        className="h-24 w-full rounded-radius-md"
                      />
                    ))}
                  </div>
                )}
                {!queuePending && queueItems.length === 0 && (
                  <EmptyState
                    illustration={<Icon icon={Shield} size="xl" />}
                    message={intl.formatMessage({
                      id: "admin.moderation.emptyQueue",
                    })}
                    description={intl.formatMessage({
                      id: "admin.moderation.emptyQueueDescription",
                    })}
                  />
                )}
                {queueItems.map((item) => (
                  <QueueItemCard key={item.id} item={item} />
                ))}
              </div>
            ),
          },
          {
            id: "appeals",
            label: `${intl.formatMessage({ id: "admin.moderation.appeals" })} (${appealItems.length})`,
            content: (
              <div className="mt-6 space-y-3">
                {appealsPending && (
                  <div className="space-y-3">
                    {[1, 2].map((n) => (
                      <Skeleton
                        key={n}
                        className="h-32 w-full rounded-radius-md"
                      />
                    ))}
                  </div>
                )}
                {!appealsPending && appealItems.length === 0 && (
                  <EmptyState
                    illustration={<Icon icon={Shield} size="xl" />}
                    message={intl.formatMessage({
                      id: "admin.moderation.emptyAppeals",
                    })}
                    description={intl.formatMessage({
                      id: "admin.moderation.emptyAppealsDescription",
                    })}
                  />
                )}
                {appealItems.map((appeal) => (
                  <AppealCard key={appeal.id} appeal={appeal} />
                ))}
              </div>
            ),
          },
        ]}
      />
    </div>
  );
}
