import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  Button,
  Card,
  Skeleton,
  Badge,
  Modal,
  Textarea,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useAdminAppeals,
  useAdminResolveAppeal,
  type AdminAppealResponse,
} from "@/hooks/use-admin";

export function Appeals() {
  const intl = useIntl();
  const [statusFilter, setStatusFilter] = useState("pending");
  const { data, isPending } = useAdminAppeals(statusFilter);
  const resolveAppeal = useAdminResolveAppeal();

  const [selectedAppeal, setSelectedAppeal] =
    useState<AdminAppealResponse | null>(null);
  const [resolution, setResolution] = useState("");
  const [resolveStatus, setResolveStatus] = useState("granted");

  const appeals = data?.data ?? [];

  function handleResolve() {
    if (!selectedAppeal || !resolution.trim()) return;
    resolveAppeal.mutate(
      {
        appealId: selectedAppeal.id,
        status: resolveStatus,
        resolution_text: resolution.trim(),
      },
      {
        onSuccess: () => {
          setSelectedAppeal(null);
          setResolution("");
        },
      },
    );
  }

  if (isPending) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "admin.appeals.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="admin.appeals.title" />
      </h1>

      <div className="flex gap-1 bg-surface-container-low rounded-lg p-1">
        {(["pending", "granted", "denied"] as const).map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => setStatusFilter(s)}
            className={`flex-1 rounded-md px-4 py-2 type-label-lg transition-colors ${
              statusFilter === s
                ? "bg-surface-container-lowest text-on-surface shadow-ambient-sm"
                : "text-on-surface-variant hover:text-on-surface hover:bg-surface-container"
            }`}
          >
            {intl.formatMessage({ id: `admin.appeals.${s}` })}
          </button>
        ))}
      </div>

      {appeals.length === 0 ? (
        <Card className="p-card-padding">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="admin.appeals.empty" />
          </p>
        </Card>
      ) : (
        <div className="space-y-3">
          {appeals.map((appeal) => (
            <Card key={appeal.id} className="p-card-padding">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      appeal.status === "pending" ? "secondary" : "primary"
                    }
                  >
                    {appeal.status}
                  </Badge>
                  <span className="type-label-sm text-on-surface-variant">
                    {new Date(appeal.created_at).toLocaleDateString()}
                  </span>
                </div>
                {appeal.status === "pending" && (
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => setSelectedAppeal(appeal)}
                  >
                    <FormattedMessage id="admin.appeals.resolve" />
                  </Button>
                )}
              </div>

              <p className="type-body-sm text-on-surface mb-2">
                {appeal.appeal_text}
              </p>

              {appeal.original_action && (
                <div className="mt-2 pt-2 border-t border-outline-variant/10">
                  <p className="type-label-sm text-on-surface-variant">
                    <FormattedMessage id="admin.appeals.originalAction" />:{" "}
                    <Badge variant="secondary">
                      {appeal.original_action.action_type}
                    </Badge>
                  </p>
                  <p className="type-body-sm text-on-surface-variant mt-1">
                    {appeal.original_action.reason}
                  </p>
                </div>
              )}

              {appeal.resolution_text && (
                <div className="mt-2 pt-2 border-t border-outline-variant/10">
                  <p className="type-label-sm text-on-surface-variant mb-1">
                    <FormattedMessage id="admin.appeals.resolution" />
                  </p>
                  <p className="type-body-sm text-on-surface">
                    {appeal.resolution_text}
                  </p>
                </div>
              )}
            </Card>
          ))}
        </div>
      )}

      <Modal
        open={!!selectedAppeal}
        onClose={() => setSelectedAppeal(null)}
        title={intl.formatMessage({ id: "admin.appeals.resolveTitle" })}
      >
        <div className="space-y-4">
          <div>
            <label
              htmlFor="resolve-status"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="admin.appeals.decision" />
            </label>
            <select
              id="resolve-status"
              value={resolveStatus}
              onChange={(e) => setResolveStatus(e.target.value)}
              className="w-full rounded-radius-sm border border-outline-variant bg-surface px-3 py-2 type-body-sm"
            >
              <option value="granted">
                {intl.formatMessage({ id: "admin.appeals.grant" })}
              </option>
              <option value="denied">
                {intl.formatMessage({ id: "admin.appeals.deny" })}
              </option>
            </select>
          </div>
          <div>
            <label
              htmlFor="resolution-text"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="admin.appeals.resolutionText" />
            </label>
            <Textarea
              id="resolution-text"
              value={resolution}
              onChange={(e) => setResolution(e.target.value)}
              rows={4}
              required
            />
          </div>
          <div className="flex justify-end gap-3">
            <Button
              variant="tertiary"
              onClick={() => setSelectedAppeal(null)}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              variant="primary"
              onClick={handleResolve}
              disabled={!resolution.trim()}
              loading={resolveAppeal.isPending}
            >
              <FormattedMessage id="admin.appeals.submitResolution" />
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
