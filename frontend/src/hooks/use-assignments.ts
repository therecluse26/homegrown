import {
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type AssignmentStatus =
  | "assigned"
  | "in_progress"
  | "completed"
  | "skipped";

export interface AssignmentResponse {
  id: string;
  student_id: string;
  assigned_by: string;
  content_type: string;
  content_id: string;
  due_date?: string;
  status: AssignmentStatus;
  assigned_at: string;
  completed_at?: string;
  created_at: string;
}

interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useAssignments(
  studentId: string,
  params?: { status?: AssignmentStatus; due_before?: string },
) {
  return useInfiniteQuery({
    queryKey: ["learning", "assignments", studentId, params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.status) sp.set("status", params.status);
      if (params?.due_before) sp.set("due_before", params.due_before);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<AssignmentResponse>>(
        `/v1/learning/students/${studentId}/assignments${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!studentId,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useCreateAssignment(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      content_type: string;
      content_id: string;
      due_date?: string;
    }) =>
      apiClient<AssignmentResponse>(
        `/v1/learning/students/${studentId}/assignments`,
        { method: "POST", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "assignments", studentId],
      });
    },
  });
}

export function useUpdateAssignment(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...cmd
    }: {
      id: string;
      status?: AssignmentStatus;
      due_date?: string;
    }) =>
      apiClient<AssignmentResponse>(
        `/v1/learning/students/${studentId}/assignments/${id}`,
        { method: "PATCH", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "assignments", studentId],
      });
    },
  });
}

export function useDeleteAssignment(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/learning/students/${studentId}/assignments/${id}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "assignments", studentId],
      });
    },
  });
}
