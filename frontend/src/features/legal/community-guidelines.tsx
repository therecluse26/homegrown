import { useIntl } from "react-intl";
import { PageTitle } from "@/components/common";

/**
 * Community Guidelines page.
 *
 * Route: /legal/guidelines
 * Linked from the report dialog and registration ToS checkbox.
 * Content is a placeholder pending community standards review.
 *
 * @see SPEC §11 (safety domain)
 * @see 11-safety §3.1 (report categories)
 */
export function CommunityGuidelines() {
  const intl = useIntl();

  return (
    <article className="mx-auto max-w-prose px-4 py-12 print:py-4">
      <PageTitle
        title={intl.formatMessage({ id: "legal.guidelines.title" })}
        className="mb-6"
      />

      <p className="mb-8 text-body-md text-on-surface-variant">
        Homegrown Academy is a community for homeschooling families. These
        guidelines help keep our community safe, respectful, and welcoming for
        everyone.
      </p>

      <div className="space-y-8 text-body-md text-on-surface">
        <section aria-labelledby="gl-respectful">
          <h2
            id="gl-respectful"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Be Respectful
          </h2>
          <p>
            Treat other families with kindness and respect. Constructive
            discussion about homeschooling approaches is welcome; personal
            attacks, harassment, or bullying are not.
          </p>
        </section>

        <section aria-labelledby="gl-methodology">
          <h2
            id="gl-methodology"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Methodology Inclusivity
          </h2>
          <p>
            Homegrown Academy supports all homeschooling methodologies —
            Charlotte Mason, Classical, Unschooling, Eclectic, and more. Hostile
            commentary about other families&apos; chosen approaches is not
            permitted.
          </p>
        </section>

        <section aria-labelledby="gl-children">
          <h2
            id="gl-children"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Protect Children
          </h2>
          <p>
            Do not share content that exploits, harms, or endangers children.
            Any content that sexualizes minors will be reported to the National
            Center for Missing &amp; Exploited Children (NCMEC) and law
            enforcement, and the account will be immediately terminated.
          </p>
        </section>

        <section aria-labelledby="gl-accuracy">
          <h2
            id="gl-accuracy"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Share Accurate Information
          </h2>
          <p>
            Do not spread misinformation, especially regarding legal homeschool
            requirements, health, or safety. If you&apos;re unsure about
            something, say so.
          </p>
        </section>

        <section aria-labelledby="gl-spam">
          <h2
            id="gl-spam"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            No Spam or Self-Promotion Abuse
          </h2>
          <p>
            Commercial content belongs in the Marketplace. Do not flood the
            social feed with promotional material. Creators may share their work
            organically; systematic self-promotion without community value is
            not permitted.
          </p>
        </section>

        <section aria-labelledby="gl-report">
          <h2
            id="gl-report"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Reporting
          </h2>
          <p>
            If you see content that violates these guidelines, use the Report
            button on any post, comment, or listing. Our moderation team reviews
            all reports. You can appeal moderation decisions from Settings →
            Account → Moderation Appeals.
          </p>
        </section>

        <section aria-labelledby="gl-enforcement">
          <h2
            id="gl-enforcement"
            className="mb-3 text-title-lg font-semibold text-on-surface"
          >
            Enforcement
          </h2>
          <p>
            Violations may result in content removal, temporary suspension, or
            permanent account termination depending on severity. We aim to
            enforce these guidelines fairly and consistently.
          </p>
        </section>
      </div>
    </article>
  );
}
