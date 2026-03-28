import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

type MethodologySummary =
  components["schemas"]["method.MethodologySummaryResponse"];
type MethodologyDetail =
  components["schemas"]["method.MethodologyDetailResponse"];

export function useMethodologyList() {
  return useQuery({
    queryKey: ["methodologies"],
    queryFn: () => apiClient<MethodologySummary[]>("/v1/methodologies"),
    staleTime: 1000 * 60 * 10,
  });
}

export function useMethodologyDetail(slug: string | undefined) {
  return useQuery({
    queryKey: ["methodologies", slug],
    queryFn: () =>
      apiClient<MethodologyDetail>(`/v1/methodologies/${slug ?? ""}`),
    enabled: !!slug,
    staleTime: 1000 * 60 * 10,
  });
}
