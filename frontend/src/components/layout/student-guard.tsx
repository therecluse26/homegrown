import { Navigate, Outlet } from "react-router";
import { useStudentIdentity } from "@/hooks/use-student-identity";
import { Spinner } from "@/components/ui";

export function StudentGuard() {
  const { data: session, isLoading, isError } = useStudentIdentity();

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
