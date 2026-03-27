import { type AnchorHTMLAttributes, type ReactNode } from "react";

type LinkProps = Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "className"> & {
  href: string;
  children: ReactNode;
  /** Adds rel="noopener noreferrer" and target="_blank" */
  external?: boolean;
  className?: string;
};

export function Link({
  href,
  children,
  external = false,
  className = "",
  ...props
}: LinkProps) {
  return (
    <a
      href={href}
      className={`text-primary no-underline transition-colors duration-[var(--duration-fast)] hover:text-primary-container hover:underline active:text-primary active:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ${className}`}
      {...(external
        ? { target: "_blank", rel: "noopener noreferrer" }
        : {})}
      {...props}
    >
      {children}
    </a>
  );
}
