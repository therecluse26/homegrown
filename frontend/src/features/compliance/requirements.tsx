import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft, MapPin, BookOpen, Clock, Calendar } from "lucide-react";
import { Card, Icon, Skeleton } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useComplianceConfig,
  useStateRequirements,
} from "@/hooks/use-compliance";

export function Requirements() {
  const intl = useIntl();
  const { data: config, isPending: configLoading } = useComplianceConfig();
  const { data: requirements, isPending: reqLoading } = useStateRequirements(
    config?.state_code,
  );

  const isPending = configLoading || reqLoading;

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "requirements.title" })}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/compliance"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="requirements.backToCompliance" />
        </RouterLink>
      </div>

      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="requirements.title" />
      </h1>

      {!config?.configured ? (
        <Card className="p-card-padding">
          <p className="type-body-sm text-on-surface-variant mb-3">
            <FormattedMessage id="requirements.notConfigured" />
          </p>
          <RouterLink
            to="/compliance"
            className="type-label-md text-primary hover:underline"
          >
            <FormattedMessage id="requirements.configureLink" />
          </RouterLink>
        </Card>
      ) : requirements ? (
        <>
          {/* State header */}
          <Card className="p-card-padding">
            <div className="flex items-center gap-2 mb-4">
              <Icon icon={MapPin} size="md" className="text-primary" />
              <h2 className="type-headline-sm text-on-surface">
                {requirements.state_name}
              </h2>
            </div>
            {requirements.description && (
              <p className="type-body-sm text-on-surface-variant">
                {requirements.description}
              </p>
            )}
          </Card>

          {/* Requirements grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Card className="p-card-padding">
              <div className="flex items-center gap-2 mb-2">
                <Icon
                  icon={Calendar}
                  size="sm"
                  className="text-on-surface-variant"
                />
                <h3 className="type-title-md text-on-surface">
                  <FormattedMessage id="requirements.daysRequired" />
                </h3>
              </div>
              <p className="type-headline-sm text-primary">
                {requirements.days_required}
              </p>
              <p className="type-label-sm text-on-surface-variant mt-1">
                <FormattedMessage id="requirements.daysPerYear" />
              </p>
            </Card>

            <Card className="p-card-padding">
              <div className="flex items-center gap-2 mb-2">
                <Icon
                  icon={Clock}
                  size="sm"
                  className="text-on-surface-variant"
                />
                <h3 className="type-title-md text-on-surface">
                  <FormattedMessage id="requirements.hoursRequired" />
                </h3>
              </div>
              <p className="type-headline-sm text-primary">
                {requirements.hours_required}
              </p>
              <p className="type-label-sm text-on-surface-variant mt-1">
                <FormattedMessage id="requirements.hoursPerYear" />
              </p>
            </Card>
          </div>

          {/* Required subjects */}
          {requirements.subjects_required.length > 0 && (
            <Card className="p-card-padding">
              <div className="flex items-center gap-2 mb-3">
                <Icon
                  icon={BookOpen}
                  size="sm"
                  className="text-on-surface-variant"
                />
                <h3 className="type-title-md text-on-surface">
                  <FormattedMessage id="requirements.requiredSubjects" />
                </h3>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {requirements.subjects_required.map((subject) => (
                  <div
                    key={subject}
                    className="px-3 py-2 rounded-radius-sm bg-surface-container-low type-body-sm text-on-surface"
                  >
                    {subject}
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* Notification threshold */}
          {requirements.notification_threshold_days > 0 && (
            <Card className="p-card-padding">
              <h3 className="type-title-md text-on-surface mb-2">
                <FormattedMessage id="requirements.notificationThreshold" />
              </h3>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage
                  id="requirements.notificationThresholdDesc"
                  values={{
                    days: requirements.notification_threshold_days,
                  }}
                />
              </p>
            </Card>
          )}
        </>
      ) : (
        <Card className="p-card-padding">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="requirements.noData" />
          </p>
        </Card>
      )}
    </div>
  );
}
