import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  Calendar,
  MapPin,
  Video,
  Users,
  Plus,
  Clock,
  Check,
  Star,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
  Badge,
  Modal,
  Input,
  Checkbox,
  FormField,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useEvents,
  useCreateEvent,
  useRSVP,
} from "@/hooks/use-social";
import type {
  EventDetailResponse,
  CreateEventCommand,
  RSVPCommand,
} from "@/hooks/use-social";

// ─── RSVP button ────────────────────────────────────────────────────────────

function RSVPButton({ event }: { event: EventDetailResponse }) {
  const rsvp = useRSVP(event.id);

  const isFull =
    event.capacity != null && event.attendee_count >= event.capacity;

  const handleRSVP = useCallback(
    (status: RSVPCommand["status"]) => {
      rsvp.mutate({ status });
    },
    [rsvp],
  );

  if (isFull && event.my_rsvp !== "going") {
    return (
      <Badge variant="secondary">
        <FormattedMessage id="social.events.rsvp.full" />
      </Badge>
    );
  }

  return (
    <div className="flex gap-1.5">
      <button
        onClick={() => handleRSVP("going")}
        className={`flex items-center gap-1 px-3 py-1.5 rounded-radius-sm type-label-md transition-colors ${
          event.my_rsvp === "going"
            ? "bg-primary text-on-primary"
            : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
        }`}
        aria-pressed={event.my_rsvp === "going"}
        disabled={rsvp.isPending}
      >
        <Icon icon={Check} size="xs" />
        <FormattedMessage id="social.events.rsvp.going" />
      </button>
      <button
        onClick={() => handleRSVP("interested")}
        className={`flex items-center gap-1 px-3 py-1.5 rounded-radius-sm type-label-md transition-colors ${
          event.my_rsvp === "interested"
            ? "bg-secondary-container text-on-secondary-container"
            : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
        }`}
        aria-pressed={event.my_rsvp === "interested"}
        disabled={rsvp.isPending}
      >
        <Icon icon={Star} size="xs" />
        <FormattedMessage id="social.events.rsvp.interested" />
      </button>
    </div>
  );
}

// ─── Event card ─────────────────────────────────────────────────────────────

function EventCard({ event }: { event: EventDetailResponse }) {
  const eventDate = new Date(event.event_date);

  return (
    <Card className="p-card-padding">
      <div className="flex items-start gap-4">
        {/* Date badge */}
        <div className="w-14 h-14 rounded-radius-md bg-primary-container text-on-primary-container flex flex-col items-center justify-center shrink-0">
          <span className="type-label-sm uppercase">
            {eventDate.toLocaleDateString(undefined, { month: "short" })}
          </span>
          <span className="type-title-md font-bold">
            {eventDate.getDate()}
          </span>
        </div>

        <div className="flex-1 min-w-0">
          <h3 className="type-title-sm text-on-surface">{event.title}</h3>

          <div className="flex flex-wrap items-center gap-3 mt-1.5 type-label-sm text-on-surface-variant">
            <span className="flex items-center gap-1">
              <Icon icon={Clock} size="xs" />
              {eventDate.toLocaleTimeString(undefined, {
                hour: "2-digit",
                minute: "2-digit",
              })}
            </span>

            {event.is_virtual ? (
              <span className="flex items-center gap-1 text-primary">
                <Icon icon={Video} size="xs" />
                <FormattedMessage id="social.events.virtual" />
              </span>
            ) : (
              event.location_name && (
                <span className="flex items-center gap-1">
                  <Icon icon={MapPin} size="xs" />
                  {event.location_name}
                </span>
              )
            )}

            <span className="flex items-center gap-1">
              <Icon icon={Users} size="xs" />
              <FormattedMessage
                id="social.events.attendees"
                values={{ count: event.attendee_count }}
              />
            </span>

            {event.capacity != null && (
              <span>
                <FormattedMessage
                  id="social.events.spots"
                  values={{
                    remaining: event.capacity - event.attendee_count,
                    total: event.capacity,
                  }}
                />
              </span>
            )}
          </div>

          {event.description && (
            <p className="type-body-sm text-on-surface-variant mt-2 line-clamp-2">
              {event.description}
            </p>
          )}

          <div className="mt-3">
            <RSVPButton event={event} />
          </div>
        </div>
      </div>
    </Card>
  );
}

// ─── Create event modal ─────────────────────────────────────────────────────

