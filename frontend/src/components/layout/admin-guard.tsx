import { Navigate, Outlet } from "react-router";
import { useAuthContext } from "@/features/auth/auth-provider";
import { Spinner } from "@/components/ui";

export function AdminGuard() {
  const { isLoading, isAuthenticated, isPlatformAdmin } = useAuthContext();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-surface flex items-center justify-center">
        <Spinner size="lg" className="text-primary" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/auth/login" replace />;
  }

  if (!isPlatformAdmin) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
