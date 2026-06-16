import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

// ─── Type aliases (from generated schema) ──────────────────────────────────

export type LearnerProfile =
  components["schemas"]["learner_profile.LearnerProfileResponse"];
export type QuizAnswer =
  components["schemas"]["learner_profile.QuizAnswer"];
export type SubmitProfileCommand =
  components["schemas"]["learner_profile.SubmitProfileCommand"];

// ─── Hooks ───────────────────────────────────────────────────────────────────

export function useProfile(studentId: string | undefined) {
  return useQuery({
    queryKey: ["learner-profile", studentId],
    queryFn: () =>
      apiClient<LearnerProfile>(
        `/v1/students/${studentId ?? ""}/learner-profile`,
      ),
    enabled: !!studentId,
    retry: (failureCount, error) => {
      // 404 means no profile yet — don't retry
      const anyErr = error as { error?: { code?: string } };
      if (anyErr?.error?.code === "not_found") return false;
      return failureCount < 2;
    },
    staleTime: 5 * 60 * 1000,
  });
}

export function useSubmitQuiz(studentId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: SubmitProfileCommand) =>
      apiClient<LearnerProfile>(
        `/v1/students/${studentId ?? ""}/learner-profile/submissions`,
        { method: "POST", body },
      ),
    onSuccess: (data) => {
      qc.setQueryData(["learner-profile", studentId], data);
      void qc.invalidateQueries({ queryKey: ["recommendations"] });
    },
  });
}
