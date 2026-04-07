import { useState, useEffect } from "react";
import { useIntl, FormattedMessage } from "react-intl";
import {
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  EmptyState,
  FormField,
  Icon,
  Input,
  Select,
  Skeleton,
  Spinner,
  Tabs,
} from "@/components/ui";
import { MethodologyCard } from "@/components/common/methodology-card";
import {
  Mail,
  Pencil,
  Plus,
  Shield,
  Trash2,
  UserMinus,
} from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useConsent } from "@/hooks/use-consent";
import { useMethodologyList } from "@/hooks/use-methodologies";
import {
  useFamilyProfile,
  useStudents,
  useUpdateFamily,
  useCreateStudent,
  useUpdateStudent,
  useDeleteStudent,
  useUpdateMethodology,
  useInviteCoParent,
  useRemoveCoParent,
  useTransferPrimary,
} from "@/hooks/use-family";
import { US_STATES, GRADE_LEVELS } from "@/lib/constants";
import type { components } from "@/api/generated/schema";

type MethodologyID = components["schemas"]["method.MethodologyID"];

// ─── Profile Tab ────────────────────────────────────────────────────────────

function ProfileTab() {
  const intl = useIntl();
  const profile = useFamilyProfile();
  const updateFamily = useUpdateFamily();

  const [editing, setEditing] = useState(false);
  const [displayName, setDisplayName] = useState("");
  const [stateCode, setStateCode] = useState("");
  const [locationRegion, setLocationRegion] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  function startEditing() {
    setDisplayName(profile.data?.display_name ?? "");
    setStateCode(profile.data?.state_code ?? "");
    setLocationRegion(profile.data?.location_region ?? "");
    setErrors({});
    setEditing(true);
  }

  function validate() {
    const next: Record<string, string> = {};
    if (!displayName.trim()) {
      next["displayName"] = intl.formatMessage({
        id: "settings.profile.displayName.error",
      });
    }
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!validate()) return;

    await updateFamily.mutateAsync({
      display_name: displayName.trim(),
      state_code: stateCode || undefined,
      location_region: locationRegion.trim() || undefined,
    });
    setEditing(false);
  }

  if (profile.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton height="h-6" width="w-48" />
        <Skeleton height="h-10" />
        <Skeleton height="h-10" />
        <Skeleton height="h-10" />
      </div>
    );
  }

  const data = profile.data;

  if (editing) {
    return (
      <form onSubmit={handleSave} noValidate className="flex flex-col gap-6">
        <FormField
          label={intl.formatMessage({ id: "settings.profile.displayName" })}
          required
          error={errors["displayName"]}
        >
          {({ id, errorId }) => (
            <Input
              id={id}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              aria-describedby={errorId}
              error={!!errors["displayName"]}
              autoFocus
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "settings.profile.state" })}
        >
          {({ id }) => (
            <Select
              id={id}
              value={stateCode}
              onChange={(e) => setStateCode(e.target.value)}
            >
              <option value="">
                {intl.formatMessage({ id: "settings.profile.state.placeholder" })}
              </option>
              {US_STATES.map((s) => (
                <option key={s.code} value={s.code}>
                  {s.name}
                </option>
              ))}
            </Select>
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "settings.profile.region" })}
        >
          {({ id }) => (
            <Input
              id={id}
              value={locationRegion}
              onChange={(e) => setLocationRegion(e.target.value)}
              placeholder={intl.formatMessage({
                id: "settings.profile.region.placeholder",
              })}
            />
          )}
        </FormField>

        {updateFamily.error && (
          <div
            role="alert"
            aria-live="assertive"
            className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
          >
            <FormattedMessage id="error.generic" />
          </div>
        )}

        <div className="flex gap-3">
          <Button
            type="button"
            variant="tertiary"
            onClick={() => setEditing(false)}
            disabled={updateFamily.isPending}
          >
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            type="submit"
            variant="primary"
            loading={updateFamily.isPending}
            disabled={updateFamily.isPending}
          >
            <FormattedMessage id="common.save" />
          </Button>
        </div>
      </form>
    );
  }

  const stateName = US_STATES.find((s) => s.code === data?.state_code)?.name;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between">
        <div className="flex flex-col gap-4">
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="settings.profile.displayName" />
            </p>
            <p className="type-body-lg text-on-surface">
              {data?.display_name ?? "—"}
            </p>
          </div>
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="settings.profile.state" />
            </p>
            <p className="type-body-md text-on-surface">
              {stateName ?? "—"}
            </p>
          </div>
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="settings.profile.region" />
            </p>
            <p className="type-body-md text-on-surface">
              {data?.location_region ?? "—"}
            </p>
          </div>
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="settings.profile.tier" />
            </p>
            <Badge variant="secondary">
              {data?.subscription_tier ?? "free"}
            </Badge>
          </div>
        </div>
        <Button variant="tertiary" size="sm" onClick={startEditing}>
          <Icon icon={Pencil} size="xs" aria-hidden className="mr-1.5" />
          <FormattedMessage id="common.edit" />
        </Button>
      </div>

      {/* Methodology selection */}
      <MethodologySection
        currentSlug={data?.primary_methodology_slug}
      />
    </div>
  );
}

