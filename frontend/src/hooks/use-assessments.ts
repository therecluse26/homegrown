import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────
// Assessment types are modeled after the quiz engine's scoring capabilities
// plus standalone grade/test tracking. These frontend-only types will be
// replaced when the backend gets swag annotations.

export type ScoreType = "points" | "percentage" | "letter";

export interface GradingScale {
  id: string;
  name: string;
  levels: GradingLevel[];
  is_default: boolean;
}

export interface GradingLevel {
  label: string;
  min_percent: number;
  max_percent: number;
  gpa_points: number;
}

export interface AssessmentResponse {
  id: string;
  student_id: string;
  title: string;
  subject_tags: string[];
  assessment_date: string;
  score_type: ScoreType;
  score_value: number;
  max_value?: number;
  weight?: number;
  grading_scale_id?: string;
  notes?: string;
  created_at: string;
}

interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ─── Grading Scales ─────────────────────────────────────────────────────────

export function useGradingScales() {
  return useQuery({
    queryKey: ["learning", "grading-scales"],
    queryFn: () =>
      apiClient<GradingScale[]>("/v1/learning/grading-scales"),
    staleTime: 1000 * 60 * 10, // 10 min — rarely changes
  });
}

// ─── Assessment Queries ─────────────────────────────────────────────────────

export function useAssessments(
  studentId: string,
  params?: {
    subject?: string;
    date_from?: string;
    date_to?: string;
  },
) {
  return useInfiniteQuery({
    queryKey: ["learning", "assessments", studentId, params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.subject) sp.set("subject", params.subject);
      if (params?.date_from) sp.set("date_from", params.date_from);
      if (params?.date_to) sp.set("date_to", params.date_to);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<AssessmentResponse>>(
        `/v1/learning/students/${studentId}/assessments${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!studentId,
  });
}

// ─── Assessment Mutations ───────────────────────────────────────────────────

export function useCreateAssessment(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      title: string;
      subject_tags?: string[];
      assessment_date: string;
      score_type: ScoreType;
      score_value: number;
      max_value?: number;
      weight?: number;
      grading_scale_id?: string;
      notes?: string;
    }) =>
      apiClient<AssessmentResponse>(
        `/v1/learning/students/${studentId}/assessments`,
        { method: "POST", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "assessments", studentId],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}

export function useUpdateAssessment(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...cmd
    }: {
      id: string;
      title?: string;
      subject_tags?: string[];
      assessment_date?: string;
      score_type?: ScoreType;
      score_value?: number;
      max_value?: number;
      weight?: number;
      notes?: string;
    }) =>
      apiClient<AssessmentResponse>(
        `/v1/learning/students/${studentId}/assessments/${id}`,
        { method: "PATCH", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "assessments", studentId],
      });
    },
  });
}
