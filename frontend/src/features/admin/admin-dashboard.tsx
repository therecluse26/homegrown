import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  Users,
  Shield,
  AlertTriangle,
  Activity,
  Flag,
  FileText,
  Heart,
  Server,
} from "lucide-react";
import {
  Card,
  Icon,
  Skeleton,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useSystemHealth,
  useAdminSafetyDashboard,
} from "@/hooks/use-admin";

function StatCard({
  icon,
  label,
  value,
  variant = "default",
  to,
}: {
  icon: typeof Users;
  label: string;
  value: number | string;
  variant?: "default" | "warning" | "error";
  to?: string;
}) {
  const colorMap = {
    default: "bg-primary-container text-on-primary-container",
    warning: "bg-warning-container text-on-warning-container",
    error: "bg-error-container text-on-error-container",
  };

  const content = (
    <Card className="p-card-padding hover:bg-surface-container-low transition-colors">
      <div className="flex items-center gap-3">
        <div
          className={`w-10 h-10 rounded-radius-md flex items-center justify-center shrink-0 ${colorMap[variant]}`}
        >
          <Icon icon={icon} size="md" />
        </div>
        <div>
          <p className="type-headline-sm text-on-surface">{value}</p>
          <p className="type-label-sm text-on-surface-variant">{label}</p>
        </div>
      </div>
    </Card>
  );

  if (to) {
    return (
      <RouterLink to={to} className="block">
        {content}
      </RouterLink>
    );
  }

  return content;
}

export function AdminDashboard() {
  const intl = useIntl();
  const { data: health, isPending: healthPending } = useSystemHealth();
  const { data: safety, isPending: safetyPending } = useAdminSafetyDashboard();

  if (healthPending || safetyPending) {
    return (
      <div className="max-w-content-wide mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((n) => (
            <Skeleton key={n} className="h-24 rounded-radius-md" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-content-wide mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "admin.dashboard.title" })}
      />

      {/* System health */}
      {health && (
        <Card className="p-card-padding mb-6">
          <div className="flex items-center gap-3 mb-4">
            <Icon icon={Server} size="md" className="text-on-surface-variant" />
            <h2 className="type-title-md text-on-surface">
              <FormattedMessage id="admin.system.health" />
            </h2>
            <Badge
              variant={health.status === "healthy" ? "primary" : "secondary"}
            >
              {health.status}
            </Badge>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {health.components.map((comp) => (
              <div
                key={comp.name}
                className="flex items-center gap-2 px-3 py-2 rounded-radius-sm bg-surface-container-low"
              >
                <div
                  className={`w-2 h-2 rounded-full ${
                    comp.status === "healthy"
                      ? "bg-primary"
                      : comp.status === "degraded"
                        ? "bg-warning"
                        : "bg-error"
                  }`}
                />
                <span className="type-label-sm text-on-surface capitalize">
                  {comp.name}
                </span>
                {comp.latency_ms != null && (
                  <span className="type-label-sm text-on-surface-variant ml-auto">
                    {comp.latency_ms}ms
                  </span>
                )}
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Safety stats */}
      {safety && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <StatCard
            icon={Flag}
            label={intl.formatMessage({ id: "admin.stats.pendingReports" })}
            value={safety.pending_reports}
            variant={safety.pending_reports > 0 ? "warning" : "default"}
            to="/admin/moderation"
          />
          <StatCard
            icon={AlertTriangle}
            label={intl.formatMessage({ id: "admin.stats.criticalReports" })}
            value={safety.critical_reports}
            variant={safety.critical_reports > 0 ? "error" : "default"}
            to="/admin/moderation"
          />
          <StatCard
            icon={Heart}
            label={intl.formatMessage({ id: "admin.stats.pendingAppeals" })}
            value={safety.pending_appeals}
            to="/admin/moderation"
          />
          <StatCard
            icon={Shield}
            label={intl.formatMessage({ id: "admin.stats.activeSuspensions" })}
            value={safety.active_suspensions}
          />
          <StatCard
            icon={Users}
            label={intl.formatMessage({ id: "admin.stats.activeBans" })}
            value={safety.active_bans}
            variant={safety.active_bans > 0 ? "error" : "default"}
          />
          <StatCard
            icon={Flag}
            label={intl.formatMessage({ id: "admin.stats.unreviewedFlags" })}
            value={safety.unreviewed_flags}
          />
          <StatCard
            icon={Activity}
            label={intl.formatMessage({ id: "admin.stats.reports24h" })}
            value={safety.reports_last_24h}
          />
          <StatCard
            icon={FileText}
            label={intl.formatMessage({ id: "admin.stats.actions24h" })}
            value={safety.actions_last_24h}
          />
        </div>
      )}

      {/* Quick links */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <RouterLink to="/admin/users">
          <Card className="p-card-padding text-center hover:bg-surface-container-low transition-colors">
            <Icon icon={Users} size="lg" className="text-primary mx-auto mb-2" />
            <p className="type-title-sm text-on-surface">
              <FormattedMessage id="admin.nav.users" />
            </p>
          </Card>
        </RouterLink>
        <RouterLink to="/admin/moderation">
          <Card className="p-card-padding text-center hover:bg-surface-container-low transition-colors">
            <Icon icon={Shield} size="lg" className="text-primary mx-auto mb-2" />
            <p className="type-title-sm text-on-surface">
              <FormattedMessage id="admin.nav.moderation" />
            </p>
          </Card>
        </RouterLink>
        <RouterLink to="/admin/audit">
          <Card className="p-card-padding text-center hover:bg-surface-container-low transition-colors">
            <Icon icon={FileText} size="lg" className="text-primary mx-auto mb-2" />
            <p className="type-title-sm text-on-surface">
              <FormattedMessage id="admin.nav.audit" />
            </p>
          </Card>
        </RouterLink>
        <RouterLink to="/admin/system">
          <Card className="p-card-padding text-center hover:bg-surface-container-low transition-colors">
            <Icon icon={Server} size="lg" className="text-primary mx-auto mb-2" />
            <p className="type-title-sm text-on-surface">
              <FormattedMessage id="admin.nav.system" />
            </p>
          </Card>
        </RouterLink>
      </div>
    </div>
  );
}
