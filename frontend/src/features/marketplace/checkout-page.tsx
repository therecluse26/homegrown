import { useEffect, useRef, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useSearchParams, Link as RouterLink } from "react-router";
import { ArrowLeft, CheckCircle, AlertCircle } from "lucide-react";
import { Button, Card, Icon, Skeleton } from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";

type CheckoutState = "loading" | "ready" | "processing" | "success" | "error";

export function CheckoutPage() {
  const intl = useIntl();
  const [searchParams] = useSearchParams();
  const paymentId = searchParams.get("payment_id");
  const clientSecret = searchParams.get("client_secret");
  const publishableKey = searchParams.get("publishable_key");

  const mountRef = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<CheckoutState>("loading");
  const [errorMsg, setErrorMsg] = useState<string>("");

  useEffect(() => {
    if (!clientSecret || !publishableKey || !paymentId) {
      setState("error");
      setErrorMsg("Missing payment details.");
      return;
    }

    let cancelled = false;

    // Load the Hyperswitch web SDK dynamically. The SDK is served from the
    // Hyperswitch CDN and requires a publishable key + client secret.
    const script = document.createElement("script");
    script.src = "https://beta.hyperswitch.io/v1/HyperLoader.js";
    script.async = true;

    script.onerror = () => {
      if (!cancelled) {
        setState("error");
        setErrorMsg("Payment SDK could not be loaded. Please check your connection.");
      }
    };

    script.onload = () => {
      if (cancelled || !mountRef.current) return;
      try {
        // @ts-expect-error — HyperLoader is a global injected by the Hyperswitch SDK
        const hyper = window.Hyper(publishableKey, {
          customBackendUrl: undefined, // use production Hyperswitch
        });

        const appearance = { theme: "default" as const };
        const elements = hyper.elements({ appearance, clientSecret });
        const unifiedCheckout = elements.create("unifiedCheckout", {
          wallets: { applePay: "never", googlePay: "never" },
        });

        if (mountRef.current) {
          unifiedCheckout.mount(mountRef.current);
          setState("ready");
        }

        mountRef.current?.closest("form")?.addEventListener("submit", async (e) => {
          e.preventDefault();
          setState("processing");
          const { error } = await hyper.confirmPayment({
            elements,
            confirmParams: {
              return_url: `${window.location.origin}/marketplace/purchases`,
            },
          });
          if (error) {
            setErrorMsg(error.message ?? "Payment failed.");
            setState("error");
          } else {
            setState("success");
          }
        });
      } catch {
        if (!cancelled) {
          setState("error");
          setErrorMsg("Could not initialise payment form.");
        }
      }
    };

    document.head.appendChild(script);

    return () => {
      cancelled = true;
      // Script tag is intentionally left — removing it would break the SDK global.
    };
  }, [clientSecret, publishableKey, paymentId]);

  if (state === "success") {
    return (
      <div className="max-w-content-narrow mx-auto">
        <PageTitle title={intl.formatMessage({ id: "marketplace.checkout.success.title", defaultMessage: "Payment successful" })} />
        <Card className="p-card-padding text-center space-y-4">
          <Icon icon={CheckCircle} size="xl" className="text-success mx-auto" />
          <p className="type-title-md text-on-surface">
            <FormattedMessage id="marketplace.checkout.success.message" defaultMessage="Your purchase is complete!" />
          </p>
          <RouterLink to="/marketplace/purchases">
            <Button variant="primary">
              <FormattedMessage id="marketplace.viewPurchases" defaultMessage="View my purchases" />
            </Button>
          </RouterLink>
        </Card>
      </div>
    );
  }

  if (state === "error" && (!clientSecret || !publishableKey)) {
    return (
      <div className="max-w-content-narrow mx-auto">
        <PageTitle title={intl.formatMessage({ id: "marketplace.checkout.title", defaultMessage: "Checkout" })} />
        <RouterLink
          to="/marketplace/cart"
          className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
        >
          <Icon icon={ArrowLeft} size="sm" />
          <FormattedMessage id="marketplace.backToCart" defaultMessage="Back to cart" />
        </RouterLink>
        <Card className="p-card-padding text-center space-y-4">
          <Icon icon={AlertCircle} size="xl" className="text-error mx-auto" />
          <p className="type-body-md text-on-surface-variant">
            {errorMsg || <FormattedMessage id="marketplace.checkout.error.generic" defaultMessage="Payment is not available right now. Please try again later." />}
          </p>
          <RouterLink to="/marketplace/cart">
            <Button variant="secondary">
              <FormattedMessage id="marketplace.backToCart" defaultMessage="Back to cart" />
            </Button>
          </RouterLink>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle title={intl.formatMessage({ id: "marketplace.checkout.title", defaultMessage: "Checkout" })} />

      <RouterLink
        to="/marketplace/cart"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.backToCart" defaultMessage="Back to cart" />
      </RouterLink>

      <Card className="p-card-padding">
        {state === "loading" && (
          <div className="space-y-3">
            <Skeleton className="h-10 w-full rounded-radius-sm" />
            <Skeleton className="h-10 w-full rounded-radius-sm" />
            <Skeleton className="h-10 w-2/3 rounded-radius-sm" />
          </div>
        )}

        {state === "error" && (
          <div className="flex flex-col items-center gap-3 py-6">
            <Icon icon={AlertCircle} size="xl" className="text-error" />
            <p className="type-body-md text-on-surface-variant text-center">{errorMsg}</p>
            <RouterLink to="/marketplace/cart">
              <Button variant="secondary">
                <FormattedMessage id="marketplace.backToCart" defaultMessage="Back to cart" />
              </Button>
            </RouterLink>
          </div>
        )}

        <form>
          {/* Hyperswitch SDK mounts its payment element here */}
          <div ref={mountRef} className="mb-4" />

          {(state === "ready" || state === "processing") && (
            <Button
              type="submit"
              variant="primary"
              className="w-full"
              disabled={state === "processing"}
            >
              {state === "processing" ? (
                <FormattedMessage id="marketplace.checkout.processing" defaultMessage="Processing…" />
              ) : (
                <FormattedMessage id="marketplace.checkout.pay" defaultMessage="Pay now" />
              )}
            </Button>
          )}
        </form>
      </Card>
    </div>
  );
}
