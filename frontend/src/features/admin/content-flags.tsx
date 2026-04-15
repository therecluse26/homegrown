import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Card, Skeleton, Badge } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useAdminReports } from "@/hooks/use-admin";

export function ContentFlags() {
  const intl = useIntl();
  const [categoryFilter, setCategoryFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("pending");

  const { data, isPending } = useAdminReports({
    status: statusFilter,
    category: categoryFilter || undefined,
  });

  const reports = data?.data ?? [];

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
        title={intl.formatMessage({ id: "admin.contentFlags.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="admin.contentFlags.title" />
      </h1>

      <div className="flex gap-1 bg-surface-container-low rounded-lg p-1">
        {(["pending", "reviewed", "dismissed"] as const).map((s) => (
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
            {intl.formatMessage({ id: `admin.contentFlags.${s}` })}
          </button>
        ))}
      </div>

      <div className="flex gap-2 flex-wrap">
        {["", "spam", "inappropriate", "harassment", "copyright"].map(
          (cat) => (
            <button
              key={cat}
              type="button"
              onClick={() => setCategoryFilter(cat)}
              className={`px-3 py-1 rounded-radius-sm type-label-sm transition-colors ${
                categoryFilter === cat
                  ? "bg-primary text-on-primary"
                  : "bg-surface-container-low text-on-surface hover:bg-surface-container"
              }`}
            >
              {cat || intl.formatMessage({ id: "admin.contentFlags.all" })}
            </button>
          ),
        )}
      </div>

      {reports.length === 0 ? (
        <Card className="p-card-padding">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="admin.contentFlags.empty" />
          </p>
        </Card>
      ) : (
        <div className="space-y-3">
          {reports.map((report) => (
            <Card key={report.id} className="p-card-padding">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">{report.category}</Badge>
                  <Badge
                    variant={
                      report.priority === "critical"
                        ? "primary"
                        : "secondary"
                    }
                  >
                    {report.priority}
                  </Badge>
                </div>
                <span className="type-label-sm text-on-surface-variant">
                  {new Date(report.created_at).toLocaleDateString()}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="type-label-sm text-on-surface-variant">
                    <FormattedMessage id="admin.contentFlags.targetType" />
                  </p>
                  <p className="type-body-sm text-on-surface">
                    {report.target_type}
                  </p>
                </div>
                <div>
                  <p className="type-label-sm text-on-surface-variant">
                    <FormattedMessage id="admin.contentFlags.status" />
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
