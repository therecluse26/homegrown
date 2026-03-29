import { FormattedMessage, useIntl } from "react-intl";
import { Save } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Select,
  Skeleton,
} from "@/components/ui";
import { TierGate } from "@/components/common/tier-gate";
import { FormField } from "@/components/ui/form-field";
import {
  useComplianceConfig,
  useStateRequirements,
  useSaveComplianceConfig,
} from "@/hooks/use-compliance";
import { useAuth } from "@/hooks/use-auth";
import { useState, useEffect, useRef } from "react";
import { Link } from "react-router";

// ─── US state options (subset; real app would have all 50) ──────────────────

const US_STATES = [
  { value: "AL", label: "Alabama" },
  { value: "AK", label: "Alaska" },
  { value: "AZ", label: "Arizona" },
  { value: "AR", label: "Arkansas" },
  { value: "CA", label: "California" },
  { value: "CO", label: "Colorado" },
  { value: "CT", label: "Connecticut" },
  { value: "DE", label: "Delaware" },
  { value: "FL", label: "Florida" },
  { value: "GA", label: "Georgia" },
  { value: "HI", label: "Hawaii" },
  { value: "ID", label: "Idaho" },
  { value: "IL", label: "Illinois" },
  { value: "IN", label: "Indiana" },
  { value: "IA", label: "Iowa" },
  { value: "KS", label: "Kansas" },
  { value: "KY", label: "Kentucky" },
  { value: "LA", label: "Louisiana" },
  { value: "ME", label: "Maine" },
  { value: "MD", label: "Maryland" },
  { value: "MA", label: "Massachusetts" },
  { value: "MI", label: "Michigan" },
  { value: "MN", label: "Minnesota" },
  { value: "MS", label: "Mississippi" },
  { value: "MO", label: "Missouri" },
  { value: "MT", label: "Montana" },
  { value: "NE", label: "Nebraska" },
  { value: "NV", label: "Nevada" },
  { value: "NH", label: "New Hampshire" },
  { value: "NJ", label: "New Jersey" },
  { value: "NM", label: "New Mexico" },
  { value: "NY", label: "New York" },
  { value: "NC", label: "North Carolina" },
  { value: "ND", label: "North Dakota" },
  { value: "OH", label: "Ohio" },
  { value: "OK", label: "Oklahoma" },
  { value: "OR", label: "Oregon" },
  { value: "PA", label: "Pennsylvania" },
  { value: "RI", label: "Rhode Island" },
  { value: "SC", label: "South Carolina" },
  { value: "SD", label: "South Dakota" },
  { value: "TN", label: "Tennessee" },
  { value: "TX", label: "Texas" },
  { value: "UT", label: "Utah" },
  { value: "VT", label: "Vermont" },
  { value: "VA", label: "Virginia" },
  { value: "WA", label: "Washington" },
  { value: "WV", label: "West Virginia" },
  { value: "WI", label: "Wisconsin" },
  { value: "WY", label: "Wyoming" },
];

// ─── Component ─────────────────────────────────────────────────────────────

export function ComplianceSetup() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { tier } = useAuth();
  const config = useComplianceConfig();
  const saveConfig = useSaveComplianceConfig();

  const [stateCode, setStateCode] = useState("");
  const [daysRequired, setDaysRequired] = useState("");
  const [hoursRequired, setHoursRequired] = useState("");

  const stateReqs = useStateRequirements(stateCode || undefined);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "compliance.setup.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  // Populate form from existing config
  useEffect(() => {
    if (config.data) {
      setStateCode(config.data.state_code || "");
      setDaysRequired(String(config.data.days_required || ""));
      setHoursRequired(String(config.data.hours_required || ""));
    }
  }, [config.data]);

  // Auto-fill thresholds from state requirements
  useEffect(() => {
    if (stateReqs.data) {
      setDaysRequired(String(stateReqs.data.days_required));
      setHoursRequired(String(stateReqs.data.hours_required));
    }
  }, [stateReqs.data]);

  const handleSave = () => {
    saveConfig.mutate({
      state_code: stateCode,
      days_required: Number(daysRequired) || 0,
      hours_required: Number(hoursRequired) || 0,
    });
  };

  // Tier gate for free users
  if (tier === "free") {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.setup.title" />
        </h1>
        <TierGate featureName="Compliance Tracking" />
      </div>
    );
  }

  if (config.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  if (config.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="compliance.setup.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="compliance.setup.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="compliance.setup.description" />
      </p>

      <Card>
        <div className="flex flex-col gap-4">
          {/* State selection */}
          <FormField
            label={intl.formatMessage({ id: "compliance.setup.selectState" })}
          >
            {({ id }) => (
              <Select
                id={id}
                value={stateCode}
                onChange={(e) => setStateCode(e.target.value)}
              >
                <option value="">
                  {intl.formatMessage({
                    id: "compliance.setup.selectState",
                  })}
                </option>
                {US_STATES.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </Select>
            )}
          </FormField>

          {/* State requirements display */}
          {stateReqs.data && (
            <div className="bg-surface-container-low rounded-radius-md p-4">
              <h3 className="type-title-sm text-on-surface font-medium mb-2">
                <FormattedMessage id="compliance.setup.requirements" />
              </h3>
              <p className="type-body-sm text-on-surface-variant">
                {stateReqs.data.description}
              </p>
            </div>
          )}

          {stateCode && !stateReqs.data && !stateReqs.isPending && (
            <p className="type-body-sm text-on-surface-variant">
              <FormattedMessage id="compliance.setup.noRequirements" />
            </p>
          )}

          {/* Threshold configuration */}
          <h3 className="type-title-sm text-on-surface font-medium mt-2">
            <FormattedMessage id="compliance.setup.thresholds" />
          </h3>
          <div className="grid grid-cols-2 gap-4">
            <FormField
              label={intl.formatMessage({
                id: "compliance.setup.daysRequired",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="number"
                  value={daysRequired}
                  onChange={(e) => setDaysRequired(e.target.value)}
                  min={0}
                  max={365}
                />
              )}
            </FormField>
            <FormField
              label={intl.formatMessage({
                id: "compliance.setup.hoursRequired",
              })}
            >
              {({ id }) => (
                <Input
                  id={id}
                  type="number"
                  value={hoursRequired}
                  onChange={(e) => setHoursRequired(e.target.value)}
                  min={0}
                  max={2000}
                />
              )}
            </FormField>
          </div>

          <div className="flex justify-between items-center mt-2">
            <div className="flex gap-3">
              <Link
                to="/compliance/attendance"
                className="type-label-md text-primary hover:underline"
              >
                <FormattedMessage id="compliance.attendance.title" />
              </Link>
              <Link
                to="/compliance/tests"
                className="type-label-md text-primary hover:underline"
              >
                <FormattedMessage id="compliance.tests.title" />
              </Link>
            </div>
            <Button
              variant="primary"
              onClick={handleSave}
              disabled={!stateCode || saveConfig.isPending}
            >
              <Icon icon={Save} size="xs" aria-hidden className="mr-1.5" />
              <FormattedMessage id="compliance.setup.save" />
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
