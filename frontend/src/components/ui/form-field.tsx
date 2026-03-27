import { type ReactNode, useId } from "react";

type FormFieldProps = {
  label: string;
  /** Error message text. When set, the field shows error styling. */
  error?: string;
  /** Optional hint text shown below the input */
  hint?: string;
  /** Required field indicator */
  required?: boolean;
  children: (props: { id: string; errorId: string | undefined; hintId: string | undefined }) => ReactNode;
  className?: string;
};

export function FormField({
  label,
  error,
  hint,
  required = false,
  children,
  className = "",
}: FormFieldProps) {
  const autoId = useId();
  const inputId = `field-${autoId}`;
  const errorId = error ? `error-${autoId}` : undefined;
  const hintId = hint && !error ? `hint-${autoId}` : undefined;

  return (
    <div className={`flex flex-col gap-1.5 ${className}`}>
      <label htmlFor={inputId} className="type-label-lg text-on-surface">
        {label}
        {required && (
          <span className="text-error ml-0.5" aria-hidden="true">
            *
          </span>
        )}
      </label>

      {children({
        id: inputId,
        errorId,
        hintId,
      })}

      {error && (
        <p
          id={errorId}
          className="type-body-sm text-error"
          role="alert"
          aria-live="assertive"
        >
          {error}
        </p>
      )}

      {hint && !error && (
        <p id={hintId} className="type-body-sm text-on-surface-variant">
          {hint}
        </p>
      )}
    </div>
  );
}
