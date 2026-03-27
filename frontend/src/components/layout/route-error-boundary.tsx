import { useRouteError, Link } from "react-router";
import { useIntl } from "react-intl";
import { Button, EmptyState, Icon } from "@/components/ui";
import { AlertTriangle } from "lucide-react";

export function RouteErrorBoundary() {
  const error = useRouteError();
  const intl = useIntl();

  // Log error for debugging (never expose details to user)
  console.error("Route error:", error);

  return (
    <div className="min-h-[60vh] flex items-center justify-center px-spacing-page-x">
      <EmptyState
        message={intl.formatMessage({ id: "error.generic", defaultMessage: "Something went wrong" })}
        description={intl.formatMessage({
          id: "error.notFoundDescription",
          defaultMessage: "The page you're looking for doesn't exist or has been moved.",
        })}
        illustration={<Icon icon={AlertTriangle} size="2xl" className="text-warning" />}
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
