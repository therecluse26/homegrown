import { Navigate, Outlet, useLocation } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import { useAuthContext } from "@/features/auth/auth-provider";
import { Spinner } from "@/components/ui";
import type { WizardProgress } from "@/types";

export function OnboardingGuard() {
  const { isAuthenticated } = useAuthContext();
  const location = useLocation();

  const { data: progress, isLoading } = useQuery({
    queryKey: ["onboarding", "progress"],
    queryFn: () => apiClient<WizardProgress>("/v1/onboarding/progress"),
    enabled: isAuthenticated,
    retry: false,
  });

  if (isLoading) {
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
