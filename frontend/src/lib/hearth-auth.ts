/**
 * Thin BFF auth client — all OAuth token operations are delegated to the
 * backend Hearth BFF endpoints. The browser never sees OAuth tokens; session
 * state is an HttpOnly `sid` cookie. [ARCH ADR-020]
 *
 * @see ARCHITECTURE §6.2 (Hearth BFF pattern)
 */

/** Redirect the browser to the BFF login endpoint to start the Hearth PKCE flow. */
export function redirectToLogin(): void {
  window.location.href = "/v1/auth/login";
}

/** Command shape for POST /v1/auth/register. [§10.1 ADR-019] */
export interface RegisterCommand {
  email: string;
  display_name: string;
  family_display_name: string;
  primary_methodology_slug: string;
}

/** Error thrown by {@link register} when the server returns a non-2xx response. */
export interface AuthError {
  /** HTTP status code as a string (e.g. "400", "409"). */
  code: string;
  message: string;
}

/** Type guard for {@link AuthError}. */
export function isAuthError(e: unknown): e is AuthError {
  return (
    typeof e === "object" &&
    e !== null &&
    "code" in e &&
    "message" in e
  );
}

/**
 * POST /v1/auth/register — app-orchestrated Hearth registration.
 *
 * Creates the Hearth identity + family + parent records in a single
 * backend-coordinated transaction. On success Hearth emails an activation
 * link; the user sets their password through Hearth's hosted UI, then logs
 * in via the PKCE flow.
 *
 * @throws {AuthError} on 4xx/5xx response.
 */
export async function register(cmd: RegisterCommand): Promise<void> {
  const res = await fetch("/v1/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    credentials: "include",
    body: JSON.stringify(cmd),
  });

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { message?: string };
    throw {
      code: String(res.status),
      message: body.message ?? "Registration failed. Please try again.",
    } satisfies AuthError;
  }
}

/**
 * POST /v1/auth/logout — revokes the refresh token (RFC 7009), deletes the
 * BFF session, and clears the sid cookie. Idempotent: safe to call even when
 * no session is active.
 */
export async function logout(): Promise<void> {
  // Best-effort; ignore errors — the cookie will expire naturally.
  await fetch("/v1/auth/logout", {
    method: "POST",
    credentials: "include",
  }).catch(() => undefined);
}

/**
 * POST /v1/auth/refresh — silently rotates the access token inside the BFF
 * session. The browser never sees the new token; only the sid cookie is
 * updated.
 *
 * @throws {Error} on 401/non-ok response (caller should trigger logout).
 */
export async function refresh(): Promise<void> {
  const res = await fetch("/v1/auth/refresh", {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error("Session refresh failed");
  }
}
