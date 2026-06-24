import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { logout } from "@/lib/hearth-auth";
import { Spinner } from "@/components/ui";

/**
 * Route component that performs BFF logout and redirects to the login page.
 *
 * Calls POST /v1/auth/logout which revokes the refresh token (RFC 7009),
 * deletes the server-side session, and clears the sid cookie. [ARCH ADR-020]
 */
export function LogoutRoute() {
  const [done, setDone] = useState(false);
  const queryClient = useQueryClient();

  useEffect(() => {
    let cancelled = false;

    async function doLogout() {
      await logout();
      queryClient.clear();
      if (!cancelled) setDone(true);
    }

    void doLogout();
    return () => {
      cancelled = true;
    };
  }, [queryClient]);

  if (done) {
    window.location.href = "/auth/login";
    return null;
  }

  return (
    <div className="flex justify-center py-12">
      <Spinner size="lg" />
    </div>
  );
}
