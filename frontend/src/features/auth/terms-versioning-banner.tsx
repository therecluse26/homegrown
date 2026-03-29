import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { AlertTriangle } from "lucide-react";
import { Button, Icon } from "@/components/ui";
import { useTermsAcceptance } from "@/hooks/use-terms";
import { useAuth } from "@/hooks/use-auth";
import { Link } from "react-router";

/**
 * Terms versioning re-acceptance banner.
 *
 * Shown when the platform's Terms of Service or Privacy Policy version has
 * changed since the user last accepted. Dismissable only by accepting the
 * new version. If the family has students, COPPA re-verification is also
 * triggered.
 *
 * @see SPEC §7.3, Phase 5 P2
 */
export function TermsVersioningBanner() {
  const intl = useIntl();
  const { isParent } = useAuth();
  const termsAcceptance = useTermsAcceptance();
  const [isAccepting, setIsAccepting] = useState(false);

  // Don't show if terms acceptance status is loading or there's no pending update
  if (
    termsAcceptance.isPending ||
    termsAcceptance.error ||
    !termsAcceptance.data?.needs_acceptance
  ) {
    return null;
  }

  const { document_type, current_version } = termsAcceptance.data;
  const hasStudents = isParent;

  const documentLabel =
    document_type === "terms"
      ? intl.formatMessage({ id: "terms.versioning.termsOfService" })
      : intl.formatMessage({ id: "terms.versioning.privacyPolicy" });

  const handleAccept = async () => {
    setIsAccepting(true);
    try {
      await termsAcceptance.accept({
        document_type,
        version: current_version,
      });
    } finally {
      setIsAccepting(false);
    }
  };

  return (
    <div
      role="alert"
      className="bg-warning-container px-4 py-3"
    >
      <div className="mx-auto max-w-content flex items-center gap-3 flex-wrap">
        <Icon
          icon={AlertTriangle}
          size="md"
          className="text-on-warning-container shrink-0"
          aria-hidden
        />
        <div className="flex-1 min-w-0">
          <p className="type-body-sm text-on-warning-container">
            <FormattedMessage
              id="terms.versioning.banner"
              values={{ document: documentLabel }}
            />
          </p>
          {hasStudents && (
            <p className="type-body-sm text-on-warning-container mt-1 font-medium">
              <FormattedMessage id="terms.versioning.coppaReVerification" />
            </p>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Link
            to={
              document_type === "terms"
                ? "/legal/terms"
                : "/legal/privacy"
            }
            className="type-label-md text-on-warning-container underline"
          >
            <FormattedMessage id="terms.versioning.review" />
          </Link>
          <Button
            variant="primary"
            size="sm"
            onClick={() => void handleAccept()}
            disabled={isAccepting}
          >
            <FormattedMessage id="terms.versioning.accept" />
          </Button>
        </div>
      </div>
    </div>
  );
}
