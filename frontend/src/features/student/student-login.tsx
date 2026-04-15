import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import { UserCircle } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useStudents } from "@/hooks/use-family";
import { useCreateStudentSession } from "@/hooks/use-student-identity";

export function StudentLogin() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: students, isPending: studentsLoading } = useStudents();
  const createSession = useCreateStudentSession();

  const [studentId, setStudentId] = useState("");
  const [pin, setPin] = useState("");
  const [error, setError] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!studentId || !pin) return;
    setError("");
    createSession.mutate(
      { studentId, pin },
      {
        onSuccess: () => {
          void navigate("/student");
        },
        onError: () => {
          setError(
            intl.formatMessage({ id: "studentLogin.invalidPin" }),
          );
        },
      },
    );
  }

  return (
    <div className="mx-auto max-w-sm mt-12 space-y-6">
      <PageTitle
        title={intl.formatMessage({ id: "studentLogin.title" })}
      />

      <div className="text-center">
        <Icon icon={UserCircle} size="lg" className="text-primary mx-auto mb-3" />
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="studentLogin.title" />
        </h1>
        <p className="type-body-sm text-on-surface-variant mt-1">
          <FormattedMessage id="studentLogin.subtitle" />
        </p>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Student selector */}
          <div>
            <label
              htmlFor="login-student"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="studentLogin.selectStudent" />
            </label>
            {studentsLoading ? (
              <Skeleton height="h-11" />
            ) : (
              <Select
                id="login-student"
                value={studentId}
                onChange={(e) => setStudentId(e.target.value)}
                required
              >
                <option value="">
                  {intl.formatMessage({
                    id: "studentLogin.chooseName",
                  })}
                </option>
                {students?.map((s) => (
                  <option key={s.id} value={s.id ?? ""}>
                    {s.display_name}
                  </option>
                ))}
              </Select>
            )}
          </div>

          {/* PIN */}
          <div>
            <label
              htmlFor="login-pin"
              className="block type-label-md text-on-surface-variant mb-1.5"
            >
              <FormattedMessage id="studentLogin.pin" />
            </label>
            <Input
              id="login-pin"
              type="password"
              inputMode="numeric"
              maxLength={6}
              value={pin}
              onChange={(e) => setPin(e.target.value)}
              placeholder={intl.formatMessage({
                id: "studentLogin.pin.placeholder",
              })}
              required
            />
          </div>

          {error && (
            <p className="type-body-sm text-error">{error}</p>
          )}

          <Button
            variant="primary"
            type="submit"
            className="w-full"
            loading={createSession.isPending}
            disabled={!studentId || !pin}
          >
            <FormattedMessage id="studentLogin.signIn" />
          </Button>
        </form>
      </Card>
    </div>
  );
}
