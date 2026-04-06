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

interface PaginatedResponse<T> {
  data: T[];
  has_more: boolean;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useProjects(studentId: string) {
  return useQuery({
    queryKey: ["learning", "projects", studentId],
    queryFn: async () => {
      const resp = await apiClient<PaginatedResponse<Project>>(
        `/v1/learning/students/${studentId}/projects`,
      );
      return resp.data;
    },
    enabled: !!studentId,
    staleTime: 1000 * 60, // 1 min
  });
}

export function useProject(studentId: string, id: string) {
  return useQuery({
    queryKey: ["learning", "projects", "detail", studentId, id],
    queryFn: () =>
      apiClient<Project>(
        `/v1/learning/students/${studentId}/projects/${id}`,
      ),
    enabled: !!studentId && !!id,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateProjectRequest) =>
      apiClient<Project>(
        `/v1/learning/students/${body.student_id}/projects`,
        { method: "POST", body },
      ),
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
      studentId,
      ...body
    }: UpdateProjectRequest & { id: string; studentId: string }) =>
      apiClient<Project>(
        `/v1/learning/students/${studentId}/projects/${id}`,
        { method: "PATCH", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useDeleteProject(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/learning/students/${studentId}/projects/${id}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useAddMilestone(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectId,
      ...body
    }: CreateMilestoneRequest & { projectId: string }) =>
      apiClient<Milestone>(
        `/v1/learning/students/${studentId}/projects/${projectId}/milestones`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}

export function useToggleMilestone(studentId: string) {
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
        `/v1/learning/students/${studentId}/projects/${projectId}/milestones/${milestoneId}`,
        { method: "PATCH", body: { completed } },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["learning", "projects"],
      });
    },
  });
}
