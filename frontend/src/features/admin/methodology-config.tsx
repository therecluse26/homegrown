import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { ChevronDown, ChevronRight } from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  Skeleton,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";
import {
  useMethodologyConfigs,
  useUpdateMethodologyConfig,
  type MethodologyConfigFull,
  type MethodologyTool,
} from "@/hooks/use-admin";

function MethodologyRow({ config }: { config: MethodologyConfigFull }) {
  const intl = useIntl();
  const updateConfig = useUpdateMethodologyConfig();
  const [isExpanded, setIsExpanded] = useState(false);
  const [displayName, setDisplayName] = useState(config.display_name);
  const [philosophy, setPhilosophy] = useState(config.philosophy);
  const [tools, setTools] = useState<MethodologyTool[]>(config.tools);
  const [dirty, setDirty] = useState(false);

  function handleDisplayNameChange(v: string) {
    setDisplayName(v);
    setDirty(true);
  }

  function handlePhilosophyChange(v: string) {
    setPhilosophy(v);
    setDirty(true);
  }

  function handleToolChange(
    key: string,
    field: keyof MethodologyTool,
    value: string | boolean,
  ) {
    setTools((prev) =>
      prev.map((t) => (t.key === key ? { ...t, [field]: value } : t)),
    );
    setDirty(true);
  }

  async function handleSave() {
    await updateConfig.mutateAsync({
      slug: config.slug,
      display_name: displayName,
      philosophy,
      tools,
    });
    setDirty(false);
  }

  function handleReset() {
    setDisplayName(config.display_name);
    setPhilosophy(config.philosophy);
    setTools(config.tools);
    setDirty(false);
  }

  return (
    <Card className="overflow-hidden">
      {/* Header: expand/collapse */}
      <button
        type="button"
        className="w-full flex items-center justify-between gap-3 text-left focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary"
        onClick={() => setIsExpanded((prev) => !prev)}
        aria-expanded={isExpanded}
        aria-controls={`methodology-${config.slug}-body`}
      >
        <div>
          <span className="type-title-sm text-on-surface font-semibold block">
            {displayName}
          </span>
          <span className="type-label-sm text-on-surface-variant font-mono block">
            {config.slug}
          </span>
        </div>
        <Icon
          icon={isExpanded ? ChevronDown : ChevronRight}
          size="sm"
          className="text-on-surface-variant shrink-0"
          aria-hidden
        />
      </button>

      {/* Body */}
      {isExpanded && (
        <div
          id={`methodology-${config.slug}-body`}
          className="pt-4 mt-4 border-t border-outline-variant flex flex-col gap-4"
        >
          {/* Display name */}
          <div>
            <label
              htmlFor={`display-name-${config.slug}`}
              className="type-label-sm text-on-surface-variant block mb-1"
            >
              <FormattedMessage id="admin.methodologyConfig.field.displayName" />
            </label>
            <Input
              id={`display-name-${config.slug}`}
              type="text"
              value={displayName}
              onChange={(e) => handleDisplayNameChange(e.target.value)}
            />
          </div>

          {/* Philosophy */}
          <div>
            <label
              htmlFor={`philosophy-${config.slug}`}
              className="type-label-sm text-on-surface-variant block mb-1"
            >
              <FormattedMessage id="admin.methodologyConfig.field.philosophy" />
            </label>
            <textarea
              id={`philosophy-${config.slug}`}
              value={philosophy}
              onChange={(e) => handlePhilosophyChange(e.target.value)}
              rows={4}
              className="w-full rounded-radius-sm border border-outline bg-surface px-3 py-2 type-body-sm text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary resize-y"
              placeholder={intl.formatMessage({
                id: "admin.methodologyConfig.field.philosophy.placeholder",
              })}
            />
          </div>

          {/* Tools */}
          {tools.length > 0 && (
            <div>
              <h4 className="type-label-sm text-on-surface-variant font-semibold mb-2">
                <FormattedMessage id="admin.methodologyConfig.tools.title" />
              </h4>
              <div className="flex flex-col gap-3">
                {tools.map((tool) => (
                  <div
                    key={tool.key}
                    className="rounded-radius-sm bg-surface-container-low px-3 py-3 flex flex-col gap-2"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <span className="type-label-sm text-on-surface font-mono">
                        {tool.key}
                      </span>
                      <button
                        type="button"
                        role="switch"
                        aria-checked={tool.enabled}
                        aria-label={intl.formatMessage(
                          { id: "admin.methodologyConfig.tools.enabled.label" },
                          { key: tool.key },
                        )}
                        onClick={() =>
                          handleToolChange(tool.key, "enabled", !tool.enabled)
                        }
                        className={[
                          "relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1 shrink-0",
                          tool.enabled ? "bg-primary" : "bg-outline-variant",
                        ].join(" ")}
                      >
                        <span
                          className={[
                            "inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform",
                            tool.enabled ? "translate-x-4" : "translate-x-0.5",
                          ].join(" ")}
                          aria-hidden
                        />
                      </button>
                    </div>

                    <div>
                      <label
                        htmlFor={`tool-label-${config.slug}-${tool.key}`}
                        className="type-label-xs text-on-surface-variant block mb-0.5"
                      >
                        <FormattedMessage id="admin.methodologyConfig.tools.label" />
                      </label>
                      <Input
                        id={`tool-label-${config.slug}-${tool.key}`}
                        type="text"
                        value={tool.label}
                        onChange={(e) =>
                          handleToolChange(tool.key, "label", e.target.value)
                        }
                        className="text-sm"
                      />
                    </div>

                    <div>
                      <label
                        htmlFor={`tool-desc-${config.slug}-${tool.key}`}
                        className="type-label-xs text-on-surface-variant block mb-0.5"
                      >
                        <FormattedMessage id="admin.methodologyConfig.tools.description" />
                      </label>
                      <Input
                        id={`tool-desc-${config.slug}-${tool.key}`}
                        type="text"
                        value={tool.description}
                        onChange={(e) =>
                          handleToolChange(
                            tool.key,
                            "description",
                            e.target.value,
                          )
                        }
                        className="text-sm"
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Save / cancel */}
          {dirty && (
            <div className="flex items-center gap-2 pt-2">
              <Button
                variant="primary"
                size="sm"
                onClick={handleSave}
                loading={updateConfig.isPending}
                disabled={updateConfig.isPending}
              >
                <FormattedMessage id="admin.methodologyConfig.save" />
              </Button>
              <Button
                variant="tertiary"
                size="sm"
                onClick={handleReset}
                disabled={updateConfig.isPending}
              >
                <FormattedMessage id="common.cancel" />
              </Button>
            </div>
          )}

          {updateConfig.error && (
            <p
              role="alert"
              aria-live="assertive"
              className="type-body-sm text-error"
            >
              <FormattedMessage id="error.generic" />
            </p>
          )}

          {updateConfig.isSuccess && !dirty && (
            <p
              aria-live="polite"
              className="type-body-sm text-primary"
            >
              <FormattedMessage id="admin.methodologyConfig.saved" />
            </p>
          )}
        </div>
      )}
    </Card>
  );
}

export function MethodologyConfig() {
  const intl = useIntl();
  const configsQuery = useMethodologyConfigs();

  if (configsQuery.isPending) {
    return (
      <div className="max-w-2xl mx-auto">
        <Skeleton className="h-8 w-64 mb-2" />
        <Skeleton className="h-4 w-80 mb-6" />
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-16 rounded-radius-md" />
          ))}
        </div>
      </div>
    );
  }

  if (configsQuery.error) {
    return (
      <div className="max-w-2xl mx-auto">
        <PageTitle
          title={intl.formatMessage({ id: "admin.methodologyConfig.title" })}
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

  const configs = configsQuery.data ?? [];

  return (
    <div className="max-w-2xl mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "admin.methodologyConfig.title" })}
        subtitle={intl.formatMessage({ id: "admin.methodologyConfig.subtitle" })}
        className="mb-6"
      />

      {configs.length === 0 ? (
        <div className="rounded-radius-md bg-surface-container-low px-4 py-8 text-center">
          <p className="type-body-sm text-on-surface-variant">
            <FormattedMessage id="admin.methodologyConfig.empty" />
          </p>
        </div>
      ) : (
        <ul className="flex flex-col gap-3" role="list">
          {configs.map((config) => (
            <li key={config.slug}>
              <MethodologyRow config={config} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
