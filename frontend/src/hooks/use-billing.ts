import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ───────────────────────────────────────────────────────────────────

export interface MicroChargeStatus {
  id: string;
  amount_cents: number;
  status: "pending" | "verified" | "failed";
  created_at: string;
}

// ─── COPPA Micro-Charge Verification ─────────────────────────────────────────

export function useMicroChargeStatus() {
  return useQuery({
    queryKey: ["billing", "micro-charge"],
    queryFn: () =>
      apiClient<MicroChargeStatus>("/v1/billing/micro-charge/status"),
  });
}

export function useInitMicroCharge() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<MicroChargeStatus>("/v1/billing/micro-charge/init", {
        method: "POST",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["billing", "micro-charge"] });
    },
  });
}

export function useVerifyMicroCharge() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (amount_cents: number) =>
      apiClient<void>("/v1/billing/micro-charge/verify", {
        method: "POST",
        body: JSON.stringify({ amount_cents }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["billing", "micro-charge"] });
      qc.invalidateQueries({ queryKey: ["family", "consent"] });
    },
  });
}
