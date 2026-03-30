import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// Session types matching backend lifecycle.SessionInfo.
// Will be replaced by generated types once lifecycle handler gets swagger annotations.

export interface Session {
  session_id: string;
  device_type: string | null;
  user_agent: string | null;
  ip_address: string | null;
  last_active: string;
  is_current: boolean;
}

interface SessionListResponse {
  sessions: Session[];
}

export function useSessions() {
  return useQuery({
    queryKey: ["auth", "sessions"],
    queryFn: async () => {
      const resp = await apiClient<SessionListResponse>("/v1/account/sessions");
      return resp.sessions;
    },
    staleTime: 1000 * 30,
  });
}

export function useRevokeSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (sessionId: string) =>
      apiClient<void>(`/v1/account/sessions/${sessionId}`, { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "sessions"] });
    },
  });
}

export function useRevokeAllSessions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/account/sessions", { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "sessions"] });
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}
