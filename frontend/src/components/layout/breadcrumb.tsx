import { Link } from "react-router";

type BreadcrumbItem = { label: string; to?: string };

export function Breadcrumb({ items }: { items: BreadcrumbItem[] }) {
  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-1 type-label-sm text-on-surface-variant mb-4">
      {items.map((item, index) => {
        const isLast = index === items.length - 1;
        return (
          <span key={index} className="flex items-center gap-1">
            {index > 0 && (
              <span className="text-on-surface-variant/50" aria-hidden>›</span>
            )}
            {isLast || !item.to ? (
              <span className={isLast ? "text-on-surface font-medium" : undefined}>
                {item.label}
              </span>
            ) : (
              <Link to={item.to} className="hover:text-primary transition-colors">
                {item.label}
              </Link>
            )}
          </span>
        );
      })}
    </nav>
  );
}
