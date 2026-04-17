import { FormattedMessage } from "react-intl";
import { Button } from "../ui/button";

export type RequiredTier = "plus" | "premium";

type TierGateProps = {
  /** Name of the feature being gated. Shown in the headline. */
  featureName: string;
  /**
   * The minimum tier needed to unlock the feature. Determines which CTA copy
   * is rendered. Defaults to "premium" for backwards compatibility. [S§3.2]
   */
  requiredTier?: RequiredTier;
  className?: string;
};

export function TierGate({
  featureName,
  requiredTier = "premium",
  className = "",
}: TierGateProps) {
  const bodyId =
    requiredTier === "plus" ? "tier.gate.body.plus" : "tier.gate.body.premium";

  return (
    <div
      className={`flex flex-col items-center justify-center gap-4 rounded-xl bg-surface-container-low p-8 text-center ${className}`}
      data-testid="tier-gate"
      data-required-tier={requiredTier}
    >
      <p className="type-title-md text-on-surface">
        <FormattedMessage
          id="tier.gate.title"
          values={{ featureName }}
        />
      </p>
      <p className="type-body-md text-on-surface-variant max-w-sm">
        <FormattedMessage id={bodyId} />
      </p>
      <Button
        variant="gradient"
        onClick={() => {
          window.location.href = "/billing";
        }}
      >
        <FormattedMessage id="tier.gate.cta" />
      </Button>
    </div>
  );
}
