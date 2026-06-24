import { createContext, useContext, type ReactNode } from "react";
import { createHearth, HearthProvider } from "@hearth-auth/sdk";
import { useAuth } from "@/hooks/use-auth";
import type { CurrentUser } from "@/types";

// BFF mode: browser never holds a JWT — getToken always returns null.
// All hasPermission/hasRole checks return false; the provider is wired so
// future claim-surfacing work (e.g. thin-JWT endpoint) can opt in. [ARCH ADR-020]
const hearthFacade = createHearth({
  baseUrl: (import.meta.env.VITE_HEARTH_URL as string | undefined) ?? "http://localhost:4933",
  realmId: (import.meta.env.VITE_HEARTH_REALM_ID as string | undefined) ?? "homegrown",
  getToken: () => null,
});

type AuthContextValue = {
  user: CurrentUser | undefined;
  isLoading: boolean;
  isAuthenticated: boolean;
  isParent: boolean;
  isPrimaryParent: boolean;
  isPlatformAdmin: boolean;
  tier: string;
  coppaStatus: string;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  return (
    <HearthProvider client={hearthFacade}>
      <AuthContext.Provider value={auth}>{children}</AuthContext.Provider>
    </HearthProvider>
  );
}

export function useAuthContext() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuthContext must be used within <AuthProvider>");
  }
  return ctx;
}
