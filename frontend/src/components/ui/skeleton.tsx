import { type HTMLAttributes } from "react";

type SkeletonProps = HTMLAttributes<HTMLDivElement> & {
  /** Width class, e.g. "w-full" or "w-32" */
  width?: string;
  /** Height class, e.g. "h-4" or "h-10" */
  height?: string;
  /** Use rounded-full for circular skeletons (avatars) */
  rounded?: boolean;
};

export function Skeleton({
  width = "w-full",
  height = "h-4",
  rounded = false,
  className = "",
  ...props
}: SkeletonProps) {
  return (
    <div
      className={`animate-pulse bg-surface-container-high ${width} ${height} ${
        rounded ? "rounded-full" : "rounded-md"
      } ${className}`}
      aria-hidden="true"
      {...props}
    />
  );
}
