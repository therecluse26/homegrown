import { forwardRef, type InputHTMLAttributes } from "react";

type RadioProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "className"> & {
  label: string;
  className?: string;
};

export const Radio = forwardRef<HTMLInputElement, RadioProps>(
  function Radio({ label, id, className = "", ...props }, ref) {
    const inputId = id ?? `radio-${label.toLowerCase().replace(/\s+/g, "-")}`;

    return (
      <label
        htmlFor={inputId}
        className={`inline-flex items-center gap-3 cursor-pointer select-none type-body-md text-on-surface ${className}`}
      >
        <input
          ref={ref}
          type="radio"
          id={inputId}
          className="h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-full bg-surface-container-highest transition-colors checked:bg-primary checked:shadow-[inset_0_0_0_4px_var(--color-on-primary)] hover:bg-surface-container-high checked:hover:bg-primary-container focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:opacity-[var(--opacity-disabled)] disabled:pointer-events-none"
          {...props}
        />
        <span>{label}</span>
      </label>
    );
  },
);
