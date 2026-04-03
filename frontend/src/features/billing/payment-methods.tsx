import { FormattedMessage, useIntl } from "react-intl";
import { CreditCard, Trash2 } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import {
  usePaymentMethods,
  useSetDefaultPaymentMethod,
  useRemovePaymentMethod,
} from "@/hooks/use-subscription";
import { useState, useEffect, useRef } from "react";

// ─── Component ─────────────────────────────────────────────────────────────

export function PaymentMethods() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const methods = usePaymentMethods();
  const setDefault = useSetDefaultPaymentMethod();
  const remove = useRemovePaymentMethod();

  const [removeTarget, setRemoveTarget] = useState<string | null>(null);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "billing.paymentMethods.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  if (methods.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-20" />
          <Skeleton height="h-20" />
        </div>
      </div>
    );
  }

  if (methods.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="billing.paymentMethods.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const methodList = methods.data ?? [];

  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-center justify-between mb-2">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          <FormattedMessage id="billing.paymentMethods.title" />
        </h1>
        <Button variant="primary" size="sm" disabled>
          <FormattedMessage id="billing.paymentMethods.add" />
        </Button>
      </div>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="billing.paymentMethods.description" />
      </p>

      {methodList.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "billing.paymentMethods.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {methodList.map((method) => (
            <li key={method.id}>
              <Card className="flex items-center justify-between">
                <div className="flex items-start gap-3">
                  <Icon
                    icon={CreditCard}
                    size="md"
                    aria-hidden
                    className="text-on-surface-variant mt-0.5 shrink-0"
                  />
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="type-title-sm text-on-surface font-medium">
                        {method.brand}{" "}
                        <FormattedMessage
                          id="billing.paymentMethods.cardEnding"
                          values={{ last4: method.last_four }}
                        />
                      </p>
                      {method.is_default && (
                        <Badge variant="primary">
                          <FormattedMessage id="billing.paymentMethods.default" />
                        </Badge>
                      )}
                    </div>
                    <p className="type-body-sm text-on-surface-variant">
                      <FormattedMessage
                        id="billing.paymentMethods.expires"
                        values={{
                          month: String(method.exp_month).padStart(2, "0"),
                          year: method.exp_year,
                        }}
                      />
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {!method.is_default && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      onClick={() => {
                        void setDefault.mutateAsync(method.id ?? "");
                      }}
                      disabled={setDefault.isPending}
                    >
                      <FormattedMessage id="billing.paymentMethods.makeDefault" />
                    </Button>
                  )}
                  <Button
                    variant="tertiary"
                    size="sm"
                    onClick={() => setRemoveTarget(method.id ?? null)}
                    className="text-error"
                  >
                    <Icon icon={Trash2} size="xs" aria-hidden />
                  </Button>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}

      <ConfirmationDialog
        open={!!removeTarget}
        onClose={() => setRemoveTarget(null)}
        onConfirm={() => {
          if (removeTarget) {
            void remove.mutateAsync(removeTarget).then(() => {
              setRemoveTarget(null);
            });
          }
        }}
        title={intl.formatMessage({
          id: "billing.paymentMethods.remove.title",
        })}
        confirmLabel={intl.formatMessage({
          id: "billing.paymentMethods.remove.confirm",
        })}
        destructive
        loading={remove.isPending}
      >
        <FormattedMessage id="billing.paymentMethods.remove.description" />
      </ConfirmationDialog>
    </div>
  );
}
