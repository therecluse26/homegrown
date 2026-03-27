import { Navigate, Outlet } from "react-router";
import { useAuthContext } from "@/features/auth/auth-provider";
import { Spinner } from "@/components/ui";

export function AdminGuard() {
  const { isLoading, isAuthenticated } = useAuthContext();

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

  // TODO: Check is_platform_admin once backend adds this field to CurrentUserResponse.
  // For now, all authenticated users can access admin routes during development.
  // In production, the backend enforces admin-only access via middleware.
  const isAdmin = true;

  if (!isAdmin) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
