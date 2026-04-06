import { describe, it, expect } from "vitest";
import {
  extractFieldErrors,
  extractGlobalMessages,
  extractCsrfToken,
  extractOAuthProviders,
  type KratosFlow,
  type KratosNode,
} from "./kratos";

function makeFlow(overrides: Partial<KratosFlow> = {}): KratosFlow {
  return {
    id: "test-flow",
    type: "login",
    expires_at: "2099-01-01T00:00:00Z",
    issued_at: "2026-01-01T00:00:00Z",
    request_url: "http://localhost:4933/self-service/login/browser",
    ui: {
      action: "http://localhost:4933/self-service/login",
      method: "POST",
      nodes: [],
    },
    ...overrides,
  };
}

function makeNode(overrides: Partial<KratosNode> = {}): KratosNode {
  return {
    type: "input",
    group: "default",
    attributes: {
      name: "field",
      type: "text",
      node_type: "input",
    },
    messages: [],
    meta: {},
    ...overrides,
  };
}

describe("extractFieldErrors", () => {
  it("returns empty object when no nodes have errors", () => {
    const flow = makeFlow({
      ui: { action: "", method: "POST", nodes: [makeNode()] },
    });
    expect(extractFieldErrors(flow)).toEqual({});
  });

  it("extracts first error per field", () => {
    const flow = makeFlow({
      ui: {
        action: "",
        method: "POST",
        nodes: [
          makeNode({
            attributes: { name: "identifier", type: "text", node_type: "input" },
            messages: [
              { id: 1, text: "Required", type: "error" },
              { id: 2, text: "Too short", type: "error" },
            ],
          }),
          makeNode({
            attributes: { name: "password", type: "password", node_type: "input" },
            messages: [{ id: 3, text: "Too weak", type: "error" }],
          }),
        ],
      },
    });

    const errors = extractFieldErrors(flow);
    expect(errors).toEqual({
      identifier: "Required",
      password: "Too weak",
    });
  });

  it("ignores info messages", () => {
    const flow = makeFlow({
      ui: {
        action: "",
        method: "POST",
        nodes: [
          makeNode({
            attributes: { name: "email", type: "text", node_type: "input" },
            messages: [{ id: 1, text: "Verified", type: "info" }],
          }),
        ],
      },
    });
    expect(extractFieldErrors(flow)).toEqual({});
  });
});

describe("extractGlobalMessages", () => {
  it("returns empty array when no ui.messages", () => {
    const flow = makeFlow();
    expect(extractGlobalMessages(flow)).toEqual([]);
  });

  it("returns all global messages", () => {
    const messages = [
      { id: 4000006, text: "Invalid credentials", type: "error" as const },
      { id: 1, text: "Welcome", type: "info" as const },
    ];
    const flow = makeFlow({
      ui: { action: "", method: "POST", nodes: [], messages },
    });
    expect(extractGlobalMessages(flow)).toEqual(messages);
  });
});

describe("extractCsrfToken", () => {
  it("returns empty string when no csrf node exists", () => {
    const flow = makeFlow();
    expect(extractCsrfToken(flow)).toBe("");
  });

  it("extracts csrf_token value from hidden input", () => {
    const flow = makeFlow({
      ui: {
        action: "",
        method: "POST",
        nodes: [
          makeNode({
            attributes: {
              name: "csrf_token",
              type: "hidden",
              node_type: "input",
              value: "abc123",
            },
          }),
        ],
      },
    });
    expect(extractCsrfToken(flow)).toBe("abc123");
  });

  it("returns empty string when value is not a string", () => {
    const flow = makeFlow({
      ui: {
        action: "",
        method: "POST",
        nodes: [
          makeNode({
            attributes: {
              name: "csrf_token",
              type: "hidden",
              node_type: "input",
              value: undefined,
            },
          }),
        ],
      },
    });
    expect(extractCsrfToken(flow)).toBe("");
  });
});

describe("extractOAuthProviders", () => {
  it("returns empty array when no oidc nodes", () => {
    const flow = makeFlow();
    expect(extractOAuthProviders(flow)).toEqual([]);
  });

  it("extracts provider names from oidc group nodes", () => {
    const flow = makeFlow({
      ui: {
        action: "",
        method: "POST",
        nodes: [
          makeNode({
            group: "oidc",
            attributes: {
              name: "provider",
              type: "submit",
              node_type: "input",
              value: "google",
            },
          }),
          makeNode({
            group: "oidc",
            attributes: {
              name: "provider",
              type: "submit",
              node_type: "input",
              value: "apple",
            },
          }),
          // Non-provider oidc node (different name) — should be excluded
          makeNode({
            group: "oidc",
            attributes: {
              name: "csrf_token",
              type: "hidden",
              node_type: "input",
              value: "tok",
            },
          }),
        ],
      },
    });
    expect(extractOAuthProviders(flow)).toEqual(["google", "apple"]);
  });

  it("filters out empty provider values", () => {
    const flow = makeFlow({
      ui: {
        action: "",
        method: "POST",
        nodes: [
          makeNode({
            group: "oidc",
            attributes: {
              name: "provider",
              type: "submit",
              node_type: "input",
              value: "",
            },
          }),
        ],
      },
    });
    expect(extractOAuthProviders(flow)).toEqual([]);
  });
});
