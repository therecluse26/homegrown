import { useParams, Link } from "react-router";
import { Card, Skeleton } from "@/components/ui";
import { LearnerProfileQuiz } from "./learner-profile-quiz";
import { useProfile } from "./use-learner-profile";
import { useStudents } from "@/hooks/use-family";
import { ProfileSummary } from "./profile-summary";
import { useState } from "react";

// Q1-Q12 are scored questions; Q13 is the interest question (index 12).
const INTEREST_STEP = 12;

export function LearnerProfilePage() {
  const { studentId } = useParams<{ studentId: string }>();
  const students = useStudents();
  const student = students.data?.find((s) => s.id === studentId);
  const profileQuery = useProfile(studentId);
  const [retaking, setRetaking] = useState(false);
  const [startAtStep, setStartAtStep] = useState(0);

  const studentName = student?.display_name ?? "your child";

  const hasProfile = !!profileQuery.data;

  const customAttrs = Object.entries(student?.custom_attributes ?? {});

  // 3.2: Loading guard — prevent quiz flash while profile loads
  if (profileQuery.isPending) {
    return (
      <div className="max-w-2xl mx-auto py-6 px-6">
        <Skeleton height="h-8" width="w-64" className="mb-6" />
        <Card><Skeleton height="h-48" /></Card>
      </div>
    );
  }

  if (hasProfile && !retaking) {
    return (
      <div className="max-w-2xl mx-auto py-6 px-6">
        <h1 className="type-headline-sm text-on-surface font-semibold mb-4">
          {studentName}'s Learning Profile
        </h1>
        <Card className="mt-4">
          <ProfileSummary
            studentName={studentName}
            summaryText={profileQuery.data!.summary_text ?? ""}
            interests={profileQuery.data!.interests ?? []}
            dimensions={{
              activity_format: profileQuery.data!.activity_format,
              session_length: profileQuery.data!.session_length,
              motivation: profileQuery.data!.motivation,
              solo_collaborative: profileQuery.data!.solo_collaborative,
              structure: profileQuery.data!.structure,
              outdoor_kinesthetic: profileQuery.data!.outdoor_kinesthetic,
            }}
            onRetake={() => {
              setStartAtStep(0);
              setRetaking(true);
            }}
            onEditInterests={() => {
              setStartAtStep(INTEREST_STEP);
              setRetaking(true);
            }}
          />
        </Card>
        {customAttrs.length > 0 && (
          <Card className="mt-4">
            <h2 className="type-title-sm text-on-surface font-semibold mb-3">
              Custom Attributes
            </h2>
            <dl className="flex flex-col gap-2">
              {customAttrs.map(([k, v]) => (
                <div key={k} className="flex gap-2">
                  <dt className="type-label-sm text-on-surface-variant min-w-28 shrink-0">
                    {k}
                  </dt>
                  <dd className="type-body-sm text-on-surface">{String(v)}</dd>
                </div>
              ))}
            </dl>
          </Card>
        )}
      </div>
    );
  }

  // 3.5: px-6 to match --spacing-page-x (was px-4)
  return (
    <div className="max-w-2xl mx-auto py-6 px-6">
      <h1 className="type-headline-sm text-on-surface font-semibold mb-4">
        {studentName}'s Learning Profile
      </h1>
      {/* 3.3: Privacy note kept only in ProfileSummary post-quiz state. Single inline sentence here. */}
      <p className="type-body-sm text-on-surface-variant mb-4">
        Only your family can see this.{" "}
        <Link to="/legal/privacy" className="underline hover:text-on-surface transition-colors">
          Learn more
        </Link>
      </p>
      <Card className="mt-4">
        {studentId && (
          <LearnerProfileQuiz
            studentId={studentId}
            onComplete={() => setRetaking(false)}
            startAtStep={startAtStep}
          />
        )}
      </Card>
    </div>
  );
}
