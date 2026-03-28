import { useState } from "react";
import { useIntl, FormattedMessage } from "react-intl";
import { useQuery } from "@tanstack/react-query";
import {
  Button,
  FormField,
  Input,
  Select,
  Card,
  Spinner,
} from "@/components/ui";
import { Icon } from "@/components/ui";
import { Trash2, UserPlus } from "lucide-react";
import { useAddChild, useRemoveChild } from "@/hooks/use-onboarding";
import { useConsent } from "@/hooks/use-consent";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

type Student = components["schemas"]["iam.StudentResponse"];

const GRADE_LEVELS = [
  { value: "pre-k", label: "Pre-K" },
  { value: "kindergarten", label: "Kindergarten" },
  { value: "grade-1", label: "Grade 1" },
  { value: "grade-2", label: "Grade 2" },
  { value: "grade-3", label: "Grade 3" },
  { value: "grade-4", label: "Grade 4" },
  { value: "grade-5", label: "Grade 5" },
  { value: "grade-6", label: "Grade 6" },
  { value: "grade-7", label: "Grade 7" },
  { value: "grade-8", label: "Grade 8" },
  { value: "grade-9", label: "Grade 9" },
  { value: "grade-10", label: "Grade 10" },
  { value: "grade-11", label: "Grade 11" },
  { value: "grade-12", label: "Grade 12" },
] as const;

const CURRENT_YEAR = new Date().getFullYear();
const MIN_BIRTH_YEAR = CURRENT_YEAR - 22;
const MAX_BIRTH_YEAR = CURRENT_YEAR - 2;

type ChildrenStepProps = {
  onNext: () => void;
  onBack: () => void;
};

type AddChildForm = {
  displayName: string;
  birthYear: string;
  gradeLevel: string;
};

const EMPTY_FORM: AddChildForm = {
  displayName: "",
  birthYear: "",
  gradeLevel: "",
};

/**
 * Onboarding Step 2 — Children.
 * Add student profiles (optional). Requires COPPA consent first.
 *
 * COPPA: consent must be provided before creating student profiles. [SPEC §7.3]
 * This step is optional — clicking Next without adding children is allowed.
 */
