import { Link } from "react-router";
import { useIntl } from "react-intl";
import { Button, EmptyState, Icon } from "@/components/ui";
import { FileQuestion } from "lucide-react";

export function NotFoundPage() {
  const intl = useIntl();

  return (
    <div className="min-h-[60vh] flex items-center justify-center">
      <EmptyState
        message={intl.formatMessage({ id: "error.notFound", defaultMessage: "Page not found" })}
        description={intl.formatMessage({
          id: "error.notFoundDescription",
          defaultMessage: "The page you're looking for doesn't exist or has been moved.",
        })}
        illustration={<Icon icon={FileQuestion} size="2xl" className="text-on-surface-variant" />}
        action={
          <Link to="/">
            <Button variant="primary">
              {intl.formatMessage({ id: "error.goHome", defaultMessage: "Go Home" })}
            </Button>
          </Link>
        }
      />
    </div>
  );
}
