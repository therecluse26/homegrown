const API_BASE = import.meta.env["VITE_API_BASE_URL"] ?? "";

function getCookie(name: string): string | undefined {
  const match = document.cookie
    .split("; ")
    .find((c) => c.startsWith(`${name}=`));
  if (!match) return undefined;
  const value = match.split("=")[1];
  return value !== undefined ? decodeURIComponent(value) : undefined;
}

const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

type RequestOptions = {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
};

export async function apiClient<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", body, headers = {} } = options;

  const csrfHeaders: Record<string, string> = {};
  if (MUTATING_METHODS.has(method.toUpperCase())) {
    const token = getCookie("_csrf");
    if (token) {
      csrfHeaders["X-CSRF-Token"] = token;
    }
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...csrfHeaders,
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
    credentials: "include", // Send cookies for Kratos session
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { code: "unknown", message: "An error occurred" },
    }));
     
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}
