import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

// ─── Type aliases (from generated schema) ──────────────────────────────────

type FamilyProfile = components["schemas"]["iam.FamilyProfileResponse"];
type StudentResponse = components["schemas"]["iam.StudentResponse"];
type ActiveToolResponse = components["schemas"]["method.ActiveToolResponse"];
type CoParentInviteResponse =
  components["schemas"]["iam.CoParentInviteResponse"];
type MethodologySelectionResponse =
  components["schemas"]["method.MethodologySelectionResponse"];

type UpdateFamilyCommand = components["schemas"]["iam.UpdateFamilyCommand"];
type CreateStudentCommand = components["schemas"]["iam.CreateStudentCommand"];
type UpdateStudentCommand = components["schemas"]["iam.UpdateStudentCommand"];
type UpdateMethodologyCommand =
  components["schemas"]["method.UpdateMethodologyCommand"];
type InviteCoParentCommand =
  components["schemas"]["iam.InviteCoParentCommand"];
type TransferPrimaryCommand =
  components["schemas"]["iam.TransferPrimaryCommand"];

// ─── Queries ────────────────────────────────────────────────────────────────

export function useFamilyProfile() {
  return useQuery({
    queryKey: ["family", "profile"],
    queryFn: () => apiClient<FamilyProfile>("/v1/families/profile"),
    staleTime: 1000 * 60 * 2,
  });
}

export function useStudents() {
  return useQuery({
    queryKey: ["family", "students"],
    queryFn: () => apiClient<StudentResponse[]>("/v1/families/students"),
    staleTime: 1000 * 60,
  });
}

export function useFamilyTools() {
  return useQuery({
    queryKey: ["family", "tools"],
    queryFn: () => apiClient<ActiveToolResponse[]>("/v1/families/tools"),
    staleTime: 1000 * 60 * 5,
  });
}

export function useStudentTools(studentId: string | undefined) {
  return useQuery({
    queryKey: ["family", "students", studentId, "tools"],
    queryFn: () =>
      apiClient<ActiveToolResponse[]>(
        `/v1/families/students/${studentId ?? ""}/tools`,
      ),
    enabled: !!studentId,
    staleTime: 1000 * 60 * 5,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useUpdateFamily() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateFamilyCommand) =>
      apiClient<FamilyProfile>("/v1/families/profile", {
        method: "PATCH",
        body,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(["family", "profile"], data);
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useCreateStudent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateStudentCommand) =>
      apiClient<StudentResponse>("/v1/families/students", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "students"] });
      void queryClient.invalidateQueries({ queryKey: ["family", "profile"] });
    },
  });
}

export function useUpdateStudent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...body
    }: UpdateStudentCommand & { id: string }) =>
      apiClient<StudentResponse>(`/v1/families/students/${id}`, {
        method: "PATCH",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "students"] });
    },
  });
}

export function useDeleteStudent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/families/students/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "students"] });
      void queryClient.invalidateQueries({ queryKey: ["family", "profile"] });
    },
  });
}

export function useUpdateMethodology() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateMethodologyCommand) =>
      apiClient<MethodologySelectionResponse>("/v1/families/methodology", {
        method: "PATCH",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "profile"] });
      void queryClient.invalidateQueries({ queryKey: ["family", "tools"] });
      void queryClient.invalidateQueries({ queryKey: ["methodologies"] });
    },
  });
}

export function useInviteCoParent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: InviteCoParentCommand) =>
      apiClient<CoParentInviteResponse>("/v1/families/invites", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "profile"] });
    },
  });
}

export function useRemoveCoParent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (parentId: string) =>
      apiClient<void>(`/v1/families/parents/${parentId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "profile"] });
    },
  });
}

export function useTransferPrimary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: TransferPrimaryCommand) =>
      apiClient<void>("/v1/families/primary-parent", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["family", "profile"] });
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}
