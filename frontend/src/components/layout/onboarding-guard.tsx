import { Navigate, Outlet, useLocation } from "react-router";
import { useOnboardingProgress } from "@/hooks/use-onboarding";
import { useAuthContext } from "@/features/auth/auth-provider";
import { Spinner } from "@/components/ui";

export function OnboardingGuard() {
  const { isAuthenticated } = useAuthContext();
  const location = useLocation();

  const { data: progress, isLoading } = useOnboardingProgress();

  if (!isAuthenticated || isLoading) {
    return (
      <div className="min-h-screen bg-surface flex items-center justify-center">
        <Spinner size="lg" className="text-primary" />
      </div>
    );
  }

  const isComplete =
    progress?.status === "completed" || progress?.status === "skipped";

  if (!isComplete && location.pathname !== "/onboarding") {
    return <Navigate to="/onboarding" replace />;
  }

  return <Outlet />;
}
