import { forwardRef, type TextareaHTMLAttributes } from "react";

type TextareaProps = Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, "className"> & {
  error?: boolean;
  className?: string;
};

const baseClasses =
  "w-full rounded-button bg-surface-container-highest px-4 py-3 type-body-md text-on-surface placeholder:text-on-surface-variant/60 transition-colors outline-none resize-y min-h-24 disabled:opacity-[var(--opacity-disabled)] disabled:pointer-events-none";

const stateClasses =
  "hover:bg-surface-container-high focus:input-focus";

const errorClasses =
  "bg-error-container shadow-[inset_0_0_0_2px_var(--color-error)] hover:bg-error-container";

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  function Textarea({ error = false, className = "", ...props }, ref) {
    return (
      <textarea
        ref={ref}
        className={`${baseClasses} ${error ? errorClasses : stateClasses} ${className}`}
        aria-invalid={error || undefined}
        {...props}
      />
    );
  },
);
