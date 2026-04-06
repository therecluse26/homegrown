import { FormattedMessage, useIntl } from "react-intl";
import {
  FolderKanban,
  Plus,
  Trash2,
  CheckCircle2,
  Circle,
} from "lucide-react";
import {
  Button,
  Card,
  ConfirmationDialog,
  EmptyState,
  Icon,
  Input,
  ProgressBar,
  Select,
  Skeleton,
} from "@/components/ui";
import { FormField } from "@/components/ui/form-field";
import {
  useProjects,
  useCreateProject,
  useUpdateProject,
  useDeleteProject,
  useAddMilestone,
  useToggleMilestone,
  type ProjectStatus,
} from "@/hooks/use-projects";
import { useStudents } from "@/hooks/use-family";
import { useState, useEffect, useRef, useCallback } from "react";

// ─── Component ─────────────────────────────────────────────────────────────

export function Projects() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const students = useStudents();
  const [studentFilter, setStudentFilter] = useState("");
  const projects = useProjects(studentFilter);
  const createProject = useCreateProject();
  const updateProject = useUpdateProject();
  const deleteProject = useDeleteProject(studentFilter);
  const addMilestone = useAddMilestone(studentFilter);
  const toggleMilestone = useToggleMilestone(studentFilter);

  // Auto-select first student
  useEffect(() => {
    const first = students.data?.[0];
    if (first?.id && !studentFilter) setStudentFilter(first.id);
  }, [students.data, studentFilter]);

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [newMilestoneProject, setNewMilestoneProject] = useState<string | null>(
    null,
  );
  const [newMilestoneTitle, setNewMilestoneTitle] = useState("");

  // Create form state
  const [formTitle, setFormTitle] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formStudentId, setFormStudentId] = useState("");
  const [formDueDate, setFormDueDate] = useState("");

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "projects.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const handleCreate = useCallback(() => {
    if (!formTitle.trim() || !formStudentId) return;
    createProject.mutate(
      {
        title: formTitle.trim(),
        description: formDescription.trim(),
        student_id: formStudentId,
        due_date: formDueDate || undefined,
      },
      {
        onSuccess: () => {
          setShowCreateForm(false);
          setFormTitle("");
          setFormDescription("");
          setFormStudentId("");
          setFormDueDate("");
        },
      },
    );
  }, [formTitle, formDescription, formStudentId, formDueDate, createProject]);

  const handleAddMilestone = useCallback(
    (projectId: string) => {
      if (!newMilestoneTitle.trim()) return;
      addMilestone.mutate(
        { projectId, title: newMilestoneTitle.trim() },
        {
          onSuccess: () => {
            setNewMilestoneProject(null);
            setNewMilestoneTitle("");
          },
        },
      );
    },
    [newMilestoneTitle, addMilestone],
  );

  const handleStatusChange = useCallback(
    (id: string, status: ProjectStatus) => {
      updateProject.mutate({ id, studentId: studentFilter, status });
    },
    [updateProject, studentFilter],
  );

  // ─── Loading ──────────────────────────────────────────────────────────

  if (projects.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <div className="flex flex-col gap-4">
          <Skeleton height="h-32" />
          <Skeleton height="h-32" />
        </div>
      </div>
    );
  }

  // ─── Error ────────────────────────────────────────────────────────────

  if (projects.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="projects.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const projectList = projects.data ?? [];
  const studentList = students.data ?? [];

  return (
    <div className="mx-auto max-w-3xl">
      <div className="flex items-center justify-between mb-2">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          <FormattedMessage id="projects.title" />
        </h1>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowCreateForm(true)}
        >
          <Icon icon={Plus} size="xs" aria-hidden className="mr-1.5" />
          <FormattedMessage id="projects.create" />
        </Button>
      </div>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="projects.description" />
      </p>

      {/* Create form */}
      {showCreateForm && (
        <Card className="mb-6">
          <h2 className="type-title-md text-on-surface font-semibold mb-4">
            <FormattedMessage id="projects.create" />
          </h2>
          <div className="flex flex-col gap-3">
            <FormField
              label={intl.formatMessage({ id: "projects.form.title" })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  value={formTitle}
                  onChange={(e) => setFormTitle(e.target.value)}
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({ id: "projects.form.description" })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  value={formDescription}
                  onChange={(e) => setFormDescription(e.target.value)}
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({ id: "projects.form.student" })}
            >
              {({ id }) => (
                <Select
                  id={id}
                  value={formStudentId}
                  onChange={(e) => setFormStudentId(e.target.value)}
                >
                  <option value="">
                    {intl.formatMessage({
                      id: "projects.form.student",
                    })}
                  </option>
                  {studentList.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.display_name}
                    </option>
                  ))}
                </Select>
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({ id: "projects.form.dueDate" })}
            >
              {({ id }) => (
                <input
                  id={id}
                  type="date"
                  value={formDueDate}
                  onChange={(e) => setFormDueDate(e.target.value)}
                  className="type-body-md text-on-surface bg-surface-container-highest px-3 py-2 rounded-radius-sm w-full"
                />
              )}
            </FormField>
            <div className="flex gap-2 justify-end">
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => setShowCreateForm(false)}
              >
                <FormattedMessage id="action.cancel" />
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={handleCreate}
                disabled={
                  !formTitle.trim() ||
                  !formStudentId ||
                  createProject.isPending
                }
              >
                <FormattedMessage id="projects.form.submit" />
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Project list */}
      {projectList.length === 0 && !showCreateForm ? (
        <EmptyState
          message={intl.formatMessage({ id: "projects.empty" })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowCreateForm(true)}
            >
              <FormattedMessage id="projects.create" />
            </Button>
          }
        />
      ) : (
        <ul className="flex flex-col gap-4" role="list">
          {projectList.map((project) => {
            const completedCount = project.milestones.filter(
              (m) => m.completed,
            ).length;
            const totalMilestones = project.milestones.length;
            const progress =
              totalMilestones > 0 ? (completedCount / totalMilestones) * 100 : 0;

            return (
              <li key={project.id}>
                <Card>
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-start gap-3">
                      <Icon
                        icon={FolderKanban}
                        size="md"
                        className="text-primary mt-0.5 shrink-0"
                        aria-hidden
                      />
                      <div>
                        <h3 className="type-title-md text-on-surface font-semibold">
                          {project.title}
                        </h3>
                        <p className="type-body-sm text-on-surface-variant">
                          {project.student_name}
                          {project.due_date && (
                            <>
                              {" · "}
                              {intl.formatDate(project.due_date, {
                                month: "short",
                                day: "numeric",
                                year: "numeric",
                              })}
                            </>
                          )}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Select
                        value={project.status}
                        onChange={(e) =>
                          handleStatusChange(
                            project.id,
                            e.target.value as ProjectStatus,
                          )
                        }
                      >
                        <option value="planning">
                          {intl.formatMessage({
                            id: "projects.status.planning",
                          })}
                        </option>
                        <option value="in_progress">
                          {intl.formatMessage({
                            id: "projects.status.in_progress",
                          })}
                        </option>
                        <option value="completed">
                          {intl.formatMessage({
                            id: "projects.status.completed",
                          })}
                        </option>
                      </Select>
                      <Button
                        variant="tertiary"
                        size="sm"
                        onClick={() => setDeleteTarget(project.id)}
                        className="text-error"
                      >
                        <Icon icon={Trash2} size="xs" aria-hidden />
                      </Button>
                    </div>
                  </div>

                  {project.description && (
                    <p className="type-body-sm text-on-surface-variant mb-3 ml-8">
                      {project.description}
                    </p>
                  )}

                  {/* Progress bar */}
                  {totalMilestones > 0 && (
                    <div className="ml-8 mb-3">
                      <ProgressBar value={progress} />
                      <p className="type-label-sm text-on-surface-variant mt-1">
                        <FormattedMessage
                          id="projects.progress"
                          values={{
                            completed: completedCount,
                            total: totalMilestones,
                          }}
                        />
                      </p>
                    </div>
                  )}

                  {/* Milestones */}
                  <div className="ml-8">
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="type-label-md text-on-surface font-medium">
                        <FormattedMessage id="projects.milestones" />
                      </h4>
                      <Button
                        variant="tertiary"
                        size="sm"
                        onClick={() =>
                          setNewMilestoneProject(project.id)
                        }
                      >
                        <Icon
                          icon={Plus}
                          size="xs"
                          aria-hidden
                          className="mr-1"
                        />
                        <FormattedMessage id="projects.milestones.add" />
                      </Button>
                    </div>

                    {project.milestones.length === 0 ? (
                      <p className="type-body-sm text-on-surface-variant">
                        <FormattedMessage id="projects.milestones.empty" />
                      </p>
                    ) : (
                      <ul className="flex flex-col gap-1.5" role="list">
                        {project.milestones.map((milestone) => (
                          <li
                            key={milestone.id}
                            className="flex items-center gap-2"
                          >
                            <button
                              type="button"
                              onClick={() =>
                                toggleMilestone.mutate({
                                  projectId: project.id,
                                  milestoneId: milestone.id,
                                  completed: !milestone.completed,
                                })
                              }
                              className="shrink-0 touch-target"
                              aria-label={intl.formatMessage({
                                id: "projects.milestones.complete",
                              })}
                            >
                              <Icon
                                icon={
                                  milestone.completed
                                    ? CheckCircle2
                                    : Circle
                                }
                                size="sm"
                                className={
                                  milestone.completed
                                    ? "text-primary"
                                    : "text-on-surface-variant"
                                }
                                aria-hidden
                              />
                            </button>
                            <span
                              className={`type-body-sm ${
                                milestone.completed
                                  ? "text-on-surface-variant line-through"
                                  : "text-on-surface"
                              }`}
                            >
                              {milestone.title}
                            </span>
                            {milestone.due_date && (
                              <span className="type-label-sm text-on-surface-variant ml-auto">
                                {intl.formatDate(milestone.due_date, {
                                  month: "short",
                                  day: "numeric",
                                })}
                              </span>
                            )}
                          </li>
                        ))}
                      </ul>
                    )}

                    {/* Add milestone inline form */}
                    {newMilestoneProject === project.id && (
                      <div className="flex items-center gap-2 mt-2">
                        <Input
                          value={newMilestoneTitle}
                          onChange={(e) =>
                            setNewMilestoneTitle(e.target.value)
                          }
                          placeholder={intl.formatMessage({
                            id: "projects.milestones.title",
                          })}
                          className="flex-1"
                          autoFocus
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              handleAddMilestone(project.id);
                            } else if (e.key === "Escape") {
                              setNewMilestoneProject(null);
                              setNewMilestoneTitle("");
                            }
                          }}
                        />
                        <Button
                          variant="primary"
                          size="sm"
                          onClick={() => handleAddMilestone(project.id)}
                          disabled={
                            !newMilestoneTitle.trim() ||
                            addMilestone.isPending
                          }
                        >
                          <FormattedMessage id="projects.milestones.add" />
                        </Button>
                      </div>
                    )}
                  </div>
                </Card>
              </li>
            );
          })}
        </ul>
      )}

      {/* Delete project dialog */}
      <ConfirmationDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget) {
            void deleteProject.mutateAsync(deleteTarget).then(() => {
              setDeleteTarget(null);
            });
          }
        }}
        title={intl.formatMessage({ id: "projects.delete.title" })}
        confirmLabel={intl.formatMessage({ id: "projects.delete.confirm" })}
        destructive
        loading={deleteProject.isPending}
      >
        <FormattedMessage id="projects.delete.description" />
      </ConfirmationDialog>
    </div>
  );
}
