import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import { ArrowLeft, AlertTriangle } from "lucide-react";
import {
  Button,
  Card,
  Icon,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";

const REFUND_REASONS = [
  "not_as_described",
  "technical_issue",
  "accidental_purchase",
  "quality_issue",
  "other",
] as const;

export function RefundRequest() {
  const intl = useIntl();
  const { id: _purchaseId } = useParams<{ id: string }>();
  const [reason, setReason] = useState("");
  const [details, setDetails] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!reason) return;
    // In a real implementation this would call a refund mutation
    setSubmitted(true);
  };

  if (submitted) {
    return (
      <div className="max-w-content-narrow mx-auto">
        <PageTitle
          title={intl.formatMessage({ id: "marketplace.refund.title" })}
        />
        <Card className="p-card-padding text-center">
          <Icon
            icon={AlertTriangle}
            size="xl"
            className="text-primary mx-auto mb-3"
          />
          <h2 className="type-title-md text-on-surface mb-2">
            <FormattedMessage id="marketplace.refund.submitted.title" />
          </h2>
          <p className="type-body-md text-on-surface-variant mb-4">
            <FormattedMessage id="marketplace.refund.submitted.description" />
          </p>
          <RouterLink to="/marketplace/purchases">
            <Button variant="primary">
              <FormattedMessage id="marketplace.refund.backToPurchases" />
            </Button>
          </RouterLink>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.refund.title" })}
      />

      <RouterLink
        to="/marketplace/purchases"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.purchases.title" />
      </RouterLink>

      <Card className="p-card-padding">
        <h2 className="type-title-md text-on-surface mb-4">
          <FormattedMessage id="marketplace.refund.title" />
        </h2>

        <div className="bg-warning-container/30 rounded-radius-md p-3 mb-4 flex items-start gap-2">
          <Icon icon={AlertTriangle} size="sm" className="text-warning shrink-0 mt-0.5" />
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="marketplace.refund.eligibility" />
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="type-label-md text-on-surface block mb-2">
              <FormattedMessage id="marketplace.refund.reason" />
            </label>
            <select
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              required
            >
              <option value="">
                {intl.formatMessage({
                  id: "marketplace.refund.selectReason",
                })}
              </option>
              {REFUND_REASONS.map((r) => (
                <option key={r} value={r}>
                  {intl.formatMessage({
                    id: `marketplace.refund.reason.${r}`,
                  })}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="type-label-md text-on-surface block mb-2">
              <FormattedMessage id="marketplace.refund.details" />
            </label>
            <textarea
              value={details}
              onChange={(e) => setDetails(e.target.value)}
              className="w-full min-h-[100px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              placeholder={intl.formatMessage({
                id: "marketplace.refund.detailsPlaceholder",
              })}
            />
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <RouterLink to="/marketplace/purchases">
              <Button type="button" variant="tertiary">
                <FormattedMessage id="common.cancel" />
              </Button>
            </RouterLink>
            <Button type="submit" variant="primary" disabled={!reason}>
              <FormattedMessage id="marketplace.refund.submit" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
