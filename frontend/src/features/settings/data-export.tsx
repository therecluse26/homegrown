import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  EmptyState,
  Icon,
  Select,
  Skeleton,
  Spinner,
} from "@/components/ui";
import { Download, FileDown, Clock, CheckCircle, XCircle } from "lucide-react";
import {
  useExportList,
  useRequestExport,
  type ExportFormat,
} from "@/hooks/use-data-lifecycle";
import { PageTitle } from "@/components/common/page-title";

const EXPORT_DOMAINS = [
  { id: "learning", labelId: "dataExport.domain.learning" },
  { id: "social", labelId: "dataExport.domain.social" },
  { id: "compliance", labelId: "dataExport.domain.compliance" },
  { id: "marketplace", labelId: "dataExport.domain.marketplace" },
  { id: "settings", labelId: "dataExport.domain.settings" },
] as const;

function StatusBadge({ status }: { status: string }) {
  switch (status) {
    case "ready":
      return (
        <Badge variant="primary">
          <Icon icon={CheckCircle} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="dataExport.status.ready" />
        </Badge>
      );
    case "processing":
    case "pending":
      return (
        <Badge variant="secondary">
          <Spinner size="sm" className="mr-1" />
          <FormattedMessage id="dataExport.status.processing" />
        </Badge>
      );
    case "expired":
      return (
        <Badge variant="default">
          <Icon icon={Clock} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="dataExport.status.expired" />
        </Badge>
      );
    case "failed":
      return (
        <Badge variant="error">
          <Icon icon={XCircle} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="dataExport.status.failed" />
        </Badge>
      );
    default:
      return null;
  }
}

export function DataExport() {
  const intl = useIntl();
  const exportList = useExportList();
  const requestExport = useRequestExport();

  const [format, setFormat] = useState<ExportFormat>("json");
  const [selectedDomains, setSelectedDomains] = useState<Set<string>>(
    new Set(EXPORT_DOMAINS.map((d) => d.id)),
  );

  function toggleDomain(id: string) {
    setSelectedDomains((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  async function handleRequestExport(e: React.FormEvent) {
    e.preventDefault();
    if (selectedDomains.size === 0) return;
    await requestExport.mutateAsync({
      format,
      domains: Array.from(selectedDomains),
    });
  }

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle
        title={intl.formatMessage({ id: "dataExport.title" })}
        subtitle={intl.formatMessage({ id: "dataExport.description" })}
        className="mb-6"
      />

      {/* Request new export */}
      <Card className="mb-6">
        <h2 className="type-title-sm text-on-surface font-semibold mb-4">
          <FormattedMessage id="dataExport.new.title" />
        </h2>
        <form onSubmit={handleRequestExport} className="flex flex-col gap-4">
          <div>
            <p className="type-label-sm text-on-surface-variant mb-2">
              <FormattedMessage id="dataExport.format" />
            </p>
            <Select
              value={format}
              onChange={(e) => setFormat(e.target.value as ExportFormat)}
              className="w-40"
              aria-label={intl.formatMessage({ id: "dataExport.format" })}
            >
              <option value="json">JSON</option>
              <option value="csv">CSV</option>
            </Select>
          </div>

          <div>
            <p className="type-label-sm text-on-surface-variant mb-2">
              <FormattedMessage id="dataExport.domains" />
            </p>
            <div className="flex flex-col gap-2">
              {EXPORT_DOMAINS.map((domain) => (
                <Checkbox
                  key={domain.id}
                  checked={selectedDomains.has(domain.id)}
                  onChange={() => toggleDomain(domain.id)}
                  label={intl.formatMessage({ id: domain.labelId })}
                />
              ))}
            </div>
          </div>

          {requestExport.error && (
            <div
              role="alert"
              aria-live="assertive"
              className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
            >
              <FormattedMessage id="error.generic" />
            </div>
          )}

          <Button
            type="submit"
            variant="primary"
            loading={requestExport.isPending}
            disabled={requestExport.isPending || selectedDomains.size === 0}
            className="self-start"
          >
            <Icon icon={FileDown} size="xs" aria-hidden className="mr-1.5" />
            <FormattedMessage id="dataExport.request" />
          </Button>
        </form>
      </Card>

      {/* Past exports */}
      <h2 className="type-title-sm text-on-surface font-semibold mb-3">
        <FormattedMessage id="dataExport.history" />
      </h2>

      {exportList.isPending ? (
        <div className="flex flex-col gap-3">
          <Skeleton height="h-16" />
          <Skeleton height="h-16" />
        </div>
      ) : !exportList.data || exportList.data.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "dataExport.history.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-2" role="list">
          {exportList.data.map((exp) => (
            <li key={exp.id}>
              <Card className="flex items-center justify-between">
                <div>
                  <p className="type-body-sm text-on-surface">
                    {exp.format.toUpperCase()} —{" "}
                    {exp.domains.join(", ")}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {new Date(exp.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <StatusBadge status={exp.status} />
                  {exp.status === "ready" && exp.download_url && (
                    <a
                      href={exp.download_url}
                      download
                      className="p-2 rounded-radius-button text-primary hover:bg-primary-container transition-colors"
                      aria-label={intl.formatMessage({
                        id: "dataExport.download",
                      })}
                    >
                      <Icon icon={Download} size="sm" aria-hidden />
                    </a>
                  )}
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
