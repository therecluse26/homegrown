import { FormattedMessage } from "react-intl";
import { Badge, Button, Card, Icon } from "@/components/ui";
import { Link } from "@/components/ui";
import {
  Clock,
  Download,
  Key,
  MessageSquareWarning,
  Trash2,
} from "lucide-react";
import { useAuth } from "@/hooks/use-auth";

export function AccountSettings() {
  const { user } = useAuth();

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="type-headline-md text-on-surface font-semibold mb-6">
        <FormattedMessage id="settings.account.title" />
      </h1>

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
          <Button variant="secondary" size="sm" disabled>
            <Icon icon={Key} size="xs" aria-hidden className="mr-1.5" />
            <FormattedMessage id="settings.account.password.change" />
          </Button>
        </div>
        <Badge variant="secondary" className="mt-2">
          <FormattedMessage id="settings.comingSoon" />
        </Badge>
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
