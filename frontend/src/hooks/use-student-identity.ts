import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

type StudentSessionIdentity =
  components["schemas"]["iam.StudentSessionIdentityResponse"];

export function useStudentIdentity() {
  return useQuery({
    queryKey: ["student", "session"],
    queryFn: () =>
      apiClient<StudentSessionIdentity>("/v1/student/session"),
    retry: false,
  });
}

export interface StudentLoginResponse {
  session_token: string;
  student_id: string;
  student_name: string;
  expires_at: string;
}

export function useCreateStudentSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      studentId,
      pin,
    }: {
      studentId: string;
      pin: string;
    }) =>
      apiClient<StudentLoginResponse>(
        `/v1/families/students/${studentId}/sessions`,
        { method: "POST", body: { pin } },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["student", "session"] });
    },
  });
}
