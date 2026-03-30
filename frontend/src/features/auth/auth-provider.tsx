import { createContext, useContext, type ReactNode } from "react";
import { useAuth } from "@/hooks/use-auth";
import type { CurrentUser } from "@/types";

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
  return <AuthContext.Provider value={auth}>{children}</AuthContext.Provider>;
}

export function useAuthContext() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuthContext must be used within <AuthProvider>");
  }
  return ctx;
}
