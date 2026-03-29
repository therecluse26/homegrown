import { FormattedMessage, useIntl } from "react-intl";
import { useParams } from "react-router";
import { Shield, ShieldOff, UserMinus, UserCheck, UserX } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  EmptyState,
  Icon,
  Skeleton,
  Tabs,
} from "@/components/ui";
import {
  useGroupDetail,
  useGroupMembers,
  usePendingJoinRequests,
  usePromoteMember,
  useDemoteMember,
  useRemoveGroupMember,
  useApproveJoinRequest,
  useDenyJoinRequest,
} from "@/hooks/use-social";
import { useState, useEffect, useRef } from "react";

// ─── Role badge ─────────────────────────────────────────────────────────────

function RoleBadge({ role }: { role: string }) {
  const variant =
    role === "owner" ? "primary" : role === "moderator" ? "secondary" : "secondary";
  const labelId = `groups.manage.role.${role}`;
  return (
    <Badge variant={variant}>
      <FormattedMessage id={labelId} />
    </Badge>
  );
}

// ─── Component ─────────────────────────────────────────────────────────────

export function GroupManagement() {
  const intl = useIntl();
  const { groupId } = useParams<{ groupId: string }>();
  const headingRef = useRef<HTMLHeadingElement>(null);

  const groupDetail = useGroupDetail(groupId);
  const members = useGroupMembers(groupId);
  const pendingRequests = usePendingJoinRequests(groupId);
  const promoteMember = usePromoteMember();
  const demoteMember = useDemoteMember();
  const removeMember = useRemoveGroupMember();
  const approveRequest = useApproveJoinRequest();
  const denyRequest = useDenyJoinRequest();

  const [removeTarget, setRemoveTarget] = useState<{
    familyId: string;
    name: string;
  } | null>(null);
  const [promoteTarget, setPromoteTarget] = useState<{
    familyId: string;
    name: string;
  } | null>(null);
  const [demoteTarget, setDemoteTarget] = useState<{
    familyId: string;
    name: string;
  } | null>(null);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "groups.manage.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // ─── Loading ──────────────────────────────────────────────────────────

  if (groupDetail.isPending || members.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  // ─── Error ────────────────────────────────────────────────────────────

  if (groupDetail.error || members.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="groups.manage.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const group = groupDetail.data;
  const memberList = members.data ?? [];
  const pendingList = pendingRequests.data ?? [];
  const myRole = group?.my_role;
  const isOwnerOrMod = myRole === "owner" || myRole === "moderator";

  const tabs = [
    {
      id: "members",
      label: intl.formatMessage(
        { id: "groups.manage.members" },
      ),
      content: (
        <div>
          {memberList.length === 0 ? (
            <EmptyState
              message={intl.formatMessage({ id: "groups.manage.members" })}
            />
          ) : (
            <ul className="flex flex-col gap-2" role="list">
              {memberList.map((member) => (
                <li key={member.family_id}>
                  <Card className="flex items-center justify-between py-3">
                    <div className="flex items-center gap-3">
                      <div>
                        <p className="type-title-sm text-on-surface font-medium">
                          {member.display_name}
                        </p>
                        <div className="flex items-center gap-1.5 mt-0.5">
                          <RoleBadge role={member.role} />
                          {member.joined_at && (
                            <span className="type-label-sm text-on-surface-variant">
                              {intl.formatDate(member.joined_at, {
                                month: "short",
                                year: "numeric",
                              })}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Actions (only for owner/moderator and not on self or owner) */}
                    {isOwnerOrMod && member.role !== "owner" && (
                      <div className="flex items-center gap-1 shrink-0">
                        {myRole === "owner" && member.role === "member" && (
                          <Button
                            variant="tertiary"
                            size="sm"
                            onClick={() =>
                              setPromoteTarget({
                                familyId: member.family_id,
                                name: member.display_name,
                              })
                            }
                            title={intl.formatMessage({
                              id: "groups.manage.promote",
                            })}
                          >
                            <Icon icon={Shield} size="xs" aria-hidden />
                          </Button>
                        )}
                        {myRole === "owner" && member.role === "moderator" && (
                          <Button
                            variant="tertiary"
                            size="sm"
                            onClick={() =>
                              setDemoteTarget({
                                familyId: member.family_id,
                                name: member.display_name,
                              })
                            }
                            title={intl.formatMessage({
                              id: "groups.manage.demote",
                            })}
                          >
                            <Icon icon={ShieldOff} size="xs" aria-hidden />
                          </Button>
                        )}
                        <Button
                          variant="tertiary"
                          size="sm"
                          onClick={() =>
                            setRemoveTarget({
                              familyId: member.family_id,
                              name: member.display_name,
                            })
                          }
                          className="text-error"
                          title={intl.formatMessage({
                            id: "groups.manage.remove",
                          })}
                        >
                          <Icon icon={UserMinus} size="xs" aria-hidden />
                        </Button>
                      </div>
                    )}
                  </Card>
                </li>
              ))}
            </ul>
          )}
        </div>
      ),
    },
    {
      id: "pending",
      label: `${intl.formatMessage({ id: "groups.manage.pendingRequests" })} (${pendingList.length})`,
      content: (
        <div>
          {pendingList.length === 0 ? (
            <EmptyState
              message={intl.formatMessage({
                id: "groups.manage.noPending",
              })}
            />
          ) : (
            <ul className="flex flex-col gap-2" role="list">
              {pendingList.map((request) => (
                <li key={request.family_id}>
                  <Card className="flex items-center justify-between py-3">
                    <p className="type-title-sm text-on-surface font-medium">
                      {request.display_name}
                    </p>
                    <div className="flex items-center gap-2 shrink-0">
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() =>
                          approveRequest.mutate({
                            groupId: groupId ?? "",
                            familyId: request.family_id,
                          })
                        }
                        disabled={approveRequest.isPending}
                      >
                        <Icon
                          icon={UserCheck}
                          size="xs"
                          aria-hidden
                          className="mr-1"
                        />
                        <FormattedMessage id="groups.manage.approve" />
                      </Button>
                      <Button
                        variant="tertiary"
                        size="sm"
                        onClick={() =>
                          denyRequest.mutate({
                            groupId: groupId ?? "",
                            familyId: request.family_id,
                          })
                        }
                        disabled={denyRequest.isPending}
                        className="text-error"
                      >
                        <Icon
                          icon={UserX}
                          size="xs"
                          aria-hidden
                          className="mr-1"
                        />
                        <FormattedMessage id="groups.manage.deny" />
                      </Button>
                    </div>
                  </Card>
                </li>
              ))}
            </ul>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="groups.manage.title" />
      </h1>
      {group && (
        <p className="type-body-md text-on-surface-variant mb-6">
          {group.summary.name}
        </p>
      )}

      <Tabs tabs={tabs} defaultTab="members" />

      {/* Promote member dialog */}
      <ConfirmationDialog
        open={!!promoteTarget}
        onClose={() => setPromoteTarget(null)}
        onConfirm={() => {
          if (promoteTarget && groupId) {
            void promoteMember
              .mutateAsync({
                groupId,
                familyId: promoteTarget.familyId,
              })
              .then(() => {
                setPromoteTarget(null);
              });
          }
        }}
        title={intl.formatMessage({ id: "groups.manage.promote.title" })}
        confirmLabel={intl.formatMessage({
          id: "groups.manage.promote.confirm",
        })}
        loading={promoteMember.isPending}
      >
        <FormattedMessage
          id="groups.manage.promote.description"
          values={{ name: promoteTarget?.name ?? "" }}
        />
      </ConfirmationDialog>

      {/* Demote member dialog */}
      <ConfirmationDialog
        open={!!demoteTarget}
        onClose={() => setDemoteTarget(null)}
        onConfirm={() => {
          if (demoteTarget && groupId) {
            void demoteMember
              .mutateAsync({
                groupId,
                familyId: demoteTarget.familyId,
              })
              .then(() => {
                setDemoteTarget(null);
              });
          }
        }}
        title={intl.formatMessage({ id: "groups.manage.demote.title" })}
        confirmLabel={intl.formatMessage({
          id: "groups.manage.demote.confirm",
        })}
        loading={demoteMember.isPending}
      >
        <FormattedMessage
          id="groups.manage.demote.description"
          values={{ name: demoteTarget?.name ?? "" }}
        />
      </ConfirmationDialog>

      {/* Remove member dialog */}
      <ConfirmationDialog
        open={!!removeTarget}
        onClose={() => setRemoveTarget(null)}
        onConfirm={() => {
          if (removeTarget && groupId) {
            void removeMember
              .mutateAsync({
                groupId,
                familyId: removeTarget.familyId,
              })
              .then(() => {
                setRemoveTarget(null);
              });
          }
        }}
        title={intl.formatMessage({ id: "groups.manage.remove.title" })}
        confirmLabel={intl.formatMessage({
          id: "groups.manage.remove.confirm",
        })}
        destructive
        loading={removeMember.isPending}
      >
        <FormattedMessage id="groups.manage.remove.description" />
      </ConfirmationDialog>
    </div>
  );
}
