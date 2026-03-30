import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Learning domain types are not yet in the generated schema. These frontend-only
// types match the Go response structs and will be replaced once swag annotations
// are added to the backend handlers.

export interface AttachmentInput {
  url: string;
  content_type: string;
  filename?: string;
}

export interface ActivityDefResponse {
  id: string;
  publisher_id: string;
  title: string;
  description?: string;
  subject_tags: string[];
  methodology_id?: string;
  tool_id?: string;
  est_duration_minutes?: number;
  attachments: AttachmentInput[];
  created_at: string;
  updated_at: string;
}

export interface ActivityDefSummaryResponse {
  id: string;
  title: string;
  subject_tags: string[];
  methodology_id?: string;
  est_duration_minutes?: number;
}

export interface ActivityLogResponse {
  id: string;
  student_id: string;
  title: string;
  description?: string;
  subject_tags: string[];
  content_id?: string;
  content_title?: string;
  methodology_id?: string;
  tool_id?: string;
  duration_minutes?: number;
  attachments: AttachmentInput[];
  activity_date: string;
  created_at: string;
}

interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ─── Activity Definition Queries ────────────────────────────────────────────

export function useActivityDefs(params?: {
  subject?: string;
  methodology_id?: string;
  search?: string;
}) {
  return useInfiniteQuery({
    queryKey: ["learning", "activity-defs", params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.subject) sp.set("subject", params.subject);
      if (params?.methodology_id)
        sp.set("methodology_id", params.methodology_id);
      if (params?.search) sp.set("search", params.search);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<ActivityDefSummaryResponse>>(
        `/v1/learning/activity-defs${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
  });
}

export function useActivityDef(id: string) {
  return useQuery({
    queryKey: ["learning", "activity-defs", id],
    queryFn: () =>
      apiClient<ActivityDefResponse>(`/v1/learning/activity-defs/${id}`),
    enabled: !!id,
  });
}

// ─── Activity Definition Mutations ──────────────────────────────────────────

export function useCreateActivityDef() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      title: string;
      description?: string;
      subject_tags?: string[];
      methodology_id?: string;
      tool_id?: string;
      est_duration_minutes?: number;
      attachments?: AttachmentInput[];
    }) =>
      apiClient<ActivityDefResponse>("/v1/learning/activity-defs", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "activity-defs"],
      });
    },
  });
}

// ─── Activity Log Queries ───────────────────────────────────────────────────

export function useActivityLog(
  studentId: string,
  params?: {
    subject?: string;
    date_from?: string;
    date_to?: string;
  },
) {
  return useInfiniteQuery({
    queryKey: ["learning", "activity-log", studentId, params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.subject) sp.set("subject", params.subject);
      if (params?.date_from) sp.set("date_from", params.date_from);
      if (params?.date_to) sp.set("date_to", params.date_to);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<ActivityLogResponse>>(
        `/v1/learning/students/${studentId}/activities${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!studentId,
  });
}

export function useActivityLogEntry(studentId: string, id: string) {
  return useQuery({
    queryKey: ["learning", "activity-log", studentId, id],
    queryFn: () =>
      apiClient<ActivityLogResponse>(
        `/v1/learning/students/${studentId}/activities/${id}`,
      ),
    enabled: !!studentId && !!id,
  });
}

// ─── Activity Log Mutations ─────────────────────────────────────────────────

export function useLogActivity(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      title: string;
      description?: string;
      subject_tags?: string[];
      content_id?: string;
      methodology_id?: string;
      tool_id?: string;
      duration_minutes?: number;
      attachments?: AttachmentInput[];
      activity_date?: string;
    }) =>
      apiClient<ActivityLogResponse>(
        `/v1/learning/students/${studentId}/activities`,
        { method: "POST", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "activity-log", studentId],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}

export function useUpdateActivityLog(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...cmd
    }: {
      id: string;
      title?: string;
      description?: string;
      subject_tags?: string[];
      duration_minutes?: number;
      attachments?: AttachmentInput[];
      activity_date?: string;
    }) =>
      apiClient<ActivityLogResponse>(
        `/v1/learning/students/${studentId}/activities/${id}`,
        { method: "PATCH", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "activity-log", studentId],
      });
    },
  });
}

export function useDeleteActivityLog(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/learning/students/${studentId}/activities/${id}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "activity-log", studentId],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}
