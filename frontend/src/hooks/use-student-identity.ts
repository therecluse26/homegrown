import { useQuery } from "@tanstack/react-query";
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
