import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ArrowLeft } from "lucide-react";
import {
  Card,
  Icon,
  Skeleton,
  Badge,
  EmptyState,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useMicroChargeHistory } from "@/hooks/use-billing";

export function MicroChargeHistory() {
  const intl = useIntl();
  const { data: charges, isPending } = useMicroChargeHistory();

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "billing.microCharges.title" })}
      />

      <div className="flex items-center gap-3">
        <RouterLink
          to="/billing"
          className="inline-flex items-center gap-1 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="billing.microCharges.backToBilling" />
        </RouterLink>
      </div>

      <h1 className="type-headline-md text-on-surface font-semibold">
        <FormattedMessage id="billing.microCharges.title" />
      </h1>

      <p className="type-body-sm text-on-surface-variant">
        <FormattedMessage id="billing.microCharges.description" />
      </p>

      {!charges || charges.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "billing.microCharges.empty" })}
          description={intl.formatMessage({
            id: "billing.microCharges.emptyDescription",
          })}
        />
      ) : (
        <Card className="p-card-padding">
          <div className="space-y-2">
            {charges.map((charge) => (
              <div
                key={charge.id}
                className="flex items-center justify-between py-3 border-b border-outline-variant/10 last:border-0"
              >
                <div>
                  <p className="type-body-sm text-on-surface">
                    ${(charge.amount_cents / 100).toFixed(2)}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {new Date(charge.created_at).toLocaleDateString()}
                    {charge.purpose && ` — ${charge.purpose}`}
                  </p>
                </div>
                <Badge
                  variant={
                    charge.status === "verified" ? "primary" : "secondary"
                  }
                >
                  {charge.status === "verified"
                    ? intl.formatMessage({
                        id: "billing.microCharges.verified",
                      })
                    : charge.status === "pending"
                      ? intl.formatMessage({
                          id: "billing.microCharges.pending",
                        })
                      : intl.formatMessage({
                          id: "billing.microCharges.failed",
                        })}
                </Badge>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
