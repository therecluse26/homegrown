import { Button } from "../ui/button";

type TierGateProps = {
  /** Name of the feature being gated */
  featureName: string;
  className?: string;
};

export function TierGate({ featureName, className = "" }: TierGateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center gap-4 rounded-xl bg-surface-container-low p-8 text-center ${className}`}
    >
      <p className="type-title-md text-on-surface">
        Upgrade to unlock {featureName}
      </p>
      <p className="type-body-md text-on-surface-variant max-w-sm">
        This feature is available on premium plans. Upgrade your subscription to
        access it.
      </p>
      <Button
        variant="gradient"
        onClick={() => {
          window.location.href = "/settings/subscription";
        }}
      >
        View Plans
      </Button>
    </div>
  );
}
