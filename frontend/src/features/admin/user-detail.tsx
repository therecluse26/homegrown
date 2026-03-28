import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, ShieldAlert, Ban, ShieldCheck } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
  Modal,
  Input,
  FormField,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useAdminUserDetail,
  useAdminSuspendUser,
  useAdminBanUser,
  useAdminUnsuspendUser,
} from "@/hooks/use-admin";

export function UserDetail() {
  const intl = useIntl();
  const { id } = useParams<{ id: string }>();
  const { data: user, isPending } = useAdminUserDetail(id);
  const suspendUser = useAdminSuspendUser();
  const banUser = useAdminBanUser();
  const unsuspendUser = useAdminUnsuspendUser();

  const [showSuspendModal, setShowSuspendModal] = useState(false);
  const [showBanModal, setShowBanModal] = useState(false);
  const [actionReason, setActionReason] = useState("");

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!user) return null;

  const handleSuspend = () => {
    if (!actionReason.trim() || !id) return;
    suspendUser.mutate(
      { userId: id, reason: actionReason },
      {
        onSuccess: () => {
          setShowSuspendModal(false);
          setActionReason("");
        },
      },
    );
  };

  const handleBan = () => {
    if (!actionReason.trim() || !id) return;
    banUser.mutate(
      { userId: id, reason: actionReason },
      {
        onSuccess: () => {
          setShowBanModal(false);
          setActionReason("");
        },
      },
    );
  };

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage(
          { id: "admin.user.title" },
          { name: user.family.name },
        )}
      />

      <RouterLink
        to="/admin/users"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="admin.users.title" />
      </RouterLink>

      {/* Family info */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="type-headline-sm text-on-surface">
            {user.family.name}
          </h2>
          <div className="flex items-center gap-2">
            <Badge
              variant={
                user.family.account_status === "active"
                  ? "primary"
                  : "secondary"
              }
            >
              {user.family.account_status}
            </Badge>
            {user.subscription && (
              <Badge variant="secondary">{user.subscription.tier}</Badge>
            )}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="admin.user.created" />
            </p>
            <p className="type-body-sm text-on-surface">
              {new Date(user.family.created_at).toLocaleDateString()}
            </p>
          </div>
          {user.family.last_active_at && (
            <div>
              <p className="type-label-sm text-on-surface-variant mb-1">
                <FormattedMessage id="admin.user.lastActive" />
              </p>
              <p className="type-body-sm text-on-surface">
                {new Date(user.family.last_active_at).toLocaleDateString()}
              </p>
            </div>
          )}
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="admin.user.activity7d" />
            </p>
            <p className="type-body-sm text-on-surface">
              {user.recent_activity.activity_count_7d} actions
            </p>
          </div>
        </div>
      </Card>

      {/* Parents */}
      <Card className="p-card-padding mb-6">
        <h3 className="type-title-md text-on-surface mb-3">
          <FormattedMessage id="admin.user.parents" />
        </h3>
        <div className="space-y-2">
          {user.parents.map((parent) => (
            <div
              key={parent.id}
              className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0"
            >
              <div>
                <p className="type-body-sm text-on-surface">
                  {parent.display_name}
                </p>
                <p className="type-label-sm text-on-surface-variant">
                  {parent.email}
                </p>
              </div>
              {parent.is_primary && (
                <Badge variant="primary">Primary</Badge>
              )}
            </div>
          ))}
        </div>
      </Card>

      {/* Students */}
      {user.students.length > 0 && (
        <Card className="p-card-padding mb-6">
          <h3 className="type-title-md text-on-surface mb-3">
            <FormattedMessage id="admin.user.students" />
          </h3>
          <div className="space-y-2">
            {user.students.map((student) => (
              <div
                key={student.id}
                className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0"
              >
                <p className="type-body-sm text-on-surface">
                  {student.display_name}
                </p>
                {student.grade_level && (
                  <Badge variant="secondary">{student.grade_level}</Badge>
                )}
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Moderation history */}
      {user.moderation_history.length > 0 && (
        <Card className="p-card-padding mb-6">
          <h3 className="type-title-md text-on-surface mb-3">
            <FormattedMessage id="admin.user.moderationHistory" />
          </h3>
          <div className="space-y-2">
            {user.moderation_history.map((action, i) => (
              <div
                key={i}
                className="py-2 border-b border-outline-variant/10 last:border-0"
              >
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">{action.action}</Badge>
                  <span className="type-label-sm text-on-surface-variant">
                    {new Date(action.created_at).toLocaleDateString()}
                  </span>
                </div>
                <p className="type-body-sm text-on-surface-variant mt-1">
                  {action.reason}
                </p>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Actions */}
      <Card className="p-card-padding">
        <h3 className="type-title-md text-on-surface mb-3">
          <FormattedMessage id="admin.user.actions" />
        </h3>
        <div className="flex gap-2">
          {user.family.account_status === "active" && (
            <>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setShowSuspendModal(true)}
              >
                <Icon icon={ShieldAlert} size="sm" className="mr-1" />
                <FormattedMessage id="admin.user.suspend" />
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setShowBanModal(true)}
              >
                <Icon icon={Ban} size="sm" className="mr-1" />
                <FormattedMessage id="admin.user.ban" />
              </Button>
            </>
          )}
          {user.family.account_status === "suspended" && (
            <Button
              variant="primary"
              size="sm"
              onClick={() => id && unsuspendUser.mutate(id)}
              disabled={unsuspendUser.isPending}
            >
              <Icon icon={ShieldCheck} size="sm" className="mr-1" />
              <FormattedMessage id="admin.user.unsuspend" />
            </Button>
          )}
        </div>
      </Card>

      {/* Suspend modal */}
      <Modal
        open={showSuspendModal}
        onClose={() => setShowSuspendModal(false)}
        title={intl.formatMessage({ id: "admin.user.suspendTitle" })}
      >
        <div className="space-y-4">
          <FormField
            label={intl.formatMessage({ id: "admin.user.reason" })}
            required
          >
            {({ id: fieldId }) => (
              <Input
                id={fieldId}
                value={actionReason}
                onChange={(e) => setActionReason(e.target.value)}
                required
              />
            )}
          </FormField>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowSuspendModal(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleSuspend}
              disabled={!actionReason.trim() || suspendUser.isPending}
            >
              <FormattedMessage id="admin.user.suspend" />
            </Button>
          </div>
        </div>
      </Modal>

      {/* Ban modal */}
      <Modal
        open={showBanModal}
        onClose={() => setShowBanModal(false)}
        title={intl.formatMessage({ id: "admin.user.banTitle" })}
      >
        <div className="space-y-4">
          <FormField
            label={intl.formatMessage({ id: "admin.user.reason" })}
            required
          >
            {({ id: fieldId }) => (
              <Input
                id={fieldId}
                value={actionReason}
                onChange={(e) => setActionReason(e.target.value)}
                required
              />
            )}
          </FormField>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setShowBanModal(false)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleBan}
              disabled={!actionReason.trim() || banUser.isPending}
            >
              <FormattedMessage id="admin.user.ban" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
