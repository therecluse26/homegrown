import { useState } from "react";
import { useIntl, FormattedMessage } from "react-intl";
import { Button, FormField, Input, Select } from "@/components/ui";
import { useUpdateFamilyProfile } from "@/hooks/use-onboarding";
import { US_STATES } from "@/lib/constants";

type FamilyProfileStepProps = {
  onNext: () => void;
};

/**
 * Onboarding Step 1 — Family Profile.
 * Collects family display name, US state, and optional location region.
 *
 * State selection feeds into compliance domain for surfacing state-specific
 * homeschooling requirements later. [04-onboard §9.1]
 */
export function FamilyProfileStep({ onNext }: FamilyProfileStepProps) {
  const intl = useIntl();
  const updateProfile = useUpdateFamilyProfile();

  const [displayName, setDisplayName] = useState("");
  const [stateCode, setStateCode] = useState("");
  const [locationRegion, setLocationRegion] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  function validate() {
    const next: Record<string, string> = {};
    if (!displayName.trim()) {
      next["displayName"] = intl.formatMessage({
        id: "onboarding.familyProfile.displayName.error",
      });
    }
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!validate()) return;

    await updateProfile.mutateAsync({
      display_name: displayName.trim(),
      state_code: stateCode || undefined,
      location_region: locationRegion.trim() || undefined,
    });

    onNext();
  }

  return (
    <div>
      <h2 className="type-headline-sm text-on-surface font-semibold mb-2">
        <FormattedMessage id="onboarding.familyProfile.title" />
      </h2>
      <p className="type-body-md text-on-surface-variant mb-8">
        <FormattedMessage id="onboarding.familyProfile.subtitle" />
      </p>

      <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-6">
        <FormField
          label={intl.formatMessage({ id: "onboarding.familyProfile.displayName" })}
          required
          error={errors["displayName"]}
        >
          {({ id, errorId }) => (
            <Input
              id={id}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={intl.formatMessage({
                id: "onboarding.familyProfile.displayName.placeholder",
              })}
              aria-describedby={errorId}
              error={!!errors["displayName"]}
              autoFocus
              autoComplete="organization"
            />
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "onboarding.familyProfile.state" })}
          hint={intl.formatMessage({ id: "onboarding.familyProfile.state.hint" })}
        >
          {({ id, hintId }) => (
            <Select
              id={id}
              value={stateCode}
              onChange={(e) => setStateCode(e.target.value)}
              aria-describedby={hintId}
            >
              <option value="">
                {intl.formatMessage({
                  id: "onboarding.familyProfile.state.placeholder",
                })}
              </option>
              {US_STATES.map((s) => (
                <option key={s.code} value={s.code}>
                  {s.name}
                </option>
              ))}
            </Select>
          )}
        </FormField>

        <FormField
          label={intl.formatMessage({ id: "onboarding.familyProfile.region" })}
          hint={intl.formatMessage({ id: "onboarding.familyProfile.region.hint" })}
        >
          {({ id, hintId }) => (
            <Input
              id={id}
              value={locationRegion}
              onChange={(e) => setLocationRegion(e.target.value)}
              placeholder={intl.formatMessage({
                id: "onboarding.familyProfile.region.placeholder",
              })}
              aria-describedby={hintId}
              autoComplete="address-level2"
            />
          )}
        </FormField>

        {updateProfile.error && (
          <div
            role="alert"
            aria-live="assertive"
            className="rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
          >
            <FormattedMessage id="error.generic" />
          </div>
        )}

        <Button
          type="submit"
          variant="primary"
          loading={updateProfile.isPending}
          disabled={updateProfile.isPending}
          className="w-full"
        >
          <FormattedMessage id="common.next" />
        </Button>
      </form>
    </div>
  );
}
