import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useNavigate } from "react-router";
import {
  ArrowLeft,
  Calendar,
  MapPin,
  Video,
  Globe,
  RefreshCw,
} from "lucide-react";
import { Link as RouterLink } from "react-router";
import {
  Button,
  Card,
  Icon,
  Input,
  Checkbox,
  FormField,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import { useCreateEvent, useMyGroups } from "@/hooks/use-social";
import type { CreateEventCommand, RecurrencePattern } from "@/hooks/use-social";
import { useMethodologyContext } from "@/features/auth/methodology-provider";

type LocationType = "in_person" | "virtual" | "hybrid";

export function EventCreation() {
  const intl = useIntl();
  const navigate = useNavigate();
  const createEvent = useCreateEvent();
  const { data: myGroups } = useMyGroups();
  const methodology = useMethodologyContext();

  const [form, setForm] = useState<Partial<CreateEventCommand>>({
    visibility: "friends",
    is_virtual: false,
  });
  const [locationType, setLocationType] = useState<LocationType>("in_person");
  const [linkToGroup, setLinkToGroup] = useState(false);
  const [addMethodologyTag, setAddMethodologyTag] = useState(false);
  const [isRecurring, setIsRecurring] = useState(false);
  const [recurrence, setRecurrence] = useState<RecurrencePattern>({
    frequency: "weekly",
  });

  const DAY_NAMES: { value: number; labelId: string }[] = [
    { value: 1, labelId: "social.events.recurrence.day.mon" },
    { value: 2, labelId: "social.events.recurrence.day.tue" },
    { value: 3, labelId: "social.events.recurrence.day.wed" },
    { value: 4, labelId: "social.events.recurrence.day.thu" },
    { value: 5, labelId: "social.events.recurrence.day.fri" },
    { value: 6, labelId: "social.events.recurrence.day.sat" },
    { value: 0, labelId: "social.events.recurrence.day.sun" },
  ];

  const toggleDayOfWeek = (day: number) => {
    setRecurrence((prev) => {
      const current = prev.days_of_week ?? [];
      const updated = current.includes(day)
        ? current.filter((d) => d !== day)
        : [...current, day];
      return { ...prev, days_of_week: updated };
    });
  };

  const updateField = <K extends keyof CreateEventCommand>(
    key: K,
    value: CreateEventCommand[K],
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const handleLocationTypeChange = useCallback((type: LocationType) => {
    setLocationType(type);
    setForm((prev) => ({
      ...prev,
      is_virtual: type === "virtual" || type === "hybrid",
    }));
  }, []);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (!form.title || !form.event_date) return;

      const data: CreateEventCommand = {
        title: form.title,
        description: form.description,
        event_date: new Date(form.event_date).toISOString(),
        end_date: form.end_date
          ? new Date(form.end_date).toISOString()
          : undefined,
        is_virtual: locationType === "virtual" || locationType === "hybrid",
        visibility: form.visibility ?? "friends",
        location_name:
          locationType !== "virtual" ? form.location_name : undefined,
        location_region:
          locationType !== "virtual" ? form.location_region : undefined,
        virtual_url:
          locationType !== "in_person" ? form.virtual_url : undefined,
        capacity: form.capacity,
        group_id: linkToGroup ? form.group_id : undefined,
        methodology_slug:
          addMethodologyTag && methodology?.primarySlug
            ? methodology.primarySlug
            : undefined,
        recurrence: isRecurring ? recurrence : undefined,
      };

      createEvent.mutate(data, {
        onSuccess: () => navigate("/events"),
      });
    },
    [form, locationType, linkToGroup, addMethodologyTag, methodology, createEvent, navigate, isRecurring, recurrence],
  );

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "social.events.create.pageTitle" })}
      />

      <RouterLink
        to="/events"
        className="inline-flex items-center gap-1 mb-6 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="social.events.backToEvents" />
      </RouterLink>

      <Card className="p-card-padding">
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Title */}
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

          {/* Description */}
          <FormField
            label={intl.formatMessage({ id: "social.events.form.description" })}
          >
            {({ id }) => (
              <textarea
                id={id}
                value={form.description ?? ""}
                onChange={(e) => updateField("description", e.target.value)}
                rows={3}
                className="w-full min-h-[80px] resize-none bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              />
            )}
          </FormField>

          {/* Date/Time */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <FormField
              label={intl.formatMessage({ id: "social.events.form.startDate" })}
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
            <FormField
              label={intl.formatMessage({ id: "social.events.form.endDate" })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="datetime-local"
                  value={form.end_date ?? ""}
                  onChange={(e) => updateField("end_date", e.target.value)}
                />
              )}
            </FormField>
          </div>

          {/* Recurrence */}
          <div className="space-y-3">
            <Checkbox
              label={intl.formatMessage({ id: "social.events.recurrence.enable" })}
              checked={isRecurring}
              onChange={(e) => setIsRecurring(e.target.checked)}
            />
            {isRecurring && (
              <Card className="p-4 bg-surface-container-low space-y-4">
                <div className="flex items-center gap-2 mb-1">
                  <Icon icon={RefreshCw} size="sm" className="text-primary" />
                  <span className="type-label-lg text-on-surface">
                    <FormattedMessage id="social.events.recurrence.title" />
                  </span>
                </div>

                <FormField
                  label={intl.formatMessage({ id: "social.events.recurrence.frequency" })}
                >
                  {({ id }) => (
                    <select
                      id={id}
                      value={recurrence.frequency}
                      onChange={(e) =>
                        setRecurrence((prev) => ({
                          ...prev,
                          frequency: e.target.value as RecurrencePattern["frequency"],
                        }))
                      }
                      className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                    >
                      <option value="weekly">
                        {intl.formatMessage({ id: "social.events.recurrence.frequency.weekly" })}
                      </option>
                      <option value="biweekly">
                        {intl.formatMessage({ id: "social.events.recurrence.frequency.biweekly" })}
                      </option>
                      <option value="monthly">
                        {intl.formatMessage({ id: "social.events.recurrence.frequency.monthly" })}
                      </option>
                      <option value="custom">
                        {intl.formatMessage({ id: "social.events.recurrence.frequency.custom" })}
                      </option>
                    </select>
                  )}
                </FormField>

                {recurrence.frequency === "custom" && (
                  <fieldset>
                    <legend className="type-label-md text-on-surface mb-2">
                      <FormattedMessage id="social.events.recurrence.daysOfWeek" />
                    </legend>
                    <div className="flex flex-wrap gap-2">
                      {DAY_NAMES.map(({ value, labelId }) => {
                        const checked = recurrence.days_of_week?.includes(value) ?? false;
                        return (
                          <button
                            key={value}
                            type="button"
                            onClick={() => toggleDayOfWeek(value)}
                            className={`px-3 py-1.5 rounded-radius-sm type-label-md transition-colors ${
                              checked
                                ? "bg-primary text-on-primary"
                                : "bg-surface-container-high text-on-surface-variant hover:bg-surface-container-highest"
                            }`}
                          >
                            <FormattedMessage id={labelId} />
                          </button>
                        );
                      })}
                    </div>
                  </fieldset>
                )}

                <FormField
                  label={intl.formatMessage({ id: "social.events.recurrence.endDate" })}
                >
                  {({ id }) => (
                    <Input
                      id={id}
                      type="date"
                      value={recurrence.end_date ?? ""}
                      onChange={(e) =>
                        setRecurrence((prev) => ({
                          ...prev,
                          end_date: e.target.value || undefined,
                        }))
                      }
                    />
                  )}
                </FormField>
              </Card>
            )}
          </div>

          {/* Location type selector */}
          <fieldset>
            <legend className="type-label-lg text-on-surface mb-2">
              <FormattedMessage id="social.events.form.locationType" />
            </legend>
            <div className="flex gap-2">
              {(
                [
                  { value: "in_person", icon: MapPin, labelId: "social.events.form.locationType.inPerson" },
                  { value: "virtual", icon: Video, labelId: "social.events.form.locationType.virtual" },
                  { value: "hybrid", icon: Globe, labelId: "social.events.form.locationType.hybrid" },
                ] as const
              ).map(({ value, icon, labelId }) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => handleLocationTypeChange(value)}
                  className={`flex items-center gap-2 px-4 py-2.5 rounded-radius-sm type-label-md transition-colors ${
                    locationType === value
                      ? "bg-primary text-on-primary"
                      : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
                  }`}
                >
                  <Icon icon={icon} size="sm" />
                  <FormattedMessage id={labelId} />
                </button>
              ))}
            </div>
          </fieldset>

          {/* In-person location fields */}
          {locationType !== "virtual" && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <FormField
                label={intl.formatMessage({
                  id: "social.events.form.location",
                })}
              >
                {({ id }) => (
                  <Input
                    id={id}
                    value={form.location_name ?? ""}
                    onChange={(e) => updateField("location_name", e.target.value)}
                    placeholder={intl.formatMessage({
                      id: "social.events.form.location.placeholder",
                    })}
                  />
                )}
              </FormField>
              <FormField
                label={intl.formatMessage({
                  id: "social.events.form.region",
                })}
              >
                {({ id }) => (
                  <Input
                    id={id}
                    value={form.location_region ?? ""}
                    onChange={(e) =>
                      updateField("location_region", e.target.value)
                    }
                  />
                )}
              </FormField>
            </div>
          )}

          {/* Virtual URL */}
          {locationType !== "in_person" && (
            <FormField
              label={intl.formatMessage({
                id: "social.events.form.virtualUrl",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="url"
                  value={form.virtual_url ?? ""}
                  onChange={(e) => updateField("virtual_url", e.target.value)}
                  placeholder="https://meet.example.com/..."
                />
              )}
            </FormField>
          )}

          {/* Capacity */}
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
                placeholder={intl.formatMessage({
                  id: "social.events.form.capacity.placeholder",
                })}
              />
            )}
          </FormField>

          {/* Visibility */}
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
                  {intl.formatMessage({
                    id: "social.events.form.visibility.friends",
                  })}
                </option>
                <option value="group">
                  {intl.formatMessage({
                    id: "social.events.form.visibility.group",
                  })}
                </option>
                <option value="discoverable">
                  {intl.formatMessage({
                    id: "social.events.form.visibility.discoverable",
                  })}
                </option>
              </select>
            )}
          </FormField>

          {/* Group linking */}
          <div className="space-y-3">
            <Checkbox
              label={intl.formatMessage({
                id: "social.events.form.linkToGroup",
              })}
              checked={linkToGroup}
              onChange={(e) => setLinkToGroup(e.target.checked)}
            />
            {linkToGroup && myGroups && myGroups.length > 0 && (
              <select
                value={form.group_id ?? ""}
                onChange={(e) => updateField("group_id", e.target.value || undefined)}
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
                aria-label={intl.formatMessage({
                  id: "social.events.form.selectGroup",
                })}
              >
                <option value="">
                  {intl.formatMessage({
                    id: "social.events.form.selectGroup",
                  })}
                </option>
                {myGroups.map((g) => (
                  <option key={g.summary.id} value={g.summary.id}>
                    {g.summary.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Methodology tag */}
          {methodology?.primarySlug && (
            <Checkbox
              label={intl.formatMessage(
                { id: "social.events.form.methodologyTag" },
                { methodology: methodology.primarySlug },
              )}
              checked={addMethodologyTag}
              onChange={(e) => setAddMethodologyTag(e.target.checked)}
            />
          )}

          {/* Submit */}
          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="tertiary"
              onClick={() => navigate("/events")}
            >
              <FormattedMessage id="common.cancel" />
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={
                !form.title || !form.event_date || createEvent.isPending
              }
            >
              <Icon icon={Calendar} size="sm" className="mr-1" />
              <FormattedMessage id="social.events.create.submit" />
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
