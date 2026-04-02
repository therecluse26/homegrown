import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Hand-written until billing swag annotations produce matching generated types.
// Tracked: specs/gaps_03_31_26.md §FE-6

export type BillingInterval = "monthly" | "annual";

export interface Subscription {
  id: string;
  plan_id: string;
  plan_name: string;
  tier: "free" | "plus" | "premium";
  interval: BillingInterval;
  status: "active" | "cancelled" | "past_due" | "trialing";
  current_period_start: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
  cancelled_at: string | null;
  amount_cents: number;
  currency: string;
}

export interface ChangePlanRequest {
  plan_id: string;
  interval: BillingInterval;
}

export interface PaymentMethod {
  id: string;
  type: "card" | "bank_account";
  brand: string;
  last4: string;
  exp_month: number;
  exp_year: number;
  is_default: boolean;
}

export interface Transaction {
  id: string;
  type: "subscription" | "purchase" | "payout" | "refund";
  status: "completed" | "pending" | "failed" | "refunded";
  amount_cents: number;
  currency: string;
  description: string;
  created_at: string;
}

export interface TransactionFilters {
  type?: Transaction["type"];
  from?: string;
  to?: string;
  page?: number;
}

// ─── Subscription Queries ───────────────────────────────────────────────────

export function useSubscription() {
  return useQuery({
    queryKey: ["billing", "subscription"],
    queryFn: () => apiClient<Subscription>("/v1/billing/subscription"),
    staleTime: 1000 * 60, // 1 min
  });
}

export function useChangePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: ChangePlanRequest) =>
      apiClient<Subscription>("/v1/billing/subscription/change", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing"] });
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useCancelSubscription() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/billing/subscription/cancel", {
        method: "POST",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing"] });
    },
  });
}

export function useReactivateSubscription() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/billing/subscription/reactivate", {
        method: "POST",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing"] });
    },
  });
}

// ─── Payment Method Queries ─────────────────────────────────────────────────

export function usePaymentMethods() {
  return useQuery({
    queryKey: ["billing", "payment-methods"],
    queryFn: () => apiClient<PaymentMethod[]>("/v1/billing/payment-methods"),
    staleTime: 1000 * 60, // 1 min
  });
}

export function useSetDefaultPaymentMethod() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/billing/payment-methods/${id}/default`, {
        method: "POST",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["billing", "payment-methods"],
      });
    },
  });
}

export function useRemovePaymentMethod() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/billing/payment-methods/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["billing", "payment-methods"],
      });
    },
  });
}

// ─── Transaction Queries ────────────────────────────────────────────────────

export function useTransactions(filters?: TransactionFilters) {
  return useQuery({
    queryKey: ["billing", "transactions", filters],
    queryFn: () => {
      const params = new URLSearchParams();
      if (filters?.type) params.set("type", filters.type);
      if (filters?.from) params.set("from", filters.from);
      if (filters?.to) params.set("to", filters.to);
      if (filters?.page) params.set("page", String(filters.page));
      const qs = params.toString();
      return apiClient<{ transactions: Transaction[]; total: number }>(
        `/v1/billing/transactions${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 30, // 30s
  });
}
