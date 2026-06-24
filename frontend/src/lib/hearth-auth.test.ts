import { describe, it, expect, vi, afterEach } from "vitest";
import { register, logout, refresh } from "./hearth-auth";
import type { RegisterCommand } from "./hearth-auth";

// ─── helpers ─────────────────────────────────────────────────────────────────

function mockFetch(status: number, body: unknown) {
  return vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
    new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

const cmd: RegisterCommand = {
  email: "jane@example.com",
  display_name: "Jane Doe",
  family_display_name: "The Doe Family",
  primary_methodology_slug: "charlotte-mason",
};

afterEach(() => vi.restoreAllMocks());

// ─── register ─────────────────────────────────────────────────────────────────

describe("register", () => {
  it("resolves on 201", async () => {
    mockFetch(201, { message: "Registration successful. Please log in." });
    await expect(register(cmd)).resolves.toBeUndefined();
  });

  it("throws AuthError on 400 with backend message", async () => {
    mockFetch(400, { message: "Invalid email address" });
    await expect(register(cmd)).rejects.toMatchObject({
      code: "400",
      message: "Invalid email address",
    });
  });

  it("throws AuthError on 409 (email already registered)", async () => {
    mockFetch(409, { message: "Email already registered" });
    await expect(register(cmd)).rejects.toMatchObject({ code: "409" });
  });

  it("uses generic message when response body is unparseable", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("not json", { status: 500 }),
    );
    await expect(register(cmd)).rejects.toMatchObject({
      code: "500",
      message: "Registration failed. Please try again.",
    });
  });

  it("sends credentials:include and Content-Type: application/json", async () => {
    const spy = mockFetch(201, {});
    await register(cmd);
    expect(spy).toHaveBeenCalledWith(
      "/v1/auth/register",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        headers: expect.objectContaining({ "Content-Type": "application/json" }),
      }),
    );
  });
});

// ─── logout ───────────────────────────────────────────────────────────────────

describe("logout", () => {
  it("calls POST /v1/auth/logout with credentials:include", async () => {
    const spy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    await logout();
    expect(spy).toHaveBeenCalledWith(
      "/v1/auth/logout",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });

  it("resolves even if the server returns an error (best-effort)", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValueOnce(
      new Error("network error"),
    );
    await expect(logout()).resolves.toBeUndefined();
  });
});

// ─── refresh ─────────────────────────────────────────────────────────────────

describe("refresh", () => {
  it("resolves on 204", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(null, { status: 204 }),
    );
    await expect(refresh()).resolves.toBeUndefined();
  });

  it("throws on 401", async () => {
    mockFetch(401, {});
    await expect(refresh()).rejects.toThrow("Session refresh failed");
  });
});
