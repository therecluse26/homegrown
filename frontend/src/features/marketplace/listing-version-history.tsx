import { FormattedMessage, useIntl } from "react-intl";
import { useParams } from "react-router";
import { FileText } from "lucide-react";
import { Badge, Card, Icon, Skeleton } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useListingVersions } from "@/hooks/use-marketplace";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function ListingVersionHistory() {
  const intl = useIntl();
  const { listingId = "" } = useParams<{ listingId: string }>();
  const versionsQuery = useListingVersions(listingId);

  if (versionsQuery.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton className="h-8 w-64 mb-2" />
        <Skeleton className="h-4 w-80 mb-6" />
        <div className="flex flex-col gap-3">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-16 rounded-radius-sm" />
          ))}
        </div>
      </div>
    );
  }

  if (versionsQuery.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageTitle
          title={intl.formatMessage({ id: "marketplace.versions.title" })}
          className="mb-6"
        />
        <Card className="rounded-radius-md bg-error-container p-card-padding">
          <p className="type-body-sm text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const versions = versionsQuery.data ?? [];

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.versions.title" })}
        subtitle={intl.formatMessage(
          { id: "marketplace.versions.subtitle" },
          { count: versions.length },
        )}
        className="mb-6"
      />

      {versions.length === 0 ? (
        <div className="rounded-radius-md bg-surface-container-low px-4 py-8 text-center">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="marketplace.versions.empty" />
          </p>
        </div>
      ) : (
        <ul
          className="flex flex-col gap-3"
          role="list"
          aria-label={intl.formatMessage({ id: "marketplace.versions.list.label" })}
        >
          {versions.map((version) => (
            <li key={version.id}>
              <Card className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-radius-sm bg-surface-container flex items-center justify-center shrink-0">
                    <Icon
                      icon={FileText}
                      size="sm"
                      className="text-on-surface-variant"
                      aria-hidden
                    />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="type-body-sm text-on-surface font-medium">
                        <FormattedMessage
                          id="marketplace.versions.versionNumber"
                          values={{ number: version.version_number }}
                        />
                      </p>
                      {version.is_current && (
                        <Badge variant="primary">
                          <FormattedMessage id="marketplace.versions.current" />
                        </Badge>
                      )}
                    </div>
                    <p className="type-label-sm text-on-surface-variant">
                      {version.file_name}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="type-label-sm text-on-surface">
                    {formatBytes(version.file_size_bytes)}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {new Date(version.uploaded_at).toLocaleDateString()}
                  </p>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}

      <p
        role="note"
        className="mt-4 type-label-sm text-on-surface-variant text-center"
      >
        <FormattedMessage id="marketplace.versions.viewOnly" />
      </p>
    </div>
  );
}
