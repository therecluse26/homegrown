import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";
import { useAuthContext } from "@/features/auth/auth-provider";

type MethodologyContext = components["schemas"]["method.MethodologyContext"];
type ActiveTool = components["schemas"]["method.ActiveToolResponse"];

export function useMethodology() {
  const { isAuthenticated } = useAuthContext();

  const contextQuery = useQuery({
    queryKey: ["family", "methodology-context"],
    queryFn: () => apiClient<MethodologyContext>("/v1/families/methodology-context"),
    enabled: isAuthenticated,
  });

  const toolsQuery = useQuery({
    queryKey: ["family", "tools"],
    queryFn: () => apiClient<ActiveTool[]>("/v1/families/tools"),
    enabled: isAuthenticated,
  });

  const ctx = contextQuery.data;

  return {
    isLoading: contextQuery.isLoading || toolsQuery.isLoading,
    primarySlug: ctx?.primary?.slug ?? null,
    primaryName: ctx?.primary?.display_name ?? null,
    secondarySlugs: ctx?.secondary?.map((s) => s.slug).filter(Boolean) ?? [],
    terminology: ctx?.terminology ?? {},
    tools: toolsQuery.data ?? [],
  };
}