export function ChildrenStep({ onNext, onBack }: ChildrenStepProps) {
  const intl = useIntl();
  const { isConsented, isLoading: consentLoading, provideConsent, isConsenting } = useConsent();
  const addChild = useAddChild();
  const removeChild = useRemoveChild();
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<AddChildForm>(EMPTY_FORM);
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [consentAcknowledged, setConsentAcknowledged] = useState(false);

  const studentsQuery = useQuery({
    queryKey: ["family", "students"],
    queryFn: () => apiClient<Student[]>("/v1/families/students"),
    staleTime: 1000 * 60,
  });

  const students = studentsQuery.data ?? [];

  function validateForm() {
    const errs: Record<string, string> = {};
    if (!form.displayName.trim()) {
      errs["displayName"] = intl.formatMessage({
        id: "onboarding.children.name.error",
      });
    }
    setFormErrors(errs);
    return Object.keys(errs).length === 0;
  }

  async function handleAddChild(e: React.FormEvent) {
    e.preventDefault();
    if (!validateForm()) return;

    await addChild.mutateAsync({
      display_name: form.displayName.trim(),
      birth_year: form.birthYear ? parseInt(form.birthYear, 10) : undefined,
      grade_level: form.gradeLevel || undefined,
    });

    setForm(EMPTY_FORM);
    setFormErrors({});
    setShowForm(false);
  }

  async function handleRemove(studentId: string) {
    await removeChild.mutateAsync(studentId);
  }

  async function handleConsent() {
    if (!consentAcknowledged) return;
    await provideConsent({
      coppa_notice_acknowledged: true,
      method: "explicit",
      verification_token: "",
    });
  }

  if (consentLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="md" />
      </div>
    );
  }

  return (
    <div>
      <h2 className="type-headline-sm text-on-surface font-semibold mb-2">
        <FormattedMessage id="onboarding.children.title" />
      </h2>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="onboarding.children.subtitle" />
      </p>

      {/* COPPA consent gate */}
      {!isConsented && (
        <Card className="mb-6 bg-secondary-container">
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
        </Card>
      )}

      {/* Student list */}
      {students.length > 0 && (
        <ul className="flex flex-col gap-3 mb-6" aria-label={intl.formatMessage({ id: "onboarding.children.list.label" })}>
          {students.map((student) => (
            <li key={student.id}>
              <Card className="flex items-center justify-between gap-4">
                <div>
                  <p className="type-title-sm text-on-surface font-medium">
                    {student.display_name}
                  </p>
                  {(student.birth_year ?? student.grade_level) && (
                    <p className="type-body-sm text-on-surface-variant">
                      {[
                        student.birth_year && `Born ${student.birth_year}`,
                        student.grade_level,
                      ]
                        .filter(Boolean)
                        .join(" · ")}
                    </p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => void handleRemove(student.id ?? "")}
                  disabled={removeChild.isPending}
                  aria-label={intl.formatMessage(
                    { id: "onboarding.children.remove.label" },
                    { name: student.display_name },
                  )}
                  className="flex-shrink-0 p-2 rounded-button text-on-surface-variant hover:text-error hover:bg-error-container transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:opacity-[var(--opacity-disabled)]"
                >
                  <Icon icon={Trash2} size="sm" aria-hidden />
                </button>
              </Card>
            </li>
          ))}
        </ul>
      )}

      {/* Add child form */}
      {isConsented && (
        <>
          {showForm ? (
            <Card className="mb-6">
              <h3 className="type-title-sm text-on-surface font-semibold mb-4">
                <FormattedMessage id="onboarding.children.add.title" />
              </h3>
              <form
                onSubmit={handleAddChild}
                noValidate
                className="flex flex-col gap-4"
              >
                <FormField
                  label={intl.formatMessage({ id: "onboarding.children.name" })}
                  required
                  error={formErrors["displayName"]}
                >
                  {({ id, errorId }) => (
                    <Input
                      id={id}
                      value={form.displayName}
                      onChange={(e) =>
                        setForm((f) => ({ ...f, displayName: e.target.value }))
                      }
                      placeholder={intl.formatMessage({
                        id: "onboarding.children.name.placeholder",
                      })}
                      aria-describedby={errorId}
                      error={!!formErrors["displayName"]}
                      autoFocus
                    />
                  )}
                </FormField>

                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    label={intl.formatMessage({ id: "onboarding.children.birthYear" })}
                  >
                    {({ id }) => (
                      <Input
                        id={id}
                        type="number"
                        value={form.birthYear}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, birthYear: e.target.value }))
                        }
                        min={MIN_BIRTH_YEAR}
                        max={MAX_BIRTH_YEAR}
                        placeholder={String(CURRENT_YEAR - 8)}
                      />
                    )}
                  </FormField>

                  <FormField
                    label={intl.formatMessage({ id: "onboarding.children.gradeLevel" })}
                  >
                    {({ id }) => (
                      <Select
                        id={id}
                        value={form.gradeLevel}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, gradeLevel: e.target.value }))
                        }
                      >
                        <option value="">
                          {intl.formatMessage({
                            id: "onboarding.children.gradeLevel.placeholder",
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

                {addChild.error && (
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
                      setShowForm(false);
                      setForm(EMPTY_FORM);
                      setFormErrors({});
                    }}
                  >
                    <FormattedMessage id="common.cancel" />
                  </Button>
                  <Button
                    type="submit"
                    variant="secondary"
                    loading={addChild.isPending}
                    disabled={addChild.isPending}
                  >
                    <FormattedMessage id="onboarding.children.add.submit" />
                  </Button>
                </div>
              </form>
            </Card>
          ) : (
            <button
              type="button"
              onClick={() => setShowForm(true)}
              className="mb-6 flex w-full items-center gap-3 rounded-button border-2 border-dashed border-outline-variant px-4 py-3 type-body-md text-on-surface-variant hover:border-primary hover:text-primary hover:bg-surface-container-low transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            >
              <Icon icon={UserPlus} size="sm" aria-hidden />
              <FormattedMessage id="onboarding.children.addButton" />
            </button>
          )}
        </>
      )}

      {/* Navigation */}
      <div className="flex gap-3 pt-2">
        <Button type="button" variant="tertiary" onClick={onBack}>
          <FormattedMessage id="common.back" />
        </Button>
        <Button
          type="button"
          variant="primary"
          onClick={onNext}
          className="flex-1"
        >
          <FormattedMessage id="common.next" />
        </Button>
      </div>
    </div>
  );
}
