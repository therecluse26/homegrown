import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { AttachmentInput } from "./use-activities";

// ─── Types ──────────────────────────────────────────────────────────────────

export type JournalEntryType = "freeform" | "narration" | "reflection";

export interface JournalEntryResponse {
  id: string;
  student_id: string;
  entry_type: JournalEntryType;
  title?: string;
  content: string;
  subject_tags: string[];
  attachments: AttachmentInput[];
  entry_date: string;
  created_at: string;
}

interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useJournalEntries(
  studentId: string,
  params?: {
    entry_type?: JournalEntryType;
    date_from?: string;
    date_to?: string;
    search?: string;
  },
) {
  return useInfiniteQuery({
    queryKey: ["learning", "journals", studentId, params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.entry_type) sp.set("entry_type", params.entry_type);
      if (params?.date_from) sp.set("date_from", params.date_from);
      if (params?.date_to) sp.set("date_to", params.date_to);
      if (params?.search) sp.set("search", params.search);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<JournalEntryResponse>>(
        `/v1/learning/students/${studentId}/journal${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!studentId,
  });
}

export function useJournalEntry(studentId: string, id: string) {
  return useQuery({
    queryKey: ["learning", "journals", studentId, id],
    queryFn: () =>
      apiClient<JournalEntryResponse>(
        `/v1/learning/students/${studentId}/journal/${id}`,
      ),
    enabled: !!studentId && !!id,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useCreateJournalEntry(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      entry_type: JournalEntryType;
      title?: string;
      content: string;
      subject_tags?: string[];
      content_id?: string;
      attachments?: AttachmentInput[];
      entry_date?: string;
    }) =>
      apiClient<JournalEntryResponse>(
        `/v1/learning/students/${studentId}/journal`,
        { method: "POST", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "journals", studentId],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}

export function useUpdateJournalEntry(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...cmd
    }: {
      id: string;
      entry_type?: JournalEntryType;
      title?: string;
      content?: string;
      subject_tags?: string[];
      attachments?: AttachmentInput[];
      entry_date?: string;
    }) =>
      apiClient<JournalEntryResponse>(
        `/v1/learning/students/${studentId}/journal/${id}`,
        { method: "PATCH", body: JSON.stringify(cmd) },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "journals", studentId],
      });
    },
  });
}

export function useDeleteJournalEntry(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/learning/students/${studentId}/journal/${id}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "journals", studentId],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}