// ─── Methodology Section ────────────────────────────────────────────────────

function MethodologySection({
  currentSlug,
}: {
  currentSlug: string | undefined;
}) {
  const methodologies = useMethodologyList();
  const updateMethodology = useUpdateMethodology();
  const [changing, setChanging] = useState(false);

  if (!changing) {
    const current = methodologies.data?.find((m) => m.slug === currentSlug);
    return (
      <div>
        <div className="flex items-center justify-between mb-3">
          <p className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="settings.profile.methodology" />
          </p>
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setChanging(true)}
          >
            <FormattedMessage id="settings.profile.methodology.change" />
          </Button>
        </div>
        {current ? (
          <MethodologyCard methodology={current} selected />
        ) : (
          <p className="type-body-md text-on-surface-variant">
            <FormattedMessage id="settings.profile.methodology.none" />
          </p>
        )}
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <p className="type-title-sm text-on-surface font-semibold">
          <FormattedMessage id="settings.profile.methodology.select" />
        </p>
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => setChanging(false)}
          disabled={updateMethodology.isPending}
        >
          <FormattedMessage id="common.cancel" />
        </Button>
      </div>

      {updateMethodology.error && (
        <div
          role="alert"
          aria-live="assertive"
          className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container mb-4"
        >
          <FormattedMessage id="error.generic" />
        </div>
      )}

      {methodologies.isPending ? (
        <div className="flex justify-center py-8">
          <Spinner size="md" />
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {methodologies.data?.map((m) => (
            <MethodologyCard
              key={m.slug}
              methodology={m}
              selected={m.slug === currentSlug}
              onClick={() => {
                if (m.slug && m.slug !== currentSlug) {
                  void updateMethodology
                    .mutateAsync({
                      primary_methodology_slug: m.slug as MethodologyID,
                    })
                    .then(() => setChanging(false));
                }
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Students Tab ───────────────────────────────────────────────────────────

function StudentsTab() {
  const intl = useIntl();
  const studentsQuery = useStudents();
  const createStudent = useCreateStudent();
  const updateStudent = useUpdateStudent();
  const deleteStudent = useDeleteStudent();
  const methodologies = useMethodologyList();
  const { isConsented, isLoading: consentLoading, provideConsent, isConsenting, consentError } = useConsent();

  const [showAddForm, setShowAddForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [consentAcknowledged, setConsentAcknowledged] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);

  // Add form state
  const [addForm, setAddForm] = useState({
    displayName: "",
    birthYear: "",
    gradeLevel: "",
  });
  const [addErrors, setAddErrors] = useState<Record<string, string>>({});

  // Edit form state
  const [editForm, setEditForm] = useState({
    displayName: "",
    birthYear: "",
    gradeLevel: "",
    methodologyOverride: "",
  });

  const students = studentsQuery.data ?? [];

  function validateAddForm() {
    const errs: Record<string, string> = {};
    if (!addForm.displayName.trim()) {
      errs["displayName"] = intl.formatMessage({
        id: "settings.students.name.error",
      });
    }
    setAddErrors(errs);
    return Object.keys(errs).length === 0;
  }

  async function handleAddStudent(e: React.FormEvent) {
    e.preventDefault();
    if (!validateAddForm()) return;

    await createStudent.mutateAsync({
      display_name: addForm.displayName.trim(),
      birth_year: addForm.birthYear
        ? parseInt(addForm.birthYear, 10)
        : undefined,
      grade_level: addForm.gradeLevel || undefined,
    });
    setAddForm({ displayName: "", birthYear: "", gradeLevel: "" });
    setAddErrors({});
    setShowAddForm(false);
  }

  function startEditing(student: (typeof students)[0]) {
    setEditingId(student.id ?? null);
    setEditForm({
      displayName: student.display_name ?? "",
      birthYear: student.birth_year ? String(student.birth_year) : "",
      gradeLevel: student.grade_level ?? "",
      methodologyOverride: student.methodology_override_slug ?? "",
    });
  }

  async function handleUpdateStudent(e: React.FormEvent) {
    e.preventDefault();
    if (!editingId) return;

    await updateStudent.mutateAsync({
      id: editingId,
      display_name: editForm.displayName.trim(),
      birth_year: editForm.birthYear
        ? parseInt(editForm.birthYear, 10)
        : undefined,
      grade_level: editForm.gradeLevel || undefined,
      methodology_override_slug: editForm.methodologyOverride || undefined,
    });
    setEditingId(null);
  }

  async function handleDeleteStudent() {
    if (!deleteTarget) return;
    await deleteStudent.mutateAsync(deleteTarget.id);
    setDeleteTarget(null);
  }

  function handleConsent() {
    if (!consentAcknowledged) return;
    void provideConsent({
      coppa_notice_acknowledged: true,
      method: "checkbox",
      verification_token: "parent_acknowledged",
    });
  }

  if (studentsQuery.isPending || consentLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton height="h-20" />
        <Skeleton height="h-20" />
      </div>
    );
  }

  const currentYear = new Date().getFullYear();

  return (
    <div className="flex flex-col gap-6">
      {/* COPPA consent gate */}
      {!isConsented && (
        <Card className="bg-secondary-container">
          <h3 className="type-title-sm text-on-surface font-semibold mb-2">
            <FormattedMessage id="coppa.title" />
          </h3>
          <p className="type-body-sm text-on-surface-variant mb-4">
            <FormattedMessage id="coppa.description" />
          </p>
          <label className="mb-4 flex cursor-pointer select-none items-start gap-3">
            <input
              type="checkbox"
              checked={consentAcknowledged}
              onChange={(e) => setConsentAcknowledged(e.target.checked)}
              className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-primary checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            />
            <span className="type-body-sm text-on-surface">
              <FormattedMessage id="coppa.acknowledge" />
            </span>
          </label>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleConsent}
            loading={isConsenting}
            disabled={!consentAcknowledged || isConsenting}
          >
            <FormattedMessage id="coppa.submit" />
          </Button>
          {consentError && (
            <div
              role="alert"
              aria-live="assertive"
              className="mt-3 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
            >
              <FormattedMessage id="error.generic" />
            </div>
          )}
        </Card>
      )}

      {/* Student list */}
      {students.length === 0 && isConsented && !showAddForm && (
        <EmptyState
          message={intl.formatMessage({ id: "settings.students.empty" })}
          description={intl.formatMessage({
            id: "settings.students.empty.description",
          })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowAddForm(true)}
            >
              <Icon icon={Plus} size="xs" aria-hidden className="mr-1.5" />
              <FormattedMessage id="settings.students.add" />
            </Button>
          }
        />
      )}

      {students.map((student) =>
        editingId === student.id ? (
          <Card key={student.id}>
            <form
              onSubmit={handleUpdateStudent}
              noValidate
              className="flex flex-col gap-4"
            >
              <FormField
                label={intl.formatMessage({ id: "settings.students.name" })}
                required
              >
                {({ id }) => (
                  <Input
                    id={id}
                    value={editForm.displayName}
                    onChange={(e) =>
                      setEditForm((f) => ({
                        ...f,
                        displayName: e.target.value,
                      }))
                    }
                    autoFocus
                  />
                )}
              </FormField>

              <div className="grid grid-cols-2 gap-4">
                <FormField
                  label={intl.formatMessage({
                    id: "settings.students.birthYear",
                  })}
                >
                  {({ id }) => (
                    <Input
                      id={id}
                      type="number"
                      value={editForm.birthYear}
                      onChange={(e) =>
                        setEditForm((f) => ({
                          ...f,
                          birthYear: e.target.value,
                        }))
                      }
                      min={currentYear - 22}
                      max={currentYear - 2}
                    />
                  )}
                </FormField>

                <FormField
                  label={intl.formatMessage({
                    id: "settings.students.gradeLevel",
                  })}
                >
                  {({ id }) => (
                    <Select
                      id={id}
                      value={editForm.gradeLevel}
                      onChange={(e) =>
                        setEditForm((f) => ({
                          ...f,
                          gradeLevel: e.target.value,
                        }))
                      }
                    >
                      <option value="">—</option>
                      {GRADE_LEVELS.map((g) => (
                        <option key={g.value} value={g.value}>
                          {g.label}
                        </option>
                      ))}
                    </Select>
                  )}
                </FormField>
              </div>

              <FormField
                label={intl.formatMessage({
                  id: "settings.students.methodologyOverride",
                })}
              >
                {({ id }) => (
                  <Select
                    id={id}
                    value={editForm.methodologyOverride}
                    onChange={(e) =>
                      setEditForm((f) => ({
                        ...f,
                        methodologyOverride: e.target.value,
                      }))
                    }
                  >
                    <option value="">
                      {intl.formatMessage({
                        id: "settings.students.methodologyOverride.inherit",
                      })}
                    </option>
                    {methodologies.data?.map((m) => (
                      <option key={m.slug} value={m.slug}>
                        {m.display_name}
                      </option>
                    ))}
                  </Select>
                )}
              </FormField>

              {updateStudent.error && (
                <div
                  role="alert"
                  aria-live="assertive"
                  className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
                >
                  <FormattedMessage id="error.generic" />
                </div>
              )}

              <div className="flex gap-3">
                <Button
                  type="button"
                  variant="tertiary"
                  onClick={() => setEditingId(null)}
                  disabled={updateStudent.isPending}
                >
                  <FormattedMessage id="common.cancel" />
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  loading={updateStudent.isPending}
                  disabled={updateStudent.isPending}
                >
                  <FormattedMessage id="common.save" />
                </Button>
              </div>
            </form>
          </Card>
        ) : (
          <Card key={student.id} className="flex items-center justify-between">
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                {student.display_name}
              </p>
              <p className="type-body-sm text-on-surface-variant">
                {[
                  student.birth_year && `Born ${student.birth_year}`,
                  student.grade_level &&
                    GRADE_LEVELS.find((g) => g.value === student.grade_level)
                      ?.label,
                  student.methodology_override_slug &&
                    methodologies.data?.find(
                      (m) => m.slug === student.methodology_override_slug,
                    )?.display_name,
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </p>
            </div>
            <div className="flex gap-1">
              <button
                type="button"
                onClick={() => startEditing(student)}
                className="p-2 rounded-button text-on-surface-variant hover:text-on-surface hover:bg-surface-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                aria-label={intl.formatMessage(
                  { id: "settings.students.edit.label" },
                  { name: student.display_name },
                )}
              >
                <Icon icon={Pencil} size="sm" aria-hidden />
              </button>
              <button
                type="button"
                onClick={() =>
                  setDeleteTarget({
                    id: student.id ?? "",
                    name: student.display_name ?? "",
                  })
                }
                className="p-2 rounded-button text-on-surface-variant hover:text-error hover:bg-error-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                aria-label={intl.formatMessage(
                  { id: "settings.students.delete.label" },
                  { name: student.display_name },
                )}
              >
                <Icon icon={Trash2} size="sm" aria-hidden />
              </button>
            </div>
          </Card>
        ),
      )}

      {/* Add student form */}
      {isConsented && students.length > 0 && !showAddForm && (
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => setShowAddForm(true)}
          className="self-start"
        >
          <Icon icon={Plus} size="xs" aria-hidden className="mr-1.5" />
          <FormattedMessage id="settings.students.add" />
        </Button>
      )}

      {showAddForm && isConsented && (
        <Card>
          <h3 className="type-title-sm text-on-surface font-semibold mb-4">
            <FormattedMessage id="settings.students.add" />
          </h3>
          <form
            onSubmit={handleAddStudent}
            noValidate
            className="flex flex-col gap-4"
          >
            <FormField
              label={intl.formatMessage({ id: "settings.students.name" })}
              required
              error={addErrors["displayName"]}
            >
              {({ id, errorId }) => (
                <Input
                  id={id}
                  value={addForm.displayName}
                  onChange={(e) =>
                    setAddForm((f) => ({ ...f, displayName: e.target.value }))
                  }
                  aria-describedby={errorId}
                  error={!!addErrors["displayName"]}
                  autoFocus
                />
              )}
            </FormField>

            <div className="grid grid-cols-2 gap-4">
              <FormField
                label={intl.formatMessage({
                  id: "settings.students.birthYear",
                })}
              >
                {({ id }) => (
                  <Input
                    id={id}
                    type="number"
                    value={addForm.birthYear}
                    onChange={(e) =>
                      setAddForm((f) => ({ ...f, birthYear: e.target.value }))
                    }
                    min={currentYear - 22}
                    max={currentYear - 2}
                    placeholder={String(currentYear - 8)}
                  />
                )}
              </FormField>

              <FormField
                label={intl.formatMessage({
                  id: "settings.students.gradeLevel",
                })}
              >
                {({ id }) => (
                  <Select
                    id={id}
                    value={addForm.gradeLevel}
                    onChange={(e) =>
                      setAddForm((f) => ({ ...f, gradeLevel: e.target.value }))
                    }
                  >
                    <option value="">
                      {intl.formatMessage({
                        id: "settings.students.gradeLevel.placeholder",
                      })}
                    </option>
                    {GRADE_LEVELS.map((g) => (
                      <option key={g.value} value={g.value}>
                        {g.label}
                      </option>
                    ))}
                  </Select>
                )}
              </FormField>
            </div>

            {createStudent.error && (
              <div
                role="alert"
                aria-live="assertive"
                className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
              >
                <FormattedMessage id="error.generic" />
              </div>
            )}

            <div className="flex gap-3">
              <Button
                type="button"
                variant="tertiary"
                onClick={() => {
                  setShowAddForm(false);
                  setAddForm({ displayName: "", birthYear: "", gradeLevel: "" });
                  setAddErrors({});
                }}
              >
                <FormattedMessage id="common.cancel" />
              </Button>
              <Button
                type="submit"
                variant="primary"
                loading={createStudent.isPending}
                disabled={createStudent.isPending}
              >
                <FormattedMessage id="settings.students.add" />
              </Button>
            </div>
          </form>
        </Card>
      )}

      {/* Delete confirmation */}
      <ConfirmationDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void handleDeleteStudent()}
        title={intl.formatMessage({ id: "settings.students.delete.title" })}
        confirmLabel={intl.formatMessage({ id: "common.delete" })}
        destructive
        loading={deleteStudent.isPending}
      >
        <FormattedMessage
          id="settings.students.delete.confirm"
          values={{ name: deleteTarget?.name }}
        />
      </ConfirmationDialog>
    </div>
  );
}

// ─── Co-Parents Tab ─────────────────────────────────────────────────────────

function CoParentsTab() {
  const intl = useIntl();
  const { isPrimaryParent } = useAuth();
  const profile = useFamilyProfile();
  const inviteCoParent = useInviteCoParent();
  const removeCoParent = useRemoveCoParent();
  const transferPrimary = useTransferPrimary();

  const [inviteEmail, setInviteEmail] = useState("");
  const [removeTarget, setRemoveTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);
  const [transferTarget, setTransferTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);

  const parents = profile.data?.parents ?? [];

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    if (!inviteEmail.trim()) return;
    await inviteCoParent.mutateAsync({ email: inviteEmail.trim() });
    setInviteEmail("");
  }

  async function handleRemove() {
    if (!removeTarget) return;
    await removeCoParent.mutateAsync(removeTarget.id);
    setRemoveTarget(null);
  }

  async function handleTransfer() {
    if (!transferTarget) return;
    await transferPrimary.mutateAsync({
      new_primary_parent_id: transferTarget.id,
    });
    setTransferTarget(null);
  }

  if (profile.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton height="h-16" />
        <Skeleton height="h-16" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Parent list */}
      {parents.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "settings.coparents.empty" })}
        />
      ) : (
        <ul className="flex flex-col gap-3">
          {parents.map((parent) => (
            <li key={parent.id}>
              <Card className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div>
                    <p className="type-title-sm text-on-surface font-medium">
                      {parent.display_name}
                    </p>
                  </div>
                  {parent.is_primary ? (
                    <Badge variant="primary">
                      <Icon
                        icon={Shield}
                        size="xs"
                        aria-hidden
                        className="mr-1"
                      />
                      <FormattedMessage id="settings.coparents.primary" />
                    </Badge>
                  ) : (
                    <Badge variant="secondary">
                      <FormattedMessage id="settings.coparents.coparent" />
                    </Badge>
                  )}
                </div>

                {/* Actions: only primary parent can remove or transfer */}
                {isPrimaryParent && !parent.is_primary && (
                  <div className="flex gap-1">
                    <button
                      type="button"
                      onClick={() =>
                        setTransferTarget({
                          id: parent.id ?? "",
                          name: parent.display_name ?? "",
                        })
                      }
                      className="p-2 rounded-button text-on-surface-variant hover:text-primary hover:bg-primary-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                      aria-label={intl.formatMessage(
                        { id: "settings.coparents.transfer.label" },
                        { name: parent.display_name },
                      )}
                    >
                      <Icon icon={Shield} size="sm" aria-hidden />
                    </button>
                    <button
                      type="button"
                      onClick={() =>
                        setRemoveTarget({
                          id: parent.id ?? "",
                          name: parent.display_name ?? "",
                        })
                      }
                      className="p-2 rounded-button text-on-surface-variant hover:text-error hover:bg-error-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                      aria-label={intl.formatMessage(
                        { id: "settings.coparents.remove.label" },
                        { name: parent.display_name },
                      )}
                    >
                      <Icon icon={UserMinus} size="sm" aria-hidden />
                    </button>
                  </div>
                )}
              </Card>
            </li>
          ))}
        </ul>
      )}

      {/* Invite form */}
      {isPrimaryParent && (
        <Card>
          <h3 className="type-title-sm text-on-surface font-semibold mb-3">
            <FormattedMessage id="settings.coparents.invite.title" />
          </h3>
          <form
            onSubmit={handleInvite}
            noValidate
            className="flex gap-3 items-end"
          >
            <div className="flex-1">
              <FormField
                label={intl.formatMessage({
                  id: "settings.coparents.invite.email",
                })}
              >
                {({ id }) => (
                  <Input
                    id={id}
                    type="email"
                    value={inviteEmail}
                    onChange={(e) => setInviteEmail(e.target.value)}
                    placeholder={intl.formatMessage({
                      id: "settings.coparents.invite.email.placeholder",
                    })}
                  />
                )}
              </FormField>
            </div>
            <Button
              type="submit"
              variant="primary"
              size="sm"
              loading={inviteCoParent.isPending}
              disabled={inviteCoParent.isPending || !inviteEmail.trim()}
            >
              <Icon icon={Mail} size="xs" aria-hidden className="mr-1.5" />
              <FormattedMessage id="settings.coparents.invite.send" />
            </Button>
          </form>
          {inviteCoParent.error && (
            <div
              role="alert"
              aria-live="assertive"
              className="mt-3 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
            >
              <FormattedMessage id="error.generic" />
            </div>
          )}
          {inviteCoParent.isSuccess && (
            <div
              role="status"
              className="mt-3 rounded-lg bg-success-container px-4 py-3 type-body-sm text-on-success-container"
            >
              <FormattedMessage id="settings.coparents.invite.success" />
            </div>
          )}
        </Card>
      )}

      {/* Remove confirmation */}
      <ConfirmationDialog
        open={!!removeTarget}
        onClose={() => setRemoveTarget(null)}
        onConfirm={() => void handleRemove()}
        title={intl.formatMessage({ id: "settings.coparents.remove.title" })}
        confirmLabel={intl.formatMessage({
          id: "settings.coparents.remove.confirm",
        })}
        destructive
        loading={removeCoParent.isPending}
      >
        <FormattedMessage
          id="settings.coparents.remove.description"
          values={{ name: removeTarget?.name }}
        />
      </ConfirmationDialog>

      {/* Transfer confirmation */}
      <ConfirmationDialog
        open={!!transferTarget}
        onClose={() => setTransferTarget(null)}
        onConfirm={() => void handleTransfer()}
        title={intl.formatMessage({ id: "settings.coparents.transfer.title" })}
        confirmLabel={intl.formatMessage({
          id: "settings.coparents.transfer.confirm",
        })}
        loading={transferPrimary.isPending}
      >
        <FormattedMessage
          id="settings.coparents.transfer.description"
          values={{ name: transferTarget?.name }}
        />
      </ConfirmationDialog>
    </div>
  );
}

// ─── Main Settings Page ─────────────────────────────────────────────────────

export function FamilySettings() {
  const intl = useIntl();

  // Set document title for browser tab
  const pageTitle = intl.formatMessage({ id: "settings.title" });
  useEffect(() => {
    const appName = intl.formatMessage({ id: "app.name" });
    document.title = `${pageTitle} — ${appName}`;
  }, [pageTitle, intl]);

  const tabs = [
    {
      id: "profile",
      label: intl.formatMessage({ id: "settings.tabs.profile" }),
      content: <ProfileTab />,
    },
    {
      id: "students",
      label: intl.formatMessage({ id: "settings.tabs.students" }),
      content: <StudentsTab />,
    },
    {
      id: "coparents",
      label: intl.formatMessage({ id: "settings.tabs.coparents" }),
      content: <CoParentsTab />,
    },
  ];

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="type-headline-md text-on-surface font-semibold mb-6">
        <FormattedMessage id="settings.title" />
      </h1>
      <Tabs tabs={tabs} defaultTab="profile" />
    </div>
  );
}
