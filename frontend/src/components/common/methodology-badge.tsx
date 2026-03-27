import { Badge } from "../ui/badge";

type MethodologyBadgeProps = {
  /** The methodology slug from the API — never hardcode names */
  slug: string;
  /** Display label from methodology config — never derive from slug */
  label: string;
  className?: string;
};

export function MethodologyBadge({ slug, label, className = "" }: MethodologyBadgeProps) {
  return (
    <Badge variant="secondary" className={className} data-methodology={slug}>
      {label}
    </Badge>
  );
}
