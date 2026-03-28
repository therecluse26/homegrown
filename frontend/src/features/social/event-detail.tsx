import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, Link as RouterLink, useNavigate } from "react-router";
import {
  ArrowLeft,
  Clock,
  MapPin,
  Video,
  Users,
  Download,
  XCircle,
  Check,
  Star,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Skeleton,
  Avatar,
  Badge,
  ConfirmationDialog,
  Tabs,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { ReportButton } from "@/components/common/report-button";
import {
  useEventDetail,
  useRSVP,
  useCancelEvent,
} from "@/hooks/use-social";
import type { EventRsvpResponse, RSVPCommand } from "@/hooks/use-social";
import { useAuth } from "@/hooks/use-auth";

// ─── RSVP button ────────────────────────────────────────────────────────────

function RSVPButton({
  eventId,
  myRsvp,
  isFull,
}: {
  eventId: string;
  myRsvp?: string;
  isFull: boolean;
}) {
  const rsvp = useRSVP(eventId);

  const handleRSVP = useCallback(
    (status: RSVPCommand["status"]) => {
      rsvp.mutate({ status });
    },
    [rsvp],
  );

  if (isFull && myRsvp !== "going") {
    return (
      <Badge variant="secondary">
        <FormattedMessage id="social.events.rsvp.full" />
      </Badge>
    );
  }

  return (
    <div className="flex gap-2">
      {(["going", "interested", "not_going"] as const).map((status) => (
        <button
          key={status}
          onClick={() => handleRSVP(status)}
          className={`flex items-center gap-1.5 px-4 py-2 rounded-radius-sm type-label-md transition-colors ${
            myRsvp === status
              ? status === "going"
                ? "bg-primary text-on-primary"
                : status === "interested"
                  ? "bg-secondary-container text-on-secondary-container"
                  : "bg-surface-container-high text-on-surface"
              : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
          }`}
          aria-pressed={myRsvp === status}
          disabled={rsvp.isPending}
        >
          <Icon
            icon={status === "going" ? Check : status === "interested" ? Star : XCircle}
            size="xs"
          />
          <FormattedMessage id={`social.events.rsvp.${status}`} />
        </button>
      ))}
    </div>
  );
}

// ─── Attendee list ──────────────────────────────────────────────────────────

function AttendeeList({
  rsvps,
  eventTitle,
}: {
  rsvps: EventRsvpResponse[];
  eventTitle: string;
}) {
  const intl = useIntl();

  const going = rsvps.filter((r) => r.status === "going");
  const interested = rsvps.filter((r) => r.status === "interested");
  const notGoing = rsvps.filter((r) => r.status === "not_going");

  const handleExportCSV = useCallback(() => {
    const headers = ["Name", "RSVP Status", "Response Date"];
    const rows = rsvps.map((r) => [
      r.display_name,
      r.status,
      new Date(r.created_at).toLocaleDateString(),
    ]);
    const csv = [headers, ...rows].map((row) => row.join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${eventTitle.replace(/[^a-z0-9]/gi, "-")}-attendees.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }, [rsvps, eventTitle]);

  const renderList = (items: EventRsvpResponse[]) => (
    <div className="space-y-2">
      {items.length === 0 && (
        <p className="type-body-sm text-on-surface-variant text-center py-4">
          <FormattedMessage id="social.events.attendees.none" />
        </p>
      )}
      {items.map((r) => (
        <div key={r.family_id} className="flex items-center gap-3 py-2">
          <Avatar size="sm" name={r.display_name} />
          <span className="type-body-md text-on-surface flex-1">
            {r.display_name}
          </span>
          <span className="type-label-sm text-on-surface-variant">
            {new Date(r.created_at).toLocaleDateString()}
          </span>
        </div>
      ))}
    </div>
  );

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="type-title-sm text-on-surface">
          <FormattedMessage id="social.events.attendees.title" />
        </h3>
        <Button variant="tertiary" size="sm" onClick={handleExportCSV}>
          <Icon icon={Download} size="xs" className="mr-1" />
          {intl.formatMessage({ id: "social.events.attendees.export" })}
        </Button>
      </div>

      <Tabs
        tabs={[
          {
            id: "going",
            label: `${intl.formatMessage({ id: "social.events.rsvp.going" })} (${going.length})`,
            content: renderList(going),
          },
          {
            id: "interested",
            label: `${intl.formatMessage({ id: "social.events.rsvp.interested" })} (${interested.length})`,
            content: renderList(interested),
          },
          {
            id: "not_going",
            label: `${intl.formatMessage({ id: "social.events.rsvp.not_going" })} (${notGoing.length})`,
            content: renderList(notGoing),
          },
        ]}
        defaultTab="going"
      />
    </div>
  );
}

// ─── Event detail page ──────────────────────────────────────────────────────

export function EventDetail() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { eventId } = useParams<{ eventId: string }>();
  const { user } = useAuth();
  const { data: event, isPending } = useEventDetail(eventId);
  const cancelEvent = useCancelEvent();
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full rounded-radius-md" />
      </div>
    );
  }

  if (!event) return null;

  const isCreator = event.creator_family_id === user?.family_id;
  const isFull =
    event.capacity != null && event.attendee_count >= event.capacity;
  const eventDate = new Date(event.event_date);
  const isCancelled = event.status === "cancelled";

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={event.title} />

      <RouterLink
        to="/events"
        className="inline-flex items-center gap-1 mb-6 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="social.events.backToEvents" />
      </RouterLink>

      <Card className="p-card-padding mb-6">
        {isCancelled && (
          <div className="mb-4 px-4 py-3 bg-error-container text-on-error-container rounded-radius-sm type-label-md">
            <FormattedMessage id="social.events.cancelled" />
          </div>
        )}

        <div className="flex items-start gap-4 mb-4">
          <div className="w-16 h-16 rounded-radius-md bg-primary-container text-on-primary-container flex flex-col items-center justify-center shrink-0">
            <span className="type-label-sm uppercase">
              {eventDate.toLocaleDateString(undefined, { month: "short" })}
            </span>
            <span className="type-title-lg font-bold">
              {eventDate.getDate()}
            </span>
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="type-headline-sm text-on-surface">{event.title}</h2>
            <p className="type-label-md text-on-surface-variant mt-1">
              {event.creator_family_name}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <ReportButton targetType="event" targetId={event.id} />
          </div>
        </div>

        {/* Event details */}
        <div className="flex flex-wrap gap-4 mb-4 type-body-md text-on-surface-variant">
          <span className="flex items-center gap-1.5">
            <Icon icon={Clock} size="sm" />
            {eventDate.toLocaleString(undefined, {
              weekday: "long",
              month: "long",
              day: "numeric",
              hour: "2-digit",
              minute: "2-digit",
            })}
          </span>

          {event.is_virtual ? (
            <span className="flex items-center gap-1.5 text-primary">
              <Icon icon={Video} size="sm" />
              <FormattedMessage id="social.events.virtual" />
            </span>
          ) : (
            event.location_name && (
              <span className="flex items-center gap-1.5">
                <Icon icon={MapPin} size="sm" />
                {event.location_name}
                {event.location_region && `, ${event.location_region}`}
              </span>
            )
          )}

          <span className="flex items-center gap-1.5">
            <Icon icon={Users} size="sm" />
            <FormattedMessage
              id="social.events.attendees"
              values={{ count: event.attendee_count }}
            />
            {event.capacity != null && (
              <>
                {" / "}
                {event.capacity}
              </>
            )}
          </span>
        </div>

        {event.methodology_name && (
          <Badge variant="secondary" className="mb-4">
            {event.methodology_name}
          </Badge>
        )}

        {event.description && (
          <p className="type-body-md text-on-surface whitespace-pre-wrap mb-6">
            {event.description}
          </p>
        )}

        {/* Virtual URL (only shown to RSVPed attendees) */}
        {event.virtual_url && event.my_rsvp === "going" && (
          <div className="mb-6 px-4 py-3 bg-primary-container/30 rounded-radius-md">
            <p className="type-label-md text-on-surface mb-1">
              <FormattedMessage id="social.events.meetingLink" />
            </p>
            <a
              href={event.virtual_url}
              target="_blank"
              rel="noopener noreferrer"
              className="type-body-md text-primary hover:underline break-all"
            >
              {event.virtual_url}
            </a>
          </div>
        )}

        {/* RSVP */}
        {!isCancelled && (
          <div className="pt-4 border-t border-outline-variant/10">
            <RSVPButton
              eventId={event.id}
              myRsvp={event.my_rsvp}
              isFull={isFull}
            />
          </div>
        )}

        {/* Creator actions */}
        {isCreator && !isCancelled && (
          <div className="flex gap-2 mt-4 pt-4 border-t border-outline-variant/10">
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => setShowCancelConfirm(true)}
              className="text-error"
            >
              <Icon icon={XCircle} size="sm" className="mr-1" />
              <FormattedMessage id="social.events.cancel" />
            </Button>
          </div>
        )}
      </Card>

      {/* Attendee list (visible to creator) */}
      {isCreator && event.rsvps && (
        <Card className="p-card-padding">
          <AttendeeList rsvps={event.rsvps} eventTitle={event.title} />
        </Card>
      )}

      {/* Cancel confirmation */}
      <ConfirmationDialog
        open={showCancelConfirm}
        onClose={() => setShowCancelConfirm(false)}
        title={intl.formatMessage({ id: "social.events.cancel.title" })}
        confirmLabel={intl.formatMessage({
          id: "social.events.cancel.confirm",
        })}
        destructive
        onConfirm={() => {
          cancelEvent.mutate(event.id, {
            onSuccess: () => {
              setShowCancelConfirm(false);
              navigate("/events");
            },
          });
        }}
        loading={cancelEvent.isPending}
      >
        <FormattedMessage
          id="social.events.cancel.description"
          values={{ count: event.attendee_count }}
        />
      </ConfirmationDialog>
    </div>
  );
}
