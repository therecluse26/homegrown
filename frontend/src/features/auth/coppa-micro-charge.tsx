import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { ShieldCheck, CreditCard, CheckCircle } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useMicroChargeStatus,
  useInitMicroCharge,
  useVerifyMicroCharge,
} from "@/hooks/use-billing";

export function CoppaMicroCharge() {
  const intl = useIntl();
  const navigate = useNavigate();
  const statusQuery = useMicroChargeStatus();
  const initMutation = useInitMicroCharge();
  const verifyMutation = useVerifyMicroCharge();

  const [dollarInput, setDollarInput] = useState("");
  const [verifyError, setVerifyError] = useState(false);

  const status = statusQuery.data;

  async function handleInit() {
    setVerifyError(false);
    await initMutation.mutateAsync();
  }

  async function handleVerify(e: React.FormEvent) {
    e.preventDefault();
    setVerifyError(false);
    const parsed = parseFloat(dollarInput.replace(/[^0-9.]/g, ""));
    if (isNaN(parsed) || parsed <= 0) {
      setVerifyError(true);
      return;
    }
    const cents = Math.round(parsed * 100);
    try {
      await verifyMutation.mutateAsync(cents);
      navigate("/", { replace: true });
    } catch {
      setVerifyError(true);
      setDollarInput("");
    }
  }

  if (statusQuery.isPending) {
    return (
      <div className="mx-auto max-w-lg">
        <Skeleton className="h-8 w-64 mb-2" />
        <Skeleton className="h-4 w-96 mb-6" />
        <Skeleton className="h-48 rounded-radius-md" />
      </div>
    );
  }

  const verified = status?.status === "verified";
  const pending = status?.status === "pending";

  return (
    <div className="mx-auto max-w-lg">
      <PageTitle
        title={intl.formatMessage({ id: "coppa.microCharge.title" })}
        subtitle={intl.formatMessage({ id: "coppa.microCharge.subtitle" })}
        className="mb-6"
      />

      {verified ? (
        <Card className="flex flex-col items-center gap-4 py-8 text-center">
          <div className="w-16 h-16 rounded-full bg-success-container flex items-center justify-center">
            <Icon
              icon={CheckCircle}
              size="xl"
              className="text-on-success-container"
              aria-hidden
            />
          </div>
          <h2 className="type-title-lg text-on-surface">
            <FormattedMessage id="coppa.microCharge.verified.title" />
          </h2>
          <p className="type-body-md text-on-surface-variant">
            <FormattedMessage id="coppa.microCharge.verified.description" />
          </p>
          <Button
            variant="primary"
            onClick={() => navigate("/", { replace: true })}
          >
            <FormattedMessage id="coppa.microCharge.verified.continue" />
          </Button>
        </Card>
      ) : !pending ? (
        /* Step 1: Explain and initiate */
        <Card className="flex flex-col gap-5">
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-radius-md bg-primary-container flex items-center justify-center shrink-0">
              <Icon
                icon={ShieldCheck}
                size="md"
                className="text-on-primary-container"
                aria-hidden
              />
            </div>
            <div>
              <h2 className="type-title-sm text-on-surface font-semibold mb-1">
                <FormattedMessage id="coppa.microCharge.how.title" />
              </h2>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="coppa.microCharge.how.description" />
              </p>
            </div>
          </div>

          <ul className="flex flex-col gap-2 pl-1" aria-label={intl.formatMessage({ id: "coppa.microCharge.steps.label" })}>
            {(["step1", "step2", "step3"] as const).map((step, i) => (
              <li key={step} className="flex items-start gap-3">
                <span
                  className="flex-shrink-0 w-6 h-6 rounded-full bg-secondary-container type-label-sm text-on-secondary-container flex items-center justify-center"
                  aria-hidden
                >
                  {i + 1}
                </span>
                <span className="type-body-sm text-on-surface-variant">
                  <FormattedMessage id={`coppa.microCharge.${step}`} />
                </span>
              </li>
            ))}
          </ul>

          <div
            role="note"
            className="rounded-radius-md bg-surface-container-low px-4 py-3 type-body-sm text-on-surface-variant"
          >
            <FormattedMessage id="coppa.microCharge.refundNote" />
          </div>

          {initMutation.error && (
            <div
              role="alert"
              aria-live="assertive"
              className="rounded-radius-md bg-error-container px-4 py-3 type-body-sm text-on-error-container"
            >
              <FormattedMessage id="error.generic" />
            </div>
          )}

          <Button
            variant="primary"
            loading={initMutation.isPending}
            disabled={initMutation.isPending}
            onClick={handleInit}
            className="self-start"
          >
            <Icon icon={CreditCard} size="xs" aria-hidden className="mr-1.5" />
            <FormattedMessage id="coppa.microCharge.start" />
          </Button>
        </Card>
      ) : (
        /* Step 2: Enter charged amount */
        <Card className="flex flex-col gap-5">
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-radius-md bg-tertiary-container flex items-center justify-center shrink-0">
              <Icon
                icon={CreditCard}
                size="md"
                className="text-on-tertiary-container"
                aria-hidden
              />
            </div>
            <div>
              <h2 className="type-title-sm text-on-surface font-semibold mb-1">
                <FormattedMessage id="coppa.microCharge.enter.title" />
              </h2>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="coppa.microCharge.enter.description" />
              </p>
            </div>
          </div>

          <form onSubmit={handleVerify} className="flex flex-col gap-4">
            <div>
              <label
                htmlFor="micro-charge-amount"
                className="type-label-md text-on-surface-variant block mb-1.5"
              >
                <FormattedMessage id="coppa.microCharge.amount.label" />
              </label>
              <div className="flex items-center gap-2">
                <span className="type-body-lg text-on-surface-variant">$</span>
                <Input
                  id="micro-charge-amount"
                  type="text"
                  inputMode="decimal"
                  value={dollarInput}
                  onChange={(e) => {
                    setVerifyError(false);
                    setDollarInput(e.target.value);
                  }}
                  placeholder="0.50"
                  className="w-32"
                  aria-describedby={verifyError ? "micro-charge-error" : undefined}
                  aria-invalid={verifyError}
                  autoFocus
                />
              </div>
              {verifyError && (
                <p
                  id="micro-charge-error"
                  role="alert"
                  aria-live="assertive"
                  className="mt-1.5 type-body-sm text-error"
                >
                  <FormattedMessage id="coppa.microCharge.amount.error" />
                </p>
              )}
            </div>

            <div
              role="note"
              className="rounded-radius-md bg-surface-container-low px-4 py-3 type-body-sm text-on-surface-variant"
            >
              <FormattedMessage id="coppa.microCharge.enter.hint" />
            </div>

            <div className="flex items-center gap-3">
              <Button
                type="submit"
                variant="primary"
                loading={verifyMutation.isPending}
                disabled={verifyMutation.isPending || !dollarInput}
              >
                <FormattedMessage id="coppa.microCharge.verify" />
              </Button>
              <Button
                type="button"
                variant="tertiary"
                onClick={handleInit}
                disabled={initMutation.isPending}
              >
                <FormattedMessage id="coppa.microCharge.retry" />
              </Button>
            </div>
          </form>
        </Card>
      )}
    </div>
  );
}
