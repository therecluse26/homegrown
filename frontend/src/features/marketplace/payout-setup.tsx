import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { DollarSign, Plus, Trash2, Star } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Icon,
  Select,
  Skeleton,
  Input,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  usePayoutConfig,
  usePayoutMethods,
  usePayoutHistory,
  useAddPayoutMethod,
  useRemovePayoutMethod,
  useSetDefaultPayoutMethod,
  type PayoutMethod,
} from "@/hooks/use-marketplace";
import { useCreatorVerification } from "@/hooks/use-marketplace";

function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

function PayoutStatusBadge({ status }: { status: string }) {
  switch (status) {
    case "completed":
      return (
        <Badge variant="primary">
          <FormattedMessage id="marketplace.payouts.status.completed" />
        </Badge>
      );
    case "processing":
      return (
        <Badge variant="secondary">
          <FormattedMessage id="marketplace.payouts.status.processing" />
        </Badge>
      );
    case "pending":
      return (
        <Badge variant="default">
          <FormattedMessage id="marketplace.payouts.status.pending" />
        </Badge>
      );
    case "failed":
      return (
        <Badge variant="error">
          <FormattedMessage id="marketplace.payouts.status.failed" />
        </Badge>
      );
    default:
      return null;
  }
}

function MethodCard({
  method,
  onRemove,
  onMakeDefault,
}: {
  method: PayoutMethod;
  onRemove: (id: string) => void;
  onMakeDefault: (id: string) => void;
}) {
  const intl = useIntl();
  return (
    <Card className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-radius-sm bg-surface-container flex items-center justify-center">
          <Icon icon={DollarSign} size="sm" className="text-on-surface-variant" aria-hidden />
        </div>
        <div>
          <p className="type-body-sm text-on-surface font-medium">
            {method.label}
          </p>
          <p className="type-label-sm text-on-surface-variant">
            {method.type === "bank_account"
              ? intl.formatMessage(
                  { id: "marketplace.payouts.method.bankEnding" },
                  { last4: method.last_four },
                )
              : intl.formatMessage(
                  { id: "marketplace.payouts.method.paypal" },
                )}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {method.is_default ? (
          <Badge variant="primary">
            <FormattedMessage id="marketplace.payouts.method.default" />
          </Badge>
        ) : (
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => onMakeDefault(method.id)}
            aria-label={intl.formatMessage(
              { id: "marketplace.payouts.method.makeDefault.label" },
              { label: method.label },
            )}
          >
            <Icon icon={Star} size="xs" aria-hidden className="mr-1" />
            <FormattedMessage id="marketplace.payouts.method.makeDefault" />
          </Button>
        )}
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => onRemove(method.id)}
          aria-label={intl.formatMessage(
            { id: "marketplace.payouts.method.remove.label" },
            { label: method.label },
          )}
          className="text-error hover:bg-error-container"
        >
          <Icon icon={Trash2} size="xs" aria-hidden />
        </Button>
      </div>
    </Card>
  );
}

