import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Card, Skeleton, Badge, StatCard } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useAdminSafetyDashboard,
  useAdminReports,
} from "@/hooks/use-admin";

export function SafetyReports() {
  const intl = useIntl();
  const { data: dashboard, isPending: dashLoading } =
    useAdminSafetyDashboard();
  const [statusFilter, setStatusFilter] = useState("pending");
  const { data: reports, isPending: reportsLoading } = useAdminReports({
    status: statusFilter,
  });

  const isPending = dashLoading || reportsLoading;
  const reportList = reports?.data ?? [];

  if (isPending) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-4 gap-4">
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
          <Skeleton className="h-24 rounded-radius-md" />
        </div>
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "admin.safetyReports.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="admin.safetyReports.title" />
      </h1>

      {/* Dashboard stats */}
      {dashboard && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <StatCard
            label={intl.formatMessage({
              id: "admin.safetyReports.pendingReports",
            })}
            value={dashboard.pending_reports}
          />
          <StatCard
            label={intl.formatMessage({
              id: "admin.safetyReports.criticalReports",
            })}
            value={dashboard.critical_reports}
          />
          <StatCard
            label={intl.formatMessage({
              id: "admin.safetyReports.activeSuspensions",
            })}
            value={dashboard.active_suspensions}
          />
          <StatCard
            label={intl.formatMessage({
              id: "admin.safetyReports.reportsLast24h",
            })}
            value={dashboard.reports_last_24h}
          />
        </div>
      )}

      <div className="flex gap-1 bg-surface-container-low rounded-lg p-1">
        {(["pending", "investigating", "resolved"] as const).map((s) => (
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
            {intl.formatMessage({ id: `admin.safetyReports.${s}` })}
          </button>
        ))}
      </div>

      {reportList.length === 0 ? (
        <Card className="p-card-padding">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="admin.safetyReports.empty" />
          </p>
        </Card>
      ) : (
        <div className="space-y-3">
          {reportList.map((report) => (
            <Card key={report.id} className="p-card-padding">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      report.priority === "critical"
                        ? "primary"
                        : "secondary"
                    }
                  >
                    {report.priority}
                  </Badge>
                  <Badge variant="secondary">{report.category}</Badge>
                </div>
                <span className="type-label-sm text-on-surface-variant">
                  {new Date(report.created_at).toLocaleDateString()}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="type-label-sm text-on-surface-variant">
                    <FormattedMessage id="admin.safetyReports.target" />
                  </p>
                  <p className="type-body-sm text-on-surface">
                    {report.target_type}: {report.target_id.slice(0, 8)}...
                  </p>
                </div>
                <div>
                  <p className="type-label-sm text-on-surface-variant">
                    <FormattedMessage id="admin.safetyReports.status" />
                  </p>
                  <Badge variant="secondary">{report.status}</Badge>
                </div>
              </div>

              {report.description && (
                <p className="type-body-sm text-on-surface-variant mt-2">
                  {report.description}
                </p>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
