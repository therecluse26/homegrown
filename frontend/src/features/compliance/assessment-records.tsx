import { FormattedMessage, useIntl } from "react-intl";
import { FileCheck } from "lucide-react";
import {
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { TierGate } from "@/components/common/tier-gate";
import { useComplianceAssessments } from "@/hooks/use-compliance";
import { useAuth } from "@/hooks/use-auth";
import { useEffect, useRef } from "react";

export function AssessmentRecords() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier } = useAuth();
  const assessments = useComplianceAssessments();

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "compliance.assessments.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  if (tier === "free") {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.assessments.title" />
        </h1>
        <TierGate featureName="Compliance Tracking" />
      </div>
    );
  }

  if (assessments.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <div className="flex flex-col gap-3">
          <Skeleton height="h-16" />
          <Skeleton height="h-16" />
        </div>
      </div>
    );
  }

  if (assessments.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.assessments.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const records = assessments.data ?? [];

  return (
    <div className="mx-auto max-w-3xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="compliance.assessments.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="compliance.assessments.description" />
      </p>

      {records.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({
            id: "compliance.assessments.empty",
          })}
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {records.map((record) => (
            <li key={record.id}>
              <Card className="flex items-center justify-between">
                <div className="flex items-start gap-3">
                  <Icon
                    icon={FileCheck}
                    size="md"
                    className="text-primary mt-0.5 shrink-0"
                    aria-hidden
                  />
                  <div>
                    <p className="type-title-sm text-on-surface font-medium">
                      {record.assessment_title}
                    </p>
                    <p className="type-body-sm text-on-surface-variant">
                      {record.student_name} · {record.requirement_name}
                    </p>
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <p className="type-title-sm text-on-surface font-medium">
                    {record.score}
                  </p>
                  <p className="type-label-sm text-on-surface-variant">
                    {intl.formatDate(record.date, {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    })}
                  </p>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
