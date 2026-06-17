import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

// ─── Type aliases (from generated schema) ──────────────────────────────────

export type RecommendationListResponse =
  components["schemas"]["recs.RecommendationListResponse"];
export type Recommendation =
  components["schemas"]["recs.RecommendationResponse"];
export type RecommendationPreferences =
  components["schemas"]["recs.RecommendationPreferencesResponse"];
export type UpdatePreferencesCommand =
  components["schemas"]["recs.UpdatePreferencesCommand"];

// ─── Hooks ───────────────────────────────────────────────────────────────────

export function useRecommendations(options?: { forStudentId?: string }) {
  const forStudentId = options?.forStudentId;
  return useQuery({
    queryKey: ["recommendations", forStudentId],
    queryFn: () => {
      const url = forStudentId
        ? `/v1/recommendations?for_student_id=${encodeURIComponent(forStudentId)}`
        : "/v1/recommendations";
      return apiClient<RecommendationListResponse>(url);
    },
    select: (data) => data.recommendations ?? [],
    staleTime: 5 * 60 * 1000,
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
      void qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}

// Undo a dismiss or block action — DELETE /recommendations/:id/feedback [13-recs §13.2]
export function useUndoFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/recommendations/${id}/feedback`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}

// Block a specific recommendation by ID — POST /recommendations/:id/block [13-recs §13.2]
export function useBlockRecommendation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/recommendations/${id}/block`, { method: "POST" }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}

export function useRecommendationPreferences() {
  return useQuery({
    queryKey: ["recommendations", "preferences"],
    queryFn: () =>
      apiClient<RecommendationPreferences>("/v1/recommendations/preferences"),
    staleTime: 5 * 60 * 1000,
  });
}

export function useUpdatePreferences() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdatePreferencesCommand) =>
      apiClient<RecommendationPreferences>("/v1/recommendations/preferences", {
        method: "PATCH",
        body,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["recommendations", "preferences"] });
      void qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}
