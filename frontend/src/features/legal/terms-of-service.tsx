import { useIntl } from "react-intl";
import { PageTitle } from "@/components/common";
import { Link } from "react-router";

/**
 * Terms of Service page.
 *
 * Route: /legal/terms
 * Rendered as plain prose — no API calls required.
 * Content is a placeholder pending legal review.
 */
export function TermsOfService() {
  const intl = useIntl();

  return (
    <article className="mx-auto max-w-prose px-4 py-12 print:py-4">
      <PageTitle
        title={intl.formatMessage({ id: "legal.terms.title" })}
        className="mb-6"
      />

      <p className="mb-4 text-label-sm text-on-surface-variant">
        Last updated: March 2026
      </p>

      <div className="space-y-6 text-body-md text-on-surface">
        <section aria-labelledby="terms-intro">
          <h2
            id="terms-intro"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            1. Introduction
          </h2>
          <p>
            Welcome to Homegrown Academy (&quot;we,&quot; &quot;us,&quot; or
            &quot;our&quot;). By accessing or using the Homegrown Academy
            platform, you agree to be bound by these Terms of Service.
          </p>
        </section>

        <section aria-labelledby="terms-eligibility">
          <h2
            id="terms-eligibility"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            2. Eligibility
          </h2>
          <p>
            You must be 18 years or older to create an account. By registering,
            you represent that you are a parent or legal guardian with authority
            to consent to the creation of student profiles for children in your
            care.
          </p>
        </section>

        <section aria-labelledby="terms-privacy">
          <h2
            id="terms-privacy"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            3. Privacy
          </h2>
          <p>
            Your use of Homegrown Academy is governed by our{" "}
            <Link to="/legal/privacy" className="font-medium text-primary hover:underline">
              Privacy Policy
            </Link>
            , which is incorporated into these Terms by reference. We take
            privacy seriously, especially for children under 13 (COPPA).
          </p>
        </section>

        <section aria-labelledby="terms-content">
          <h2
            id="terms-content"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            4. User Content
          </h2>
          <p>
            You retain ownership of content you create on Homegrown Academy.
            By posting content, you grant us a license to display it to other
            users in accordance with your privacy settings. You are responsible
            for ensuring your content complies with our{" "}
            <Link to="/legal/guidelines" className="font-medium text-primary hover:underline">
              Community Guidelines
            </Link>
            .
          </p>
        </section>

        <section aria-labelledby="terms-subscription">
          <h2
            id="terms-subscription"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            5. Subscriptions and Billing
          </h2>
          <p>
            Homegrown Academy offers free and paid subscription tiers. Paid
            subscriptions are billed in advance. You may cancel at any time;
            your access continues until the end of the current billing period.
          </p>
        </section>

        <section aria-labelledby="terms-termination">
          <h2
            id="terms-termination"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            6. Termination
          </h2>
          <p>
            We reserve the right to suspend or terminate accounts that violate
            these Terms or our Community Guidelines. You may delete your account
            at any time from Settings → Account.
          </p>
        </section>

        <section aria-labelledby="terms-contact">
          <h2
            id="terms-contact"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            7. Contact
          </h2>
          <p>
            If you have questions about these Terms, please contact us at{" "}
            <a
              href="mailto:legal@homegrown.academy"
              className="font-medium text-primary hover:underline"
            >
              legal@homegrown.academy
            </a>
            .
          </p>
        </section>
      </div>
    </article>
  );
}
