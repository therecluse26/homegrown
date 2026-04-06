import { PageTitle } from "@/components/common/page-title";
import { Card } from "@/components/ui";
import { ArrowLeft } from "lucide-react";
import { Link as RouterLink } from "react-router";

type ComingSoonStubProps = {
  /** Page title displayed as h1 and in document.title */
  title: string;
  /** Optional description below the title */
  description?: string;
  /** Where the back link navigates (defaults to "/") */
  backTo?: string;
  /** Label for the back link */
  backLabel?: string;
};

/**
 * Lightweight placeholder page for routes that are planned but not yet built.
 * Uses design token classes only — no hardcoded colours or spacing.
 */
export function ComingSoonStub({
  title,
  description,
  backTo = "/",
  backLabel = "Go back",
}: ComingSoonStubProps) {
  return (
    <div className="max-w-content-narrow mx-auto py-12">
      <PageTitle title={title} />
      <Card className="mt-6 p-card-padding text-center">
        <p className="type-title-md text-on-surface mb-2">Coming soon</p>
        {description && (
          <p className="type-body-sm text-on-surface-variant mb-4">
            {description}
          </p>
        )}
        {!description && (
          <p className="type-body-sm text-on-surface-variant mb-4">
            This feature is currently under development.
          </p>
        )}
        <RouterLink
          to={backTo}
          className="inline-flex items-center gap-1.5 type-label-md text-primary hover:underline"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          {backLabel}
        </RouterLink>
      </Card>
    </div>
  );
}
