import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Shield, Send } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Modal,
  FormField,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useMyAppeals, useSubmitAppeal, useAccountStatus } from "@/hooks/use-admin";
import type { AppealResponse } from "@/hooks/use-admin";

function AppealCard({ appeal }: { appeal: AppealResponse }) {
  const statusColor =
    appeal.status === "approved"
      ? "primary"
      : appeal.status === "denied"
        ? "secondary"
        : "default";

  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-2 mb-2">
        <Badge variant={statusColor}>{appeal.status}</Badge>
        <span className="type-label-sm text-on-surface-variant">
          {new Date(appeal.created_at).toLocaleDateString()}
        </span>
      </div>
      <p className="type-body-sm text-on-surface mb-2">{appeal.appeal_text}</p>
      {appeal.resolution_text && (
        <div className="bg-surface-container-low rounded-radius-sm p-3 mt-2">
          <p className="type-label-sm text-on-surface-variant mb-1">
            <FormattedMessage id="settings.appeals.resolution" />
          </p>
          <p className="type-body-sm text-on-surface">
            {appeal.resolution_text}
          </p>
          {appeal.resolved_at && (
            <p className="type-label-sm text-on-surface-variant mt-1">
              {new Date(appeal.resolved_at).toLocaleDateString()}
            </p>
          )}
        </div>
      )}
    </Card>
  );
}

export function ModerationAppeals() {
  const intl = useIntl();
  const { data: appeals, isPending: appealsPending } = useMyAppeals();
  const { data: accountStatus } = useAccountStatus();
  const submitAppeal = useSubmitAppeal();

  const [showSubmitModal, setShowSubmitModal] = useState(false);
  const [actionId, setActionId] = useState("");
  const [appealText, setAppealText] = useState("");

  const handleSubmit = useCallback(() => {
    if (!actionId || !appealText.trim()) return;
    submitAppeal.mutate(
      { action_id: actionId, appeal_text: appealText },
      {
        onSuccess: () => {
          setShowSubmitModal(false);
          setActionId("");
          setAppealText("");
        },
      },
    );
  }, [actionId, appealText, submitAppeal]);

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "settings.appeals.title" })}
      />

      {/* Account status */}
      {accountStatus && accountStatus.status !== "active" && (
        <Card className="p-card-padding mb-6 border-l-4 border-warning">
          <div className="flex items-center gap-2 mb-2">
            <Icon icon={Shield} size="md" className="text-warning" />
            <h2 className="type-title-md text-on-surface">
              <FormattedMessage id="settings.appeals.accountStatus" />
            </h2>
            <Badge variant="secondary">{accountStatus.status}</Badge>
          </div>
          {accountStatus.suspension_reason && (
            <p className="type-body-sm text-on-surface-variant">
              {accountStatus.suspension_reason}
            </p>
          )}
          {accountStatus.suspension_expires_at && (
            <p className="type-label-sm text-on-surface-variant mt-1">
              <FormattedMessage id="settings.appeals.expiresAt" />{" "}
              {new Date(
                accountStatus.suspension_expires_at,
              ).toLocaleDateString()}
            </p>
          )}
          <Button
            variant="secondary"
            size="sm"
            className="mt-3"
            onClick={() => setShowSubmitModal(true)}
          >
            <Icon icon={Send} size="sm" className="mr-1" />
            <FormattedMessage id="settings.appeals.submitAppeal" />
          </Button>
        </Card>
      )}

      {/* Appeals list */}
      {appealsPending && (
        <div className="space-y-3">
          {[1, 2].map((n) => (
            <Skeleton key={n} className="h-24 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {appeals && appeals.length === 0 && (
        <EmptyState
          illustration={<Icon icon={Shield} size="xl" />}
          message={intl.formatMessage({ id: "settings.appeals.empty" })}
          description={intl.formatMessage({
            id: "settings.appeals.emptyDescription",
          })}
        />
      )}

      <div className="space-y-3">
        {appeals?.map((appeal) => (
          <AppealCard key={appeal.id} appeal={appeal} />
        ))}
      </div>

      {/* Submit appeal modal */}
      <Modal
        open={showSubmitModal}
        onClose={() => setShowSubmitModal(false)}
        title={intl.formatMessage({ id: "settings.appeals.submitTitle" })}
      >
        <div className="space-y-4">
          <FormField
            label={intl.formatMessage({ id: "settings.appeals.actionId" })}
            required
          >
            {({ id }) => (
              <input
                id={id}
                value={actionId}
                onChange={(e) => setActionId(e.target.value)}
                required
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                placeholder="Moderation action ID"
              />
            )}
          </FormField>
          <FormField
            label={intl.formatMessage({ id: "settings.appeals.text" })}
            required
          >
            {({ id }) => (
              <textarea
                id={id}
                value={appealText}
                onChange={(e) => setAppealText(e.target.value)}
                required
                minLength={10}
                className="w-full min-h-[120px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                placeholder={intl.formatMessage({
                  id: "settings.appeals.textPlaceholder",
                })}
              />
            )}
          </FormField>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowSubmitModal(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleSubmit}
              disabled={
                !actionId || appealText.length < 10 || submitAppeal.isPending
              }
            >
              <FormattedMessage id="settings.appeals.submit" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
