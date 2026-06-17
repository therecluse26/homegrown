import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router";
import { Mail, BookOpen, Users, FileText, Shield } from "lucide-react";
import { PageTitle } from "@/components/common";
import { Card, Icon } from "@/components/ui";

type FaqItem = { q: string; a: string };

const FAQ_ITEMS: FaqItem[] = [
  { q: "help.faq.q1", a: "help.faq.a1" },
  { q: "help.faq.q2", a: "help.faq.a2" },
  { q: "help.faq.q3", a: "help.faq.a3" },
  { q: "help.faq.q4", a: "help.faq.a4" },
];

type QuickLink = { labelId: string; to: string; icon: typeof BookOpen };

const QUICK_LINKS: QuickLink[] = [
  { labelId: "nav.learning", to: "/learning", icon: BookOpen },
  { labelId: "nav.social", to: "/friends", icon: Users },
  { labelId: "nav.settings", to: "/settings", icon: Shield },
  { labelId: "nav.compliance", to: "/compliance", icon: FileText },
];

export function HelpPage() {
  const intl = useIntl();

  return (
    <div className="max-w-prose mx-auto py-8 space-y-8">
      <PageTitle title={intl.formatMessage({ id: "help.title", defaultMessage: "Help & Support" })} />

      {/* Contact card */}
      <Card className="p-card-padding">
        <h2 className="type-title-md text-on-surface font-semibold mb-3">
          <FormattedMessage id="help.contact.title" defaultMessage="Contact Us" />
        </h2>
        <p className="type-body-md text-on-surface-variant mb-4">
          <FormattedMessage id="help.contact.description" defaultMessage="Our team is here to help homeschooling families succeed. Reach out any time." />
        </p>
        <a
          href={`mailto:${intl.formatMessage({ id: "help.email", defaultMessage: "support@homegrownacademy.com" })}`}
          className="inline-flex items-center gap-2 type-label-lg text-primary hover:underline underline-offset-4"
        >
          <Icon icon={Mail} size="sm" />
          <FormattedMessage id="help.email" defaultMessage="support@homegrownacademy.com" />
        </a>
      </Card>

      {/* FAQ */}
      <section aria-labelledby="faq-heading">
        <h2 id="faq-heading" className="type-title-md text-on-surface font-semibold mb-4">
          <FormattedMessage id="help.faq.title" defaultMessage="Frequently Asked Questions" />
        </h2>
        <div className="space-y-3">
          {FAQ_ITEMS.map((item) => (
            <Card key={item.q} className="p-card-padding">
              <h3 className="type-label-lg text-on-surface font-semibold mb-1">
                {intl.formatMessage({ id: item.q })}
              </h3>
              <p className="type-body-md text-on-surface-variant">
                {intl.formatMessage({ id: item.a })}
              </p>
            </Card>
          ))}
        </div>
      </section>

      {/* Quick navigation links */}
      <section aria-labelledby="quick-nav-heading">
        <h2 id="quick-nav-heading" className="type-title-md text-on-surface font-semibold mb-4">
          <FormattedMessage id="help.quickNav" defaultMessage="Quick Links" />
        </h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {QUICK_LINKS.map((ql) => (
            <Link
              key={ql.to}
              to={ql.to}
              className="flex flex-col items-center gap-2 p-4 rounded-lg bg-surface-container hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)] text-center"
            >
              <Icon icon={ql.icon} size="md" className="text-primary" />
              <span className="type-label-sm text-on-surface">
                {intl.formatMessage({ id: ql.labelId })}
              </span>
            </Link>
          ))}
        </div>
      </section>

      {/* Legal links */}
      <div className="flex flex-wrap gap-4 pt-2 border-t border-outline-variant/20">
        <Link to="/legal/privacy" className="type-label-sm text-on-surface-variant hover:text-primary underline-offset-4 hover:underline">
          <FormattedMessage id="legal.privacy.title" defaultMessage="Privacy Policy" />
        </Link>
        <Link to="/legal/terms" className="type-label-sm text-on-surface-variant hover:text-primary underline-offset-4 hover:underline">
          <FormattedMessage id="legal.terms.title" defaultMessage="Terms of Service" />
        </Link>
        <Link to="/legal/guidelines" className="type-label-sm text-on-surface-variant hover:text-primary underline-offset-4 hover:underline">
          <FormattedMessage id="legal.guidelines.title" defaultMessage="Community Guidelines" />
        </Link>
      </div>
    </div>
  );
}
