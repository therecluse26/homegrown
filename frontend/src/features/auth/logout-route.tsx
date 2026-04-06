import { useEffect, useState } from "react";
import { initLogout, performLogout } from "@/lib/kratos";
import { Spinner } from "@/components/ui";

/**
 * Route component that performs logout and redirects to the login page.
 * Handles the Kratos logout flow (init -> perform -> redirect).
 */
export function LogoutRoute() {
  const [done, setDone] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function doLogout() {
      try {
        const { logout_token } = await initLogout();
        await performLogout(logout_token);
      } catch {
        // If logout fails (no session, network error), still redirect to login
      }
      if (!cancelled) {
        setDone(true);
      }
    }

    void doLogout();

    return () => {
      cancelled = true;
    };
  }, []);

  if (done) {
    // Use window.location to ensure a full page reload clears cached auth state
    window.location.href = "/auth/login";
    return null;
  }

  return (
    <div className="flex justify-center py-12">
      <Spinner size="lg" />
    </div>
  );
}
