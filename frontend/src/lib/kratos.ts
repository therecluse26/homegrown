/**
 * Kratos Browser API helpers for SPA auth flows.
 *
 * Uses Kratos "browser" flow endpoints with Accept: application/json so Kratos
 * returns JSON rather than redirecting. CSRF tokens are embedded in flow nodes
 * and must be echoed back on submission.
 *
 * Proxied through Vite dev server at /self-service → http://localhost:4933.
 *
 * @see ARCHITECTURE §6.1 (Kratos configuration)
 * @see ARCHITECTURE §11.2 (frontend auth strategy)
 */

// ─── Minimal Kratos types ─────────────────────────────────────────────────────

export type KratosFlowType =
  | "login"
  | "registration"
  | "recovery"
  | "verification";

export interface KratosMessage {
  id: number;
  text: string;
  type: "info" | "error" | "success";
  context?: Record<string, unknown>;
}

export interface KratosNodeAttributes {
  name: string;
  type: string;
  value?: string | boolean | number;
  required?: boolean;
  disabled?: boolean;
  node_type: "input" | "a" | "img" | "script" | "text";
  onclick?: string;
  onclickTrigger?: string;
  href?: string;
  label?: { id: number; text: string; type: string };
}

export interface KratosNode {
  type: "input" | "img" | "text" | "a" | "script";
  group: string;
  attributes: KratosNodeAttributes;
  messages: KratosMessage[];
  meta: { label?: { id: number; text: string; type: string } };
}

export interface KratosUi {
  action: string;
  method: string;
  nodes: KratosNode[];
  messages?: KratosMessage[];
}

export interface KratosFlow {
  id: string;
  type: string;
  expires_at: string;
  issued_at: string;
  request_url: string;
  ui: KratosUi;
  state?: string;
}

export interface KratosSession {
  id: string;
  active: boolean;
  expires_at: string;
  authenticated_at: string;
  identity: {
    id: string;
    traits: {
      email: string;
      name?: string;
    };
  };
}

export interface KratosError {
  error?: { code: number; status: string; message: string; reason?: string };
  ui?: KratosUi;
  id?: string;
  redirect_browser_to?: string;
}

/** Discriminated union returned by submitFlow. */
export type FlowResult =
  | { kind: "flow"; flow: KratosFlow }         // validation errors; re-render form
  | { kind: "success"; session?: KratosSession } // auth succeeded; invalidate auth query
  | { kind: "redirect"; url: string };           // follow a URL (recovery steps, etc.)

// ─── Field error extraction ───────────────────────────────────────────────────

/**
 * Extract field-level validation errors from a Kratos flow's UI nodes.
 * Returns a map of field name → first error message text.
 */
export function extractFieldErrors(flow: KratosFlow): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const node of flow.ui.nodes) {
    const first = node.messages.find((m) => m.type === "error");
    if (first) {
      errors[node.attributes.name] = first.text;
    }
  }
  return errors;
}

/**
 * Extract global (non-field) messages from a flow's ui.messages array.
 */
export function extractGlobalMessages(flow: KratosFlow): KratosMessage[] {
  return flow.ui.messages ?? [];
}

/**
 * Find the CSRF token value from a flow's nodes (hidden input with name=csrf_token).
 */
export function extractCsrfToken(flow: KratosFlow): string {
  const node = flow.ui.nodes.find(
    (n) =>
      n.attributes.name === "csrf_token" && n.attributes.node_type === "input",
  );
  return typeof node?.attributes.value === "string" ? node.attributes.value : "";
}

/**
 * Extract OAuth provider names configured in a flow (group === "oidc").
 */
export function extractOAuthProviders(flow: KratosFlow): string[] {
  return flow.ui.nodes
    .filter(
      (n) =>
        n.group === "oidc" &&
        n.attributes.node_type === "input" &&
        n.attributes.name === "provider",
    )
    .map((n) => String(n.attributes.value ?? ""))
    .filter(Boolean);
}

// ─── Flow init helpers ────────────────────────────────────────────────────────

async function kratosGet<T>(path: string): Promise<T> {
  const res = await fetch(path, {
    method: "GET",
    headers: { Accept: "application/json" },
    credentials: "include",
  });

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as KratosError;
    throw body;
  }

  return res.json() as Promise<T>;
}

