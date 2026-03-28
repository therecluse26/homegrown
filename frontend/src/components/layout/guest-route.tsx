import { Navigate, Outlet } from "react-router";
import { useAuthContext } from "@/features/auth/auth-provider";
import { Spinner } from "@/components/ui";

/**
 * Blocks authenticated users from accessing guest-only routes (login, register).
 * Mirrors ProtectedRoute: shows a spinner while auth loads, then redirects
 * authenticated users to "/" and renders Outlet for unauthenticated users.
 */
export function GuestRoute() {
  const { isLoading, isAuthenticated } = useAuthContext();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-surface flex items-center justify-center">
        <Spinner size="lg" className="text-primary" />
      </div>
    );
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
