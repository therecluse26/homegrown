import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Data lifecycle endpoints exist in the backend but aren't yet fully annotated
// in swag. We define lightweight frontend-only types until they are.

export type ExportFormat = "json" | "csv";

export type ExportStatus = "pending" | "processing" | "ready" | "expired" | "failed";

export interface ExportRequest {
  id: string;
  format: ExportFormat;
  domains: string[];
  status: ExportStatus;
  download_url?: string;
  expires_at?: string;
  created_at: string;
}

export type DeletionStatus =
  | "none"
  | "pending"
  | "grace_period"
  | "processing"
  | "completed";

export interface DeletionRequest {
  status: DeletionStatus;
  requested_at?: string;
  grace_period_ends_at?: string;
  days_remaining?: number;
}

// ─── Export Queries & Mutations ──────────────────────────────────────────────

export function useExportList() {
  return useQuery({
    queryKey: ["data", "exports"],
    queryFn: () => apiClient<ExportRequest[]>("/v1/data/exports"),
    staleTime: 1000 * 30,
  });
}

export function useExportStatus(exportId: string | undefined) {
  return useQuery({
    queryKey: ["data", "exports", exportId],
    queryFn: () =>
      apiClient<ExportRequest>(`/v1/data/exports/${exportId ?? ""}`),
    enabled: !!exportId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      // Poll every 5s while processing, stop once ready/failed/expired
      if (status === "pending" || status === "processing") return 5000;
      return false;
    },
  });
}

export function useRequestExport() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { format: ExportFormat; domains: string[] }) =>
      apiClient<ExportRequest>("/v1/data/exports", {
        method: "POST",
        body: params,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["data", "exports"] });
    },
  });
}

// ─── Deletion Queries & Mutations ───────────────────────────────────────────

export function useDeletionStatus() {
  return useQuery({
    queryKey: ["data", "deletion"],
    queryFn: () =>
      apiClient<DeletionRequest>("/v1/families/deletion-request"),
    staleTime: 1000 * 60,
    // Return a default "none" state if the endpoint 404s (no pending request)
    retry: (failureCount, error) => {
      if (error instanceof Error && error.message.includes("404"))
        return false;
      return failureCount < 3;
    },
  });
}

export function useRequestDeletion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/families/deletion-request", { method: "POST" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["data", "deletion"] });
    },
  });
}

export function useCancelDeletion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/families/deletion-request", { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["data", "deletion"] });
    },
  });
}
