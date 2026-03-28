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

// ─── Content type (frontend-only until backend types generated) ──────────────

interface ContentResponse {
  id: string;
  title: string;
  description?: string;
  content_type: string; // "pdf" | "html" | "external_link"
  content_url: string;
  subject_tags: string[];
  created_at: string;
}

// Placeholder hook — will be replaced when backend has swag annotations
function useContentDef(id: string) {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  void id;
  return {
    data: undefined as ContentResponse | undefined,
    isPending: false,
  };
}

// ─── Main component ──────────────────────────────────────────────────────────

export function ContentViewer() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { contentId } = useParams<{ contentId: string }>();

  const { data: content, isPending } = useContentDef(contentId ?? "");

  if (!contentId) {
    return (
      <EmptyState message={intl.formatMessage({ id: "content.notFound" })} />
    );
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
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/learning")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {content?.title ?? intl.formatMessage({ id: "content.title" })}
        </h1>
      </div>

      {/* Content area */}
      {content ? (
        <>
          {content.content_type === "pdf" && (
            <Card className="overflow-hidden p-0">
              <iframe
                src={content.content_url}
                title={content.title}
                className="w-full h-[70vh] border-0"
              />
            </Card>
          )}

          {content.content_type === "html" && (
            <Card className="overflow-hidden p-0">
              <iframe
                src={content.content_url}
                title={content.title}
                className="w-full h-[70vh] border-0"
                sandbox="allow-same-origin"
              />
            </Card>
          )}

          {content.content_type === "external_link" && (
            <Card className="text-center space-y-4 py-12">
              <div className="mx-auto w-16 h-16 rounded-full bg-primary-container flex items-center justify-center">
                <Icon
                  icon={ExternalLink}
                  size="xl"
                  className="text-on-primary-container"
                  aria-hidden
                />
              </div>
              <p className="type-title-md text-on-surface font-medium">
                {content.title}
              </p>
              {content.description && (
                <p className="type-body-md text-on-surface-variant max-w-md mx-auto">
                  {content.description}
                </p>
              )}
              <a
                href={content.content_url}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="primary" size="sm">
                  <Icon icon={ExternalLink} size="sm" aria-hidden />
                  <span className="ml-1">
                    <FormattedMessage id="content.openExternal" />
                  </span>
                </Button>
              </a>
            </Card>
          )}

          {/* Progress */}
          <Card className="bg-surface-container-low">
            <div className="flex items-center justify-between mb-2">
              <span className="type-label-md text-on-surface-variant">
                <FormattedMessage id="content.progress" />
              </span>
              <span className="type-label-sm text-on-surface-variant">0%</span>
            </div>
            <ProgressBar value={0} />
          </Card>

          {/* Description */}
          {content.description && (
            <Card>
              <p className="type-body-md text-on-surface-variant">
                {content.description}
              </p>
            </Card>
          )}
        </>
      ) : (
        <EmptyState
          illustration={<Icon icon={FileText} size="xl" className="text-on-surface-variant" aria-hidden />}
          message={intl.formatMessage({ id: "content.notFound" })}
          description={intl.formatMessage({
            id: "content.notFound.description",
          })}
        />
      )}
    </div>
  );
}
