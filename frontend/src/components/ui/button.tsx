import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { Spinner } from "./spinner";

type ButtonVariant = "primary" | "secondary" | "tertiary" | "gradient";

type ButtonSize = "sm" | "md" | "lg";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  /** Icon element to render before children */
  leadingIcon?: ReactNode;
  /** Icon element to render after children */
  trailingIcon?: ReactNode;
};

const baseClasses =
  "relative inline-flex items-center justify-center gap-2 rounded-button font-body transition-all touch-target select-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:opacity-[var(--opacity-disabled)] disabled:pointer-events-none";

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-on-primary hover:state-hover active:state-pressed",
  secondary:
    "bg-secondary-container text-on-secondary-container hover:state-hover active:state-pressed",
  tertiary:
    "bg-transparent text-primary hover:bg-surface-container-low active:bg-surface-container",
  gradient:
    "text-on-primary hover:state-hover active:state-pressed",
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 type-label-md",
  md: "px-5 py-2.5 type-label-lg",
  lg: "px-7 py-3 type-title-sm",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = "primary",
    size = "md",
    loading = false,
    disabled,
    leadingIcon,
    trailingIcon,
    children,
    className = "",
    type = "button",
    ...props
  },
  ref,
) {
  const isDisabled = disabled || loading;

  return (
    <button
      ref={ref}
      type={type}
      disabled={isDisabled}
      className={`${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${
        variant === "gradient" ? "bg-[image:var(--gradient-primary)]" : ""
      } parent:uppercase student:normal-case ${className}`}
      aria-busy={loading || undefined}
      {...props}
    >
      {loading ? (
        <Spinner size={size === "lg" ? "md" : "sm"} />
      ) : (
        leadingIcon
      )}
      <span className={`inline-flex items-center gap-2${loading ? " invisible" : ""}`}>{children}</span>
      {!loading && trailingIcon}
      {loading && (
        <span className="sr-only">Loading</span>
      )}
    </button>
  );
});
