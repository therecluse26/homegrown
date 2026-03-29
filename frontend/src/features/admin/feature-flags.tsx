import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Plus } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Icon,
  Input,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useFeatureFlags,
  useUpdateFeatureFlag,
  useCreateFeatureFlag,
  type FeatureFlag,
} from "@/hooks/use-admin";

function FlagRow({ flag }: { flag: FeatureFlag }) {
  const intl = useIntl();
  const updateFlag = useUpdateFeatureFlag();
  const [rollout, setRollout] = useState(flag.rollout_percentage ?? 100);
  const [whitelist, setWhitelist] = useState(
    (flag.allowed_family_ids ?? []).join(", "),
  );
  const [dirty, setDirty] = useState(false);

  function handleRolloutChange(v: number) {
    setRollout(v);
    setDirty(true);
  }

  function handleWhitelistChange(v: string) {
    setWhitelist(v);
    setDirty(true);
  }

  async function handleSave() {
    const families = whitelist
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    await updateFlag.mutateAsync({
      id: flag.id,
      rollout_percentage: rollout,
      whitelisted_families: families,
    });
    setDirty(false);
  }

  async function handleToggle() {
    await updateFlag.mutateAsync({ id: flag.id, enabled: !flag.enabled });
  }

  return (
    <Card className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="type-label-md text-on-surface font-mono font-semibold">
              {flag.key}
            </span>
            <Badge variant={flag.enabled ? "primary" : "default"}>
              {flag.enabled
                ? intl.formatMessage({ id: "admin.featureFlags.status.enabled" })
                : intl.formatMessage({
                    id: "admin.featureFlags.status.disabled",
                  })}
            </Badge>
          </div>
          <p className="type-body-sm text-on-surface-variant mt-0.5">
            {flag.description}
          </p>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={flag.enabled}
          aria-label={intl.formatMessage(
            { id: "admin.featureFlags.toggle.label" },
            { key: flag.key },
          )}
          onClick={handleToggle}
          disabled={updateFlag.isPending}
          className={[
            "relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 shrink-0",
            flag.enabled ? "bg-primary" : "bg-outline-variant",
            updateFlag.isPending ? "opacity-50 cursor-not-allowed" : "cursor-pointer",
          ].join(" ")}
        >
          <span
            className={[
              "inline-block h-4 w-4 transform rounded-full bg-white transition-transform",
              flag.enabled ? "translate-x-6" : "translate-x-1",
            ].join(" ")}
            aria-hidden
          />
        </button>
      </div>

      {/* Rollout % */}
      <div>
        <label
          htmlFor={`rollout-${flag.id}`}
          className="type-label-sm text-on-surface-variant flex items-center justify-between"
        >
          <FormattedMessage id="admin.featureFlags.rollout.label" />
          <span className="font-medium text-on-surface">{rollout}%</span>
        </label>
        <input
          id={`rollout-${flag.id}`}
          type="range"
          min={0}
          max={100}
          value={rollout}
          onChange={(e) => handleRolloutChange(Number(e.target.value))}
          className="mt-1.5 w-full accent-primary"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={rollout}
        />
      </div>

      {/* Family whitelist */}
      <div>
        <label
          htmlFor={`whitelist-${flag.id}`}
          className="type-label-sm text-on-surface-variant block mb-1"
        >
          <FormattedMessage id="admin.featureFlags.whitelist.label" />
        </label>
        <Input
          id={`whitelist-${flag.id}`}
          type="text"
          value={whitelist}
          onChange={(e) => handleWhitelistChange(e.target.value)}
          placeholder={intl.formatMessage({
            id: "admin.featureFlags.whitelist.placeholder",
          })}
        />
        <p className="mt-1 type-label-sm text-on-surface-variant">
          <FormattedMessage id="admin.featureFlags.whitelist.hint" />
        </p>
      </div>

      {dirty && (
        <div className="flex items-center gap-2">
          <Button
            variant="primary"
            size="sm"
            onClick={handleSave}
            loading={updateFlag.isPending}
            disabled={updateFlag.isPending}
          >
            <FormattedMessage id="common.save" />
          </Button>
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => {
              setRollout(flag.rollout_percentage ?? 100);
              setWhitelist((flag.allowed_family_ids ?? []).join(", "));
              setDirty(false);
            }}
          >
            <FormattedMessage id="common.cancel" />
          </Button>
        </div>
      )}

      {updateFlag.error && (
        <p role="alert" aria-live="assertive" className="type-body-sm text-error">
          <FormattedMessage id="error.generic" />
        </p>
      )}
    </Card>
  );
}

