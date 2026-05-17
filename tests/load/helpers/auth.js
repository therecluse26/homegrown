import http from 'k6/http';
import { check, fail } from 'k6';
import { KRATOS_URL, BASE_URL, TEST_EMAIL, TEST_PASSWORD } from './config.js';

/**
 * Authenticate via Kratos native API flow and return a session token + student ID.
 *
 * Uses the Kratos API flow (not browser flow): init → submit credentials → session_token.
 * The backend auth middleware accepts the token via X-Session-Token header, which it
 * forwards to Kratos /sessions/whoami using the X-Session-Token header (not a cookie).
 */
export function login(email = TEST_EMAIL, password = TEST_PASSWORD) {
  // Step 1: Initialise a native API login flow.
  const initRes = http.get(`${KRATOS_URL}/self-service/login/api`, {
    tags: { type: 'auth-setup' },
  });
  if (!check(initRes, { 'login flow init 200': (r) => r.status === 200 })) {
    fail(`Kratos login flow init failed: ${initRes.status} ${initRes.body}`);
  }

  const flow = initRes.json();
  const actionURL = flow.ui.action;

  // Step 2: Submit credentials to the flow action URL.
  const submitRes = http.post(
    actionURL,
    JSON.stringify({ identifier: email, password, method: 'password' }),
    { headers: { 'Content-Type': 'application/json' }, tags: { type: 'auth-setup' } },
  );
  if (!check(submitRes, { 'login credentials accepted 200': (r) => r.status === 200 })) {
    fail(`Kratos login submit failed: ${submitRes.status} ${submitRes.body}`);
  }

  const sessionToken = submitRes.json('session_token');
  if (!sessionToken) {
    fail('No session_token in Kratos login response');
  }

  return sessionToken;
}

/**
 * Fetch the first student ID for the authenticated user.
 * Called once in setup() so VUs can reuse the ID.
 */
export function fetchFirstStudentId(sessionToken) {
  const res = http.get(`${BASE_URL}/v1/students`, {
    headers: { 'X-Session-Token': sessionToken },
    tags: { type: 'auth-setup' },
  });
  if (!check(res, { 'list students 200': (r) => r.status === 200 })) {
    // Non-fatal: learning scenario will skip if no student ID available.
    return null;
  }
  const students = res.json('students') || res.json() || [];
  return Array.isArray(students) && students.length > 0 ? students[0].id : null;
}

/** Build request params with the session token header and a named tag. */
export function authParams(sessionToken, tag) {
  return {
    headers: { 'X-Session-Token': sessionToken },
    tags: { type: tag || 'read' },
  };
}
