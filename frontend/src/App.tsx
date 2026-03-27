import { RouterProvider } from "react-router";
import { IntlProvider } from "react-intl";
import { ToastProvider } from "@/components/ui";
import { NetworkStatus } from "@/components/common";
import { AuthProvider } from "@/features/auth/auth-provider";
import { MethodologyProvider } from "@/features/auth/methodology-provider";
import { defaultLocale, getMessages } from "@/lib/i18n";
import { router } from "@/routes";

export function App() {
  return (
    <IntlProvider locale={defaultLocale} messages={getMessages(defaultLocale)}>
      <ToastProvider>
        <AuthProvider>
          <MethodologyProvider>
            <NetworkStatus />
            <RouterProvider router={router} />
          </MethodologyProvider>
        </AuthProvider>
      </ToastProvider>
    </IntlProvider>
  );
}
