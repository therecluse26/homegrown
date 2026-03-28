import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { CurrentUser } from "@/types";

export function useAuth() {
  const query = useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => apiClient<CurrentUser>("/v1/auth/me"),
    retry: false,
    staleTime: 1000 * 60 * 5,
  });

  const user = query.data;

  return {
    user,
    isLoading: query.isPending,
    isAuthenticated: query.isSuccess && !!user,
    isParent: query.isSuccess && !!user?.parent_id,
    isPrimaryParent: !!user?.is_primary_parent,
    tier: user?.subscription_tier ?? "free",
    coppaStatus: user?.coppa_consent_status ?? "unknown",
    error: query.error,
  };
}
