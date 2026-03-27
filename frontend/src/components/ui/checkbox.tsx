import { forwardRef, type InputHTMLAttributes } from "react";

type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "className"> & {
  label: string;
  className?: string;
};

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
  function Checkbox({ label, id, className = "", ...props }, ref) {
    const inputId = id ?? `checkbox-${label.toLowerCase().replace(/\s+/g, "-")}`;

    return (
      <label
        htmlFor={inputId}
        className={`inline-flex items-center gap-3 cursor-pointer select-none type-body-md text-on-surface ${className}`}
      >
        <input
          ref={ref}
          type="checkbox"
          id={inputId}
          className="h-5 w-5 shrink-0 cursor-pointer appearance-none rounded-sm bg-surface-container-highest transition-colors checked:bg-primary checked:bg-[image:url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2214%22%20height%3D%2214%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22white%22%20stroke-width%3D%223%22%3E%3Cpath%20d%3D%22M20%206%209%2017l-5-5%22%2F%3E%3C%2Fsvg%3E')] bg-center bg-no-repeat hover:bg-surface-container-high checked:hover:bg-primary-container focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:opacity-[var(--opacity-disabled)] disabled:pointer-events-none"
          {...props}
        />
        <span>{label}</span>
      </label>
    );
  },
);
