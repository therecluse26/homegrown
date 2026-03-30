import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type ReadingStatus = "to_read" | "in_progress" | "completed";

export interface ReadingItemSummaryResponse {
  id: string;
  title: string;
  author?: string;
  subject_tags: string[];
  cover_image_url?: string;
}

export interface ReadingItemResponse {
  id: string;
  publisher_id: string;
  title: string;
  author?: string;
  isbn?: string;
  subject_tags: string[];
  description?: string;
  cover_image_url?: string;
  page_count?: number;
  created_at: string;
}

export interface ReadingProgressResponse {
  id: string;
  student_id: string;
  reading_item: ReadingItemSummaryResponse;
  reading_list_id?: string;
  status: ReadingStatus;
  started_at?: string;
  completed_at?: string;
  notes?: string;
}

export interface ReadingListSummaryResponse {
  id: string;
  name: string;
  description?: string;
  student_id?: string;
  item_count: number;
  completed_count: number;
}

export interface ReadingListDetailResponse {
  id: string;
  name: string;
  description?: string;
  student_id?: string;
  items: ReadingListItemWithProgress[];
  created_at: string;
}

export interface ReadingListItemWithProgress {
  reading_item: ReadingItemSummaryResponse;
  sort_order: number;
  progress?: ReadingProgressResponse;
}

interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// ─── Reading Item Queries ───────────────────────────────────────────────────

export function useReadingItems(params?: {
  search?: string;
  subject?: string;
  isbn?: string;
}) {
  return useInfiniteQuery({
    queryKey: ["learning", "reading-items", params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.search) sp.set("search", params.search);
      if (params?.subject) sp.set("subject", params.subject);
      if (params?.isbn) sp.set("isbn", params.isbn);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<ReadingItemSummaryResponse>>(
        `/v1/learning/reading-items${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
  });
}

export function useReadingItem(id: string) {
  return useQuery({
    queryKey: ["learning", "reading-items", id],
    queryFn: () =>
      apiClient<ReadingItemResponse & { linked_artifacts: unknown[] }>(
        `/v1/learning/reading-items/${id}`,
      ),
    enabled: !!id,
  });
}

// ─── Reading Item Mutations ─────────────────────────────────────────────────

export function useCreateReadingItem() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      title: string;
      author?: string;
      isbn?: string;
      subject_tags?: string[];
      description?: string;
      cover_image_url?: string;
      page_count?: number;
    }) =>
      apiClient<ReadingItemResponse>("/v1/learning/reading-items", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "reading-items"],
      });
    },
  });
}

// ─── Reading Progress Queries ───────────────────────────────────────────────

export function useReadingProgress(
  studentId: string,
  params?: { status?: ReadingStatus },
) {
  return useInfiniteQuery({
    queryKey: ["learning", "reading-progress", studentId, params],
    queryFn: ({ pageParam }) => {
      const sp = new URLSearchParams();
      if (params?.status) sp.set("status", params.status);
      if (pageParam) sp.set("cursor", pageParam);
      const qs = sp.toString();
      return apiClient<PaginatedResponse<ReadingProgressResponse>>(
        `/v1/learning/students/${studentId}/reading${qs ? `?${qs}` : ""}`,
      );
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!studentId,
  });
}

// ─── Reading Progress Mutations ─────────────────────────────────────────────

export function useStartReading(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      reading_item_id: string;
      reading_list_id?: string;
    }) =>
      apiClient<ReadingProgressResponse>(
        `/v1/learning/students/${studentId}/reading`,
        { method: "POST", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "reading-progress", studentId],
      });
    },
  });
}

export function useUpdateReadingProgress(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...cmd
    }: {
      id: string;
      status?: ReadingStatus;
      notes?: string;
    }) =>
      apiClient<ReadingProgressResponse>(
        `/v1/learning/students/${studentId}/reading/${id}`,
        { method: "PATCH", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "reading-progress", studentId],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}

// ─── Reading List Queries ───────────────────────────────────────────────────

export function useReadingLists() {
  return useQuery({
    queryKey: ["learning", "reading-lists"],
    queryFn: () =>
      apiClient<ReadingListSummaryResponse[]>(
        "/v1/learning/reading-lists",
      ),
  });
}

export function useReadingListDetail(id: string) {
  return useQuery({
    queryKey: ["learning", "reading-lists", id],
    queryFn: () =>
      apiClient<ReadingListDetailResponse>(
        `/v1/learning/reading-lists/${id}`,
      ),
    enabled: !!id,
  });
}

// ─── Reading List Mutations ─────────────────────────────────────────────────

export function useCreateReadingList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      name: string;
      description?: string;
      student_id?: string;
      reading_item_ids?: string[];
    }) =>
      apiClient<{ id: string; name: string; created_at: string }>(
        "/v1/learning/reading-lists",
        { method: "POST", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "reading-lists"],
      });
    },
  });
}

export function useUpdateReadingList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...cmd
    }: {
      id: string;
      name?: string;
      description?: string;
      add_item_ids?: string[];
      remove_item_ids?: string[];
    }) =>
      apiClient<{ id: string; name: string }>(
        `/v1/learning/reading-lists/${id}`,
        { method: "PATCH", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "reading-lists"],
      });
    },
  });
}

export function useDeleteReadingList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/learning/reading-lists/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "reading-lists"],
      });
    },
  });
}
