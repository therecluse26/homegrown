import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Button, Card, Icon, Input, FormField } from "@/components/ui";
import { Link } from "@/components/ui";
import {
  Clock,
  Download,
  Key,
  MessageSquareWarning,
  Trash2,
} from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { PageTitle } from "@/components/common/page-title";
import {
  initSettingsFlow,
  submitFlow,
  extractCsrfToken,
  extractFieldErrors,
} from "@/lib/kratos";
import type { KratosFlow } from "@/lib/kratos";

// ─── Password change form ────────────────────────────────────────────────────

function PasswordChangeForm({ onDone }: { onDone: () => void }) {
  const intl = useIntl();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [successMsg, setSuccessMsg] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const next: Record<string, string> = {};
      if (password !== confirm) {
        next["confirm"] = intl.formatMessage({
          id: "settings.account.password.mismatch",
        });
      }
      if (Object.keys(next).length > 0) {
        setErrors(next);
        return;
      }
      setErrors({});
      setSubmitting(true);
      try {
        const flow: KratosFlow = await initSettingsFlow();
        const csrf = extractCsrfToken(flow);
        const result = await submitFlow(flow.ui.action, flow.ui.method, {
          csrf_token: csrf,
          method: "password",
          password,
        });
        if (result.kind === "flow") {
          const fieldErrors = extractFieldErrors(result.flow);
          setErrors(fieldErrors);
        } else {
          setSuccessMsg(true);
          setPassword("");
          setConfirm("");
          setTimeout(onDone, 1500);
        }
      } catch {
        setErrors({
          _global: intl.formatMessage({
            id: "settings.account.password.error.generic",
          }),
        });
      } finally {
        setSubmitting(false);
      }
    },
    [password, confirm, intl, onDone],
  );

  if (successMsg) {
    return (
      <div
        role="status"
        className="rounded-radius-md bg-success-container px-4 py-3 type-body-sm text-on-success-container"
      >
        <FormattedMessage id="settings.account.password.success" />
      </div>
    );
  }

  return (
    <form onSubmit={(e) => void handleSubmit(e)} noValidate className="flex flex-col gap-4 mt-4">
      <FormField
        label={intl.formatMessage({ id: "settings.account.password.new" })}
        error={errors["password"] ?? errors["_global"]}
      >
        {({ id, errorId }) => (
          <Input
            id={id}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={intl.formatMessage({
              id: "settings.account.password.new.placeholder",
            })}
            aria-describedby={errorId}
            error={!!errors["password"] || !!errors["_global"]}
            autoComplete="new-password"
            autoFocus
          />
        )}
      </FormField>
      <FormField
        label={intl.formatMessage({ id: "settings.account.password.confirm" })}
        error={errors["confirm"]}
      >
        {({ id, errorId }) => (
          <Input
            id={id}
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder={intl.formatMessage({
              id: "settings.account.password.confirm.placeholder",
            })}
            aria-describedby={errorId}
            error={!!errors["confirm"]}
            autoComplete="new-password"
          />
        )}
      </FormField>
      <div className="flex gap-3">
        <Button
          type="button"
          variant="tertiary"
          size="sm"
          onClick={onDone}
          disabled={submitting}
        >
          <FormattedMessage id="common.cancel" />
        </Button>
        <Button
          type="submit"
          variant="primary"
          size="sm"
          loading={submitting}
          disabled={submitting || !password || !confirm}
        >
          <FormattedMessage id="settings.account.password.change" />
        </Button>
      </div>
    </form>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function AccountSettings() {
  const intl = useIntl();
  const { user } = useAuth();
  const [changingPassword, setChangingPassword] = useState(false);

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle title={intl.formatMessage({ id: "settings.account.title" })} className="mb-6" />

      {/* Email */}
      <Card className="mb-4">
        <p className="type-label-sm text-on-surface-variant mb-1">
          <FormattedMessage id="settings.account.email" />
        </p>
        <p className="type-body-lg text-on-surface">{user?.email ?? "—"}</p>
      </Card>

      {/* Password */}
      <Card className="mb-4">
        <div className="flex items-center justify-between">
          <div>
            <p className="type-label-sm text-on-surface-variant mb-1">
              <FormattedMessage id="settings.account.password" />
            </p>
            <p className="type-body-md text-on-surface-variant">
              <FormattedMessage id="settings.account.password.hint" />
            </p>
          </div>
          {!changingPassword && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setChangingPassword(true)}
            >
              <Icon icon={Key} size="xs" aria-hidden className="mr-1.5" />
              <FormattedMessage id="settings.account.password.change" />
            </Button>
          )}
        </div>
        {changingPassword && (
          <PasswordChangeForm onDone={() => setChangingPassword(false)} />
        )}
      </Card>

      {/* Sub-page links */}
      <div className="flex flex-col gap-2">
        <Link href="/settings/account/sessions" className="block">
          <Card interactive className="flex items-center gap-3">
            <Icon
              icon={Clock}
              size="sm"
              aria-hidden
              className="text-on-surface-variant"
            />
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id="settings.account.sessions" />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="settings.account.sessions.description" />
              </p>
            </div>
          </Card>
        </Link>

        <Link href="/settings/account/export" className="block">
          <Card interactive className="flex items-center gap-3">
            <Icon
              icon={Download}
              size="sm"
              aria-hidden
              className="text-on-surface-variant"
            />
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id="settings.account.export" />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="settings.account.export.description" />
              </p>
            </div>
          </Card>
        </Link>

        <Link href="/settings/account/delete" className="block">
          <Card interactive className="flex items-center gap-3">
            <Icon
              icon={Trash2}
              size="sm"
              aria-hidden
              className="text-error"
            />
            <div>
              <p className="type-title-sm text-error font-medium">
                <FormattedMessage id="settings.account.delete" />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="settings.account.delete.description" />
              </p>
            </div>
          </Card>
        </Link>

        <Link href="/settings/account/appeals" className="block">
          <Card interactive className="flex items-center gap-3">
            <Icon
              icon={MessageSquareWarning}
              size="sm"
              aria-hidden
              className="text-on-surface-variant"
            />
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id="settings.account.appeals" />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id="settings.account.appeals.description" />
              </p>
            </div>
          </Card>
        </Link>
      </div>
    </div>
  );
}
