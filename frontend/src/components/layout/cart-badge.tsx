import { NavLink } from "react-router";
import { ShoppingCart } from "lucide-react";
import { useIntl } from "react-intl";
import { Icon } from "@/components/ui";
import { useCart } from "@/hooks/use-marketplace";

export function CartBadge() {
  const intl = useIntl();
  const { data } = useCart();
  const count = data?.item_count ?? 0;

  return (
    <NavLink
      to="/marketplace/cart"
      className="relative p-2 min-w-11 min-h-11 flex items-center justify-center rounded-radius-button text-on-surface-variant hover:bg-surface-container-high transition-colors duration-[var(--duration-normal)]"
      aria-label={intl.formatMessage(
        { id: "cart.badge.label" },
        { count },
      )}
    >
      <Icon icon={ShoppingCart} size="md" />
      {count > 0 && (
        <span
          aria-hidden="true"
          className="absolute top-1 right-1 flex items-center justify-center min-w-[1.125rem] h-[1.125rem] px-1 rounded-full bg-primary text-on-primary type-label-sm font-semibold leading-none"
        >
          {count > 99 ? "99+" : count}
        </span>
      )}
      <span className="sr-only" aria-live="polite" aria-atomic="true">
        {count > 0
          ? intl.formatMessage({ id: "cart.badge.sr" }, { count })
          : ""}
      </span>
    </NavLink>
  );
}
