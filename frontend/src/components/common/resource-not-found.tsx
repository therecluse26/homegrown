import { FormattedMessage } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft } from "lucide-react";
import { Card, Icon } from "@/components/ui";

/**
 * Shown when a detail page's resource fetch returns no data after loading.
 * Handles invalid UUIDs, deleted resources, and permission errors gracefully.
 */
export function ResourceNotFound({
  backTo,
  backLabelId = "common.goBack",
}: {
  backTo: string;
  backLabelId?: string;
}) {
  return (
    <div className="max-w-content-narrow mx-auto py-12">
      <Card className="p-card-padding text-center">
        <p className="type-title-md text-on-surface mb-2">
          <FormattedMessage
            id="error.resourceNotFound"
            defaultMessage="Resource not found"
          />
        </p>
        <p className="type-body-sm text-on-surface-variant mb-4">
          <FormattedMessage
            id="error.resourceNotFound.description"
            defaultMessage="This item may have been removed or the link is invalid."
          />
        </p>
        <RouterLink
          to={backTo}
          className="inline-flex items-center gap-1.5 type-label-md text-primary hover:underline"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id={backLabelId} defaultMessage="Go back" />
        </RouterLink>
      </Card>
    </div>
  );
}
