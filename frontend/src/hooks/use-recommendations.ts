import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ───────────────────────────────────────────────────────────────────

export interface Recommendation {
  id: string;
  type: "content" | "activity" | "resource";
  title: string;
  description: string;
  reason: string;
  ai_generated: boolean;
  category: string;
  link?: string;
}

export interface RecommendationPreferences {
  blocked_categories: string[];
  show_ai_recommendations: boolean;
}

// ─── Hooks ───────────────────────────────────────────────────────────────────

export function useRecommendations() {
  return useQuery({
    queryKey: ["recommendations"],
    queryFn: () =>
      apiClient<Recommendation[]>("/v1/recommendations"),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useDismissRecommendation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/recommendations/${id}/dismiss`, {
        method: "POST",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}

export function useUndoDismiss() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/recommendations/${id}/undo-dismiss`, {
        method: "POST",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}

export function useBlockCategory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (category: string) =>
      apiClient<void>(
        `/v1/recommendations/categories/${encodeURIComponent(category)}/block`,
        { method: "POST" },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["recommendations"] });
      qc.invalidateQueries({ queryKey: ["recommendations", "preferences"] });
    },
  });
}

export function useRecommendationPreferences() {
  return useQuery({
    queryKey: ["recommendations", "preferences"],
    queryFn: () =>
      apiClient<RecommendationPreferences>("/v1/recommendations/preferences"),
  });
}
