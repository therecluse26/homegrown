import { useParams, Link } from "react-router";
import { Card } from "@/components/ui";
import { LearnerProfileQuiz } from "./learner-profile-quiz";
import { useProfile } from "./use-learner-profile";
import { useStudents } from "@/hooks/use-family";
import { ProfileSummary } from "./profile-summary";
import { useState } from "react";

export function LearnerProfilePage() {
  const { studentId } = useParams<{ studentId: string }>();
  const students = useStudents();
  const student = students.data?.find((s) => s.id === studentId);
  const profileQuery = useProfile(studentId);
  const [retaking, setRetaking] = useState(false);

  const studentName = student?.display_name ?? "your child";

  const hasProfile = !!profileQuery.data;

  if (hasProfile && !retaking) {
    return (
      <div className="max-w-2xl mx-auto py-6 px-4">
        <h1 className="type-headline-sm text-on-surface font-semibold mb-4">
          {studentName}'s Learning Profile
        </h1>
        <Card className="mt-4">
          <ProfileSummary
            studentName={studentName}
            summaryText={profileQuery.data!.summary_text ?? ""}
            interests={profileQuery.data!.interests ?? []}
            onRetake={() => setRetaking(true)}
            onEditInterests={() => setRetaking(true)}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto py-6 px-4">
      <h1 className="type-headline-sm text-on-surface font-semibold mb-4">
        {studentName}'s Learning Profile
      </h1>
      <div className="mb-4 rounded-lg bg-surface-container-low px-4 py-3 type-body-sm text-on-surface-variant shadow-ghost-border">
        Only your family can see this. Learner Profiles are never shared or used for ads.{" "}
        <Link to="/legal/privacy" className="underline hover:text-on-surface transition-colors">
          Learn more
        </Link>
      </div>
      <Card className="mt-4">
        {studentId && (
          <LearnerProfileQuiz
            studentId={studentId}
            onComplete={() => setRetaking(false)}
          />
        )}
      </Card>
    </div>
  );
}