function CreateEventModal({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const intl = useIntl();
  const createEvent = useCreateEvent();
  const [form, setForm] = useState<Partial<CreateEventCommand>>({
    visibility: "friends",
    is_virtual: false,
  });

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!form.title || !form.event_date) return;

      // Ensure datetime-local values are sent as RFC3339 (Go expects trailing :00Z)
      const payload: CreateEventCommand = {
        ...(form as CreateEventCommand),
        event_date: new Date(form.event_date).toISOString(),
        end_date: form.end_date
          ? new Date(form.end_date).toISOString()
          : undefined,
      };

      createEvent.mutate(payload, {
        onSuccess: () => {
          onClose();
          setForm({ visibility: "friends", is_virtual: false });
        },
      });
    },
    [form, createEvent, onClose],
  );

  const updateField = <K extends keyof CreateEventCommand>(
    key: K,
    value: CreateEventCommand[K],
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={intl.formatMessage({ id: "social.events.create.title" })}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <FormField
          label={intl.formatMessage({ id: "social.events.form.title" })}
          required
        >
          {({ id }) => (
            <Input
              id={id}
              value={form.title ?? ""}
              onChange={(e) => updateField("title", e.target.value)}
              required
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "social.events.form.description" })}
        >
          {({ id }) => (
            <textarea
              id={id}
              value={form.description ?? ""}
              onChange={(e) => updateField("description", e.target.value)}
              className="w-full min-h-[80px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "social.events.form.date" })}
          required
        >
          {({ id }) => (
            <Input
              id={id}
              type="datetime-local"
              value={form.event_date ?? ""}
              onChange={(e) => updateField("event_date", e.target.value)}
              required
            />
          )}
        </FormField>

        <Checkbox
          label={intl.formatMessage({ id: "social.events.form.virtual" })}
          checked={form.is_virtual ?? false}
          onChange={(e) => updateField("is_virtual", e.target.checked)}
        />

        {form.is_virtual && (
          <FormField
            label={intl.formatMessage({ id: "social.events.form.virtualUrl" })}
          >
            {({ id }) => (
              <Input
                id={id}
                type="url"
                value={form.virtual_url ?? ""}
                onChange={(e) => updateField("virtual_url", e.target.value)}
              />
            )}
          </FormField>
        )}

        {!form.is_virtual && (
          <>
            <FormField
              label={intl.formatMessage({ id: "social.events.form.location" })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  value={form.location_name ?? ""}
                  onChange={(e) => updateField("location_name", e.target.value)}
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({ id: "social.events.form.region" })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  value={form.location_region ?? ""}
                  onChange={(e) => updateField("location_region", e.target.value)}
                />
              )}
            </FormField>
          </>
        )}

        <FormField
          label={intl.formatMessage({ id: "social.events.form.capacity" })}
        >
          {({ id }) => (
            <Input
              id={id}
              type="number"
              min={1}
              value={form.capacity ?? ""}
              onChange={(e) =>
                updateField(
                  "capacity",
                  e.target.value ? Number(e.target.value) : undefined,
                )
              }
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "social.events.form.visibility" })}
        >
          {({ id }) => (
            <select
              id={id}
              value={form.visibility ?? "friends"}
              onChange={(e) => updateField("visibility", e.target.value)}
              className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
            >
              <option value="friends">
                {intl.formatMessage({ id: "social.events.form.visibility.friends" })}
              </option>
              <option value="group">
                {intl.formatMessage({ id: "social.events.form.visibility.group" })}
              </option>
              <option value="discoverable">
                {intl.formatMessage({ id: "social.events.form.visibility.discoverable" })}
              </option>
            </select>
          )}
        </FormField>

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="tertiary" onClick={onClose}>
            <FormattedMessage id="common.cancel" />
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={!form.title || !form.event_date || createEvent.isPending}
          >
            <FormattedMessage id="common.create" />
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// ─── Events list page ───────────────────────────────────────────────────────

export function EventsList() {
  const intl = useIntl();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const { data: events, isPending } = useEvents();

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "social.events.title" })} />

      <div className="flex items-center justify-end mb-6">
        <Button
          variant="primary"
          size="sm"
          onClick={() => setShowCreateModal(true)}
        >
          <Icon icon={Plus} size="sm" className="mr-1" />
          <FormattedMessage id="social.events.create" />
        </Button>
      </div>

      {isPending && (
        <div className="space-y-4">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-28 w-full rounded-radius-md" />
          ))}
        </div>
      )}

      {events && events.length === 0 && (
        <EmptyState
          illustration={<Icon icon={Calendar} size="xl" />}
          message={intl.formatMessage({ id: "social.events.empty.title" })}
          description={intl.formatMessage({ id: "social.events.empty.description" })}
          action={
            <Button variant="primary" onClick={() => setShowCreateModal(true)}>
              <FormattedMessage id="social.events.empty.cta" />
            </Button>
          }
        />
      )}

      <div className="space-y-4">
        {events?.map((event) => <EventCard key={event.id} event={event} />)}
      </div>

      <CreateEventModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
      />
    </div>
  );
}
