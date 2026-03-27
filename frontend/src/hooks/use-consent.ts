import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

type ConsentStatus = components["schemas"]["iam.ConsentStatusResponse"];
type CoppaConsentCommand = components["schemas"]["iam.CoppaConsentCommand"];

/**
 * Hook for COPPA consent status and mutations.
 *
 * Consent lifecycle: registered → noticed → consented
 * Must be completed before any student can be created.
 *
 * @see SPEC §7.3 (COPPA consent flow)
 * @see 01-iam §8 (consent endpoints)
 */
export function useConsent() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["family", "consent"],
    queryFn: () => apiClient<ConsentStatus>("/v1/families/consent"),
    staleTime: 1000 * 60 * 5,
    retry: 1,
  });

  const consentMutation = useMutation({
    mutationFn: (body: CoppaConsentCommand) =>
      apiClient<ConsentStatus>("/v1/families/consent", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "consent"] });
      void queryClient.invalidateQueries({ queryKey: ["family", "students"] });
    },
  });

  return {
    consentStatus: query.data,
    isLoading: query.isLoading,
    isConsented: query.data?.can_create_students === true,
    provideConsent: consentMutation.mutateAsync,
    isConsenting: consentMutation.isPending,
    consentError: consentMutation.error,
  };
}
