import { useQuery, useInfiniteQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export interface SubjectHoursResponse {
  subject_slug: string;
  subject_name: string;
  hours: number;
}

export interface ProgressSummaryResponse {
  student_id: string;
  date_from: string;
  date_to: string;
  total_activities: number;
  total_hours: number;
  hours_by_subject: SubjectHoursResponse[];
  books_completed: number;
  journal_entries: number;
}

export interface SubjectProgressResponse {
  subject_slug: string;
  subject_name: string;
  activity_count: number;
  total_hours: number;
  journal_count: number;
  books_completed: number;
}

export type TimelineEntryType =
  | "activity"
  | "journal"
  | "reading_completed";

export interface TimelineEntryResponse {
  id: string;
  entry_type: TimelineEntryType;
  title: string;
  description?: string;
  subject_tags: string[];
  date: string;
  created_at: string;
}

export interface StreakResponse {
  student_id: string;
  current_streak: number;
  longest_streak: number;
  last_activity_date: string;
  milestones_reached: number[]; // e.g. [7, 14, 30]
}

interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ─── Streak ──────────────────────────────────────────────────────────────

export function useStreak(studentId: string) {
  return useQuery({
    queryKey: ["learning", "streak", studentId],
    queryFn: () =>
      apiClient<StreakResponse>(
        `/v1/learning/students/${studentId}/streak`,
      ),
    enabled: !!studentId,
  });
}

// ─── Progress Summary ───────────────────────────────────────────────────────

export function useStudentProgress(
  studentId: string,
  params?: { date_from?: string; date_to?: string },
) {
  return useQuery({
    queryKey: ["learning", "progress", studentId, params],
    queryFn: () => {
      const sp = new URLSearchParams();
      if (params?.date_from) sp.set("date_from", params.date_from);
      if (params?.date_to) sp.set("date_to", params.date_to);
      const qs = sp.toString();
      return apiClient<ProgressSummaryResponse>(
        `/v1/learning/students/${studentId}/progress${qs ? `?${qs}` : ""}`,
      );
    },
    enabled: !!studentId,
  });
}

// ─── Subject Breakdown ──────────────────────────────────────────────────────

export function useSubjectProgress(
  studentId: string,
  params?: { date_from?: string; date_to?: string },
) {
  return useQuery({
    queryKey: ["learning", "progress", studentId, "subjects", params],
    queryFn: () => {
      const sp = new URLSearchParams();
      if (params?.date_from) sp.set("date_from", params.date_from);
      if (params?.date_to) sp.set("date_to", params.date_to);
      const qs = sp.toString();
      return apiClient<SubjectProgressResponse[]>(
        `/v1/learning/students/${studentId}/progress/subjects${qs ? `?${qs}` : ""}`,
      );
    },
    enabled: !!studentId,
  });
}

// ─── Timeline ───────────────────────────────────────────────────────────────

export function useProgressTimeline(
  studentId: string,
  params?: { date_from?: string; date_to?: string },
) {
  return useInfiniteQuery({
    queryKey: ["learning", "progress", studentId, "timeline", params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.date_from) sp.set("date_from", params.date_from);
      if (params?.date_to) sp.set("date_to", params.date_to);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<TimelineEntryResponse>>(
        `/v1/learning/students/${studentId}/progress/timeline${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!studentId,
  });
}
