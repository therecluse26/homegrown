import { useIntl } from "react-intl";
import { Card } from "@/components/ui";
import { Icon } from "@/components/ui";
import { BookOpen, Check } from "lucide-react";
import type { components } from "@/api/generated/schema";

type MethodologySummary =
  components["schemas"]["method.MethodologySummaryResponse"];

type MethodologyCardProps = {
  methodology: MethodologySummary;
  selected?: boolean;
  onClick?: () => void;
};

/**
 * Methodology summary card for browse/selection during onboarding
 * and settings. Shows display name, description, and selection state.
 *
 * @see SPEC §2 (methodology selection)
 * @see 04-onboard §9.2 (exploration path)
 */
export function MethodologyCard({
  methodology,
  selected = false,
  onClick,
}: MethodologyCardProps) {
  const intl = useIntl();

  const label = selected
    ? intl.formatMessage(
        { id: "methodology.card.selectedLabel" },
        { name: methodology.display_name },
      )
    : methodology.display_name;

  return (
    <Card
      interactive={!!onClick}
      onClick={onClick}
      className={`relative transition-all ${
        selected
          ? "ring-2 ring-primary bg-primary-container"
          : "hover:bg-surface-container-low"
      }`}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      aria-pressed={onClick ? selected : undefined}
      aria-label={onClick ? label : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onClick();
              }
            }
          : undefined
      }
    >
      {/* Selected checkmark */}
      {selected && (
        <div className="absolute top-3 right-3 flex h-6 w-6 items-center justify-center rounded-full bg-primary text-on-primary">
          <Icon icon={Check} size="xs" aria-hidden />
        </div>
      )}

      {/* Icon / image */}
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-secondary-container text-on-secondary-container">
        {methodology.icon_url ? (
          <img
            src={methodology.icon_url}
            alt=""
            className="h-8 w-8 object-contain"
            aria-hidden
          />
        ) : (
          <Icon icon={BookOpen} size="md" aria-hidden />
        )}
      </div>

      <h3 className="type-title-sm text-on-surface font-semibold mb-1">
        {methodology.display_name}
      </h3>

      {methodology.short_desc && (
        <p className="type-body-sm text-on-surface-variant line-clamp-3">
          {methodology.short_desc}
        </p>
      )}
    </Card>
  );
}
