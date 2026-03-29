import { forwardRef, type SelectHTMLAttributes, type ReactNode } from "react";

type SelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, "className"> & {
  error?: boolean;
  className?: string;
  children: ReactNode;
};

const baseClasses =
  "w-full appearance-none rounded-button bg-surface-container-highest px-4 py-3 pr-10 type-body-md text-on-surface transition-colors outline-none cursor-pointer disabled:opacity-[var(--opacity-disabled)] disabled:pointer-events-none";

const stateClasses =
  "hover:bg-surface-container-high focus:input-focus";

const errorClasses =
  "bg-error-container shadow-[inset_0_0_0_2px_var(--color-error)] hover:bg-error-container";

// Chevron arrow SVG as background image using currentColor (inherits text color)
const chevronBg =
  "bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2220%22%20height%3D%2220%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22currentColor%22%20stroke-width%3D%222%22%3E%3Cpath%20d%3D%22m6%209%206%206%206-6%22%2F%3E%3C%2Fsvg%3E')] bg-[position:right_0.75rem_center] bg-no-repeat";

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  function Select({ error = false, className = "", children, ...props }, ref) {
    return (
      <select
        ref={ref}
        className={`${baseClasses} ${chevronBg} ${error ? errorClasses : stateClasses} ${className}`}
        aria-invalid={error || undefined}
        {...props}
      >
        {children}
      </select>
    );
  },
);
