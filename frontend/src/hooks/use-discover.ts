import { useQuery, useMutation } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

// ─── Type aliases (from generated schema) ──────────────────────────────────

type QuizResponse = components["schemas"]["discover.QuizResponse"];
type QuizResultResponse = components["schemas"]["discover.QuizResultResponse"];
type SubmitQuizCommand = components["schemas"]["discover.SubmitQuizCommand"];
type StateGuideSummaryResponse =
  components["schemas"]["discover.StateGuideSummaryResponse"];
type StateGuideResponse = components["schemas"]["discover.StateGuideResponse"];

// ─── Queries ────────────────────────────────────────────────────────────────

export function useDiscoverQuiz() {
  return useQuery({
    queryKey: ["discover", "quiz"],
    queryFn: () => apiClient<QuizResponse>("/v1/discovery/quiz"),
    staleTime: 1000 * 60 * 60, // 60 min — quiz definition rarely changes
  });
}

export function useQuizResult(shareId: string | undefined) {
  return useQuery({
    queryKey: ["discover", "quiz", "results", shareId],
    queryFn: () =>
      apiClient<QuizResultResponse>(
        `/v1/discovery/quiz/results/${shareId ?? ""}`,
      ),
    enabled: !!shareId,
  });
}

export function useStateGuides() {
  return useQuery({
    queryKey: ["discover", "state-guides"],
    queryFn: () =>
      apiClient<StateGuideSummaryResponse[]>("/v1/discovery/state-guides"),
    staleTime: 1000 * 60 * 60, // 60 min — state guides rarely change
  });
}

export function useStateGuide(stateCode: string | undefined) {
  return useQuery({
    queryKey: ["discover", "state-guides", stateCode],
    queryFn: () =>
      apiClient<StateGuideResponse>(
        `/v1/discovery/state-guides/${stateCode ?? ""}`,
      ),
    enabled: !!stateCode,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useSubmitQuiz() {
  return useMutation({
    mutationFn: (body: SubmitQuizCommand) =>
      apiClient<QuizResultResponse>("/v1/discovery/quiz/results", {
        method: "POST",
        body,
      }),
  });
}
