import { useState } from "react";
import { useIntl } from "react-intl";
import { FileText } from "lucide-react";
import {
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Input,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useAdminAuditLog } from "@/hooks/use-admin";

export function AuditLog() {
  const intl = useIntl();
  const [actionFilter, setActionFilter] = useState("");
  const [targetTypeFilter, setTargetTypeFilter] = useState("");

  const { data, isPending } = useAdminAuditLog({
    action: actionFilter || undefined,
    target_type: targetTypeFilter || undefined,
  });

  const entries = data?.data ?? [];

  return (
    <div className="max-w-content-wide mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "admin.audit.title" })}
      />

      {/* Filters */}
      <div className="flex gap-3 mb-6">
        <Input
          value={actionFilter}
          onChange={(e) => setActionFilter(e.target.value)}
          placeholder={intl.formatMessage({
            id: "admin.audit.filterAction",
          })}
          className="w-48"
        />
        <Input
          value={targetTypeFilter}
          onChange={(e) => setTargetTypeFilter(e.target.value)}
          placeholder={intl.formatMessage({
            id: "admin.audit.filterTarget",
          })}
          className="w-48"
        />
      </div>

      {/* Loading */}
      {isPending && (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((n) => (
            <Skeleton key={n} className="h-16 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {/* Empty */}
      {!isPending && entries.length === 0 && (
        <EmptyState
          illustration={<Icon icon={FileText} size="xl" />}
          message={intl.formatMessage({ id: "admin.audit.empty" })}
          description={intl.formatMessage({
            id: "admin.audit.emptyDescription",
          })}
        />
      )}

      {/* Entries */}
      <div className="space-y-2">
        {entries.map((entry) => (
          <Card key={entry.id} className="p-card-padding">
            <div className="flex items-center gap-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <Badge variant="secondary">{entry.action}</Badge>
                  <Badge variant="default">{entry.target_type}</Badge>
                  <span className="type-label-sm text-on-surface-variant">
                    {new Date(entry.created_at).toLocaleString()}
                  </span>
                </div>
                <div className="flex items-center gap-2 type-label-sm text-on-surface-variant">
                  {entry.admin_email && (
                    <span>by {entry.admin_email}</span>
                  )}
                  {entry.target_id && (
                    <span>target: {entry.target_id.slice(0, 8)}...</span>
                  )}
                </div>
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
