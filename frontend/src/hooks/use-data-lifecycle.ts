import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Data lifecycle endpoints exist in the backend but aren't yet fully annotated
// in swag. We define lightweight frontend-only types until they are.

export type ExportFormat = "json" | "csv";

export type ExportStatus = "pending" | "processing" | "completed" | "expired" | "failed";

/** Full export detail returned by the single-export status endpoint. */
export interface ExportRequest {
  id: string;
  format: ExportFormat;
  status: ExportStatus;
  download_url?: string;
  expires_at?: string;
  created_at: string;
}

/** Lighter summary returned in paginated list responses. */
export interface ExportSummary {
  id: string;
  status: ExportStatus;
  format: ExportFormat;
  size_bytes?: number;
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

interface PaginatedExports {
  items: ExportSummary[];
  total: number;
}

export function useExportList() {
  return useQuery({
    queryKey: ["data", "exports"],
    queryFn: () => apiClient<PaginatedExports>("/v1/account/exports"),
    staleTime: 1000 * 30,
  });
}

export function useExportStatus(exportId: string | undefined) {
  return useQuery({
    queryKey: ["data", "exports", exportId],
    queryFn: () =>
      apiClient<ExportRequest>(`/v1/account/export/${exportId ?? ""}`),
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
      apiClient<{ export_id: string; status: string }>("/v1/account/export", {
        method: "POST",
        body: { format: params.format, include_domains: params.domains },
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
    queryFn: async (): Promise<DeletionRequest> => {
      try {
        return await apiClient<DeletionRequest>("/v1/account/deletion");
      } catch (err: unknown) {
        // 404 = no active deletion request — return default "none" state
        const status = (err as { error?: { code?: number } })?.error?.code;
        if (status === 404) {
          return { status: "none" };
        }
        throw err;
      }
    },
    staleTime: 1000 * 60,
  });
}

export function useRequestDeletion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/account/deletion", { method: "POST" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["data", "deletion"] });
    },
  });
}

export function useCancelDeletion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/account/deletion", { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["data", "deletion"] });
    },
  });
}
