import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Notification types are not yet in the generated schema, so we define
// lightweight frontend-only types. These will be replaced with generated types
// once the backend notification endpoints are wired into swag annotations.

export type NotificationType =
  | "friend_request_received"
  | "friend_request_accepted"
  | "message_received"
  | "content_flagged"
  | "event_cancelled"
  | "purchase_completed"
  | "review_received"
  | "subscription_created"
  | "subscription_cancelled"
  | "subscription_renewed"
  | "streak_milestone"
  | "learning_milestone"
  | "attendance_threshold_warning"
  | "payout_completed"
  | "system";

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  deep_link?: string;
  read: boolean;
  created_at: string;
  /** Optional reference for actionable notifications (e.g. friend request ID) */
  reference_id?: string;
}

interface NotificationListResponse {
  notifications: Notification[];
  total: number;
  unread_count: number;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useNotifications(params?: {
  page?: number;
  type?: NotificationType;
  read?: boolean;
}) {
  return useQuery({
    queryKey: ["notifications", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      if (params?.page) searchParams.set("page", String(params.page));
      if (params?.type) searchParams.set("type", params.type);
      if (params?.read !== undefined)
        searchParams.set("read", String(params.read));
      const qs = searchParams.toString();
      return apiClient<NotificationListResponse>(
        `/v1/notifications${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 30, // 30s — notifications should feel fresh
  });
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ["notifications", "unread-count"],
    queryFn: () =>
      apiClient<{ count: number }>("/v1/notifications/unread-count"),
    staleTime: 1000 * 15, // 15s — poll frequently for badge
    refetchInterval: 1000 * 30, // Auto-refetch every 30s
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useMarkRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/notifications/${id}/read`, { method: "POST" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

export function useMarkAllRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<void>("/v1/notifications/read-all", { method: "POST" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}
