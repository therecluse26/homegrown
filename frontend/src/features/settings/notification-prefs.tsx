import { Fragment, useCallback, useEffect, useMemo, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Card } from "@/components/ui";
import {
  useNotificationPreferences,
  useUpdateNotificationPreferences,
  type NotificationPreference,
} from "@/hooks/use-notifications";

/** Map individual notification types to user-friendly category groups. */
const TYPE_CATEGORIES: Record<string, { category: string; labelId: string }> = {
  friend_request_sent: { category: "social", labelId: "settings.notifications.friends" },
  friend_request_accepted: { category: "social", labelId: "settings.notifications.friends" },
  message_received: { category: "social", labelId: "settings.notifications.messages" },
  event_cancelled: { category: "social", labelId: "settings.notifications.events" },
  methodology_changed: { category: "learning", labelId: "settings.notifications.learning" },
  onboarding_completed: { category: "learning", labelId: "settings.notifications.learning" },
  activity_streak: { category: "learning", labelId: "settings.notifications.learning" },
  milestone_achieved: { category: "learning", labelId: "settings.notifications.learning" },
  book_completed: { category: "learning", labelId: "settings.notifications.learning" },
  data_export_ready: { category: "system", labelId: "settings.notifications.system" },
  purchase_completed: { category: "marketplace", labelId: "settings.notifications.marketplace" },
  purchase_refunded: { category: "marketplace", labelId: "settings.notifications.marketplace" },
  creator_onboarded: { category: "marketplace", labelId: "settings.notifications.marketplace" },
  content_flagged: { category: "system", labelId: "settings.notifications.system" },
  co_parent_added: { category: "system", labelId: "settings.notifications.system" },
  family_deletion_scheduled: { category: "system", labelId: "settings.notifications.system" },
  subscription_created: { category: "system", labelId: "settings.notifications.system" },
  subscription_changed: { category: "system", labelId: "settings.notifications.system" },
  subscription_cancelled: { category: "system", labelId: "settings.notifications.system" },
  payout_completed: { category: "marketplace", labelId: "settings.notifications.marketplace" },
};

/** Human-friendly label for individual notification types. */
const TYPE_LABELS: Record<string, string> = {
  friend_request_sent: "Friend request sent",
  friend_request_accepted: "Friend request accepted",
  message_received: "Messages",
  event_cancelled: "Event cancelled",
  methodology_changed: "Methodology changed",
  onboarding_completed: "Onboarding completed",
  activity_streak: "Activity streaks",
  milestone_achieved: "Milestones achieved",
  book_completed: "Books completed",
  data_export_ready: "Data export ready",
  purchase_completed: "Purchase completed",
  purchase_refunded: "Purchase refunded",
  creator_onboarded: "Creator onboarded",
  content_flagged: "Content flagged",
  co_parent_added: "Co-parent added",
  family_deletion_scheduled: "Family deletion",
  subscription_created: "Subscription created",
  subscription_changed: "Subscription changed",
  subscription_cancelled: "Subscription cancelled",
  payout_completed: "Payout completed",
};

/** Discover unique channels from the API response. */
function getChannels(prefs: NotificationPreference[]): string[] {
  const set = new Set(prefs.map((p) => p.channel));
  // Stable order: in_app first, then email, then others alphabetically
  const order = ["in_app", "email", "push"];
  return [...set].sort(
    (a, b) => (order.indexOf(a) === -1 ? 99 : order.indexOf(a)) - (order.indexOf(b) === -1 ? 99 : order.indexOf(b)),
  );
}

const CHANNEL_LABELS: Record<string, string> = {
  in_app: "In-app",
  email: "Email",
  push: "Push",
};

/** Group notification types by their unique type string (deduplicated). */
function getUniqueTypes(prefs: NotificationPreference[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const p of prefs) {
    if (!seen.has(p.notification_type)) {
      seen.add(p.notification_type);
      result.push(p.notification_type);
    }
  }
  return result;
}

export function NotificationPrefs() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { data: preferences, isLoading } = useNotificationPreferences();
  const { mutate: updatePrefs, isPending } =
    useUpdateNotificationPreferences();

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "settings.notifications.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const channels = useMemo(
    () => (preferences ? getChannels(preferences) : []),
    [preferences],
  );
  const types = useMemo(
    () => (preferences ? getUniqueTypes(preferences) : []),
    [preferences],
  );

  const prefMap = useMemo(() => {
    const map = new Map<string, NotificationPreference>();
    if (preferences) {
      for (const p of preferences) {
        map.set(`${p.notification_type}:${p.channel}`, p);
      }
    }
    return map;
  }, [preferences]);

  const handleToggle = useCallback(
    (type: string, channel: string, currentEnabled: boolean) => {
      updatePrefs([
        { notification_type: type, channel, enabled: !currentEnabled },
      ]);
    },
    [updatePrefs],
  );

  // Group types by category for section headers
  const sections = useMemo(() => {
    const groups: { category: string; labelId: string; types: string[] }[] = [];
    const seen = new Set<string>();
    for (const t of types) {
      const cat = TYPE_CATEGORIES[t]?.category ?? "other";
      if (!seen.has(cat)) {
        seen.add(cat);
        groups.push({
          category: cat,
          labelId: TYPE_CATEGORIES[t]?.labelId ?? "settings.notifications.system",
          types: types.filter((tt) => (TYPE_CATEGORIES[tt]?.category ?? "other") === cat),
        });
      }
    }
    return groups;
  }, [types]);

  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-center gap-3 mb-6">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none"
        >
          <FormattedMessage id="settings.notifications.title" />
        </h1>
      </div>

      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="settings.notifications.description" />
      </p>

      <Card>
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading preferences"
          >
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        ) : (
          <table className="w-full" role="grid">
            <thead>
              <tr className="border-b border-outline-variant/10">
                <th className="text-left type-label-sm text-on-surface-variant pb-3">
                  <FormattedMessage id="settings.notifications.type" />
                </th>
                {channels.map((ch) => (
                  <th
                    key={ch}
                    className="text-center type-label-sm text-on-surface-variant pb-3"
                  >
                    {CHANNEL_LABELS[ch] ?? ch}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {sections.map((section) => (
                <Fragment key={section.category}>
                  <tr>
                    <td
                      colSpan={channels.length + 1}
                      className="pt-4 pb-2 type-label-md text-primary font-semibold"
                    >
                      <FormattedMessage id={section.labelId} />
                    </td>
                  </tr>
                  {section.types.map((nt) => (
                    <tr
                      key={nt}
                      className="border-b border-outline-variant/10 last:border-0"
                    >
                      <td className="py-3 type-body-sm text-on-surface pl-3">
                        {TYPE_LABELS[nt] ?? nt}
                      </td>
                      {channels.map((ch) => {
                        const pref = prefMap.get(`${nt}:${ch}`);
                        const enabled = pref?.enabled ?? false;
                        const isCritical = pref?.system_critical ?? false;
                        return (
                          <td key={ch} className="py-3 text-center">
                            <input
                              type="checkbox"
                              checked={enabled}
                              disabled={isPending || isCritical}
                              onChange={() =>
                                handleToggle(nt, ch, enabled)
                              }
                              className={`h-4 w-4 rounded-sm accent-primary ${
                                isCritical
                                  ? "opacity-[var(--opacity-disabled)] cursor-not-allowed"
                                  : "cursor-pointer"
                              }`}
                              aria-label={`${TYPE_LABELS[nt] ?? nt} ${CHANNEL_LABELS[ch] ?? ch}`}
                            />
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </Fragment>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}