/** Start a new login flow. Returns the flow with UI nodes. */
export async function initLoginFlow(): Promise<KratosFlow> {
  return kratosGet<KratosFlow>("/self-service/login/browser");
}

/** Start a new registration flow. Returns the flow with UI nodes. */
export async function initRegistrationFlow(): Promise<KratosFlow> {
  return kratosGet<KratosFlow>("/self-service/registration/browser");
}

/** Start a new account recovery flow. Returns the flow with UI nodes. */
export async function initRecoveryFlow(): Promise<KratosFlow> {
  return kratosGet<KratosFlow>("/self-service/recovery/browser");
}

/** Start a new email verification flow. Returns the flow with UI nodes. */
export async function initVerificationFlow(): Promise<KratosFlow> {
  return kratosGet<KratosFlow>("/self-service/verification/browser");
}

/**
 * Fetch an existing flow by ID and type (used when Kratos redirects back with ?flow=xxx).
 */
export async function getFlow(
  type: KratosFlowType,
  flowId: string,
): Promise<KratosFlow> {
  return kratosGet<KratosFlow>(
    `/self-service/${type}/flows?id=${encodeURIComponent(flowId)}`,
  );
}

// ─── Flow submission ──────────────────────────────────────────────────────────

/**
 * Rewrite a Kratos action URL to a relative path so it routes through the Vite
 * dev proxy instead of hitting the Kratos instance directly (which would fail
 * due to CORS). In production all requests go through the same origin anyway.
 */
function relativeAction(action: string): string {
  try {
    const url = new URL(action);
    return url.pathname + url.search + url.hash;
  } catch {
    return action;
  }
}

/**
 * Submit a self-service flow and return a discriminated FlowResult.
 *
 * On validation errors, returns `{ kind: "flow", flow }` with updated nodes.
 * On success, returns `{ kind: "success", session? }`.
 * On redirect (recovery steps), returns `{ kind: "redirect", url }`.
 *
 * @param action  The flow's `ui.action` URL (includes flow ID param)
 * @param method  The flow's `ui.method` ("POST" for most flows)
 * @param body    Key-value form data including csrf_token
 */
export async function submitFlow(
  action: string,
  method: string,
  body: Record<string, string | boolean | number>,
): Promise<FlowResult> {
  const res = await fetch(relativeAction(action), {
    method: method.toUpperCase(),
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    credentials: "include",
    body: JSON.stringify(body),
  });

  // 422 — browser location change required (Kratos wants us to redirect)
  if (res.status === 422) {
    const body422 = (await res.json()) as { redirect_browser_to?: string };
    return { kind: "redirect", url: body422.redirect_browser_to ?? "/" };
  }

  // 200/400 — updated flow (could be errors or success state)
  if (res.status === 200 || res.status === 400) {
    const data = (await res.json()) as Record<string, unknown>;

    // Kratos login success returns { session, session_token?, identity }
    if ("session" in data) {
      return { kind: "success", session: data.session as KratosSession };
    }

    // Registration/verification success returns { identity } (no session key in some versions)
    if ("identity" in data && !("ui" in data)) {
      return { kind: "success" };
    }

    // Has ui nodes — still in flow (validation errors or multi-step)
    if ("ui" in data) {
      return { kind: "flow", flow: data as unknown as KratosFlow };
    }

    // Fallback: treat as success
    return { kind: "success" };
  }

  // Error: parse and throw
  const errorBody = (await res.json().catch(() => ({}))) as KratosError;
  throw errorBody;
}

// ─── Logout ───────────────────────────────────────────────────────────────────

/**
 * Initiate a browser logout flow and get the logout token.
 */
export async function initLogout(): Promise<{
  logout_url: string;
  logout_token: string;
}> {
  return kratosGet<{ logout_url: string; logout_token: string }>(
    "/self-service/logout/browser",
  );
}

/** Complete logout using the token from initLogout(). */
export async function performLogout(token: string): Promise<void> {
  await fetch(
    `/self-service/logout?token=${encodeURIComponent(token)}`,
    {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" },
    },
  );
}

// ─── Session ──────────────────────────────────────────────────────────────────

/** Fetch the current Kratos session (null if not authenticated). */
export async function getSession(): Promise<KratosSession | null> {
  try {
    return await kratosGet<KratosSession>("/sessions/whoami");
  } catch {
    return null;
  }
}
