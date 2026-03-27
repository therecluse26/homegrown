import { useIntl } from "react-intl";
import { PageTitle } from "@/components/common";
import { Link } from "react-router";

/**
 * Privacy Policy page.
 *
 * Route: /legal/privacy
 * Privacy-first design: no third-party trackers, no GPS, COPPA-compliant.
 * Content is a placeholder pending legal review.
 *
 * @see SPEC §7 (privacy requirements)
 * @see ARCHITECTURE §8 (privacy design)
 */
export function PrivacyPolicy() {
  const intl = useIntl();

  return (
    <article className="mx-auto max-w-prose px-4 py-12 print:py-4">
      <PageTitle
        title={intl.formatMessage({ id: "legal.privacy.title" })}
        className="mb-6"
      />

      <p className="mb-4 text-label-sm text-on-surface-variant">
        Last updated: March 2026
      </p>

      <div className="space-y-6 text-body-md text-on-surface">
        <section aria-labelledby="privacy-intro">
          <h2
            id="privacy-intro"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Our Commitment to Privacy
          </h2>
          <p>
            Homegrown Academy is a privacy-first platform. We collect only what
            we need, we do not sell your data, and we are fully COPPA-compliant
            for families with children under 13.
          </p>
        </section>

        <section aria-labelledby="privacy-collect">
          <h2
            id="privacy-collect"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            What We Collect
          </h2>
          <ul className="ml-6 list-disc space-y-2">
            <li>
              <strong>Account information</strong>: email address, display name,
              password (hashed — we never store plaintext passwords)
            </li>
            <li>
              <strong>Family profile</strong>: family name, state/region (for
              compliance requirements), methodology preferences
            </li>
            <li>
              <strong>Student profiles</strong>: display name, birth year, grade
              level — with explicit parental consent required (COPPA)
            </li>
            <li>
              <strong>Learning data</strong>: activity logs, journal entries,
              progress records you create
            </li>
            <li>
              <strong>Usage data</strong>: pages visited, features used (no
              third-party analytics trackers)
            </li>
          </ul>
          <p className="mt-3">
            <strong>We never collect</strong>: GPS coordinates, device
            identifiers, or any data not directly entered by you.
          </p>
        </section>

        <section aria-labelledby="privacy-coppa">
          <h2
            id="privacy-coppa"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Children&apos;s Privacy (COPPA)
          </h2>
          <p>
            Student profiles for children under 13 are governed by the
            Children&apos;s Online Privacy Protection Act (COPPA). We require
            verifiable parental consent before creating any student profile.
            Child data is never shared with third parties or used for
            advertising.
          </p>
          <p className="mt-2">
            Parents may delete student profiles at any time from Settings →
            Account. Child data is deleted immediately upon request (no grace
            period).
          </p>
        </section>

        <section aria-labelledby="privacy-rights">
          <h2
            id="privacy-rights"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Your Rights
          </h2>
          <ul className="ml-6 list-disc space-y-2">
            <li>
              <strong>Access</strong>: Download all your data from Settings →
              Account → Export Data
            </li>
            <li>
              <strong>Delete</strong>: Request account deletion from Settings →
              Account (14-day grace period)
            </li>
            <li>
              <strong>Correct</strong>: Update your profile information at any
              time
            </li>
            <li>
              <strong>Portability</strong>: Export in JSON or CSV format
            </li>
          </ul>
        </section>

        <section aria-labelledby="privacy-contact">
          <h2
            id="privacy-contact"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Contact
          </h2>
          <p>
            For privacy questions or data requests, email{" "}
            <a
              href="mailto:privacy@homegrown.academy"
              className="font-medium text-primary hover:underline"
            >
              privacy@homegrown.academy
            </a>{" "}
            or visit our{" "}
            <Link
              to="/settings/account/export"
              className="font-medium text-primary hover:underline"
            >
              data export
            </Link>{" "}
            page.
          </p>
        </section>
      </div>
    </article>
  );
}
