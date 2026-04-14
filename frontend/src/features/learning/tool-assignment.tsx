import { FormattedMessage, useIntl } from "react-intl";
import { Wrench, RotateCcw } from "lucide-react";
import {
  Button,
  Card,
  Checkbox,
  ConfirmationDialog,
  Icon,
  Select,
  Skeleton,
} from "@/components/ui";
import { useStudents, useStudentTools } from "@/hooks/use-family";
import { useState, useEffect, useRef, useCallback } from "react";
import { apiClient } from "@/api/client";
import { useMutation, useQueryClient } from "@tanstack/react-query";

// ─── Types ──────────────────────────────────────────────────────────────────

interface ToolConfig {
  slug: string;
  label: string;
  description: string;
  enabled: boolean;
  is_default: boolean;
}

interface UpdateToolAssignmentRequest {
  student_id: string;
  tools: { slug: string; enabled: boolean }[];
}

// ─── Mutation hook ──────────────────────────────────────────────────────────

function useUpdateToolAssignment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateToolAssignmentRequest) =>
      apiClient<void>(
        `/v1/families/students/${body.student_id}/tools`,
        { method: "PUT", body: { tools: body.tools } },
      ),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["family", "students", variables.student_id, "tools"],
      });
    },
  });
}

function useResetToolAssignment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (studentId: string) =>
      apiClient<void>(
        `/v1/families/students/${studentId}/tools/reset`,
        { method: "POST" },
      ),
    onSuccess: (_data, studentId) => {
      void queryClient.invalidateQueries({
        queryKey: ["family", "students", studentId, "tools"],
      });
    },
  });
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ToolAssignment() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const students = useStudents();
  const updateTools = useUpdateToolAssignment();
  const resetTools = useResetToolAssignment();

  const [selectedStudentId, setSelectedStudentId] = useState("");
  const [showResetDialog, setShowResetDialog] = useState(false);
  const [localTools, setLocalTools] = useState<ToolConfig[]>([]);
  const [hasChanges, setHasChanges] = useState(false);

  const studentTools = useStudentTools(selectedStudentId);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "toolAssignment.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // Auto-select first student
  useEffect(() => {
    const firstStudent = students.data?.[0];
    if (!selectedStudentId && firstStudent?.id) {
      setSelectedStudentId(firstStudent.id);
    }
  }, [students.data, selectedStudentId]);

  // Sync remote tools into local state
  useEffect(() => {
    if (studentTools.data) {
      setLocalTools(studentTools.data as ToolConfig[]);
      setHasChanges(false);
    }
  }, [studentTools.data]);

  const handleToggle = useCallback(
    (slug: string, enabled: boolean) => {
      setLocalTools((prev) =>
        prev.map((t) => (t.slug === slug ? { ...t, enabled } : t)),
      );
      setHasChanges(true);
    },
    [],
  );

  const handleSave = useCallback(() => {
    if (!selectedStudentId) return;
    updateTools.mutate({
      student_id: selectedStudentId,
      tools: localTools.map((t) => ({ slug: t.slug, enabled: t.enabled })),
    });
    setHasChanges(false);
  }, [selectedStudentId, localTools, updateTools]);

  const handleReset = useCallback(() => {
    if (!selectedStudentId) return;
    resetTools.mutate(selectedStudentId, {
      onSuccess: () => {
        setShowResetDialog(false);
      },
    });
  }, [selectedStudentId, resetTools]);

  // ─── Loading ──────────────────────────────────────────────────────────

  if (students.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  // ─── Error ────────────────────────────────────────────────────────────

  if (students.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="toolAssignment.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const studentList = students.data ?? [];

  return (
    <div className="mx-auto max-w-2xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="toolAssignment.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="toolAssignment.description" />
      </p>

      {/* Student selector */}
      <div className="mb-6">
        <label className="type-label-md text-on-surface font-medium mb-1 block">
          <FormattedMessage id="toolAssignment.student" />
        </label>
        <Select
          value={selectedStudentId}
          onChange={(e) => {
            setSelectedStudentId(e.target.value);
            setHasChanges(false);
          }}
        >
          {studentList.map((s) => (
            <option key={s.id} value={s.id}>
              {s.display_name}
            </option>
          ))}
        </Select>
      </div>

      {/* Tools list */}
      {selectedStudentId && (
        <Card>
          {studentTools.isPending ? (
            <div className="flex flex-col gap-3">
              <Skeleton height="h-10" />
              <Skeleton height="h-10" />
              <Skeleton height="h-10" />
            </div>
          ) : (
            <>
              <div className="flex flex-col gap-3" role="list">
                {localTools.map((tool) => (
                  <div
                    key={tool.slug}
                    role="listitem"
                    className="flex items-center justify-between py-2"
                  >
                    <div className="flex items-start gap-3">
                      <Icon
                        icon={Wrench}
                        size="sm"
                        className="text-on-surface-variant mt-0.5 shrink-0"
                        aria-hidden
                      />
                      <div>
                        <p className="type-title-sm text-on-surface font-medium">
                          {tool.label}
                        </p>
                        <p className="type-body-sm text-on-surface-variant">
                          {tool.description}
                        </p>
                        {tool.is_default && (
                          <span className="type-label-sm text-primary">
                            <FormattedMessage id="toolAssignment.default" />
                          </span>
                        )}
                      </div>
                    </div>
                    <Checkbox
                      checked={tool.enabled ?? false}
                      onChange={(e) =>
                        handleToggle(tool.slug, e.target.checked)
                      }
                      label={
                        tool.enabled
                          ? intl.formatMessage({
                              id: "toolAssignment.enabled",
                            })
                          : intl.formatMessage({
                              id: "toolAssignment.disabled",
                            })
                      }
                    />
                  </div>
                ))}
              </div>

              {/* Actions */}
              <div className="flex items-center justify-between mt-4 pt-4 border-t border-outline-variant/20">
                <Button
                  variant="tertiary"
                  size="sm"
                  onClick={() => setShowResetDialog(true)}
                >
                  <Icon
                    icon={RotateCcw}
                    size="xs"
                    aria-hidden
                    className="mr-1.5"
                  />
                  <FormattedMessage id="toolAssignment.reset" />
                </Button>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSave}
                  disabled={!hasChanges || updateTools.isPending}
                >
                  <FormattedMessage id="toolAssignment.saved" />
                </Button>
              </div>
            </>
          )}
        </Card>
      )}

      {/* Reset dialog */}
      <ConfirmationDialog
        open={showResetDialog}
        onClose={() => setShowResetDialog(false)}
        onConfirm={handleReset}
        title={intl.formatMessage({ id: "toolAssignment.reset" })}
        confirmLabel={intl.formatMessage({ id: "toolAssignment.reset" })}
        loading={resetTools.isPending}
      >
        <FormattedMessage id="toolAssignment.reset.confirm" />
      </ConfirmationDialog>
    </div>
  );
}
