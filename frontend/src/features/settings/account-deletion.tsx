import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { AlertTriangle, Download, XCircle } from "lucide-react";
import { Button, Card, Icon, Input, Skeleton } from "@/components/ui";
import { useFamilyProfile } from "@/hooks/use-family";
import {
  useDeletionStatus,
  useRequestDeletion,
  useCancelDeletion,
} from "@/hooks/use-data-lifecycle";

export function AccountDeletion() {
  const intl = useIntl();
  const profile = useFamilyProfile();
  const deletionStatus = useDeletionStatus();
  const requestDeletion = useRequestDeletion();
  const cancelDeletion = useCancelDeletion();

  const [confirmInput, setConfirmInput] = useState("");
  const [acknowledged, setAcknowledged] = useState(false);

  const familyName = profile.data?.display_name ?? "";
  const isConfirmed =
    confirmInput.trim().toLowerCase() === familyName.toLowerCase();

  const status = deletionStatus.data?.status ?? "none";
  const daysRemaining = deletionStatus.data?.days_remaining;
  const graceEnds = deletionStatus.data?.grace_period_ends_at;

  async function handleRequestDeletion() {
    if (!isConfirmed || !acknowledged) return;
    await requestDeletion.mutateAsync();
    setConfirmInput("");
    setAcknowledged(false);
  }

  async function handleCancelDeletion() {
    await cancelDeletion.mutateAsync();
  }

  if (profile.isPending || deletionStatus.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-48" />
      </div>
    );
  }

  // Active grace period — show cancellation option
  if (status === "pending" || status === "grace_period") {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 className="type-headline-md text-error font-semibold mb-6">
          <FormattedMessage id="accountDeletion.title" />
        </h1>

        <Card className="bg-error-container mb-6">
          <div className="flex items-start gap-3">
            <Icon
              icon={AlertTriangle}
              size="lg"
              aria-hidden
              className="text-error shrink-0 mt-0.5"
            />
            <div>
              <p className="type-title-sm text-on-error-container font-semibold mb-1">
                <FormattedMessage id="accountDeletion.pending.title" />
              </p>
              <p className="type-body-md text-on-error-container">
                <FormattedMessage
                  id="accountDeletion.pending.description"
                  values={{
                    days: daysRemaining ?? 0,
                    date: graceEnds
                      ? new Date(graceEnds).toLocaleDateString()
                      : "—",
                  }}
                />
              </p>
            </div>
          </div>
        </Card>

        <div className="flex gap-3">
          <Button
            variant="primary"
            onClick={() => void handleCancelDeletion()}
            loading={cancelDeletion.isPending}
          >
            <Icon icon={XCircle} size="xs" aria-hidden className="mr-1.5" />
            <FormattedMessage id="accountDeletion.cancel" />
          </Button>
        </div>

        {cancelDeletion.error && (
          <div
            role="alert"
            aria-live="assertive"
            className="mt-4 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
          >
            <FormattedMessage id="error.generic" />
          </div>
        )}
      </div>
    );
  }

  // No pending deletion — show request form
  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="type-headline-md text-error font-semibold mb-2">
        <FormattedMessage id="accountDeletion.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="accountDeletion.description" />
      </p>

      {/* Consequences */}
      <Card className="mb-6">
        <h2 className="type-title-sm text-on-surface font-semibold mb-3">
          <FormattedMessage id="accountDeletion.consequences.title" />
        </h2>
        <ul className="flex flex-col gap-2">
          {[
            "accountDeletion.consequence.data",
            "accountDeletion.consequence.students",
            "accountDeletion.consequence.social",
            "accountDeletion.consequence.marketplace",
            "accountDeletion.consequence.irreversible",
          ].map((id) => (
            <li
              key={id}
              className="flex items-start gap-2 type-body-sm text-on-surface-variant"
            >
              <span className="text-error mt-0.5 shrink-0">•</span>
              <FormattedMessage id={id} />
            </li>
          ))}
        </ul>
      </Card>

      {/* Export offer */}
      <Card className="mb-6 bg-surface-container-low">
        <div className="flex items-center justify-between">
          <div>
            <p className="type-title-sm text-on-surface font-medium">
              <FormattedMessage id="accountDeletion.exportOffer.title" />
            </p>
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="accountDeletion.exportOffer.description" />
            </p>
          </div>
          <RouterLink to="/settings/account/export">
            <Button variant="secondary" size="sm">
              <Icon
                icon={Download}
                size="xs"
                aria-hidden
                className="mr-1.5"
              />
              <FormattedMessage id="accountDeletion.exportOffer.button" />
            </Button>
          </RouterLink>
        </div>
      </Card>

      {/* Confirmation */}
      <Card className="border-error/20">
        <h2 className="type-title-sm text-error font-semibold mb-4">
          <FormattedMessage id="accountDeletion.confirm.title" />
        </h2>

        <label className="mb-4 flex cursor-pointer select-none items-start gap-3">
          <input
            type="checkbox"
            checked={acknowledged}
            onChange={(e) => setAcknowledged(e.target.checked)}
            className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-error checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          />
          <span className="type-body-sm text-on-surface">
            <FormattedMessage id="accountDeletion.confirm.acknowledge" />
          </span>
        </label>

        <div className="mb-4">
          <p className="type-label-sm text-on-surface-variant mb-1">
            <FormattedMessage
              id="accountDeletion.confirm.typeFamily"
              values={{ name: familyName }}
            />
          </p>
          <Input
            value={confirmInput}
            onChange={(e) => setConfirmInput(e.target.value)}
            placeholder={familyName}
            aria-label={intl.formatMessage({
              id: "accountDeletion.confirm.typeFamily.label",
            })}
          />
        </div>

        {requestDeletion.error && (
          <div
            role="alert"
            aria-live="assertive"
            className="mb-4 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
          >
            <FormattedMessage id="error.generic" />
          </div>
        )}

        <Button
          variant="primary"
          onClick={() => void handleRequestDeletion()}
          loading={requestDeletion.isPending}
          disabled={
            !isConfirmed || !acknowledged || requestDeletion.isPending
          }
          className="bg-error hover:bg-error/90"
        >
          <FormattedMessage id="accountDeletion.confirm.delete" />
        </Button>
      </Card>
    </div>
  );
}
