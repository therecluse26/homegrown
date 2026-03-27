import { type LucideIcon } from "lucide-react";

type IconSize = "xs" | "sm" | "md" | "lg" | "xl" | "2xl";

const sizeMap: Record<IconSize, number> = {
  xs: 12,
  sm: 16,
  md: 20,
  lg: 24,
  xl: 32,
  "2xl": 48,
};

type IconProps = {
  icon: LucideIcon;
  size?: IconSize;
  className?: string;
  /** Accessible label. If omitted, icon is decorative (aria-hidden). */
  label?: string;
};

export function Icon({ icon: LucideIcon, size = "md", className = "", label }: IconProps) {
  return (
    <LucideIcon
      size={sizeMap[size]}
      className={`shrink-0 ${className}`}
      aria-hidden={label ? undefined : true}
      aria-label={label}
      role={label ? "img" : undefined}
    />
  );
}
