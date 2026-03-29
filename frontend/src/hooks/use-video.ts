import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export interface CaptionTrack {
  src: string;
  srclang: string;
  label: string;
  kind: "subtitles" | "captions" | "descriptions";
}

export interface VideoDefResponse {
  id: string;
  publisher_id: string;
  title: string;
  description?: string;
  subject_tags: string[];
  methodology_id?: string;
  duration_seconds?: number;
  thumbnail_url?: string;
  video_url: string;
  video_source: string;
  caption_tracks?: CaptionTrack[];
  created_at: string;
}

export interface VideoProgressResponse {
  id: string;
  student_id: string;
  video_def_id: string;
  watched_seconds: number;
  completed: boolean;
  last_position_seconds: number;
  completed_at?: string;
  created_at: string;
}

// ─── Video Definition Queries ───────────────────────────────────────────────

export function useVideoDef(id: string) {
  return useQuery({
    queryKey: ["learning", "videos", id],
    queryFn: () =>
      apiClient<VideoDefResponse>(`/v1/learning/videos/${id}`),
    enabled: !!id,
  });
}

// ─── Video Progress Queries ─────────────────────────────────────────────────

export function useVideoProgress(studentId: string, videoDefId: string) {
  return useQuery({
    queryKey: ["learning", "video-progress", studentId, videoDefId],
    queryFn: () =>
      apiClient<VideoProgressResponse>(
        `/v1/learning/students/${studentId}/video-progress/${videoDefId}`,
      ),
    enabled: !!studentId && !!videoDefId,
  });
}

// ─── Video Progress Mutations ───────────────────────────────────────────────

export function useUpdateVideoProgress(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      video_def_id: string;
      watched_seconds?: number;
      last_position_seconds?: number;
      completed?: boolean;
    }) =>
      apiClient<VideoProgressResponse>(
        `/v1/learning/students/${studentId}/video-progress`,
        { method: "PATCH", body: JSON.stringify(cmd) },
      ),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({
        queryKey: [
          "learning",
          "video-progress",
          studentId,
          vars.video_def_id,
        ],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}
