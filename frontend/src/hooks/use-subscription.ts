import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

// ─── Type aliases (from generated schema) ──────────────────────────────────

export type Subscription =
  components["schemas"]["billing.SubscriptionResponse"];
export type Transaction =
  components["schemas"]["billing.TransactionResponse"];
export type TransactionListResponse =
  components["schemas"]["billing.TransactionListResponse"];
export type PaymentMethod =
  components["schemas"]["billing.PaymentMethodResponse"];
export type ChangePlanRequest =
  components["schemas"]["billing.UpdateSubscriptionCommand"];

// ─── Local filter type (not returned by API) ────────────────────────────────

export interface TransactionFilters {
  type?: string;
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
      apiClient<Subscription>("/v1/billing/subscription", {
        method: "PATCH",
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
      apiClient<void>("/v1/billing/subscription", {
        method: "DELETE",
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

export function useAddPaymentMethod() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<PaymentMethod>("/v1/billing/payment-methods", {
        method: "POST",
        body: JSON.stringify({ setup_intent_client_secret: "" }),
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
      return apiClient<TransactionListResponse>(
        `/v1/billing/transactions${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 30, // 30s
  });
}
