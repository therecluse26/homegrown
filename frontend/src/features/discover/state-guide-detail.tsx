import { useEffect, useRef } from "react";
import { useParams, Link } from "react-router";
import {
  ArrowLeft,
  AlertTriangle,
  CheckCircle,
  XCircle,
  FileText,
  Calendar,
  BookOpen,
  ClipboardCheck,
  Building2,
  Info,
} from "lucide-react";
import { Badge, Button, Card, EmptyState, Skeleton } from "@/components/ui";
import { useStateGuide } from "@/hooks/use-discover";

const REGULATION_BADGE: Record<string, { variant: "success" | "warning" | "error"; label: string }> = {
  low: { variant: "success", label: "Low Regulation" },
  moderate: { variant: "warning", label: "Moderate Regulation" },
  high: { variant: "error", label: "High Regulation" },
};

type RequirementItemProps = {
  icon: React.ReactNode;
  label: string;
  required: boolean | undefined;
  details: string | undefined;
};

function RequirementItem({ icon, label, required, details }: RequirementItemProps) {
  if (required === undefined && !details) return null;

  return (
    <div className="flex gap-3 py-3 border-b border-outline-variant last:border-0">
      <div className="mt-0.5 text-on-surface-variant">{icon}</div>
      <div className="flex-1 space-y-1">
        <div className="flex items-center gap-2">
          <span className="type-body-md text-on-surface font-medium">
            {label}
          </span>
          {required !== undefined &&
            (required ? (
              <CheckCircle className="h-4 w-4 text-primary" />
            ) : (
              <XCircle className="h-4 w-4 text-on-surface-variant opacity-50" />
            ))}
        </div>
        {details && (
          <p className="type-body-sm text-on-surface-variant">{details}</p>
        )}
      </div>
    </div>
  );
}

export function StateGuideDetail() {
  const { stateCode } = useParams<{ stateCode: string }>();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: guide, isPending, error } = useStateGuide(stateCode);

  useEffect(() => {
    if (guide?.state_name) {
      document.title = `${guide.state_name} Homeschooling Guide - Homegrown Academy`;
    } else {
      document.title = "State Guide - Homegrown Academy";
    }
    headingRef.current?.focus();
  }, [guide?.state_name]);

  if (isPending) {
    return (
      <div className="space-y-6">
        <Skeleton width="w-32" height="h-6" />
        <Skeleton width="w-64" height="h-8" />
        <Skeleton width="w-full" height="h-64" />
      </div>
    );
  }

  if (error || !guide) {
    return (
      <EmptyState
        message="State guide not found"
        description="This state guide may not be available yet."
        action={
          <Link to="/discover/states" tabIndex={-1}>
            <Button>Browse All States</Button>
          </Link>
        }
      />
    );
  }

  const reqs = guide.requirements;
  const regulation = reqs?.regulation_level
    ? REGULATION_BADGE[reqs.regulation_level]
    : undefined;

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        to="/discover/states"
        className="inline-flex items-center gap-1 type-label-md text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring focus-visible:rounded"
      >
        <ArrowLeft className="h-4 w-4" />
        All States
      </Link>

      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <h1
            ref={headingRef}
            tabIndex={-1}
            className="type-headline-md text-on-surface font-semibold outline-none"
          >
            {guide.state_name}
          </h1>
          {regulation && (
            <Badge variant={regulation.variant}>{regulation.label}</Badge>
          )}
        </div>
        {guide.last_reviewed_at && (
          <p className="type-body-sm text-on-surface-variant">
            Last reviewed:{" "}
            {new Date(guide.last_reviewed_at).toLocaleDateString("en-US", {
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </p>
        )}
      </div>

      {/* Requirements checklist */}
      {reqs && (
        <Card>
          <h2 className="type-title-md text-on-surface font-semibold mb-2">
            Requirements
          </h2>
          <div className="divide-y-0">
            <RequirementItem
              icon={<FileText className="h-5 w-5" />}
              label="Notification Required"
              required={reqs.notification_required}
              details={reqs.notification_details}
            />
            <RequirementItem
              icon={<BookOpen className="h-5 w-5" />}
              label="Required Subjects"
              required={
                reqs.required_subjects && reqs.required_subjects.length > 0
                  ? true
                  : undefined
              }
              details={reqs.required_subjects?.join(", ")}
            />
            <RequirementItem
              icon={<ClipboardCheck className="h-5 w-5" />}
              label="Assessment Required"
              required={reqs.assessment_required}
              details={reqs.assessment_details}
            />
            <RequirementItem
              icon={<Calendar className="h-5 w-5" />}
              label="Attendance Required"
              required={reqs.attendance_required}
              details={
                reqs.attendance_days
                  ? `${String(reqs.attendance_days)} days/year${reqs.attendance_details ? ` — ${reqs.attendance_details}` : ""}`
                  : reqs.attendance_details
              }
            />
            <RequirementItem
              icon={<FileText className="h-5 w-5" />}
              label="Record Keeping"
              required={reqs.record_keeping_required}
              details={reqs.record_keeping_details}
            />
            <RequirementItem
              icon={<Building2 className="h-5 w-5" />}
              label="Umbrella School Option"
              required={reqs.umbrella_school_available}
              details={reqs.umbrella_school_details}
            />
          </div>
        </Card>
      )}

      {/* Guide content (plain text rendering) */}
      {guide.guide_content && (
        <Card className="space-y-4">
          <h2 className="type-title-md text-on-surface font-semibold">
            Guide
          </h2>
          <div className="space-y-3">
            {guide.guide_content.split("\n\n").map((paragraph, i) => (
              <p
                key={i}
                className="type-body-md text-on-surface-variant leading-relaxed"
              >
                {paragraph}
              </p>
            ))}
          </div>
        </Card>
      )}

      {/* Legal disclaimer */}
      {guide.legal_disclaimer && (
        <Card className="bg-warning-container/30 border border-warning">
          <div className="flex gap-3">
            <AlertTriangle className="h-5 w-5 text-warning mt-0.5 flex-shrink-0" />
            <div className="space-y-1">
              <h3 className="type-label-md text-on-surface font-semibold">
                Legal Disclaimer
              </h3>
              <p className="type-body-sm text-on-surface-variant">
                {guide.legal_disclaimer}
              </p>
            </div>
          </div>
        </Card>
      )}

      {/* Info note */}
      <div className="flex items-start gap-2 py-2">
        <Info className="h-4 w-4 text-on-surface-variant mt-0.5 flex-shrink-0" />
        <p className="type-body-sm text-on-surface-variant">
          Laws change frequently. Always verify current requirements with your
          state&rsquo;s department of education.
        </p>
      </div>
    </div>
  );
}
