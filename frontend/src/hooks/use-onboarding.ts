import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

type WizardProgress = components["schemas"]["onboard.WizardProgressResponse"];
type UpdateFamilyProfileCommand =
  components["schemas"]["onboard.UpdateFamilyProfileCommand"];
type AddChildCommand = components["schemas"]["onboard.AddChildCommand"];
type SelectMethodologyCommand =
  components["schemas"]["onboard.SelectMethodologyCommand"];
type ImportQuizCommand = components["schemas"]["onboard.ImportQuizCommand"];
type QuizImportResponse = components["schemas"]["onboard.QuizImportResponse"];
type RoadmapResponse = components["schemas"]["onboard.RoadmapResponse"];
type RecommendationsResponse =
  components["schemas"]["onboard.RecommendationsResponse"];
type CommunityResponse = components["schemas"]["onboard.CommunityResponse"];

export function useOnboardingProgress() {
  return useQuery({
    queryKey: ["onboarding", "progress"],
    queryFn: () => apiClient<WizardProgress>("/v1/onboarding/progress"),
    staleTime: 1000 * 30,
    retry: 1,
  });
}

export function useUpdateFamilyProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateFamilyProfileCommand) =>
      apiClient<WizardProgress>("/v1/onboarding/family-profile", {
        method: "PATCH",
        body,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(["onboarding", "progress"], data);
    },
  });
}

export function useAddChild() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: AddChildCommand) =>
      apiClient<WizardProgress>("/v1/onboarding/children", {
        method: "POST",
        body,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(["onboarding", "progress"], data);
      void queryClient.invalidateQueries({ queryKey: ["family", "students"] });
    },
  });
}

export function useRemoveChild() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (childId: string) =>
      apiClient<void>(`/v1/onboarding/children/${childId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "students"] });
    },
  });
}

export function useSelectMethodology() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: SelectMethodologyCommand) =>
      apiClient<WizardProgress>("/v1/onboarding/methodology", {
        method: "PATCH",
        body,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(["onboarding", "progress"], data);
      void queryClient.invalidateQueries({ queryKey: ["onboarding", "roadmap"] });
      void queryClient.invalidateQueries({ queryKey: ["onboarding", "recommendations"] });
      void queryClient.invalidateQueries({ queryKey: ["onboarding", "community"] });
    },
  });
}

export function useImportQuiz() {
  return useMutation({
    mutationFn: (body: ImportQuizCommand) =>
      apiClient<QuizImportResponse>("/v1/onboarding/methodology/import-quiz", {
        method: "POST",
        body,
      }),
  });
}

export function useOnboardingRoadmap(enabled = true) {
  return useQuery({
    queryKey: ["onboarding", "roadmap"],
    queryFn: () => apiClient<RoadmapResponse>("/v1/onboarding/roadmap"),
    enabled,
    staleTime: 1000 * 60 * 5,
  });
}

export function useOnboardingRecommendations(enabled = true) {
  return useQuery({
    queryKey: ["onboarding", "recommendations"],
    queryFn: () =>
      apiClient<RecommendationsResponse>("/v1/onboarding/recommendations"),
    enabled,
    staleTime: 1000 * 60 * 5,
  });
}

export function useOnboardingCommunity(enabled = true) {
  return useQuery({
    queryKey: ["onboarding", "community"],
    queryFn: () => apiClient<CommunityResponse>("/v1/onboarding/community"),
    enabled,
    staleTime: 1000 * 60 * 5,
  });
}

export function useCompleteOnboarding() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<WizardProgress>("/v1/onboarding/complete", { method: "POST" }),
    onSuccess: (data) => {
      queryClient.setQueryData(["onboarding", "progress"], data);
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useSkipOnboarding() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<WizardProgress>("/v1/onboarding/skip", { method: "POST" }),
    onSuccess: (data) => {
      queryClient.setQueryData(["onboarding", "progress"], data);
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useCompleteRoadmapItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (itemId: string) =>
      apiClient<void>(`/v1/onboarding/roadmap/${itemId}/complete`, {
        method: "PATCH",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["onboarding", "roadmap"] });
    },
  });
}
