const API_BASE = import.meta.env["VITE_API_BASE_URL"] ?? "";

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

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
    credentials: "include", // Send cookies for Kratos session
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { code: "unknown", message: "An error occurred" },
    }));
    // eslint-disable-next-line @typescript-eslint/only-throw-error
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}
