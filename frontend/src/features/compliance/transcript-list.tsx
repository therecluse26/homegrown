import { useState, useCallback, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink, useNavigate } from "react-router";
import { Plus, GraduationCap, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Select,
  Badge,
  Input,
  EmptyState,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { TierGate } from "@/components/common/tier-gate";
import { useAuth } from "@/hooks/use-auth";
import { useStudents } from "@/hooks/use-family";
import {
  useTranscripts,
  useCreateTranscript,
  useDeleteTranscript,
} from "@/hooks/use-compliance";
import type { TranscriptSummary, GpaDisplay } from "@/hooks/use-compliance";

// ─── Status badge ──────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: TranscriptSummary["status"] }) {
  const variant =
    status === "ready" ? "primary" : status === "generating" ? "secondary" : undefined;
  return (
    <Badge variant={variant}>
      <FormattedMessage id={`compliance.transcript.status.${status}`} />
    </Badge>
  );
}

// ─── Transcript card ───────────────────────────────────────────────────────

function TranscriptCard({
  transcript,
  studentId,
  onDelete,
}: {
  transcript: TranscriptSummary;
  studentId: string;
  onDelete: (id: string) => void;
}) {
  const intl = useIntl();

  return (
    <Card className="p-card-padding">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <RouterLink
              to={`/compliance/transcripts/${studentId}/${transcript.id}`}
              className="type-title-sm text-on-surface font-semibold hover:text-primary transition-colors"
            >
              {transcript.title}
            </RouterLink>
            <StatusBadge status={transcript.status} />
          </div>
          {transcript.grade_levels.length > 0 && (
            <p className="type-body-sm text-on-surface-variant">
              {transcript.grade_levels.join(", ")}
            </p>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={() => onDelete(transcript.id)}
            className="p-2 rounded-radius-sm text-on-surface-variant hover:bg-error-container hover:text-on-error-container transition-colors touch-target"
            aria-label={intl.formatMessage(
              { id: "compliance.transcript.delete.label" },
              { name: transcript.title },
            )}
          >
            <Icon icon={Trash2} size="sm" />
          </button>
        </div>
      </div>
    </Card>
  );
}

// ─── Create form ───────────────────────────────────────────────────────────

function CreateTranscriptForm({
  students,
  onClose,
}: {
  students: { id: string; display_name: string }[];
  onClose: () => void;
}) {
  const intl = useIntl();
  const navigate = useNavigate();
  const createTranscript = useCreateTranscript();

  const [title, setTitle] = useState("");
  const [studentId, setStudentId] = useState(students[0]?.id ?? "");
  const [gpaDisplay, setGpaDisplay] = useState<GpaDisplay>("four_point");

  const canSubmit = title.trim() && studentId;

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!canSubmit) return;
      createTranscript.mutate(
        {
          student_id: studentId,
          title: title.trim(),
          gpa_display: gpaDisplay,
        },
        {
          onSuccess: (data) => {
            navigate(`/compliance/transcripts/${studentId}/${data.id}`);
          },
        },
      );
    },
    [canSubmit, studentId, title, gpaDisplay, createTranscript, navigate],
  );

  return (
    <Card className="p-card-padding mb-6">
      <h2 className="type-title-sm text-on-surface mb-4">
        <FormattedMessage id="compliance.transcript.create.title" />
      </h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label
              htmlFor="transcript-title"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.transcript.form.title" />
            </label>
            <Input
              id="transcript-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={intl.formatMessage({
                id: "compliance.transcript.form.title.placeholder",
              })}
            />
          </div>
          <div>
            <label
              htmlFor="transcript-student"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.portfolio.form.student" />
            </label>
            <Select
              id="transcript-student"
              value={studentId}
              onChange={(e) => setStudentId(e.target.value)}
            >
              {students.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.display_name}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <label
              htmlFor="transcript-gpa"
              className="type-label-md text-on-surface block mb-1"
            >
              <FormattedMessage id="compliance.transcript.form.gpaDisplay" />
            </label>
            <Select
              id="transcript-gpa"
              value={gpaDisplay}
              onChange={(e) => setGpaDisplay(e.target.value as GpaDisplay)}
            >
              <option value="four_point">
                {intl.formatMessage({ id: "compliance.transcript.gpa.fourPoint" })}
              </option>
              <option value="percentage">
                {intl.formatMessage({ id: "compliance.transcript.gpa.percentage" })}
              </option>
              <option value="pass_fail">
                {intl.formatMessage({ id: "compliance.transcript.gpa.passFail" })}
              </option>
            </Select>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="tertiary" size="sm" onClick={onClose}>
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={!canSubmit || createTranscript.isPending}
          >
            <FormattedMessage id="compliance.transcript.create.submit" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

