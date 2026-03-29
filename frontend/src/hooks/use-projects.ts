import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type ProjectStatus = "planning" | "in_progress" | "completed";

export interface Milestone {
  id: string;
  title: string;
  due_date: string | null;
  completed: boolean;
  completed_at: string | null;
}

export interface Project {
  id: string;
  title: string;
  description: string;
  student_id: string;
  student_name: string;
  status: ProjectStatus;
  due_date: string | null;
  milestones: Milestone[];
  created_at: string;
  updated_at: string;
}

export interface CreateProjectRequest {
  title: string;
  description: string;
  student_id: string;
  due_date?: string;
}

export interface UpdateProjectRequest {
  title?: string;
  description?: string;
  status?: ProjectStatus;
  due_date?: string | null;
}

export interface CreateMilestoneRequest {
  title: string;
  due_date?: string;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useProjects(studentId?: string) {
  return useQuery({
    queryKey: ["learning", "projects", studentId],
    queryFn: () => {
      const params = new URLSearchParams();
      if (studentId) params.set("student_id", studentId);
      const qs = params.toString();
      return apiClient<Project[]>(
        `/v1/learning/projects${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60, // 1 min
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: ["learning", "projects", "detail", id],
    queryFn: () => apiClient<Project>(`/v1/learning/projects/${id}`),
    enabled: !!id,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateProjectRequest) =>
      apiClient<Project>("/v1/learning/projects", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useUpdateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: UpdateProjectRequest & { id: string }) =>
      apiClient<Project>(`/v1/learning/projects/${id}`, {
        method: "PATCH",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useDeleteProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/learning/projects/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useAddMilestone() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectId,
      ...body
    }: CreateMilestoneRequest & { projectId: string }) =>
      apiClient<Milestone>(
        `/v1/learning/projects/${projectId}/milestones`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useToggleMilestone() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectId,
      milestoneId,
      completed,
    }: {
      projectId: string;
      milestoneId: string;
      completed: boolean;
    }) =>
      apiClient<void>(
        `/v1/learning/projects/${projectId}/milestones/${milestoneId}`,
        { method: "PATCH", body: { completed } },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}
