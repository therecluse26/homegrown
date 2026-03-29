import { useState } from "react";
import { Link } from "react-router";
import { FormattedMessage, useIntl } from "react-intl";
import { ShieldAlert } from "lucide-react";
import { Button, Icon } from "@/components/ui";
import { useConsent } from "@/hooks/use-consent";
import { useStudents } from "@/hooks/use-family";

/**
 * COPPA re-verification banner shown when Terms of Service version changes
 * for families that already have student profiles.
 *
 * The banner is non-dismissable in the sense that it reappears on next session,
 * but can be temporarily hidden with "Remind me later".
 *
 * Detection logic: the consent status `requires_reverification` field (set by
 * backend when ToS version changes) triggers the banner. Only families with
 * existing students see it — new families without students are unaffected.
 *
 * @see SPEC §7.3 (COPPA re-verification on ToS change)
 */
export function CoppaReverificationBanner() {
  const intl = useIntl();
  const { consentStatus, isLoading: consentLoading } = useConsent();
  const { data: students, isPending: studentsLoading } = useStudents();
  const [dismissed, setDismissed] = useState(false);

  // Only show when:
  // 1. Consent data is loaded
  // 2. Family has students (COPPA only applies when child data exists)
  // 3. Backend signals re-verification is needed
  // 4. User hasn't temporarily dismissed it
  const hasStudents = (students?.length ?? 0) > 0;
  const consentStatusRaw = consentStatus as
    | (typeof consentStatus & { requires_reverification?: boolean })
    | undefined;
  const needsReverification =
    consentStatusRaw?.requires_reverification === true;

  if (
    consentLoading ||
    studentsLoading ||
    !hasStudents ||
    !needsReverification ||
    dismissed
  ) {
    return null;
  }

  return (
    <div
      className="mx-auto mb-6 flex items-start gap-3 rounded-xl bg-warning-container px-4 py-3 text-on-warning-container"
      role="alert"
    >
      <Icon icon={ShieldAlert} size="md" className="mt-0.5 shrink-0" aria-hidden />
      <div className="flex-1 space-y-2">
        <p className="type-body-md">
          <FormattedMessage id="coppa.reverify.banner" />
        </p>
        <div className="flex gap-3">
          <Link to="/legal/terms" className="no-underline">
            <Button variant="primary" size="sm">
              <FormattedMessage id="coppa.reverify.action" />
            </Button>
          </Link>
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setDismissed(true)}
            aria-label={intl.formatMessage({ id: "coppa.reverify.dismiss" })}
          >
            <FormattedMessage id="coppa.reverify.dismiss" />
          </Button>
        </div>
      </div>
    </div>
  );
}
