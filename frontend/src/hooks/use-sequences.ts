import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type SequenceProgressStatus =
  | "not_started"
  | "in_progress"
  | "completed";

export interface SequenceItemResponse {
  id: string;
  sort_order: number;
  content_type: string;
  content_id: string;
  is_required: boolean;
  unlock_after_previous: boolean;
}

export interface SequenceDefResponse {
  id: string;
  publisher_id: string;
  title: string;
  description?: string;
  subject_tags: string[];
  methodology_id?: string;
  is_linear: boolean;
  created_at: string;
}

export interface SequenceDefDetailResponse extends SequenceDefResponse {
  items: SequenceItemResponse[];
}

export interface SequenceProgressResponse {
  id: string;
  student_id: string;
  sequence_def_id: string;
  current_item_index: number;
  status: SequenceProgressStatus;
  item_completions: Record<string, unknown>;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

// ─── Sequence Definition Queries ────────────────────────────────────────────

export function useSequenceDef(id: string) {
  return useQuery({
    queryKey: ["learning", "sequences", id],
    queryFn: () =>
      apiClient<SequenceDefDetailResponse>(
        `/v1/learning/sequences/${id}`,
      ),
    enabled: !!id,
  });
}

// ─── Sequence Progress Queries ──────────────────────────────────────────────

export function useSequenceProgress(
  studentId: string,
  progressId: string,
) {
  return useQuery({
    queryKey: [
      "learning",
      "sequence-progress",
      studentId,
      progressId,
    ],
    queryFn: () =>
      apiClient<SequenceProgressResponse>(
        `/v1/learning/students/${studentId}/sequence-progress/${progressId}`,
      ),
    enabled: !!studentId && !!progressId,
  });
}

// ─── Sequence Progress Mutations ────────────────────────────────────────────

export function useStartSequence(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: { sequence_def_id: string }) =>
      apiClient<SequenceProgressResponse>(
        `/v1/learning/students/${studentId}/sequence-progress`,
        { method: "POST", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "sequence-progress", studentId],
      });
    },
  });
}

export function useUpdateSequenceProgress(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      progressId,
      ...cmd
    }: {
      progressId: string;
      complete_item_id?: string;
      skip_item_id?: string;
      unlock_item_id?: string;
    }) =>
      apiClient<SequenceProgressResponse>(
        `/v1/learning/students/${studentId}/sequence-progress/${progressId}`,
        { method: "PATCH", body: JSON.stringify(cmd) },
      ),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({
        queryKey: [
          "learning",
          "sequence-progress",
          studentId,
          vars.progressId,
        ],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}
