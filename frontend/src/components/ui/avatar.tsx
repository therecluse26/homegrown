import { type ImgHTMLAttributes } from "react";

type AvatarSize = "xs" | "sm" | "md" | "lg" | "xl";

type AvatarProps = Omit<ImgHTMLAttributes<HTMLImageElement>, "className"> & {
  /** Full name for initials fallback */
  name: string;
  size?: AvatarSize;
  src?: string;
  className?: string;
};

const sizeClasses: Record<AvatarSize, string> = {
  xs: "h-6 w-6 text-[0.5rem]",
  sm: "h-8 w-8 text-[0.625rem]",
  md: "h-10 w-10 type-label-md",
  lg: "h-12 w-12 type-label-lg",
  xl: "h-16 w-16 type-title-sm",
};

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return (parts[0]?.[0] ?? "?").toUpperCase();
  return `${(parts[0]?.[0] ?? "").toUpperCase()}${(parts[parts.length - 1]?.[0] ?? "").toUpperCase()}`;
}

export function Avatar({
  name,
  size = "md",
  src,
  className = "",
  alt,
  ...props
}: AvatarProps) {
  const initials = getInitials(name);

  if (src) {
    return (
      <img
        src={src}
        alt={alt ?? name}
        className={`${sizeClasses[size]} rounded-full object-cover ${className}`}
        {...props}
      />
    );
  }

  return (
    <div
      className={`${sizeClasses[size]} inline-flex items-center justify-center rounded-full bg-primary text-on-primary font-semibold select-none ${className}`}
      role="img"
      aria-label={alt ?? name}
    >
      {initials}
    </div>
  );
}
