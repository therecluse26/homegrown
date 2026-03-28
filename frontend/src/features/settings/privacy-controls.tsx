import { useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { MapPin } from "lucide-react";
import { Card, Checkbox, Icon, Select, Skeleton } from "@/components/ui";
import {
  useMyProfile,
  useUpdateProfile,
} from "@/hooks/use-social";
import type { PrivacySettings } from "@/hooks/use-social";

const PRIVACY_FIELD_KEYS: {
  id: keyof PrivacySettings;
  labelId: string;
  descriptionId: string;
}[] = [
  {
    id: "display_name",
    labelId: "settings.privacy.field.name",
    descriptionId: "settings.privacy.field.name.description",
  },
  {
    id: "location",
    labelId: "settings.privacy.field.location",
    descriptionId: "settings.privacy.field.location.description",
  },
  {
    id: "methodology",
    labelId: "settings.privacy.field.methodology",
    descriptionId: "settings.privacy.field.methodology.description",
  },
  {
    id: "parent_names",
    labelId: "settings.privacy.field.parentNames",
    descriptionId: "settings.privacy.field.parentNames.description",
  },
  {
    id: "children_names",
    labelId: "settings.privacy.field.childrenNames",
    descriptionId: "settings.privacy.field.childrenNames.description",
  },
  {
    id: "children_ages",
    labelId: "settings.privacy.field.childrenAges",
    descriptionId: "settings.privacy.field.childrenAges.description",
  },
];

const VISIBILITY_OPTIONS = [
  { value: "friends", labelId: "settings.privacy.visibility.friends" },
  { value: "hidden", labelId: "settings.privacy.visibility.hidden" },
] as const;

const DEFAULT_PRIVACY: PrivacySettings = {
  display_name: "friends",
  parent_names: "friends",
  children_names: "friends",
  children_ages: "friends",
  location: "friends",
  methodology: "friends",
};

export function PrivacyControls() {
  const intl = useIntl();
  const { data: profile, isPending } = useMyProfile();
  const updateProfile = useUpdateProfile();

  const currentSettings: PrivacySettings = profile?.privacy_settings ?? DEFAULT_PRIVACY;

  const handleFieldChange = useCallback(
    (field: keyof PrivacySettings, value: string) => {
      const updated = { ...currentSettings, [field]: value };
      updateProfile.mutate({ privacy_settings: updated });
    },
    [currentSettings, updateProfile],
  );

  const handleLocationToggle = useCallback(
    (checked: boolean) => {
      updateProfile.mutate({ location_visible: checked });
    },
    [updateProfile],
  );

  if (isPending) {
    return (
      <div className="mx-auto max-w-2xl space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="type-headline-md text-on-surface font-semibold mb-2">
        <FormattedMessage id="settings.privacy.title" />
      </h1>

      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="settings.privacy.description" />
      </p>

      {/* Location sharing toggle */}
      <Card className="p-card-padding mb-6">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-radius-sm bg-primary-container flex items-center justify-center shrink-0">
            <Icon icon={MapPin} size="sm" className="text-on-primary-container" />
          </div>
          <div className="flex-1">
            <Checkbox
              label={intl.formatMessage({
                id: "settings.privacy.locationSharing.label",
              })}
              checked={profile?.location_visible ?? false}
              onChange={(e) => handleLocationToggle(e.target.checked)}
              disabled={updateProfile.isPending}
            />
            <p className="type-body-sm text-on-surface-variant mt-1 ml-7">
              <FormattedMessage id="settings.privacy.locationSharing.description" />
            </p>
          </div>
        </div>
      </Card>

      {/* Per-field privacy controls */}
      <h2 className="type-title-md text-on-surface mb-3">
        <FormattedMessage id="settings.privacy.fieldVisibility.title" />
      </h2>
      <p className="type-body-sm text-on-surface-variant mb-4">
        <FormattedMessage id="settings.privacy.fieldVisibility.description" />
      </p>

      <div className="flex flex-col gap-3">
        {PRIVACY_FIELD_KEYS.map((field) => (
          <Card
            key={field.id}
            className="p-card-padding flex items-center justify-between gap-4"
          >
            <div className="flex-1 min-w-0">
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id={field.labelId} />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id={field.descriptionId} />
              </p>
            </div>
            <Select
              value={currentSettings[field.id] ?? "friends"}
              onChange={(e) => handleFieldChange(field.id, e.target.value)}
              className="w-36 shrink-0"
              aria-label={intl.formatMessage({ id: field.labelId })}
              disabled={updateProfile.isPending}
            >
              {VISIBILITY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {intl.formatMessage({ id: opt.labelId })}
                </option>
              ))}
            </Select>
          </Card>
        ))}
      </div>
    </div>
  );
}
