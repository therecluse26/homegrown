import { useState, useMemo, useCallback, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  Plus,
  Trash2,
  FileText,
  GraduationCap,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Select,
  Badge,
  Input,
  ConfirmationDialog,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { TierGate } from "@/components/common/tier-gate";
import { useAuth } from "@/hooks/use-auth";
import {
  useTranscriptDetail,
  useAddCourse,
  useUpdateCourse,
  useDeleteCourse,
  useGenerateTranscript,
} from "@/hooks/use-compliance";
import type { TranscriptCourse } from "@/hooks/use-compliance";

// ─── Level labels ──────────────────────────────────────────────────────────

const LEVEL_LABELS: Record<string, string> = {
  regular: "compliance.transcript.level.regular",
  honors: "compliance.transcript.level.honors",
  ap: "compliance.transcript.level.ap",
};

// ─── Course row ────────────────────────────────────────────────────────────

function CourseRow({
  course,
  onUpdate,
  onDelete,
}: {
  course: TranscriptCourse;
  onUpdate: (courseId: string, field: string, value: string | number) => void;
  onDelete: (courseId: string) => void;
}) {
  const intl = useIntl();

  return (
    <tr className="border-b border-outline-variant/10 last:border-b-0">
      <td className="py-2 pr-2">
        <Input
          value={course.title}
          onChange={(e) => onUpdate(course.id, "title", e.target.value)}
          className="type-body-sm"
          aria-label={intl.formatMessage({
            id: "compliance.transcript.course.title",
          })}
        />
      </td>
      <td className="py-2 pr-2">
        <Select
          value={course.level}
          onChange={(e) => onUpdate(course.id, "level", e.target.value)}
          className="type-label-sm"
          aria-label={intl.formatMessage({
            id: "compliance.transcript.course.level",
          })}
        >
          {Object.keys(LEVEL_LABELS).map((l) => (
            <option key={l} value={l}>
              {intl.formatMessage({ id: LEVEL_LABELS[l] })}
            </option>
          ))}
        </Select>
      </td>
      <td className="py-2 pr-2">
        <Input
          type="number"
          value={String(course.credits)}
          onChange={(e) =>
            onUpdate(course.id, "credits", Number(e.target.value) || 0)
          }
          className="w-16 type-body-sm text-center"
          min={0}
          step={0.5}
          aria-label={intl.formatMessage({
            id: "compliance.transcript.course.credits",
          })}
        />
      </td>
      <td className="py-2 pr-2">
        <Input
          value={course.grade_letter ?? ""}
          onChange={(e) => onUpdate(course.id, "grade_letter", e.target.value)}
          className="w-16 type-body-sm text-center"
          aria-label={intl.formatMessage({
            id: "compliance.transcript.course.grade",
          })}
        />
      </td>
      <td className="py-2 pr-2 type-label-sm text-on-surface-variant text-center whitespace-nowrap">
        {course.grade_points != null
          ? course.grade_points.toFixed(2)
          : "—"}
      </td>
      <td className="py-2 text-center">
        <button
          onClick={() => onDelete(course.id)}
          className="p-1 rounded-radius-sm text-on-surface-variant hover:bg-error-container hover:text-on-error-container transition-colors touch-target"
          aria-label={intl.formatMessage(
            { id: "compliance.transcript.course.remove" },
            { title: course.title },
          )}
        >
          <Icon icon={Trash2} size="xs" />
        </button>
      </td>
    </tr>
  );
}

// ─── Main component ────────────────────────────────────────────────────────

export function TranscriptBuilder() {
  const intl = useIntl();
  const { id, studentId: routeStudentId } = useParams<{ id: string; studentId: string }>();
  const { tier } = useAuth();
  const studentId = routeStudentId ?? "";

  const { data: transcript, isPending } = useTranscriptDetail(studentId, id);
  const addCourse = useAddCourse(studentId);
  const updateCourse = useUpdateCourse(studentId);
  const deleteCourse = useDeleteCourse(studentId);
  const generateTranscript = useGenerateTranscript(studentId, id ?? "");

  const [showGenerateConfirm, setShowGenerateConfirm] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [generateError, setGenerateError] = useState<string | null>(null);
  const gpaLiveRef = useRef<HTMLSpanElement>(null);

  const courses = useMemo(() => transcript?.courses ?? [], [transcript]);

  const totalCredits = useMemo(
    () => courses.reduce((sum, c) => sum + c.credits, 0),
    [courses],
  );

  const handleAddCourse = useCallback(() => {
    addCourse.mutate({
      title: "New Course",
      subject: "General",
      grade_level: 9,
      level: "regular",
      credits: 1,
      school_year: new Date().getFullYear().toString(),
    });
  }, [addCourse]);

  const handleUpdateCourse = useCallback(
    (courseId: string, field: string, value: string | number) => {
      const body: Record<string, string | number> = { courseId: courseId };
      body[field] = value;
      updateCourse.mutate(body as never);
    },
    [updateCourse],
  );

  const handleDeleteCourse = useCallback(() => {
    if (deleteTarget) {
      deleteCourse.mutate(deleteTarget, {
        onSuccess: () => setDeleteTarget(null),
      });
    }
  }, [deleteTarget, deleteCourse]);

  const handleGenerate = useCallback(() => {
    setGenerateError(null);
    const timeout = setTimeout(() => {
      setGenerateError(
        intl.formatMessage({
          id: "compliance.transcript.generate.timeout",
          defaultMessage:
            "PDF generation is taking longer than expected. Please try again.",
        }),
      );
    }, 30_000);

    generateTranscript.mutate(undefined, {
      onSuccess: () => {
        clearTimeout(timeout);
        setShowGenerateConfirm(false);
        setGenerateError(null);
      },
      onError: () => {
        clearTimeout(timeout);
        setShowGenerateConfirm(false);
        setGenerateError(
          intl.formatMessage({
            id: "compliance.transcript.generate.error",
            defaultMessage:
              "Failed to generate transcript PDF. Please try again later.",
          }),
        );
      },
    });
  }, [generateTranscript, intl]);

  if (tier === "free") {
    return <TierGate featureName="Transcript Builder" />;
  }

  if (isPending) {
    return (
      <div className="max-w-content mx-auto">
        <Skeleton className="h-8 w-48 rounded-radius-sm mb-4" />
        <Skeleton className="h-40 w-full rounded-radius-md mb-4" />
        <Skeleton className="h-60 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!transcript) {
    return (
      <div className="max-w-content mx-auto">
        <PageTitle
          title={intl.formatMessage({ id: "compliance.transcript.notFound" })}
        />
        <Card className="p-card-padding text-center">
          <p className="type-body-md text-on-surface-variant py-8">
            <FormattedMessage id="compliance.transcript.notFound" />
          </p>
          <RouterLink to="/compliance/transcripts">
            <Button variant="primary" size="sm">
              <FormattedMessage id="compliance.transcript.backToList" />
            </Button>
          </RouterLink>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-content mx-auto">
      <PageTitle
        title={intl.formatMessage(
          { id: "compliance.transcript.builder.pageTitle" },
          { name: transcript.title },
        )}
      />

      {/* Header */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-6">
        <div className="flex items-center gap-3">
          <RouterLink to="/compliance/transcripts">
            <Button variant="tertiary" size="sm">
              <Icon icon={ArrowLeft} size="sm" className="mr-1" />
              <FormattedMessage id="compliance.transcript.backToList" />
            </Button>
          </RouterLink>
          <h2 className="type-title-md text-on-surface font-semibold">
            {transcript.title}
          </h2>
          <span className="type-body-sm text-on-surface-variant">
            {transcript.student_name}
          </span>
        </div>
      </div>

      <span aria-live="polite" className="sr-only" ref={gpaLiveRef} />

      {/* GPA summary */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Icon
              icon={GraduationCap}
              size="lg"
              className="text-primary"
            />
            <div>
              <p className="type-label-md text-on-surface-variant">
                <FormattedMessage id="compliance.transcript.cumulativeGpa" />
              </p>
              <p
                className="type-headline-md text-on-surface font-bold"
                aria-live="polite"
              >
                {transcript.gpa_weighted != null
                  ? transcript.gpa_weighted.toFixed(2)
                  : "—"}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3 type-body-sm text-on-surface-variant">
            <span>
              <FormattedMessage
                id="compliance.transcript.totalCredits"
                values={{ count: totalCredits }}
              />
            </span>
            <span>
              <FormattedMessage
                id="compliance.transcript.totalCourses"
                values={{ count: courses.length }}
              />
            </span>
            <Badge>
              {transcript.status}
            </Badge>
          </div>
        </div>
      </Card>

      {/* Course table */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="type-title-sm text-on-surface">
            <FormattedMessage id="compliance.transcript.courses" />
          </h3>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleAddCourse}
            disabled={addCourse.isPending}
          >
            <Icon icon={Plus} size="sm" className="mr-1" />
            <FormattedMessage id="compliance.transcript.addCourse" />
          </Button>
        </div>

        {courses.length === 0 ? (
          <p className="type-body-md text-on-surface-variant text-center py-8">
            <FormattedMessage id="compliance.transcript.noCourses" />
          </p>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="type-label-sm text-on-surface-variant uppercase tracking-wide">
                <th className="text-left pb-2 pr-2 font-medium">
                  <FormattedMessage id="compliance.transcript.course.title" />
                </th>
                <th className="text-left pb-2 pr-2 font-medium">
                  <FormattedMessage id="compliance.transcript.course.level" />
                </th>
                <th className="text-center pb-2 pr-2 font-medium">
                  <FormattedMessage id="compliance.transcript.course.credits" />
                </th>
                <th className="text-center pb-2 pr-2 font-medium">
                  <FormattedMessage id="compliance.transcript.course.grade" />
                </th>
                <th className="text-center pb-2 pr-2 font-medium">
                  <FormattedMessage id="compliance.transcript.course.points" />
                </th>
                <th className="text-center pb-2 font-medium w-10" />
              </tr>
            </thead>
            <tbody>
              {courses.map((course) => (
                <CourseRow
                  key={course.id}
                  course={course}
                  onUpdate={handleUpdateCourse}
                  onDelete={setDeleteTarget}
                />
              ))}
            </tbody>
          </table>
        )}
      </Card>

      {/* Generate error */}
      {generateError && (
        <div
          role="alert"
          className="rounded-radius-md bg-error-container px-4 py-3 type-body-sm text-on-error-container mb-4"
        >
          {generateError}
        </div>
      )}

      {/* Action bar */}
      <div className="flex items-center justify-end">
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowGenerateConfirm(true)}
          disabled={
            courses.length === 0 ||
            generateTranscript.isPending
          }
        >
          <Icon icon={FileText} size="sm" className="mr-1" />
          {generateTranscript.isPending ? (
            <FormattedMessage
              id="compliance.transcript.generating"
              defaultMessage="Generating..."
            />
          ) : (
            <FormattedMessage id="compliance.transcript.generate" />
          )}
        </Button>
      </div>

      {/* Delete course confirmation */}
      <ConfirmationDialog
        open={!!deleteTarget}
        onConfirm={handleDeleteCourse}
        onClose={() => setDeleteTarget(null)}
        title={intl.formatMessage({
          id: "compliance.transcript.course.deleteTitle",
        })}
        confirmLabel={intl.formatMessage({
          id: "compliance.transcript.course.deleteConfirm",
        })}
        destructive
      >
        {intl.formatMessage({
          id: "compliance.transcript.course.deleteDescription",
        })}
      </ConfirmationDialog>

      {/* Generate confirmation */}
      <ConfirmationDialog
        open={showGenerateConfirm}
        onConfirm={handleGenerate}
        onClose={() => setShowGenerateConfirm(false)}
        title={intl.formatMessage({
          id: "compliance.transcript.generate.title",
        })}
        confirmLabel={intl.formatMessage({
          id: "compliance.transcript.generate.confirm",
        })}
      >
        {intl.formatMessage({
          id: "compliance.transcript.generate.description",
        })}
      </ConfirmationDialog>
    </div>
  );
}
