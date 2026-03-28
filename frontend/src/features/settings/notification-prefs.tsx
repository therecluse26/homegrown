import { FormattedMessage } from "react-intl";
import { Badge, Card } from "@/components/ui";

const NOTIFICATION_TYPES = [
  { id: "messages", labelId: "settings.notifications.messages" },
  { id: "friends", labelId: "settings.notifications.friends" },
  { id: "learning", labelId: "settings.notifications.learning" },
  { id: "marketplace", labelId: "settings.notifications.marketplace" },
  { id: "system", labelId: "settings.notifications.system" },
] as const;

const CHANNELS = [
  { id: "in_app", labelId: "settings.notifications.channel.inApp" },
  { id: "email", labelId: "settings.notifications.channel.email" },
  { id: "push", labelId: "settings.notifications.channel.push" },
] as const;

export function NotificationPrefs() {
  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-center gap-3 mb-6">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="settings.notifications.title" />
        </h1>
        <Badge variant="secondary">
          <FormattedMessage id="settings.comingSoon" />
        </Badge>
      </div>

      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="settings.notifications.description" />
      </p>

      <Card>
        <table className="w-full">
          <thead>
            <tr className="border-b border-outline-variant">
              <th className="text-left type-label-sm text-on-surface-variant pb-3">
                <FormattedMessage id="settings.notifications.type" />
              </th>
              {CHANNELS.map((ch) => (
                <th
                  key={ch.id}
                  className="text-center type-label-sm text-on-surface-variant pb-3"
                >
                  <FormattedMessage id={ch.labelId} />
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {NOTIFICATION_TYPES.map((nt) => (
              <tr key={nt.id} className="border-b border-outline-variant last:border-0">
                <td className="py-3 type-body-sm text-on-surface">
                  <FormattedMessage id={nt.labelId} />
                </td>
                {CHANNELS.map((ch) => (
                  <td key={ch.id} className="py-3 text-center">
                    <input
                      type="checkbox"
                      disabled
                      checked={ch.id === "in_app"}
                      className="h-4 w-4 rounded-sm bg-surface-container-highest opacity-[var(--opacity-disabled)] cursor-not-allowed"
                      aria-label={`${nt.id} ${ch.id}`}
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