// ─── Main component ────────────────────────────────────────────────────────

export function TranscriptList() {
  const intl = useIntl();
  const { tier } = useAuth();
  const [showCreate, setShowCreate] = useState(false);
  const [studentFilter, setStudentFilter] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const { data: students } = useStudents();
  const { data: transcripts, isPending } = useTranscripts(studentFilter);
  const deleteTranscript = useDeleteTranscript(studentFilter);

  // Auto-select first student
  useEffect(() => {
    const first = students?.[0];
    if (first?.id && !studentFilter) setStudentFilter(first.id);
  }, [students, studentFilter]);

  const handleDelete = useCallback(() => {
    if (!deleteTarget) return;
    deleteTranscript.mutate(deleteTarget, {
      onSuccess: () => setDeleteTarget(null),
    });
  }, [deleteTarget, deleteTranscript]);

  if (tier === "free") {
    return <TierGate featureName="Transcript Builder" />;
  }

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "compliance.transcript.pageTitle" })}
      />

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-3">
          {students && students.length > 1 && (
            <Select
              value={studentFilter}
              onChange={(e) =>
                setStudentFilter(e.target.value)
              }
              className="w-40"
              aria-label={intl.formatMessage({
                id: "compliance.transcript.studentFilter",
              })}
            >
              <option value="">
                {intl.formatMessage({ id: "planning.export.allStudents" })}
              </option>
              {students.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.display_name}
                </option>
              ))}
            </Select>
          )}
        </div>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowCreate(true)}
        >
          <Icon icon={Plus} size="sm" className="mr-1" />
          <FormattedMessage id="compliance.transcript.create" />
        </Button>
      </div>

      {/* Create form */}
      {showCreate && students && students.length > 0 && (
        <CreateTranscriptForm
          students={students
            .filter((s): s is typeof s & { id: string; display_name: string } =>
              !!s.id && !!s.display_name
            )}
          onClose={() => setShowCreate(false)}
        />
      )}

      {/* Transcript list */}
      {isPending ? (
        <div className="space-y-3">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-20 w-full rounded-radius-md" />
          ))}
        </div>
      ) : !transcripts || transcripts.length === 0 ? (
        <EmptyState
          illustration={<Icon icon={GraduationCap} size="xl" />}
          message={intl.formatMessage({
            id: "compliance.transcript.empty",
          })}
          description={intl.formatMessage({
            id: "compliance.transcript.empty.description",
          })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowCreate(true)}
            >
              <Icon icon={Plus} size="sm" className="mr-1" />
              <FormattedMessage id="compliance.transcript.create" />
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {transcripts.map((t) => (
            <TranscriptCard
              key={t.id}
              transcript={t}
              studentId={studentFilter}
              onDelete={setDeleteTarget}
            />
          ))}
        </div>
      )}

      {/* Delete confirmation */}
      <ConfirmationDialog
        open={!!deleteTarget}
        onConfirm={handleDelete}
        onClose={() => setDeleteTarget(null)}
        title={intl.formatMessage({ id: "compliance.transcript.delete.title" })}
        confirmLabel={intl.formatMessage({
          id: "compliance.transcript.delete.confirm",
        })}
        destructive
      >
        {intl.formatMessage({
          id: "compliance.transcript.delete.description",
        })}
      </ConfirmationDialog>
    </div>
  );
}
