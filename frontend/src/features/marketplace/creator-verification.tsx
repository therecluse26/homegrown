import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { ShieldCheck, Clock, CheckCircle, XCircle } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Icon,
  Input,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useCreatorVerification,
  useSubmitVerification,
} from "@/hooks/use-marketplace";

function VerificationStatusBadge({ status }: { status: string }) {
  switch (status) {
    case "verified":
      return (
        <Badge variant="primary">
          <Icon icon={CheckCircle} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="marketplace.creator.verification.status.verified" />
        </Badge>
      );
    case "pending":
      return (
        <Badge variant="secondary">
          <Icon icon={Clock} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="marketplace.creator.verification.status.pending" />
        </Badge>
      );
    case "rejected":
      return (
        <Badge variant="error">
          <Icon icon={XCircle} size="xs" aria-hidden className="mr-1" />
          <FormattedMessage id="marketplace.creator.verification.status.rejected" />
        </Badge>
      );
    default:
      return (
        <Badge variant="default">
          <FormattedMessage id="marketplace.creator.verification.status.unverified" />
        </Badge>
      );
  }
}

export function CreatorVerification() {
  const intl = useIntl();
  const verificationQuery = useCreatorVerification();
  const submitMutation = useSubmitVerification();

  const [legalName, setLegalName] = useState("");
  const [taxId, setTaxId] = useState("");

  const verification = verificationQuery.data;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await submitMutation.mutateAsync({ legal_name: legalName, tax_id: taxId });
    setLegalName("");
    setTaxId("");
  }

  if (verificationQuery.isPending) {
    return (
      <div className="mx-auto max-w-lg">
        <Skeleton className="h-8 w-64 mb-2" />
        <Skeleton className="h-4 w-80 mb-6" />
        <Skeleton className="h-64 rounded-radius-md" />
      </div>
    );
  }

  if (verificationQuery.error) {
    return (
      <div className="mx-auto max-w-lg">
        <PageTitle
          title={intl.formatMessage({
            id: "marketplace.creator.verification.title",
          })}
          className="mb-6"
        />
        <Card className="rounded-radius-md bg-error-container p-card-padding">
          <p className="type-body-sm text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const status = verification?.status ?? "unverified";

  return (
    <div className="mx-auto max-w-lg">
      <PageTitle
        title={intl.formatMessage({
          id: "marketplace.creator.verification.title",
        })}
        subtitle={intl.formatMessage({
          id: "marketplace.creator.verification.subtitle",
        })}
        className="mb-6"
      />

      {/* Status indicator */}
      <Card className="mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-radius-md bg-surface-container flex items-center justify-center">
              <Icon
                icon={ShieldCheck}
                size="md"
                className="text-on-surface-variant"
                aria-hidden
              />
            </div>
            <div>
              <p className="type-body-sm text-on-surface font-medium">
                <FormattedMessage id="marketplace.creator.verification.status.label" />
              </p>
              {verification?.submitted_at && (
                <p className="type-label-sm text-on-surface-variant">
                  <FormattedMessage
                    id="marketplace.creator.verification.submittedAt"
                    values={{
                      date: new Date(
                        verification.submitted_at,
                      ).toLocaleDateString(),
                    }}
                  />
                </p>
              )}
            </div>
          </div>
          <VerificationStatusBadge status={status} />
        </div>

        {status === "verified" && verification?.legal_name && (
          <div className="mt-4 pt-4 border-t border-outline-variant">
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="marketplace.creator.verification.legalName" />
            </p>
            <p className="type-body-sm text-on-surface">
              {verification.legal_name}
            </p>
            {verification.tax_id_last_four && (
              <>
                <p className="type-label-sm text-on-surface-variant mt-3 mb-1">
                  <FormattedMessage id="marketplace.creator.verification.taxId.masked" />
                </p>
                <p className="type-body-sm text-on-surface">
                  {intl.formatMessage(
                    { id: "marketplace.creator.verification.taxId.lastFour" },
                    { last4: verification.tax_id_last_four },
                  )}
                </p>
              </>
            )}
          </div>
        )}
      </Card>

      {/* Pending message */}
      {status === "pending" && (
        <Card className="rounded-radius-md bg-surface-container-low p-card-padding mb-6">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="marketplace.creator.verification.pending.message" />
          </p>
        </Card>
      )}

      {/* Verified: show success */}
      {status === "verified" && (
        <Card className="flex items-start gap-3 p-card-padding">
          <Icon
            icon={CheckCircle}
            size="md"
            className="text-primary shrink-0 mt-0.5"
            aria-hidden
          />
          <p className="type-body-sm text-on-surface">
            <FormattedMessage id="marketplace.creator.verification.verified.message" />
          </p>
        </Card>
      )}

      {/* Rejected or unverified: show form */}
      {(status === "unverified" || status === "rejected") && (
        <Card>
          {status === "rejected" && (
            <div
              role="alert"
              className="rounded-radius-sm bg-error-container px-4 py-3 type-body-sm text-on-error-container mb-4"
            >
              <FormattedMessage id="marketplace.creator.verification.rejected.message" />
            </div>
          )}

          <h2 className="type-title-sm text-on-surface font-semibold mb-4">
            <FormattedMessage
              id={
                status === "rejected"
                  ? "marketplace.creator.verification.resubmit.title"
                  : "marketplace.creator.verification.form.title"
              }
            />
          </h2>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div>
              <label
                htmlFor="legal-name"
                className="type-label-sm text-on-surface-variant block mb-1.5"
              >
                <FormattedMessage id="marketplace.creator.verification.form.legalName" />
              </label>
              <Input
                id="legal-name"
                type="text"
                value={legalName}
                onChange={(e) => setLegalName(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "marketplace.creator.verification.form.legalName.placeholder",
                })}
                required
                autoComplete="name"
              />
            </div>

            <div>
              <label
                htmlFor="tax-id"
                className="type-label-sm text-on-surface-variant block mb-1.5"
              >
                <FormattedMessage id="marketplace.creator.verification.form.taxId" />
              </label>
              <Input
                id="tax-id"
                type="text"
                value={taxId}
                onChange={(e) => setTaxId(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "marketplace.creator.verification.form.taxId.placeholder",
                })}
                required
                autoComplete="off"
                aria-describedby="tax-id-hint"
              />
              <p
                id="tax-id-hint"
                className="mt-1 type-label-sm text-on-surface-variant"
              >
                <FormattedMessage id="marketplace.creator.verification.form.taxId.hint" />
              </p>
            </div>

            {submitMutation.error && (
              <div
                role="alert"
                aria-live="assertive"
                className="rounded-radius-md bg-error-container px-4 py-3 type-body-sm text-on-error-container"
              >
                <FormattedMessage id="error.generic" />
              </div>
            )}

            <Button
              type="submit"
              variant="primary"
              loading={submitMutation.isPending}
              disabled={submitMutation.isPending || !legalName || !taxId}
              className="self-start"
            >
              <FormattedMessage id="marketplace.creator.verification.form.submit" />
            </Button>
          </form>
        </Card>
      )}
    </div>
  );
}
