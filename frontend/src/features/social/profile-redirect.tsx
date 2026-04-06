import { Navigate } from "react-router";
import { useAuth } from "@/hooks/use-auth";
import { Spinner } from "@/components/ui";

/**
 * Redirects /profile to /family/:familyId using the current user's family.
 * Falls back to home if no family ID is available.
 */
export function ProfileRedirect() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  const familyId = user?.family_id;
  if (familyId) {
    return <Navigate to={`/family/${familyId}`} replace />;
  }

  return <Navigate to="/" replace />;
}
