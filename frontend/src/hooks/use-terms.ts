import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export interface TermsAcceptanceStatus {
  needs_acceptance: boolean;
  document_type: "terms" | "privacy";
  current_version: string;
  accepted_version: string | null;
  accepted_at: string | null;
}

export interface AcceptTermsRequest {
  document_type: "terms" | "privacy";
  version: string;
}

// ─── Hook ───────────────────────────────────────────────────────────────────

export function useTermsAcceptance() {
  const queryClient = useQueryClient();

  const status = useQuery({
    queryKey: ["legal", "terms-acceptance"],
    queryFn: () =>
      apiClient<TermsAcceptanceStatus>("/v1/legal/terms-acceptance"),
    staleTime: 1000 * 60 * 5, // 5 min — terms versions change infrequently
  });

  const acceptMutation = useMutation({
    mutationFn: (body: AcceptTermsRequest) =>
      apiClient<void>("/v1/legal/terms-acceptance", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["legal"] });
    },
  });

  return {
    ...status,
    accept: acceptMutation.mutateAsync,
    isAccepting: acceptMutation.isPending,
  };
}
