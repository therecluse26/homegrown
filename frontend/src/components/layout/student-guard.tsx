import { Navigate, Outlet } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import { Spinner } from "@/components/ui";
import type { components } from "@/api/generated/schema";

type StudentSession = components["schemas"]["iam.StudentSessionIdentityResponse"];

export function StudentGuard() {
  const { data: session, isLoading, isError } = useQuery({
    queryKey: ["student", "session"],
    queryFn: () => apiClient<StudentSession>("/v1/student/session"),
    retry: false,
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-surface flex items-center justify-center">
        <Spinner size="lg" className="text-primary" />
      </div>
    );
  }

  if (isError || !session?.student_id) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