function CreateFlagForm({ onClose }: { onClose: () => void }) {
  const intl = useIntl();
  const createFlag = useCreateFeatureFlag();
  const [key, setKey] = useState("");
  const [description, setDescription] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!key.trim()) return;
    await createFlag.mutateAsync({
      key: key.trim(),
      description: description.trim(),
      enabled: false,
      rollout_percentage: 0,
    });
    onClose();
  }

  return (
    <Card className="mb-4">
      <h3 className="type-title-sm text-on-surface font-semibold mb-3">
        <FormattedMessage id="admin.featureFlags.create.title" />
      </h3>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        <div>
          <label
            htmlFor="flag-key"
            className="type-label-sm text-on-surface-variant block mb-1"
          >
            <FormattedMessage id="admin.featureFlags.create.key" />
          </label>
          <Input
            id="flag-key"
            type="text"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="feature_name_v1"
            required
          />
        </div>
        <div>
          <label
            htmlFor="flag-description"
            className="type-label-sm text-on-surface-variant block mb-1"
          >
            <FormattedMessage id="admin.featureFlags.create.description" />
          </label>
          <Input
            id="flag-description"
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={intl.formatMessage({
              id: "admin.featureFlags.create.description.placeholder",
            })}
          />
        </div>
        {createFlag.error && (
          <p role="alert" aria-live="assertive" className="type-body-sm text-error">
            <FormattedMessage id="error.generic" />
          </p>
        )}
        <div className="flex items-center gap-2">
          <Button
            type="submit"
            variant="primary"
            size="sm"
            loading={createFlag.isPending}
            disabled={createFlag.isPending || !key.trim()}
          >
            <FormattedMessage id="admin.featureFlags.create.submit" />
          </Button>
          <Button type="button" variant="tertiary" size="sm" onClick={onClose}>
            <FormattedMessage id="common.cancel" />
          </Button>
        </div>
      </form>
    </Card>
  );
}

export function FeatureFlags() {
  const intl = useIntl();
  const flagsQuery = useFeatureFlags();
  const [showCreate, setShowCreate] = useState(false);

  if (flagsQuery.isPending) {
    return (
      <div className="max-w-2xl mx-auto">
        <Skeleton className="h-8 w-48 mb-2" />
        <Skeleton className="h-4 w-80 mb-6" />
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-48 rounded-radius-md" />
          ))}
        </div>
      </div>
    );
  }

  if (flagsQuery.error) {
    return (
      <div className="max-w-2xl mx-auto">
        <PageTitle
          title={intl.formatMessage({ id: "admin.featureFlags.title" })}
          className="mb-6"
        />
        <Card className="rounded-radius-md bg-error-container p-card-padding">
          <p className="type-body-sm text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const flags = flagsQuery.data ?? [];

  return (
    <div className="max-w-2xl mx-auto">
      <div className="flex items-start justify-between mb-6">
        <PageTitle
          title={intl.formatMessage({ id: "admin.featureFlags.title" })}
          subtitle={intl.formatMessage({ id: "admin.featureFlags.subtitle" })}
        />
        {!showCreate && (
          <Button
            variant="primary"
            onClick={() => setShowCreate(true)}
          >
            <Icon icon={Plus} size="xs" aria-hidden className="mr-1" />
            <FormattedMessage id="admin.featureFlags.create" />
          </Button>
        )}
      </div>

      {showCreate && (
        <CreateFlagForm onClose={() => setShowCreate(false)} />
      )}

      {flags.length === 0 && !showCreate ? (
        <div className="rounded-radius-md bg-surface-container-low px-4 py-8 text-center">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="admin.featureFlags.empty" />
          </p>
        </div>
      ) : (
        <ul className="flex flex-col gap-4" role="list">
          {flags.map((flag) => (
            <li key={flag.id}>
              <FlagRow flag={flag} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