export function PayoutSetup() {
  const intl = useIntl();
  const configQuery = usePayoutConfig();
  const methodsQuery = usePayoutMethods();
  const historyQuery = usePayoutHistory();
  const verificationQuery = useCreatorVerification();
  const addMethod = useAddPayoutMethod();
  const removeMethod = useRemovePayoutMethod();
  const setDefault = useSetDefaultPayoutMethod();

  const [showAddForm, setShowAddForm] = useState(false);
  const [methodType, setMethodType] = useState<"bank_account" | "paypal">(
    "bank_account",
  );
  const [label, setLabel] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  const [routingNumber, setRoutingNumber] = useState("");
  const [paypalEmail, setPaypalEmail] = useState("");

  const isVerified = verificationQuery.data?.status === "verified";
  const isPending =
    configQuery.isPending || methodsQuery.isPending || verificationQuery.isPending;

  async function handleAddMethod(e: React.FormEvent) {
    e.preventDefault();
    await addMethod.mutateAsync({
      type: methodType,
      label: label || (methodType === "bank_account" ? "Bank Account" : "PayPal"),
      ...(methodType === "bank_account"
        ? { account_number: accountNumber, routing_number: routingNumber }
        : { paypal_email: paypalEmail }),
    });
    setShowAddForm(false);
    setLabel("");
    setAccountNumber("");
    setRoutingNumber("");
    setPaypalEmail("");
  }

  if (isPending) {
    return (
      <div className="mx-auto max-w-2xl space-y-4">
        <Skeleton className="h-8 w-48 mb-2" />
        <Skeleton className="h-4 w-72 mb-6" />
        <Skeleton className="h-32 rounded-radius-md" />
        <Skeleton className="h-48 rounded-radius-md" />
      </div>
    );
  }

  if (configQuery.error || methodsQuery.error || verificationQuery.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageTitle
          title={intl.formatMessage({ id: "marketplace.payouts.title" })}
          className="mb-6"
        />
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  if (!isVerified) {
    return (
      <div className="mx-auto max-w-2xl">
        <PageTitle
          title={intl.formatMessage({ id: "marketplace.payouts.title" })}
          className="mb-6"
        />
        <Card className="flex flex-col items-center gap-4 py-8 text-center">
          <h2 className="type-title-sm text-on-surface font-semibold">
            <FormattedMessage id="marketplace.payouts.verificationRequired.title" />
          </h2>
          <p className="type-body-sm text-on-surface-variant max-w-sm">
            <FormattedMessage id="marketplace.payouts.verificationRequired.description" />
          </p>
          <a
            href="/creator/verification"
            className="inline-flex items-center justify-center gap-2 rounded-button bg-primary text-on-primary hover:state-hover active:state-pressed font-body transition-all touch-target select-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring px-6 py-2.5"
          >
            <FormattedMessage id="marketplace.payouts.verificationRequired.cta" />
          </a>
        </Card>
      </div>
    );
  }

  const config = configQuery.data;

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.payouts.title" })}
        subtitle={intl.formatMessage({ id: "marketplace.payouts.subtitle" })}
        className="mb-6"
      />

      {/* Payout config summary */}
      {config && (
        <div className="grid grid-cols-2 gap-4 mb-6">
          {[
            {
              labelId: "marketplace.payouts.config.totalEarnings",
              value: formatCents(config.total_earnings_cents),
            },
            {
              labelId: "marketplace.payouts.config.pendingBalance",
              value: formatCents(config.pending_balance_cents),
            },
            {
              labelId: "marketplace.payouts.config.minimumThreshold",
              value: formatCents(config.minimum_threshold_cents),
            },
            {
              labelId: "marketplace.payouts.config.nextPayoutDate",
              value: config.next_payout_date
                ? new Date(config.next_payout_date).toLocaleDateString()
                : intl.formatMessage({ id: "marketplace.payouts.config.nextPayoutDate.na" }),
            },
          ].map(({ labelId, value }) => (
            <Card key={labelId} className="p-card-padding">
              <p className="type-label-sm text-on-surface-variant">
                <FormattedMessage id={labelId} />
              </p>
              <p className="type-headline-sm text-on-surface mt-0.5">{value}</p>
            </Card>
          ))}
        </div>
      )}

      {/* Payout Methods */}
      <section aria-labelledby="payout-methods-heading" className="mb-6">
        <div className="flex items-center justify-between mb-3">
          <h2
            id="payout-methods-heading"
            className="type-title-sm text-on-surface font-semibold"
          >
            <FormattedMessage id="marketplace.payouts.methods.title" />
          </h2>
          {!showAddForm && (
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => setShowAddForm(true)}
            >
              <Icon icon={Plus} size="xs" aria-hidden className="mr-1" />
              <FormattedMessage id="marketplace.payouts.methods.add" />
            </Button>
          )}
        </div>

        {showAddForm && (
          <Card className="mb-4">
            <h3 className="type-label-md text-on-surface font-semibold mb-3">
              <FormattedMessage id="marketplace.payouts.methods.form.title" />
            </h3>
            <form onSubmit={handleAddMethod} className="flex flex-col gap-3">
              <div>
                <label
                  htmlFor="payout-method-type"
                  className="type-label-sm text-on-surface-variant block mb-1"
                >
                  <FormattedMessage id="marketplace.payouts.methods.form.type" />
                </label>
                <Select
                  id="payout-method-type"
                  value={methodType}
                  onChange={(e) =>
                    setMethodType(
                      e.target.value as "bank_account" | "paypal",
                    )
                  }
                >
                  <option value="bank_account">
                    {intl.formatMessage({
                      id: "marketplace.payouts.methods.form.type.bank",
                    })}
                  </option>
                  <option value="paypal">
                    {intl.formatMessage({
                      id: "marketplace.payouts.methods.form.type.paypal",
                    })}
                  </option>
                </Select>
              </div>

              <div>
                <label
                  htmlFor="payout-method-label"
                  className="type-label-sm text-on-surface-variant block mb-1"
                >
                  <FormattedMessage id="marketplace.payouts.methods.form.label" />
                </label>
                <Input
                  id="payout-method-label"
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  placeholder={intl.formatMessage({
                    id: "marketplace.payouts.methods.form.label.placeholder",
                  })}
                />
              </div>

              {methodType === "bank_account" ? (
                <>
                  <div>
                    <label
                      htmlFor="payout-routing"
                      className="type-label-sm text-on-surface-variant block mb-1"
                    >
                      <FormattedMessage id="marketplace.payouts.methods.form.routing" />
                    </label>
                    <Input
                      id="payout-routing"
                      type="text"
                      inputMode="numeric"
                      value={routingNumber}
                      onChange={(e) => setRoutingNumber(e.target.value)}
                      placeholder="021000021"
                    />
                  </div>
                  <div>
                    <label
                      htmlFor="payout-account"
                      className="type-label-sm text-on-surface-variant block mb-1"
                    >
                      <FormattedMessage id="marketplace.payouts.methods.form.account" />
                    </label>
                    <Input
                      id="payout-account"
                      type="text"
                      inputMode="numeric"
                      value={accountNumber}
                      onChange={(e) => setAccountNumber(e.target.value)}
                      placeholder="000123456789"
                    />
                  </div>
                </>
              ) : (
                <div>
                  <label
                    htmlFor="payout-paypal-email"
                    className="type-label-sm text-on-surface-variant block mb-1"
                  >
                    <FormattedMessage id="marketplace.payouts.methods.form.paypalEmail" />
                  </label>
                  <Input
                    id="payout-paypal-email"
                    type="email"
                    value={paypalEmail}
                    onChange={(e) => setPaypalEmail(e.target.value)}
                    placeholder="you@example.com"
                  />
                </div>
              )}

              {addMethod.error && (
                <div
                  role="alert"
                  aria-live="assertive"
                  className="rounded-radius-md bg-error-container px-4 py-3 type-body-sm text-on-error-container"
                >
                  <FormattedMessage id="error.generic" />
                </div>
              )}

              <div className="flex items-center gap-3">
                <Button
                  type="submit"
                  variant="primary"
                  loading={addMethod.isPending}
                  disabled={addMethod.isPending}
                >
                  <FormattedMessage id="marketplace.payouts.methods.form.save" />
                </Button>
                <Button
                  type="button"
                  variant="tertiary"
                  onClick={() => setShowAddForm(false)}
                >
                  <FormattedMessage id="common.cancel" />
                </Button>
              </div>
            </form>
          </Card>
        )}

        {!methodsQuery.data || methodsQuery.data.length === 0 ? (
          <div className="rounded-radius-md bg-surface-container-low px-4 py-6 text-center">
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="marketplace.payouts.methods.empty" />
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-2" role="list">
            {methodsQuery.data.map((method) => (
              <li key={method.id}>
                <MethodCard
                  method={method}
                  onRemove={(id) => removeMethod.mutate(id)}
                  onMakeDefault={(id) => setDefault.mutate(id)}
                />
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Payout History */}
      <section aria-labelledby="payout-history-heading">
        <h2
          id="payout-history-heading"
          className="type-title-sm text-on-surface font-semibold mb-3"
        >
          <FormattedMessage id="marketplace.payouts.history.title" />
        </h2>

        {historyQuery.isPending ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-14 rounded-radius-sm" />
            <Skeleton className="h-14 rounded-radius-sm" />
          </div>
        ) : !historyQuery.data || historyQuery.data.length === 0 ? (
          <div className="rounded-radius-md bg-surface-container-low px-4 py-6 text-center">
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="marketplace.payouts.history.empty" />
            </p>
          </div>
        ) : (
          <div className="rounded-radius-md border border-outline-variant overflow-hidden">
            <table className="w-full" aria-label={intl.formatMessage({ id: "marketplace.payouts.history.title" })}>
              <thead className="bg-surface-container-low">
                <tr>
                  <th scope="col" className="px-4 py-2 text-left type-label-sm text-on-surface-variant">
                    <FormattedMessage id="marketplace.payouts.history.date" />
                  </th>
                  <th scope="col" className="px-4 py-2 text-right type-label-sm text-on-surface-variant">
                    <FormattedMessage id="marketplace.payouts.history.amount" />
                  </th>
                  <th scope="col" className="px-4 py-2 text-right type-label-sm text-on-surface-variant">
                    <FormattedMessage id="marketplace.payouts.history.status" />
                  </th>
                </tr>
              </thead>
              <tbody>
                {historyQuery.data.map((entry, i) => (
                  <tr
                    key={entry.id}
                    className={
                      i % 2 === 0
                        ? "bg-surface"
                        : "bg-surface-container-lowest"
                    }
                  >
                    <td className="px-4 py-2 type-body-sm text-on-surface">
                      {new Date(entry.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-2 type-body-sm text-on-surface text-right font-medium">
                      {formatCents(entry.amount_cents)}
                    </td>
                    <td className="px-4 py-2 text-right">
                      <PayoutStatusBadge status={entry.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
