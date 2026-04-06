import { useState, useMemo, useCallback, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  Plus,
  Trash2,
  Download,
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
  Tabs,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { TierGate } from "@/components/common/tier-gate";
import { useAuth } from "@/hooks/use-auth";
import {
  useTranscriptDetail,
  useUpdateTranscript,
  useAddSemester,
  useAddCourse,
  useUpdateCourse,
  useDeleteCourse,
  useGenerateTranscript,
} from "@/hooks/use-compliance";
import type {
  TranscriptSemester,
  TranscriptCourse,
  CourseLevel,
  GpaDisplay,
} from "@/hooks/use-compliance";

// ─── GPA calculation utilities ─────────────────────────────────────────────

const LETTER_TO_POINTS: Record<string, number> = {
  "A+": 4.0, A: 4.0, "A-": 3.7,
  "B+": 3.3, B: 3.0, "B-": 2.7,
  "C+": 2.3, C: 2.0, "C-": 1.7,
  "D+": 1.3, D: 1.0, "D-": 0.7,
  F: 0.0,
};

const LEVEL_WEIGHT: Record<CourseLevel, number> = {
  regular: 0,
  honors: 0.5,
  ap: 1.0,
  dual_enrollment: 1.0,
};

function calculateWeightedGPA(courses: TranscriptCourse[]): number | undefined {
  if (courses.length === 0) return undefined;
  let totalQualityPoints = 0;
  let totalCredits = 0;

  for (const course of courses) {
    const basePoints = LETTER_TO_POINTS[course.grade.toUpperCase()];
    if (basePoints === undefined) continue;
    const weightedPoints = basePoints + LEVEL_WEIGHT[course.level];
    totalQualityPoints += weightedPoints * course.credits;
    totalCredits += course.credits;
  }

  if (totalCredits === 0) return undefined;
  return totalQualityPoints / totalCredits;
}

function calculateSemesterGPA(
  courses: TranscriptCourse[],
): number | undefined {
  return calculateWeightedGPA(courses);
}

// ─── Level labels ──────────────────────────────────────────────────────────

const LEVEL_LABELS: Record<CourseLevel, string> = {
  regular: "compliance.transcript.level.regular",
  honors: "compliance.transcript.level.honors",
  ap: "compliance.transcript.level.ap",
  dual_enrollment: "compliance.transcript.level.dualEnrollment",
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
          {(Object.keys(LEVEL_LABELS) as CourseLevel[]).map((l) => (
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
          value={course.grade}
          onChange={(e) => onUpdate(course.id, "grade", e.target.value)}
          className="w-16 type-body-sm text-center"
          aria-label={intl.formatMessage({
            id: "compliance.transcript.course.grade",
          })}
        />
      </td>
      <td className="py-2 pr-2 type-label-sm text-on-surface-variant text-center whitespace-nowrap">
        {course.grade_points !== undefined
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

// ─── Semester tab content ──────────────────────────────────────────────────

function SemesterPanel({
  semester,
  studentId,
}: {
  semester: TranscriptSemester;
  studentId: string;
}) {
  const intl = useIntl();
  const addCourse = useAddCourse(studentId);
  const updateCourse = useUpdateCourse(studentId, "");
  const deleteCourse = useDeleteCourse(studentId);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const gpaLiveRef = useRef<HTMLSpanElement>(null);

  const semesterGPA = useMemo(
    () => calculateSemesterGPA(semester.courses),
    [semester.courses],
  );

  const handleAddCourse = useCallback(() => {
    addCourse.mutate({
      semester_id: semester.id,
      title: "",
      level: "regular",
      credits: 1,
      grade: "",
      sort_order: semester.courses.length,
    });
  }, [addCourse, semester.id, semester.courses.length]);

  const handleUpdateCourse = useCallback(
    (courseId: string, field: string, value: string | number) => {
      // Use the course-specific mutation
      const body: Record<string, string | number> = {};
      body[field] = value;
      updateCourse.mutate(body as never);
      // Announce GPA change
      if (gpaLiveRef.current && (field === "grade" || field === "credits" || field === "level")) {
        const newGPA = calculateWeightedGPA(
          semester.courses.map((c) =>
            c.id === courseId ? { ...c, [field]: value } : c,
          ),
        );
        if (newGPA !== undefined) {
          gpaLiveRef.current.textContent = intl.formatMessage(
            { id: "compliance.transcript.gpaUpdated" },
            { gpa: newGPA.toFixed(2) },
          );
        }
      }
    },
    [updateCourse, semester.courses, intl],
  );

  const handleDeleteCourse = useCallback(() => {
    if (deleteTarget) {
      deleteCourse.mutate(deleteTarget, {
        onSuccess: () => setDeleteTarget(null),
      });
    }
  }, [deleteTarget, deleteCourse]);

  return (
    <div>
      <span aria-live="polite" className="sr-only" ref={gpaLiveRef} />

      {/* Semester summary */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <span className="type-label-md text-on-surface-variant">
            <FormattedMessage
              id="compliance.transcript.semesterCredits"
              values={{ count: semester.semester_credits }}
            />
          </span>
          {semesterGPA !== undefined && (
            <Badge variant="primary">
              <FormattedMessage
                id="compliance.transcript.semesterGpa"
                values={{ gpa: semesterGPA.toFixed(2) }}
              />
            </Badge>
          )}
        </div>
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

      {/* Course table */}
      {semester.courses.length === 0 ? (
        <p className="type-body-md text-on-surface-variant text-center py-6">
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
            {semester.courses.map((course) => (
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
    </div>
  );
}

// ─── Main component ────────────────────────────────────────────────────────

export function TranscriptBuilder() {
  const intl = useIntl();
  const { id, studentId: routeStudentId } = useParams<{ id: string; studentId: string }>();
  const { tier } = useAuth();
  const studentId = routeStudentId ?? "";

  const { data: transcript, isPending } = useTranscriptDetail(studentId, id);
  const updateTranscript = useUpdateTranscript(studentId, id ?? "");
  const addSemester = useAddSemester(studentId, id ?? "");
  const generateTranscript = useGenerateTranscript(studentId, id ?? "");


  const [gpaDisplay, setGpaDisplay] = useState<GpaDisplay>("four_point");
  const [showGenerateConfirm, setShowGenerateConfirm] = useState(false);
  const [newSemesterName, setNewSemesterName] = useState("");
  const [showAddSemester, setShowAddSemester] = useState(false);

  // Sync GPA display from server
  const lastSyncedId = useRef<string | null>(null);
  if (transcript && transcript.id !== lastSyncedId.current) {
    lastSyncedId.current = transcript.id;
    setGpaDisplay(transcript.gpa_display);
  }

  // Calculate cumulative GPA across all semesters
  const allCourses = useMemo(() => {
    if (!transcript) return [];
    return transcript.semesters.flatMap((s) => s.courses);
  }, [transcript]);

  const cumulativeGPA = useMemo(
    () => calculateWeightedGPA(allCourses),
    [allCourses],
  );

  const handleAddSemester = useCallback(() => {
    if (!newSemesterName.trim()) return;
    addSemester.mutate(
      {
        name: newSemesterName.trim(),
        sort_order: transcript?.semesters.length ?? 0,
      },
      {
        onSuccess: () => {
          setNewSemesterName("");
          setShowAddSemester(false);
        },
      },
    );
  }, [newSemesterName, addSemester, transcript]);

  const handleGpaDisplayChange = useCallback(
    (display: GpaDisplay) => {
      setGpaDisplay(display);
      updateTranscript.mutate({ gpa_display: display });
    },
    [updateTranscript],
  );

  const handleGenerate = useCallback(() => {
    generateTranscript.mutate(undefined, {
      onSuccess: () => setShowGenerateConfirm(false),
    });
  }, [generateTranscript]);

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

  const semesterTabs = transcript.semesters.map((s) => ({
    id: s.id,
    label: s.name,
    content: (
      <SemesterPanel
        semester={s}
        studentId={studentId}
      />
    ),
  }));

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

        {/* GPA display toggle */}
        <div className="flex items-center gap-2">
          <span className="type-label-md text-on-surface-variant">
            <FormattedMessage id="compliance.transcript.form.gpaDisplay" />:
          </span>
          <div className="flex bg-surface-container-low rounded-radius-sm">
            {(["four_point", "percentage", "pass_fail"] as GpaDisplay[]).map(
              (d) => (
                <button
                  key={d}
                  onClick={() => handleGpaDisplayChange(d)}
                  className={`px-2 py-1 type-label-sm rounded-radius-sm transition-colors ${
                    gpaDisplay === d
                      ? "bg-primary text-on-primary"
                      : "text-on-surface-variant hover:bg-surface-container-high"
                  }`}
                >
                  <FormattedMessage
                    id={`compliance.transcript.gpa.${d === "four_point" ? "fourPoint" : d === "percentage" ? "percentage" : "passFail"}`}
                  />
                </button>
              ),
            )}
          </div>
        </div>
      </div>

      {/* Cumulative GPA */}
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
                {cumulativeGPA !== undefined
                  ? cumulativeGPA.toFixed(2)
                  : "—"}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3 type-body-sm text-on-surface-variant">
            <span>
              <FormattedMessage
                id="compliance.transcript.totalCredits"
                values={{ count: transcript.total_credits }}
              />
            </span>
            <span>
              <FormattedMessage
                id="compliance.transcript.totalSemesters"
                values={{ count: transcript.semesters.length }}
              />
            </span>
          </div>
        </div>
      </Card>

      {/* Semester tabs */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="type-title-sm text-on-surface">
            <FormattedMessage id="compliance.transcript.semesters.title" />
          </h3>
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setShowAddSemester(true)}
          >
            <Icon icon={Plus} size="sm" className="mr-1" />
            <FormattedMessage id="compliance.transcript.addSemester" />
          </Button>
        </div>

        {/* Add semester inline form */}
        {showAddSemester && (
          <div className="flex items-end gap-2 mb-4 p-3 bg-surface-container-low rounded-radius-sm">
            <div className="flex-1">
              <label
                htmlFor="new-semester"
                className="type-label-md text-on-surface block mb-1"
              >
                <FormattedMessage id="compliance.transcript.semesterName" />
              </label>
              <Input
                id="new-semester"
                value={newSemesterName}
                onChange={(e) => setNewSemesterName(e.target.value)}
                placeholder={intl.formatMessage({
                  id: "compliance.transcript.semesterName.placeholder",
                })}
              />
            </div>
            <Button
              variant="primary"
              size="sm"
              onClick={handleAddSemester}
              disabled={!newSemesterName.trim() || addSemester.isPending}
            >
              <FormattedMessage id="common.add" />
            </Button>
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => {
                setShowAddSemester(false);
                setNewSemesterName("");
              }}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
          </div>
        )}

        {transcript.semesters.length === 0 ? (
          <p className="type-body-md text-on-surface-variant text-center py-8">
            <FormattedMessage id="compliance.transcript.noSemesters" />
          </p>
        ) : (
          <Tabs tabs={semesterTabs} />
        )}
      </Card>

      {/* Action bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {transcript.download_url && (
            <a
              href={transcript.download_url}
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button variant="tertiary" size="sm">
                <Icon icon={Download} size="sm" className="mr-1" />
                <FormattedMessage id="compliance.transcript.download" />
              </Button>
            </a>
          )}
        </div>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowGenerateConfirm(true)}
          disabled={
            transcript.semesters.length === 0 ||
            generateTranscript.isPending
          }
        >
          <Icon icon={FileText} size="sm" className="mr-1" />
          <FormattedMessage id="compliance.transcript.generate" />
        </Button>
      </div>

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
