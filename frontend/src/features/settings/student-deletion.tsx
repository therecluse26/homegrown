import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { FormattedMessage, useIntl } from "react-intl";
import { AlertTriangle } from "lucide-react";
import { Button, Card, Icon, Input, Skeleton } from "@/components/ui";
import { useStudents, useDeleteStudent } from "@/hooks/use-family";

/**
 * COPPA-compliant student deletion page.
 * Unlike account deletion, student data deletion is immediate (no grace period)
 * per COPPA requirements for child data.
 */
export function StudentDeletion() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { studentId } = useParams<{ studentId: string }>();
  const students = useStudents();
  const deleteStudent = useDeleteStudent();

  const [confirmInput, setConfirmInput] = useState("");
  const [acknowledged, setAcknowledged] = useState(false);

  const student = students.data?.find((s) => s.id === studentId);
  const studentName = student?.display_name ?? "";
  const isConfirmed =
    confirmInput.trim().toLowerCase() === studentName.toLowerCase();

  async function handleDelete() {
    if (!studentId || !isConfirmed || !acknowledged) return;
    await deleteStudent.mutateAsync(studentId);
    void navigate("/settings", { replace: true });
  }

  if (students.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-48" />
      </div>
    );
  }

  if (!student) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1 className="type-headline-md text-on-surface font-semibold mb-6">
          <FormattedMessage id="studentDeletion.notFound" />
        </h1>
        <Button variant="tertiary" onClick={() => void navigate("/settings")}>
          <FormattedMessage id="common.backToSettings" />
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="type-headline-md text-error font-semibold mb-2">
        <FormattedMessage
          id="studentDeletion.title"
          values={{ name: studentName }}
        />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="studentDeletion.description" />
      </p>

      {/* COPPA notice */}
      <Card className="bg-warning-container mb-6">
        <div className="flex items-start gap-3">
          <Icon
            icon={AlertTriangle}
            size="md"
            aria-hidden
            className="text-warning shrink-0 mt-0.5"
          />
          <div>
            <p className="type-title-sm text-on-warning-container font-semibold mb-1">
              <FormattedMessage id="studentDeletion.coppa.title" />
            </p>
            <p className="type-body-sm text-on-warning-container">
              <FormattedMessage id="studentDeletion.coppa.description" />
            </p>
          </div>
        </div>
      </Card>

      {/* What will be deleted */}
      <Card className="mb-6">
        <h2 className="type-title-sm text-on-surface font-semibold mb-3">
          <FormattedMessage id="studentDeletion.consequences.title" />
        </h2>
        <ul className="flex flex-col gap-2">
          {[
            "studentDeletion.consequence.profile",
            "studentDeletion.consequence.learning",
            "studentDeletion.consequence.sessions",
            "studentDeletion.consequence.immediate",
          ].map((id) => (
            <li
              key={id}
              className="flex items-start gap-2 type-body-sm text-on-surface-variant"
            >
              <span className="text-error mt-0.5 shrink-0">•</span>
              <FormattedMessage id={id} />
            </li>
          ))}
        </ul>
      </Card>

      {/* Confirmation */}
      <Card>
        <label className="mb-4 flex cursor-pointer select-none items-start gap-3">
          <input
            type="checkbox"
            checked={acknowledged}
            onChange={(e) => setAcknowledged(e.target.checked)}
            className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-error checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          />
          <span className="type-body-sm text-on-surface">
            <FormattedMessage id="studentDeletion.confirm.acknowledge" />
          </span>
        </label>

        <div className="mb-4">
          <p className="type-label-sm text-on-surface-variant mb-1">
            <FormattedMessage
              id="studentDeletion.confirm.typeName"
              values={{ name: studentName }}
            />
          </p>
          <Input
            value={confirmInput}
            onChange={(e) => setConfirmInput(e.target.value)}
            placeholder={studentName}
            aria-label={intl.formatMessage({
              id: "studentDeletion.confirm.typeName.label",
            })}
          />
        </div>

        {deleteStudent.error && (
          <div
            role="alert"
            aria-live="assertive"
            className="mb-4 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
          >
            <FormattedMessage id="error.generic" />
          </div>
        )}

        <div className="flex gap-3">
          <Button
            variant="tertiary"
            onClick={() => void navigate("/settings")}
          >
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            variant="primary"
            onClick={() => void handleDelete()}
            loading={deleteStudent.isPending}
            disabled={
              !isConfirmed || !acknowledged || deleteStudent.isPending
            }
            className="bg-error hover:bg-error/90"
          >
            <FormattedMessage id="studentDeletion.confirm.delete" />
          </Button>
        </div>
      </Card>
    </div>
  );
}
