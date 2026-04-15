import { FormattedMessage, useIntl } from "react-intl";
import { RefreshCw } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useSystemHealth,
  useJobStatus,
  useDeadLetterJobs,
  useRetryDeadLetterJob,
} from "@/hooks/use-admin";

export function SystemDashboard() {
  const intl = useIntl();
  const { data: health, isPending: healthLoading } = useSystemHealth();
  const { data: jobs, isPending: jobsLoading } = useJobStatus();
  const { data: deadLetters, isPending: dlLoading } = useDeadLetterJobs();
  const retryJob = useRetryDeadLetterJob();

  const isPending = healthLoading || jobsLoading;

  if (isPending) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
        <Skeleton className="h-48 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "admin.system.title" })}
      />
      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="admin.system.title" />
      </h1>

      {/* System Health */}
      {health && (
        <Card className="p-card-padding">
          <div className="flex items-center justify-between mb-4">
            <h2 className="type-title-md text-on-surface">
              <FormattedMessage id="admin.system.health" />
            </h2>
            <Badge
              variant={
                health.status === "healthy" ? "primary" : "secondary"
              }
            >
              {health.status}
            </Badge>
          </div>

          <div className="space-y-2">
            {health.components.map((comp) => (
              <div
                key={comp.name}
                className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-0"
              >
                <div>
                  <p className="type-body-sm text-on-surface">
                    {comp.name}
                  </p>
                  {comp.details && (
                    <p className="type-label-sm text-on-surface-variant">
                      {comp.details}
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {comp.latency_ms != null && (
                    <span className="type-label-sm text-on-surface-variant">
                      {comp.latency_ms}ms
                    </span>
                  )}
                  <Badge
                    variant={
                      comp.status === "healthy" ? "primary" : "secondary"
                    }
                  >
                    {comp.status}
                  </Badge>
                </div>
              </div>
            ))}
          </div>

          <p className="type-label-sm text-on-surface-variant mt-3">
            <FormattedMessage id="admin.system.lastChecked" />{" "}
            {new Date(health.checked_at).toLocaleString()}
          </p>
        </Card>
      )}

      {/* Job Queues */}
      {jobs && (
        <Card className="p-card-padding">
          <h2 className="type-title-md text-on-surface mb-4">
            <FormattedMessage id="admin.system.jobQueues" />
          </h2>

          {jobs.queues.length === 0 ? (
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="admin.system.noQueues" />
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-outline-variant/10">
                    <th className="text-left type-label-sm text-on-surface-variant py-2">
                      <FormattedMessage id="admin.system.queueName" />
                    </th>
                    <th className="text-right type-label-sm text-on-surface-variant py-2">
                      <FormattedMessage id="admin.system.pending" />
                    </th>
                    <th className="text-right type-label-sm text-on-surface-variant py-2">
                      <FormattedMessage id="admin.system.processing" />
                    </th>
                    <th className="text-right type-label-sm text-on-surface-variant py-2">
                      <FormattedMessage id="admin.system.completed24h" />
                    </th>
                    <th className="text-right type-label-sm text-on-surface-variant py-2">
                      <FormattedMessage id="admin.system.failed24h" />
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {jobs.queues.map((q) => (
                    <tr
                      key={q.name}
                      className="border-b border-outline-variant/10 last:border-0"
                    >
                      <td className="py-2 type-body-sm text-on-surface">
                        {q.name}
                      </td>
                      <td className="py-2 text-right type-body-sm text-on-surface">
                        {q.pending}
                      </td>
                      <td className="py-2 text-right type-body-sm text-on-surface">
                        {q.processing}
                      </td>
                      <td className="py-2 text-right type-body-sm text-on-surface">
                        {q.completed_24h}
                      </td>
                      <td className="py-2 text-right type-body-sm text-on-surface">
                        {q.failed_24h > 0 ? (
                          <span className="text-error">{q.failed_24h}</span>
                        ) : (
                          q.failed_24h
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {jobs.dead_letter_count > 0 && (
            <p className="type-label-sm text-error mt-3">
              <FormattedMessage
                id="admin.system.deadLetterCount"
                values={{ count: jobs.dead_letter_count }}
              />
            </p>
          )}
        </Card>
      )}

      {/* Dead Letter Queue */}
      {!dlLoading && deadLetters && deadLetters.length > 0 && (
        <Card className="p-card-padding">
          <h2 className="type-title-md text-on-surface mb-4">
            <FormattedMessage id="admin.system.deadLetterQueue" />
          </h2>

          <div className="space-y-2">
            {deadLetters.map((job) => (
              <div
                key={job.id}
                className="flex items-center justify-between py-3 border-b border-outline-variant/10 last:border-0"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{job.queue}</Badge>
                    <span className="type-body-sm text-on-surface">
                      {job.job_type}
                    </span>
                  </div>
                  <p className="type-label-sm text-error mt-1">
                    {job.error_message}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    <FormattedMessage id="admin.system.failedAt" />{" "}
                    {new Date(job.failed_at).toLocaleString()} &middot;{" "}
                    <FormattedMessage
                      id="admin.system.retries"
                      values={{ count: job.retry_count }}
                    />
                  </p>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => retryJob.mutate(job.id)}
                  loading={retryJob.isPending}
                >
                  <Icon icon={RefreshCw} size="sm" className="mr-1" />
                  <FormattedMessage id="admin.system.retry" />
                </Button>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
