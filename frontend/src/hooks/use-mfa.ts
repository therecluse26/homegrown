import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export interface MfaStatus {
  enabled: boolean;
  method: "totp" | null;
  configured_at: string | null;
}

export interface TotpSetupResponse {
  secret: string;
  otpauth_uri: string;
  recovery_codes: string[];
}

export interface TotpVerifyRequest {
  code: string;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useMfaStatus() {
  return useQuery({
    queryKey: ["auth", "mfa", "status"],
    queryFn: () => apiClient<MfaStatus>("/v1/auth/mfa/status"),
    staleTime: 1000 * 60, // 1 min — MFA status rarely changes
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useInitTotpSetup() {
  return useMutation({
    mutationFn: () =>
      apiClient<TotpSetupResponse>("/v1/auth/mfa/totp/setup", {
        method: "POST",
      }),
  });
}

export function useVerifyTotp() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: TotpVerifyRequest) =>
      apiClient<{ recovery_codes: string[] }>("/v1/auth/mfa/totp/verify", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "mfa"] });
    },
  });
}

export function useDisableMfa() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: TotpVerifyRequest) =>
      apiClient<void>("/v1/auth/mfa/disable", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "mfa"] });
    },
  });
}
