import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, FileText, ExternalLink } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  ProgressBar,
  Skeleton,
} from "@/components/ui";

/**
 * Student-facing content viewer.
 * Simplified version of the parent content-viewer — no admin controls,
 * just content display and progress tracking.
 */
export function StudentReader() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { contentId } = useParams<{ contentId: string }>();

  // Placeholder hook — same pattern as content-viewer.tsx
  const content = undefined as
    | {
        id: string;
        title: string;
        description?: string;
        content_type: string;
        content_url: string;
      }
    | undefined;
  const isPending = false;

  if (!contentId) {
    return <EmptyState message={intl.formatMessage({ id: "content.notFound" })} />;
  }

  if (isPending) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-[500px]" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => void navigate(-1)}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {content?.title ?? intl.formatMessage({ id: "content.title" })}
        </h1>
      </div>

      {content ? (
        <>
          {(content.content_type === "pdf" || content.content_type === "html") && (
            <Card className="overflow-hidden p-0">
              <iframe
                src={content.content_url}
                title={content.title}
                className="w-full h-[70vh] border-0"
                sandbox={
                  content.content_type === "html"
                    ? "allow-same-origin"
                    : undefined
                }
              />
            </Card>
          )}

          {content.content_type === "external_link" && (
            <Card className="text-center space-y-4 py-12">
              <Icon
                icon={ExternalLink}
                size="xl"
                className="text-on-primary-container mx-auto"
                aria-hidden
              />
              <p className="type-title-md text-on-surface font-medium">
                {content.title}
              </p>
              <a
                href={content.content_url}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="primary" size="sm">
                  <FormattedMessage id="content.openExternal" />
                </Button>
              </a>
            </Card>
          )}

          <Card className="bg-surface-container-low">
            <div className="flex items-center justify-between mb-2">
              <span className="type-label-md text-on-surface-variant">
                <FormattedMessage id="content.progress" />
              </span>
              <span className="type-label-sm text-on-surface-variant">0%</span>
            </div>
            <ProgressBar value={0} />
          </Card>
        </>
      ) : (
        <EmptyState
          illustration={
            <Icon
              icon={FileText}
              size="xl"
              className="text-on-surface-variant"
              aria-hidden
            />
          }
          message={intl.formatMessage({ id: "content.notFound" })}
          description={intl.formatMessage({
            id: "content.notFound.description",
          })}
        />
      )}
    </div>
  );
}
