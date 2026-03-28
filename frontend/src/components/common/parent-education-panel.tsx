import { useState } from "react";
import { FormattedMessage } from "react-intl";
import { ChevronDown, Lightbulb } from "lucide-react";
import { Icon, Card } from "@/components/ui";

interface ParentEducationPanelProps {
  /** Tool display name from methodology config */
  toolName: string;
  /** Guidance text from ActiveToolResponse.guidance */
  guidance?: string;
  /** Philosophy explanation for "Why this tool?" */
  philosophy?: string;
}

export function ParentEducationPanel({
  toolName,
  guidance,
  philosophy,
}: ParentEducationPanelProps) {
  const [expanded, setExpanded] = useState(false);

  if (!guidance && !philosophy) return null;

  return (
    <Card className="bg-surface-container-low">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="w-full flex items-center gap-3 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring rounded-lg"
        aria-expanded={expanded}
      >
        <div className="shrink-0 text-tertiary-fixed">
          <Icon icon={Lightbulb} size="md" aria-hidden />
        </div>
        <div className="flex-1 min-w-0">
          <p className="type-title-sm text-on-surface font-medium">
            <FormattedMessage
              id="parentEducation.title"
              values={{ tool: toolName }}
            />
          </p>
        </div>
        <Icon
          icon={ChevronDown}
          size="sm"
          className={`shrink-0 text-on-surface-variant transition-transform ${
            expanded ? "rotate-180" : ""
          }`}
          aria-hidden
        />
      </button>

      {expanded && (
        <div className="mt-3 space-y-3">
          {guidance && (
            <div>
              <p className="type-label-md text-on-surface-variant font-medium mb-1">
                <FormattedMessage id="parentEducation.guidance" />
              </p>
              <p className="type-body-sm text-on-surface">{guidance}</p>
            </div>
          )}
          {philosophy && (
            <div>
              <p className="type-label-md text-on-surface-variant font-medium mb-1">
                <FormattedMessage id="parentEducation.whyThisTool" />
              </p>
              <p className="type-body-sm text-on-surface">{philosophy}</p>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}
