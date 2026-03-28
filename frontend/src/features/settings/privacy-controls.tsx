import { FormattedMessage, useIntl } from "react-intl";
import { Badge, Card, Select } from "@/components/ui";

const PRIVACY_FIELDS = [
  {
    id: "name",
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
] as const;

const VISIBILITY_OPTIONS = [
  { value: "everyone", labelId: "settings.privacy.visibility.everyone" },
  { value: "friends", labelId: "settings.privacy.visibility.friends" },
  { value: "hidden", labelId: "settings.privacy.visibility.hidden" },
] as const;

export function PrivacyControls() {
  const intl = useIntl();

  return (
    <div className="mx-auto max-w-2xl">
      <div className="flex items-center gap-3 mb-6">
        <h1 className="type-headline-md text-on-surface font-semibold">
          <FormattedMessage id="settings.privacy.title" />
        </h1>
        <Badge variant="secondary">
          <FormattedMessage id="settings.comingSoon" />
        </Badge>
      </div>

      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="settings.privacy.description" />
      </p>

      <div className="flex flex-col gap-4">
        {PRIVACY_FIELDS.map((field) => (
          <Card key={field.id} className="flex items-center justify-between gap-4">
            <div>
              <p className="type-title-sm text-on-surface font-medium">
                <FormattedMessage id={field.labelId} />
              </p>
              <p className="type-body-sm text-on-surface-variant">
                <FormattedMessage id={field.descriptionId} />
              </p>
            </div>
            <Select
              disabled
              value="friends"
              className="w-40 shrink-0"
              aria-label={intl.formatMessage({ id: field.labelId })}
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
