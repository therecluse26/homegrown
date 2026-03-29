import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// Session list/revoke endpoints aren't yet in the generated schema.
// These lightweight types will be replaced once the IAM handler is wired.

export interface Session {
  id: string;
  device: string;
  browser: string;
  ip_address: string;
  last_active: string;
  is_current: boolean;
  created_at: string;
}

export function useSessions() {
  return useQuery({
    queryKey: ["auth", "sessions"],
    queryFn: () => apiClient<Session[]>("/v1/auth/sessions"),
    staleTime: 1000 * 30,
  });
}

export function useRevokeSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (sessionId: string) =>
      apiClient<void>(`/v1/auth/sessions/${sessionId}`, { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "sessions"] });
    },
  });
}

export function useRevokeAllSessions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/auth/sessions/revoke-all", { method: "POST" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "sessions"] });
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}
