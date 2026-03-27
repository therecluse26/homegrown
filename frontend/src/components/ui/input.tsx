import { forwardRef, type InputHTMLAttributes } from "react";

type InputProps = Omit<InputHTMLAttributes<HTMLInputElement>, "className"> & {
  /** Error state — applies error styling */
  error?: boolean;
  className?: string;
};

const baseClasses =
  "w-full rounded-button bg-surface-container-highest px-4 py-3 type-body-md text-on-surface placeholder:text-on-surface-variant/60 transition-colors outline-none disabled:opacity-[var(--opacity-disabled)] disabled:pointer-events-none";

const stateClasses =
  "hover:bg-surface-container-high focus:input-focus";

const errorClasses =
  "bg-error-container shadow-[inset_0_0_0_2px_var(--color-error)] hover:bg-error-container";

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { error = false, className = "", ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      className={`${baseClasses} ${error ? errorClasses : stateClasses} ${className}`}
      aria-invalid={error || undefined}
      {...props}
    />
  );
});
