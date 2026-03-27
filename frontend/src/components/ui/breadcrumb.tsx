import { ChevronRight } from "lucide-react";
import { Icon } from "./icon";

type BreadcrumbItem = {
  label: string;
  href?: string;
};

type BreadcrumbProps = {
  items: BreadcrumbItem[];
  className?: string;
};

export function Breadcrumb({ items, className = "" }: BreadcrumbProps) {
  return (
    <nav aria-label="Breadcrumb" className={className}>
      <ol className="flex items-center gap-1.5 type-body-sm">
        {items.map((item, index) => {
          const isLast = index === items.length - 1;

          return (
            <li key={item.label} className="flex items-center gap-1.5">
              {index > 0 && (
                <Icon
                  icon={ChevronRight}
                  size="xs"
                  className="text-on-surface-variant"
                />
              )}
              {isLast || !item.href ? (
                <span
                  className={
                    isLast
                      ? "text-on-surface font-medium"
                      : "text-on-surface-variant"
                  }
                  aria-current={isLast ? "page" : undefined}
                >
                  {item.label}
                </span>
              ) : (
                <a
                  href={item.href}
                  className="text-on-surface-variant hover:text-primary transition-colors"
                >
                  {item.label}
                </a>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
