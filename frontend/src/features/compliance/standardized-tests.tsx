import { FormattedMessage, useIntl } from "react-intl";
import { GraduationCap, Plus, Trash2 } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  Select,
  Skeleton,
} from "@/components/ui";
import { TierGate } from "@/components/common/tier-gate";
import { FormField } from "@/components/ui/form-field";
import {
  useStandardizedTests,
  useCreateStandardizedTest,
  type TestSection,
} from "@/hooks/use-compliance";
import { useStudents } from "@/hooks/use-family";
import { useAuth } from "@/hooks/use-auth";
import { useState, useEffect, useRef, useCallback } from "react";

// ─── Component ─────────────────────────────────────────────────────────────

export function StandardizedTests() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier } = useAuth();
  const students = useStudents();
  const [studentFilter, setStudentFilter] = useState("");
  const tests = useStandardizedTests(studentFilter);
  const createTest = useCreateStandardizedTest();

  const [showForm, setShowForm] = useState(false);
  const [testName, setTestName] = useState("");
  const [testDate, setTestDate] = useState("");
  const [studentId, setStudentId] = useState("");
  const [sections, setSections] = useState<TestSection[]>([
    { name: "", score: "" },
  ]);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "compliance.tests.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // Auto-select first student for list filtering
  useEffect(() => {
    const first = students.data?.[0];
    if (first?.id && !studentFilter) setStudentFilter(first.id);
  }, [students.data, studentFilter]);

  const addSection = useCallback(() => {
    setSections((prev) => [...prev, { name: "", score: "" }]);
  }, []);

  const removeSection = useCallback((index: number) => {
    setSections((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const updateSection = useCallback(
    (index: number, field: keyof TestSection, value: string) => {
      setSections((prev) =>
        prev.map((s, i) => (i === index ? { ...s, [field]: value } : s)),
      );
    },
    [],
  );

  const resetForm = useCallback(() => {
    setShowForm(false);
    setTestName("");
    setTestDate("");
    setStudentId("");
    setSections([{ name: "", score: "" }]);
  }, []);

  const handleSubmit = useCallback(() => {
    if (!testName.trim() || !testDate || !studentId) return;
    const scores: Record<string, number> = {};
    for (const s of sections) {
      if (s.name.trim() && s.score.trim()) {
        const num = Number(s.score);
        if (!Number.isNaN(num)) scores[s.name.trim()] = num;
      }
    }
    createTest.mutate(
      {
        student_id: studentId,
        test_name: testName.trim(),
        test_date: `${testDate}T00:00:00Z`,
        scores,
      },
      { onSuccess: resetForm },
    );
  }, [testName, testDate, studentId, sections, createTest, resetForm]);

  // Tier gate
  if (tier === "free") {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.tests.title" />
        </h1>
        <TierGate featureName="Compliance Tracking" />
      </div>
    );
  }

  if (tests.isPending || students.isPending) {
    return (
      <div className="mx-auto max-w-3xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-48" />
      </div>
    );
  }

  if (tests.error) {
    return (
      <div className="mx-auto max-w-3xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.tests.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const testList = tests.data ?? [];
  const studentList = students.data ?? [];

  return (
    <div className="mx-auto max-w-3xl">
      <div className="flex items-center justify-between mb-2">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          <FormattedMessage id="compliance.tests.title" />
        </h1>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowForm(true)}
        >
          <Icon icon={Plus} size="xs" aria-hidden className="mr-1.5" />
          <FormattedMessage id="compliance.tests.add" />
        </Button>
      </div>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="compliance.tests.description" />
      </p>

      {/* Add test form */}
      {showForm && (
        <Card className="mb-6">
          <h2 className="type-title-md text-on-surface font-semibold mb-4">
            <FormattedMessage id="compliance.tests.add" />
          </h2>
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-2 gap-3">
              <FormField
                label={intl.formatMessage({
                  id: "compliance.tests.form.testName",
                })}
              >
                {({ id }) => (
                  <Input
                    id={id}
                    value={testName}
                    onChange={(e) => setTestName(e.target.value)}
                    autoFocus
                  />
                )}
              </FormField>
              <FormField
                label={intl.formatMessage({
                  id: "compliance.tests.form.date",
                })}
              >
                {({ id }) => (
                  <input
                    id={id}
                    type="date"
                    value={testDate}
                    onChange={(e) => setTestDate(e.target.value)}
                    className="type-body-md text-on-surface bg-surface-container-highest px-3 py-2 rounded-radius-sm w-full"
                  />
                )}
              </FormField>
            </div>
            <FormField
              label={intl.formatMessage({
                id: "compliance.tests.form.student",
              })}
            >
              {({ id }) => (
                <Select
                  id={id}
                  value={studentId}
                  onChange={(e) => setStudentId(e.target.value)}
                >
                  <option value="">
                    {intl.formatMessage({
                      id: "compliance.tests.form.student",
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

            {/* Sections */}
            <div>
              <p className="type-label-md text-on-surface font-medium mb-2">
                <FormattedMessage id="compliance.tests.form.section" />
              </p>
              {sections.map((section, i) => (
                <div key={i} className="flex items-center gap-2 mb-2">
                  <Input
                    value={section.name}
                    onChange={(e) =>
                      updateSection(i, "name", e.target.value)
                    }
                    placeholder={intl.formatMessage({
                      id: "compliance.tests.form.section",
                    })}
                    className="flex-1"
                  />
                  <Input
                    value={section.score}
                    onChange={(e) =>
                      updateSection(i, "score", e.target.value)
                    }
                    placeholder={intl.formatMessage({
                      id: "compliance.tests.form.score",
                    })}
                    className="w-24"
                  />
                  {sections.length > 1 && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      onClick={() => removeSection(i)}
                      className="text-error shrink-0"
                    >
                      <Icon icon={Trash2} size="xs" aria-hidden />
                    </Button>
                  )}
                </div>
              ))}
              <Button variant="tertiary" size="sm" onClick={addSection}>
                <Icon icon={Plus} size="xs" aria-hidden className="mr-1" />
                <FormattedMessage id="compliance.tests.form.addSection" />
              </Button>
            </div>

            <div className="flex justify-end gap-2 mt-2">
              <Button variant="tertiary" onClick={resetForm}>
                <FormattedMessage id="action.cancel" />
              </Button>
              <Button
                variant="primary"
                onClick={handleSubmit}
                disabled={
                  !testName.trim() ||
                  !testDate ||
                  !studentId ||
                  createTest.isPending
                }
              >
                <FormattedMessage id="compliance.tests.form.submit" />
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Test list */}
      {testList.length === 0 && !showForm ? (
        <EmptyState
          message={intl.formatMessage({ id: "compliance.tests.empty" })}
          action={
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowForm(true)}
            >
              <FormattedMessage id="compliance.tests.add" />
            </Button>
          }
        />
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {testList.map((test) => (
            <li key={test.id}>
              <Card>
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-start gap-3">
                    <Icon
                      icon={GraduationCap}
                      size="md"
                      className="text-primary mt-0.5 shrink-0"
                      aria-hidden
                    />
                    <div>
                      <p className="type-title-sm text-on-surface font-medium">
                        {test.test_name}
                      </p>
                      <p className="type-body-sm text-on-surface-variant">
                        {intl.formatDate(test.test_date, {
                          month: "short",
                          day: "numeric",
                          year: "numeric",
                        })}
                      </p>
                    </div>
                  </div>
                </div>
                {test.scores && Object.keys(test.scores).length > 0 && (
                  <div className="ml-8 grid grid-cols-2 gap-1">
                    {Object.entries(test.scores).map(([name, score]) => (
                      <div
                        key={name}
                        className="flex items-center justify-between bg-surface-container-low rounded-radius-sm px-3 py-1.5"
                      >
                        <span className="type-body-sm text-on-surface-variant">
                          {name}
                        </span>
                        <span className="type-title-sm text-on-surface font-medium">
                          {score}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
