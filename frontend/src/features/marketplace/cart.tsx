import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import { ShoppingCart, Trash2, ArrowLeft } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { useToast } from "@/components/ui/toast";
import { PageTitle } from "@/components/common/page-title";
import {
  useCart,
  useRemoveFromCart,
  useCheckout,
} from "@/hooks/use-marketplace";

export function Cart() {
  const intl = useIntl();
  const { data: cart, isPending } = useCart();
  const removeFromCart = useRemoveFromCart();
  const checkout = useCheckout();
  const { toast } = useToast();

  if (isPending) {
    return (
      <div className="max-w-content-narrow mx-auto space-y-4">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
        <Skeleton className="h-24 w-full rounded-radius-md" />
      </div>
    );
  }

  const handleCheckout = () => {
    checkout.mutate(undefined, {
      onSuccess: (data) => {
        if (data.checkout_url) {
          window.location.href = data.checkout_url;
        }
      },
      onError: () => {
        toast(
          intl.formatMessage({ id: "marketplace.cart.checkout.error" }),
          "error",
        );
      },
    });
  };

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.cart.title" })}
      />

      <RouterLink
        to="/marketplace"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.continueShopping" />
      </RouterLink>

      {(!cart || cart.item_count === 0) && (
        <EmptyState
          illustration={<Icon icon={ShoppingCart} size="xl" />}
          message={intl.formatMessage({ id: "marketplace.cart.empty.title" })}
          description={intl.formatMessage({
            id: "marketplace.cart.empty.description",
          })}
          action={
            <RouterLink to="/marketplace">
              <Button variant="primary">
                <FormattedMessage id="marketplace.continueShopping" />
              </Button>
            </RouterLink>
          }
        />
      )}

      {cart && cart.item_count > 0 && (
        <>
          <div className="space-y-3 mb-6">
            {cart.items.map((item) => (
              <Card
                key={item.listing_id}
                className="p-card-padding flex items-center gap-4"
              >
                {item.thumbnail_url ? (
                  <div className="w-16 h-16 rounded-radius-sm overflow-hidden shrink-0 bg-surface-container-low">
                    <img
                      src={item.thumbnail_url}
                      alt={item.title}
                      className="w-full h-full object-cover"
                    />
                  </div>
                ) : (
                  <div className="w-16 h-16 rounded-radius-sm shrink-0 bg-surface-container-low flex items-center justify-center">
                    <Icon
                      icon={ShoppingCart}
                      size="md"
                      className="text-on-surface-variant"
                    />
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <RouterLink
                    to={`/marketplace/listings/${item.listing_id}`}
                    className="type-title-sm text-on-surface hover:text-primary transition-colors"
                  >
                    {item.title}
                  </RouterLink>
                  <p className="type-title-sm text-primary mt-1">
                    ${(item.price_cents / 100).toFixed(2)}
                  </p>
                </div>
                <button
                  onClick={() => removeFromCart.mutate(item.listing_id)}
                  disabled={removeFromCart.isPending}
                  className="p-2 text-on-surface-variant hover:text-error transition-colors rounded-radius-sm"
                  aria-label={intl.formatMessage({
                    id: "marketplace.cart.remove",
                  })}
                >
                  <Icon icon={Trash2} size="sm" />
                </button>
              </Card>
            ))}
          </div>

          {/* Total & checkout */}
          <Card className="p-card-padding">
            <div className="flex items-center justify-between mb-4">
              <span className="type-title-md text-on-surface">
                <FormattedMessage id="marketplace.cart.total" />
              </span>
              <span className="type-headline-sm text-primary">
                ${(cart.total_cents / 100).toFixed(2)}
              </span>
            </div>
            <Button
              variant="primary"
              className="w-full"
              onClick={handleCheckout}
              disabled={checkout.isPending}
            >
              <FormattedMessage id="marketplace.cart.checkout" />
            </Button>
          </Card>
        </>
      )}
    </div>
  );
}
