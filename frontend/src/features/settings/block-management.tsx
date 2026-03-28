import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { ShieldOff, UserX } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Avatar,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useBlockedFamilies, useUnblockFamily } from "@/hooks/use-social";

export function BlockManagement() {
  const intl = useIntl();
  const { data: blocked, isPending } = useBlockedFamilies();
  const unblock = useUnblockFamily();
  const [unblockTarget, setUnblockTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "settings.blocks.title" })}
      />

      {isPending && (
        <div className="space-y-3">
          {[1, 2].map((n) => (
            <Skeleton key={n} className="h-16 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {blocked && blocked.length === 0 && (
        <EmptyState
          illustration={<Icon icon={ShieldOff} size="xl" />}
          message={intl.formatMessage({ id: "settings.blocks.empty" })}
          description={intl.formatMessage({
            id: "settings.blocks.emptyDescription",
          })}
        />
      )}

      <div className="space-y-2">
        {blocked?.map((family) => (
          <Card
            key={family.family_id}
            className="p-card-padding flex items-center gap-3"
          >
            <Avatar size="md" name={family.display_name} />
            <div className="flex-1 min-w-0">
              <p className="type-title-sm text-on-surface">
                {family.display_name}
              </p>
              <p className="type-label-sm text-on-surface-variant">
                <FormattedMessage id="settings.blocks.blockedOn" />{" "}
                {new Date(family.blocked_at).toLocaleDateString()}
              </p>
            </div>
            <Button
              variant="tertiary"
              size="sm"
              onClick={() =>
                setUnblockTarget({
                  id: family.family_id,
                  name: family.display_name,
                })
              }
            >
              <Icon icon={UserX} size="sm" className="mr-1" />
              <FormattedMessage id="settings.blocks.unblock" />
            </Button>
          </Card>
        ))}
      </div>

      <ConfirmationDialog
        open={!!unblockTarget}
        onClose={() => setUnblockTarget(null)}
        title={intl.formatMessage(
          { id: "settings.blocks.unblockConfirm" },
          { name: unblockTarget?.name ?? "" },
        )}
        confirmLabel={intl.formatMessage({ id: "settings.blocks.unblock" })}
        onConfirm={() => {
          if (unblockTarget) {
            unblock.mutate(unblockTarget.id, {
              onSuccess: () => setUnblockTarget(null),
            });
          }
        }}
        loading={unblock.isPending}
      >
        <FormattedMessage id="settings.blocks.unblockDescription" />
      </ConfirmationDialog>
    </div>
  );
}
